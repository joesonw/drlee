package server

import (
	"context"
	"fmt"
	"time"

	coreRPC "github.com/joesonw/drlee/pkg/core/rpc"
	"github.com/joesonw/drlee/pkg/utils"
	"go.uber.org/zap"
)

type luaRPCEnv struct {
	server        *Server
	inboxConsumer <-chan *coreRPC.Request
	logger        *zap.Logger
}

func (env *luaRPCEnv) Register(name string) {
	env.server.localServicesMu.Lock()
	env.server.localServices[name] = 1
	env.server.localServicesMu.Unlock()
}
func (env *luaRPCEnv) Call(ctx context.Context, req *coreRPC.Request, cb func(*coreRPC.Response)) {
	go func() {
		body, err := env.server.luaRPCCall(ctx, req.Name, req.Body)
		cb(&coreRPC.Response{
			Body:  body,
			Error: err,
		})
	}()
}
func (env *luaRPCEnv) Broadcast(ctx context.Context, req *coreRPC.Request, cb func([]*coreRPC.Response)) {
	go func() {
		list := env.server.luaRPCBroadcast(ctx, req.Name, req.Body)
		cb(list)
	}()
}
func (env *luaRPCEnv) Reply(id, nodeName string, isLoopBack bool, res *coreRPC.Response) {
	r := &RPCResponse{
		ID:        id,
		Timestamp: time.Now(),
		NodeName:  nodeName,
	}
	if res.Error != nil {
		r.IsError = true
		r.Result = []byte(res.Error.Error())
	} else {
		r.Result = res.Body
	}

	if isLoopBack {
		env.server.replybox.Insert(r)
		return
	}

	b, err := utils.MarshalGOB(&r)
	if err != nil {
		env.logger.Fatal("unable to marshal GOB", zap.Error(err))
		return
	}

	if err := env.server.outboxQueue.Put(b); err != nil {
		env.logger.Fatal("unable to put outbox queue", zap.Error(err))
	}
}
func (env *luaRPCEnv) ReadChan() <-chan *coreRPC.Request {
	return env.inboxConsumer
}

func (env *luaRPCEnv) Start() {
	for name, weight := range env.server.localServices {
		nodeName := env.server.members.LocalNode().Name
		env.server.broadcasts.QueueBroadcast(&RegistryBroadcast{
			NodeName:  nodeName,
			Timestamp: time.Now(),
			Name:      name,
			Weight:    weight,
		})
		env.logger.Info(fmt.Sprintf("broadcasted service \"%s\"", name))
	}

	env.logger.Info("lua rpc started")
}

func (env *luaRPCEnv) Build() *coreRPC.Env {
	return &coreRPC.Env{
		Register:  env.Register,
		Call:      env.Call,
		Broadcast: env.Broadcast,
		Reply:     env.Reply,
		ReadChan:  env.ReadChan,
		Start:     env.Start,
	}
}
