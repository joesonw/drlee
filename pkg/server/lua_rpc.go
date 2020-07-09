package server

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	coreRPC "github.com/joesonw/drlee/pkg/core/rpc"

	uuid "github.com/satori/go.uuid"

	"github.com/hashicorp/memberlist"
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

func (s *Server) luaRPCCall(ctx context.Context, req *coreRPC.Request) ([]byte, error) {
	s.localServicesMu.RLock()
	_, hasLocal := s.localServices[req.Name]
	s.localServicesMu.RUnlock()
	var timeout time.Duration
	if exp := req.ExpiresAt; !exp.IsZero() {
		timeout = time.Until(exp)
	}
	if hasLocal {
		return s.callLuaRPCMethod(ctx, &RPCRequest{
			ID:         uuid.NewV4().String(),
			Name:       req.Name,
			Body:       req.Body,
			Timestamp:  time.Now(),
			Timeout:    timeout,
			IsLoopBack: true,
		})
	}

	s.servicesMu.RLock()
	group, ok := s.services[req.Name]
	if !ok {
		s.servicesMu.RUnlock()
		return nil, fmt.Errorf("service \"%s\" is not registered in cluster", req.Name)
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

	rpc := s.getRemoteRPC(nodeName)
	if rpc == nil {
		return nil, fmt.Errorf("service \"%s\" is not registered in cluster", req.Name)
	}

	callRes, err := rpc.RPCCall(ctx, &proto.CallRequest{
		Name:                req.Name,
		Body:                req.Body,
		NodeName:            s.members.LocalNode().Name,
		TimeoutMilliseconds: timeout.Milliseconds(),
	})
	if err != nil {
		return nil, err
	}
	ch := s.replybox.Watch(callRes.ID)
	select {
	case <-ctx.Done():
		s.replybox.Delete(callRes.ID)
		return nil, ctx.Err()
	case res := <-ch:
		if res.IsError {
			return nil, errors.New(string(res.Result))
		}
		return res.Result, nil
	}
}

func (s *Server) luaRPCBroadcast(ctx context.Context, req *coreRPC.Request) []*coreRPC.Response {
	var responseIDList []string
	var timeout time.Duration
	if exp := req.ExpiresAt; !exp.IsZero() {
		timeout = time.Until(exp)
	}

	s.localServicesMu.RLock()
	_, hasLocal := s.localServices[req.Name]
	s.localServicesMu.RUnlock()
	if hasLocal {
		ids := s.inbox.Broadcast(&RPCRequest{
			ID:         uuid.NewV4().String(),
			Name:       req.Name,
			Body:       req.Body,
			Timestamp:  time.Now(),
			Timeout:    timeout,
			IsLoopBack: true,
		})
		responseIDList = append(responseIDList, ids...)
	}

	s.servicesMu.RLock()
	group, ok := s.services[req.Name]
	s.servicesMu.RUnlock()

	if ok {
		for nodeName := range group {
			rpc := s.getRemoteRPC(nodeName)
			if rpc == nil {
				id := uuid.NewV4().String()
				responseIDList = append(responseIDList, id)
				s.replybox.Insert(&RPCResponse{
					ID:        id,
					Result:    []byte(fmt.Sprintf("service \"%s\" is not registered in cluster", req.Name)),
					Timestamp: time.Now(),
					IsError:   true,
				})
				continue
			}

			res, err := rpc.RPCBroadcast(ctx, &proto.BroadcastRequest{
				Name:                req.Name,
				Body:                req.Body,
				NodeName:            s.members.LocalNode().Name,
				TimeoutMilliseconds: timeout.Milliseconds(),
			})
			if err != nil {
				id := uuid.NewV4().String()
				responseIDList = append(responseIDList, id)
				s.replybox.Insert(&RPCResponse{
					ID:        id,
					Result:    []byte(err.Error()),
					Timestamp: time.Now(),
					IsError:   true,
				})
				continue
			}
			responseIDList = append(responseIDList, res.IDLst...)
		}
	}

	var result []*coreRPC.Response
	for _, id := range responseIDList {
		res := <-s.replybox.Watch(id)
		if res.IsError {
			result = append(result, &coreRPC.Response{
				Error: errors.New(string(res.Result)),
			})
		} else {
			result = append(result, &coreRPC.Response{
				Body: res.Result,
			})
		}
	}
	return result
}

// nolint:unparam
func (s *Server) callLuaRPCMethod(ctx context.Context, req *RPCRequest) ([]byte, error) {
	ch := s.replybox.Watch(req.ID)
	if err := s.inbox.Put(req); err != nil {
		return nil, err
	}
	res := <-ch
	if res.IsError {
		return nil, errors.New(string(res.Result))
	}
	return res.Result, nil
}
