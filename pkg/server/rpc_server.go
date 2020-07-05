package server

import (
	"context"
	"time"

	"github.com/joesonw/drlee/proto"
	uuid "github.com/satori/go.uuid"
)

func (s *Server) RPCCall(ctx context.Context, req *proto.CallRequest) (res *proto.CallResponse, err error) {
	call := &RPCRequest{
		ID:        uuid.NewV4().String(),
		Name:      req.Name,
		Body:      req.Body,
		Timestamp: time.Now(),
		Timeout:   time.Millisecond * time.Duration(req.TimeoutMilliseconds),
		NodeName:  req.NodeName,
	}
	s.logger.Sugar().Debugf("received RPCCall [%s] '%s' from node (%s)", call.ID, req.NodeName, req.NodeName)
	err = s.inbox.Put(call)
	if err != nil {
		return
	}

	res = &proto.CallResponse{
		ID:            call.ID,
		TimestampNano: call.Timestamp.UnixNano(),
	}
	return
}

func (s *Server) RPCBroadcast(ctx context.Context, req *proto.BroadcastRequest) (res *proto.BroadcastResponse, err error) {
	res = &proto.BroadcastResponse{
		TimestampNano: time.Now().UnixNano(),
	}

	res.IDLst = s.inbox.Broadcast(&RPCRequest{
		Name:      req.Name,
		Body:      req.Body,
		Timestamp: time.Now(),
		Timeout:   time.Millisecond * time.Duration(req.TimeoutMilliseconds),
		NodeName:  req.NodeName,
	})

	return
}

func (s *Server) RPCReply(ctx context.Context, req *proto.ReplyRequest) (res *proto.ReplyResponse, err error) {
	res = &proto.ReplyResponse{}
	s.logger.Sugar().Debugf("received RPCReply [%s]", req.ID)

	s.replybox.Insert(&RPCResponse{
		ID:        req.ID,
		Result:    req.Result,
		Timestamp: time.Unix(0, req.TimestampNano),
		IsError:   req.IsError,
	})

	return
}
