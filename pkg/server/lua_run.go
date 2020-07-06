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
	coreWebsocket "github.com/joesonw/drlee/pkg/core/websocket"
	"github.com/joesonw/drlee/pkg/runtime"
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
	if err != nil {
		return err
	}
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
	s.replybox.Reset()
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

//nolint:unparam
func (s *Server) runLua(L *lua.LState, box packr.Box, dir, name string, id int, proto *lua.FunctionProto) {
	logger := s.logger.Named(fmt.Sprintf("lua-worker-%s-%d", name, id))
	exit := make(chan time.Duration, 1)
	s.luaExitChannelGroup = append(s.luaExitChannelGroup, exit)
	inboxConsumer := s.inbox.NewConsumer(id)
	ctx, cancel := context.WithCancel(L.Context())
	L.SetContext(ctx)

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
	coreHTTP.Open(L, ec, box, &http.Client{}, s.listeners.Listen)
	coreJSON.Open(L)
	coreLog.Open(L, logger)
	coreNetwork.Open(L, ec, s.listeners.Listen, net.Dial)
	coreWebsocket.Open(L, ec, s.listeners.Listen, net.Dial)
	coreRedis.Open(L, ec, func(options *redis.Options) coreRedis.Doable {
		return redis.NewClient(options)
	})
	env := luaRPCEnv{
		server:        s,
		inboxConsumer: inboxConsumer,
		logger:        logger,
	}
	coreRPC.Open(L, ec, env.Build())
	coreSQL.Open(L, ec, sql.Open)
	coreTime.Open(L, ec, time.Now)
	for _, plugin := range s.plugins {
		plugin.Open(L, ec)
	}

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
	cancel()
	ec.Close()
	L.Close()
}
