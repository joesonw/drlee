package core

import (
	"context"
	"time"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/atomic"
)

type ExecutionContext struct {
	L         *lua.LState
	config    Config
	exit      chan struct{}
	lCalls    chan LuaCall
	gCalls    chan GoCall
	guardPool *GuardPool
	started   *atomic.Bool
}

type Config struct {
	OnError           func(error)
	LuaStackSize      int
	GoStackSize       int
	GoCallConcurrency int
	GoCallTimeout     time.Duration
	LuaCallTimeout    time.Duration
}

func NewExecutionContext(L *lua.LState, config Config) *ExecutionContext {
	return &ExecutionContext{
		L:         L,
		config:    config,
		exit:      make(chan struct{}, 1),
		lCalls:    make(chan LuaCall, config.LuaStackSize),
		gCalls:    make(chan GoCall, config.GoStackSize),
		started:   atomic.NewBool(false),
		guardPool: NewGuardPool(64),
	}
}

func (ec *ExecutionContext) Start() {
	go ec.startLua()
	for i := 0; i < ec.config.GoCallConcurrency; i++ {
		go ec.startGo()
	}
}

func (ec *ExecutionContext) Call(call IsCall) {
	if g, ok := call.(GoCall); ok {
		ec.gCalls <- g
	} else if l, ok := call.(LuaCall); ok {
		ec.lCalls <- l
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
	oldCtx := ec.L.Context()
	ctx := oldCtx
	if ec.config.LuaCallTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ec.config.LuaCallTimeout)
		defer cancel()
	}
	ec.L.SetContext(ctx)
	defer ec.L.SetContext(oldCtx)
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
	ctx := context.Background()
	if ec.config.GoCallTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ec.config.LuaCallTimeout)
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

func (ec *ExecutionContext) Defer(guard Guard) {
	guard.setPool(ec.guardPool)
	ec.guardPool.Insert(guard)
}

func (ec *ExecutionContext) Close() {
	ec.guardPool.Close()
	ec.guardPool.ForEach(func(r Guard) {
		r.Release()
		r.setPool(nil)
		r.setNode(nil)
	})
	if !ec.started.CAS(true, false) {
		return
	}
	ec.exit <- struct{}{}
	for i := 0; i < ec.config.GoCallConcurrency; i++ {
		ec.exit <- struct{}{}
	}
}
