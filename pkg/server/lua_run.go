package server

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/gobuffalo/packr"
	"github.com/joesonw/drlee/pkg/core"
	coreFS "github.com/joesonw/drlee/pkg/core/fs"
	coreHTTP "github.com/joesonw/drlee/pkg/core/http"
	coreJSON "github.com/joesonw/drlee/pkg/core/json"
	coreLog "github.com/joesonw/drlee/pkg/core/log"
	coreNetwork "github.com/joesonw/drlee/pkg/core/network"
	coreRedis "github.com/joesonw/drlee/pkg/core/redis"
	coreRPC "github.com/joesonw/drlee/pkg/core/rpc"
	coreSQL "github.com/joesonw/drlee/pkg/core/sql"
	coreTime "github.com/joesonw/drlee/pkg/core/time"
	"github.com/joesonw/drlee/pkg/runtime"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	"go.uber.org/zap"
)

func (s *Server) LoadLua(ctx context.Context, path string) error {
	if !s.isLuaReloading.CAS(false, true) {
		return errors.New("lua is reloading")
	}
	defer s.isLuaReloading.Store(false)
	s.luaScript = path

	name := filepath.Base(path)
	src, err := ioutil.ReadFile(path)
	chunk, err := parse.Parse(bytes.NewBuffer(src), name)
	if err != nil {
		return err
	}
	proto, err := lua.Compile(chunk, name)
	if err != nil {
		return err
	}

	for i := 0; i < s.config.Concurrency; i++ {
		L := lua.NewState(lua.Options{})
		L.SetContext(context.Background())
		box := runtime.New()
		globalSrc, err := box.FindString("global.lua")
		if err != nil {
			return err
		}
		err = L.DoString(globalSrc)
		if err != nil {
			return err
		}
		go s.runLua(L, box, filepath.Dir(path), name, i, proto)
	}

	return nil
}

func (s *Server) StopLua(timeout time.Duration) error {
	if s.isLuaReloading.Load() {
		return errors.New("lua is reloading")
	}

	for _, c := range s.luaExitChannelGroup {
		c <- timeout
	}

	wgChannel := make(chan struct{}, 1)
	go func() {
		s.luaRunWg.Wait()
		wgChannel <- struct{}{}
	}()
	timer := time.NewTimer(timeout)
	select {
	case <-timer.C:
		s.logger.Info("stop lua timeout, forcing stop")
	case <-wgChannel:
	}
	timer.Stop()

	s.listeners.Reset()
	s.inbox.Reset()
	s.luaExitChannelGroup = nil
	s.localServicesMu.RLock()
	for name := range s.localServices {
		nodeName := s.members.LocalNode().Name
		s.broadcasts.QueueBroadcast(&RegistryBroadcast{
			NodeName:  nodeName,
			Timestamp: time.Now(),
			Name:      name,
			IsDeleted: true,
		})
	}
	s.localServicesMu.RUnlock()

	return nil
}

func (s *Server) runLua(L *lua.LState, box packr.Box, dir, name string, id int, proto *lua.FunctionProto) {
	logger := s.logger.Named(fmt.Sprintf("lua-worker-%s-%d", name, id))
	exit := make(chan time.Duration, 1)
	s.luaExitChannelGroup = append(s.luaExitChannelGroup, exit)
	inboxConsumer := s.inbox.NewConsumer(id)

	ec := core.NewExecutionContext(L, core.Config{
		OnError: func(err error) {
			logger.Error("uncaught lua error", zap.Error(err))
		},
		LuaStackSize:      128,
		GoStackSize:       256,
		GoCallConcurrency: 4,
	})
	ec.Start()

	coreFS.Open(L, ec, func(name string, flag, perm int) (coreFS.File, error) {
		return os.OpenFile(name, flag, os.FileMode(perm))
	}, box)
	coreHTTP.Open(L, ec, box, &http.Client{}, func(addr string) (net.Listener, error) {
		return s.listeners.Listen("tcp", addr)
	})
	coreJSON.Open(L)
	coreLog.Open(L, logger)
	coreNetwork.Open(L, ec, func(network, addr string) (net.Listener, error) {
		return s.listeners.Listen(network, addr)
	}, net.Dial)
	coreRedis.Open(L, ec, func(options *redis.Options) coreRedis.Doable {
		return redis.NewClient(options)
	})
	coreRPC.Open(L, ec, &coreRPC.Env{
		Register: func(name string) {
			s.localServicesMu.Lock()
			s.localServices[name] = 1
			s.localServicesMu.Unlock()
		},
		Call: func(ctx context.Context, req coreRPC.Request, cb func(coreRPC.Response)) {
			go func() {
				body, err := s.luaRPCCall(ctx, req.Name, req.Body)
				cb(coreRPC.Response{
					Body:  body,
					Error: err,
				})
			}()
		},
		Broadcast: func(ctx context.Context, req coreRPC.Request, cb func([]coreRPC.Response)) {
			go func() {
				list := s.luaRPCBroadcast(ctx, req.Name, req.Body)
				cb(list)
			}()
		},
		Reply: func(id, nodeName string, isLoopBack bool, res coreRPC.Response) {
			r := RPCResponse{
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
				s.replybox.Insert(r)
				return
			}

			b, err := utils.MarshalGOB(&r)
			if err != nil {
				logger.Fatal("unable to marshal GOB", zap.Error(err))
				return
			}

			if err := s.outboxQueue.Put(b); err != nil {
				logger.Fatal("unable to put outbox queue", zap.Error(err))
			}
		},
		ReadChan: func() <-chan coreRPC.Request {
			return inboxConsumer
		},
		Start: func() {
			for name, weight := range s.localServices {
				nodeName := s.members.LocalNode().Name
				s.broadcasts.QueueBroadcast(&RegistryBroadcast{
					NodeName:  nodeName,
					Timestamp: time.Now(),
					Name:      name,
					Weight:    weight,
				})
				logger.Info(fmt.Sprintf("broadcasted service \"%s\"", name))
			}

			logger.Info("lua rpc started")
		},
	})
	coreSQL.Open(L, ec, sql.Open)
	coreTime.Open(L, ec, time.Now)
	fn := &lua.LFunction{
		IsG:       false,
		Env:       L.Env,
		Proto:     proto,
		GFunction: nil,
	}
	L.Push(fn)
	if err := L.PCall(0, lua.MultRet, nil); err != nil {
		logger.Fatal("unable to run lua", zap.Error(err))
	}

	<-exit
	ec.Close()
	L.Close()
}
