package server

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/joesonw/drlee/pkg/builtin"
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

func (s *Server) LRPCRegister(ctx context.Context, name string) (chan *builtin.RPCRequest, error) {
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

func (s *Server) LRPCBroadcast(ctx context.Context, timeout time.Duration, name string, body []byte) []builtin.RPCBroadcastResult {
	if timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var resultSet []builtin.RPCBroadcastResult

	s.localServicesMu.RLock()
	_, hasLocal := s.localServices[name]
	s.localServicesMu.RUnlock()
	if hasLocal {
		res, err := s.CallRPC(ctx, name, body)
		resultSet = append(resultSet, builtin.RPCBroadcastResult{
			Body:  res,
			Error: err,
		})
	}

	s.servicesMu.RLock()
	group, ok := s.services[name]
	if !ok {
		s.servicesMu.RUnlock()
		return resultSet
	}

	callMap := map[string]chan *RPCResponse{}

	for nodeName := range group {
		rpc := s.getRemoteRPC(nodeName)
		if rpc == nil {
			resultSet = append(resultSet, builtin.RPCBroadcastResult{
				Error: fmt.Errorf("service \"%s\" is not registered in cluster", name),
			})
			continue
		}

		res, err := rpc.RPCBroadcast(ctx, &proto.BroadcastRequest{
			Name:                name,
			Body:                body,
			TimeoutMilliseconds: timeout.Milliseconds(),
			NodeName:            s.members.LocalNode().Name,
		})
		if err != nil {
			resultSet = append(resultSet, builtin.RPCBroadcastResult{
				Error: err,
			})
			continue
		}

		s.replyInboxMu.Lock()
		for _, id := range res.IDLst {
			ch := make(chan *RPCResponse, 1)
			s.replyInbox[id] = ch
			callMap[id] = ch
		}
		s.replyInboxMu.Unlock()
	}

	s.servicesMu.RUnlock()
	s.endpointMu.RLock()

	done := make(chan struct{}, 1)
	go func() {
		for id, ch := range callMap {
			res := <-ch
			if res.IsError {
				resultSet = append(resultSet, builtin.RPCBroadcastResult{
					Error: errors.New(string(res.Result)),
				})
			} else {
				resultSet = append(resultSet, builtin.RPCBroadcastResult{
					Body: res.Result,
				})
			}
			delete(callMap, id)
		}
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		for id := range callMap {
			s.replyInboxMu.Lock()
			delete(s.replyInbox, id)
			s.replyInboxMu.Unlock()
		}
	case <-done:
	}
	return resultSet
}

func (s *Server) CallRPC(ctx context.Context, name string, body []byte) ([]byte, error) {
	req := builtin.NewRPCRequest(name, body)
	s.servicesRequestCh <- req
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-req.Done():
		return req.Result(), req.Err()
	}
}
