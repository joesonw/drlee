package core

import (
	"context"
	"time"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

type ExecutionContext struct {
	L            *lua.LState
	config       Config
	exit         chan struct{}
	lCalls       chan LuaCall
	gCalls       chan GoCall
	resourcePool *ResourcePool
	closed       bool
}

type Config struct {
	OnError           func(error)
	LuaStackSize      int
	GoStackSize       int
	GoCallConcurrency int
	GoCallTimeout     time.Duration
	IsDebug           bool
	Logger            *zap.Logger
}

func NewExecutionContext(L *lua.LState, config Config) *ExecutionContext {
	return &ExecutionContext{
		L:            L,
		config:       config,
		exit:         make(chan struct{}, 1),
		lCalls:       make(chan LuaCall, config.LuaStackSize),
		gCalls:       make(chan GoCall, config.GoStackSize),
		resourcePool: NewResourcePool(64),
	}
}

func (ec *ExecutionContext) Start() {
	go ec.startLua()
	for i := 0; i < ec.config.GoCallConcurrency; i++ {
		go ec.startGo()
	}
}

func (ec *ExecutionContext) Call(call IsCall) {
	switch c := call.(type) {
	case GoCall:
		ec.gCalls <- c
	case LuaCall:
		ec.lCalls <- c
	}
}

func (ec *ExecutionContext) startLua() {
	for {
		select {
		case <-ec.exit:
			return
		case call := <-ec.lCalls:
			ec.callLua(call)
		}
	}
}

func (ec *ExecutionContext) callLua(call LuaCall) {
	if ec.closed {
		return
	}
	err := call.Call(ec.L)
	if err != nil {
		if r, ok := call.(OnError); ok {
			r.OnError(err)
		} else {
			ec.config.OnError(err)
		}
	}
}

func (ec *ExecutionContext) startGo() {
	for {
		select {
		case <-ec.exit:
			return
		case call := <-ec.gCalls:
			ec.callGo(call)
		}
	}
}

func (ec *ExecutionContext) callGo(call GoCall) {
	if ec.closed {
		return
	}
	ctx := context.Background()
	if ec.config.GoCallTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ec.config.GoCallTimeout)
		defer cancel()
	}
	err := call.Call(ctx)
	if err != nil {
		if r, ok := call.(OnError); ok {
			r.OnError(err)
		} else {
			ec.config.OnError(err)
		}
	}
}

func (ec *ExecutionContext) Guard(resource Resource) {
	resource.setPool(ec.resourcePool)
	ec.resourcePool.Insert(resource)
}

func (ec *ExecutionContext) Close() {
	ec.exit <- struct{}{}
	for i := 0; i < ec.config.GoCallConcurrency; i++ {
		ec.exit <- struct{}{}
	}

	ec.resourcePool.Close()
	ec.resourcePool.ForEach(func(g Resource) {
		if ec.config.IsDebug {
			ec.config.Logger.Debug("releasing " + g.Name())
		}
		g.Release()
		g.setPool(nil)
		g.setNode(nil)
	})
}
