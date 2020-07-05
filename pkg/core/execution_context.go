package core

import (
	"context"
	"time"

	lua "github.com/yuin/gopher-lua"
)

type ExecutionContext struct {
	L         *lua.LState
	config    Config
	exit      chan struct{}
	lCalls    chan LuaCall
	gCalls    chan GoCall
	guardPool *GuardPool
	closed    bool
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
	if ec.closed {
		return
	}
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

func (ec *ExecutionContext) Leak(guard Guard) {
	guard.setPool(ec.guardPool)
	ec.guardPool.Insert(guard)
}

func (ec *ExecutionContext) Close() {
	ec.exit <- struct{}{}
	for i := 0; i < ec.config.GoCallConcurrency; i++ {
		ec.exit <- struct{}{}
	}

	ec.guardPool.Close()
	ec.guardPool.ForEach(func(r Guard) {
		r.Release()
		r.setPool(nil)
		r.setNode(nil)
	})
}
