package server

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/joesonw/drlee/pkg/libs"
	"github.com/joesonw/drlee/proto"
)

var _ memberlist.Broadcast = &RegistryBroadcast{}

func (b RegistryBroadcast) Invalidates(other memberlist.Broadcast) bool {
	if o, ok := other.(*RegistryBroadcast); ok && o.NodeName == b.NodeName {
		return o.Timestamp.After(b.Timestamp)
	}
	return false
}

func (b RegistryBroadcast) Finished() {}

func (s *Server) LRPCRegister(ctx context.Context, name string) (chan *libs.RPCRequest, error) {
	s.localServicesMu.Lock()
	s.localServices[name] = 1
	s.localServicesMu.Unlock()
	return s.servicesRequestCh, nil
}

func (s *Server) StartRegistry(ctx context.Context) error {
	s.localServicesMu.RLock()
	defer s.localServicesMu.RUnlock()

	for name, weight := range s.localServices {
		nodeName := s.members.LocalNode().Name
		s.broadcasts.QueueBroadcast(&RegistryBroadcast{
			NodeName:  nodeName,
			Timestamp: time.Now(),
			Name:      name,
			Weight:    weight,
		})
		s.logger.Info(fmt.Sprintf("broadcasted service \"%s\"", name))
	}

	s.logger.Info("lua rpc started")
	return nil
}

func (s *Server) LRPCCall(ctx context.Context, timeout time.Duration, name string, body []byte) ([]byte, error) {
	if timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	s.localServicesMu.RLock()
	_, hasLocal := s.localServices[name]
	s.localServicesMu.RUnlock()
	if hasLocal {
		return s.CallRPC(ctx, name, body)
	}

	s.servicesMu.RLock()
	group, ok := s.services[name]
	if !ok {
		s.servicesMu.RUnlock()
		return nil, fmt.Errorf("service \"%s\" is not registered in cluster", name)
	}

	var totalWeight float64
	for _, weight := range group {
		totalWeight += weight
	}

	targetWeight := rand.Float64() * totalWeight
	nodeName := ""
	var currentWeight float64
	for name, weight := range group {
		currentWeight += weight
		if currentWeight >= targetWeight {
			nodeName = name
			break
		}
	}

	s.servicesMu.RUnlock()
	s.endpointMu.RLock()

	rpc := s.getRemoteRPC(nodeName)
	if rpc == nil {
		return nil, fmt.Errorf("service \"%s\" is not registered in cluster", name)
	}

	res, err := rpc.RPCCall(ctx, &proto.CallRequest{
		Name:                name,
		Body:                body,
		TimeoutMilliseconds: timeout.Milliseconds(),
		NodeName:            s.members.LocalNode().Name,
	})
	if err != nil {
		return nil, err
	}

	ch := make(chan *RPCResponse, 1)
	defer func() {
		close(ch)
	}()
	s.replyInboxMu.Lock()
	s.replyInbox[res.ID] = ch
	s.replyInboxMu.Unlock()

	select {
	case <-ctx.Done():
		s.replyInboxMu.Lock()
		delete(s.replyInbox, res.ID)
		s.replyInboxMu.Unlock()
		return nil, ctx.Err()
	case r := <-ch:
		if r.IsError {
			return nil, errors.New(string(r.Result))
		}
		return r.Result, nil
	}
}

func (s *Server) CallRPC(ctx context.Context, name string, body []byte) ([]byte, error) {
	req := libs.NewRPCRequest(name, body)
	s.servicesRequestCh <- req
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-req.Done():
		return req.Result(), req.Err()
	}
}
