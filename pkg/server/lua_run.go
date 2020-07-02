package server

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/joesonw/drlee/pkg/builtin"
	"github.com/joesonw/drlee/pkg/lib"
	"github.com/joesonw/drlee/pkg/utils"
	uuid "github.com/satori/go.uuid"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	"go.uber.org/zap"
)

type luaFile struct {
	id string
	s  *Server
	*os.File
}

func (f *luaFile) Close() error {
	err := f.File.Close()
	f.s.luaOpenedFileMu.Lock()
	delete(f.s.luaOpenedFiles, f.id)
	f.s.luaOpenedFileMu.Unlock()
	return err
}

func (s *Server) LoadLua(ctx context.Context, path string) error {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()
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
		s.bootstrapScript(ctx, filepath.Dir(path), name, i, proto)
	}

	for i := 0; i < s.config.Concurrency; i++ {
		go s.listenRPCForScript(i)
	}

	return nil
}

func (s *Server) StopLua(timeout time.Duration) error {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()
	s.isLuaReloading = true
	defer func() {
		s.isLuaReloading = false
	}()
	for _, c := range s.luaExitChannelGroup {
		c <- struct{}{}
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

	s.httpServerMappingMu.Lock()
	for _, hs := range s.httpServerMapping {
		hs.listener.Close()
	}
	s.httpServerMapping = map[string]*httpServer{}
	s.httpServerMappingMu.Unlock()
	s.luaExitChannelGroup = nil

	for _, L := range s.luaStates {
		L.Close()
	}
	s.luaStates = map[int]*lua.LState{}

	s.luaOpenedFileMu.Lock()
	for _, f := range s.luaOpenedFiles {
		f.Close()
	}
	s.luaOpenedFiles = map[string]builtin.File{}
	s.luaOpenedFileMu.Unlock()

	close(s.servicesRequestCh)
	s.servicesRequestCh = make(chan *builtin.RPCRequest, 1024)
	return nil
}

func (s *Server) listenRPCForScript(id int) {
	logger := s.logger.Named(fmt.Sprintf("lua-rpc-%d", id))
	logger.Info("lua rpc worker started")
	ch := s.inboxQueue.ReadChan()
	exit := make(chan struct{}, 1)
	s.luaExitChannelGroup = append(s.luaExitChannelGroup, exit)
	for {
		select {
		case <-exit:
			logger.Info("exit upon request")
			return
		case data := <-ch:
			{
				s.luaRunWg.Add(1)
				req := &RPCRequest{}
				if err := utils.UnmarshalGOB(data, req); err != nil {
					logger.Error("unable to unmarshal rpc request", zap.Error(err))
					s.luaRunWg.Done()
					continue
				}

				logger.Sugar().Debugf("lua received rpc %s", req.ID)

				result, err := s.CallRPC(context.TODO(), req.Name, req.Body)
				res := &RPCResponse{
					ID:        req.ID,
					Timestamp: time.Now(),
					NodeName:  req.NodeName,
				}
				if err != nil {
					res.IsError = true
					res.Result = []byte(err.Error())
				} else {
					res.Result = result
				}

				logger.Sugar().Debugf("lua finished rpc %s (ok: %v)", req.ID, !res.IsError)

				b, err := utils.MarshalGOB(res)
				if err != nil {
					logger.Error("unable to marshal rpc response", zap.Error(err))
					s.luaRunWg.Done()
					continue
				}

				if err := s.outboxQueue.Put(b); err != nil {
					logger.Fatal("unable to write to outbox queue", zap.Error(err))
				}
				s.luaRunWg.Done()
			}
		}
	}
}

func (s *Server) bootstrapScript(ctx context.Context, dir, name string, id int, proto *lua.FunctionProto) {
	logger := s.logger.Named(fmt.Sprintf("lua-worker-%s-%d", name, id))
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})
	L.SetContext(context.Background())
	exit := make(chan struct{}, 1)
	s.luaExitChannelGroup = append(s.luaExitChannelGroup, exit)

	stack := builtin.NewAsyncStack(L, 1024, func(err error) {
		L.RaiseError(err.Error())
	})

	mu := &sync.Mutex{}
	mu.Lock()
	builtin.OpenAll(L, &builtin.Env{
		RPC:           s,
		Logger:        logger,
		OpenSQL:       sql.Open,
		HttpClient:    http.DefaultClient,
		GlobalFuncs:   map[string]*builtin.GlobalFunc{},
		Globals:       map[string]lua.LValue{},
		ServerStartMU: mu,
		Dir:           dir,
		ServeHTTP:     s.RegisterLuaHTTPServer,
		AsyncStack:    stack,
		RedisNewClient: func(options *redis.Options) builtin.RedisDoable {
			return redis.NewClient(options)
		},
		OpenFile: func(name string, flag, perm int) (builtin.File, error) {
			f, err := os.OpenFile(name, flag, os.FileMode(perm))
			if err != nil {
				return nil, err
			}
			s.luaOpenedFileMu.Lock()
			s.luaOpenedFiles[name] = f
			s.luaOpenedFileMu.Unlock()
			id := uuid.NewV4().String()
			return &luaFile{
				id:   id,
				s:    s,
				File: f,
			}, nil
		},
	})
	if err := lib.Open(L); err != nil {
		logger.Fatal("unable to load lua libs", zap.Error(err))
	}

	f := &lua.LFunction{
		IsG:       false,
		Env:       L.Env,
		Proto:     proto,
		GFunction: nil,
	}

	L.Push(f)
	if err := L.PCall(0, lua.MultRet, nil); err != nil {
		logger.Fatal("unable to run lua", zap.Error(err))
	}
	stack.Start()
	go func() {
		<-exit
		stack.Stop()
	}()
	mu.Lock()
	mu.Unlock()
	s.luaStates[id] = L

}
