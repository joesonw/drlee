package server

import (
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/joesonw/drlee/pkg/utils"
	"github.com/joesonw/drlee/proto"
	"go.uber.org/zap"
)

type RPCRequest struct {
	ID         string
	Name       string
	Body       []byte
	Timestamp  time.Time
	Timeout    time.Duration
	NodeName   string
	IsLoopBack bool
}

type RPCResponse struct {
	ID        string
	Result    []byte
	Timestamp time.Time
	NodeName  string
	IsError   bool
}

func init() {
	gob.Register(&RPCRequest{})
	gob.Register(&RPCResponse{})
}

func (s *Server) StartReplyWorkers() {
	concurrency := s.config.RPC.ReplyConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	for i := 0; i < concurrency; i++ {
		go s.replyWorker(i)
	}
}

func (s *Server) replyWorker(i int) {
	logger := s.logger.Named(fmt.Sprintf("reply-%d", i))
	logger.Info("reply worker started")
	ch := s.outboxQueue.ReadChan()
	for data := range ch {
		if err := s.doReply(context.TODO(), logger, data); err != nil {
			logger.Error("unable to do reply", zap.Error(err))
		}
	}
}

func (s *Server) doReply(ctx context.Context, logger *zap.Logger, data []byte) error {
	res := &RPCResponse{}
	if err := utils.UnmarshalGOB(data, res); err != nil {
		return err
	}

	logger.Sugar().Debugf("reply worker received rpc %s to node %s", res.ID, res.NodeName)
	rpc := s.getRemoteRPC(res.NodeName)
	if rpc == nil {
		logger.Warn(fmt.Sprintf("remote \"%s\": not found", res.NodeName))
		return nil
	}

	_, err := rpc.RPCReply(ctx, &proto.ReplyRequest{
		ID:            res.ID,
		Result:        res.Result,
		TimestampNano: res.Timestamp.UnixNano(),
		IsError:       res.IsError,
	})
	return err
}
