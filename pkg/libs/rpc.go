package libs

import (
	"context"
	"errors"
	"fmt"
	"time"

	lua "github.com/yuin/gopher-lua"
)

type RPCRequest struct {
	name   string
	body   []byte
	result []byte
	err    error
	done   chan struct{}
	isDone bool

	_luaDone chan struct{}
}

func NewRPCRequest(name string, body []byte) *RPCRequest {
	return &RPCRequest{
		name:     name,
		body:     body,
		done:     make(chan struct{}, 1),
		_luaDone: make(chan struct{}, 1),
	}
}

func (req *RPCRequest) Result() []byte {
	return req.result
}

func (req *RPCRequest) Err() error {
	return req.err
}

func (req *RPCRequest) Done() <-chan struct{} {
	return req.done
}

func (req *RPCRequest) resolve(result []byte) {
	if req.isDone {
		return
	}
	req.isDone = true
	req.result = result
	req.done <- struct{}{}
}

func (req *RPCRequest) reject(err error) {
	if req.isDone {
		return
	}
	req.isDone = true
	req.err = err
	req.done <- struct{}{}
}

func (req *RPCRequest) reply(L *lua.LState) int {
	defer func() {
		req._luaDone <- struct{}{}
	}()
	if L.GetTop() >= 1 {
		if err := L.Get(1); err != lua.LNil {
			req.reject(errors.New(err.String()))
			return 0
		}
	}
	if L.GetTop() >= 2 {
		b, err := JSONEncode(L.Get(2))
		if err != nil {
			req.reject(err)
		} else {
			req.resolve(b)
		}
	} else {
		req.resolve(nil)
	}

	return 0
}

type RPC interface {
	LRPCRegister(ctx context.Context, name string) (chan *RPCRequest, error)
	LRPCCall(ctx context.Context, timeout time.Duration, name string, body []byte) ([]byte, error)
}

var registryFuncs = map[string]lua.LGFunction{
	"rpc_register": lRPCRegister,
	"rpc_call":     lRPCCall,
}

type lRegistry struct {
	L        *lua.LState
	methods  map[string]*lua.LFunction
	registry RPC
}

func (r *lRegistry) Call(req *RPCRequest) {
	L := r.L
	f, ok := r.methods[req.name]
	if !ok {
		req.reject(fmt.Errorf("service \"%s\" not found", req.name))
		return
	}

	val, err := JSONDecode(L, req.body)
	if err != nil {
		req.reject(err)
		return
	}

	EnqueueExecutable(L, func(err error) {
		req.reject(err)
	}, f, val, L.NewFunction(req.reply))

	<-req._luaDone
}

func OpenRPC(L *lua.LState, env *Env) {
	ud := L.NewUserData()
	ud.Value = &lRegistry{
		L:        L,
		methods:  map[string]*lua.LFunction{},
		registry: env.RPC,
	}
	RegisterGlobalFuncs(L, registryFuncs, ud)
}

func upRegistry(L *lua.LState) *lRegistry {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	if reg, ok := ud.Value.(*lRegistry); ok {
		return reg
	}
	L.RaiseError("registry expected")
	return nil
}

func lRPCRegister(L *lua.LState) int {
	registry := upRegistry(L)
	if L.GetTop() < 2 {
		L.RaiseError("rpc_register(name, fn, cb?) requires at least two arguments")
	}
	name := L.CheckString(1)
	fn := L.CheckFunction(2)
	registry.methods[name] = fn

	cb := NewCallback(L.Get(3))

	go func() {
		ch, err := registry.registry.LRPCRegister(L.Context(), name)
		if err != nil {
			cb.Reject(L, Error(err))
			return
		} else {
			cb.Resolve(L, lua.LNil)
		}

		for req := range ch {
			registry.Call(req)
		}
	}()

	return 1
}

func lRPCCall(L *lua.LState) int {
	registry := upRegistry(L)
	if L.GetTop() < 2 {
		L.RaiseError("rpc_call(name, fn, timeout?, cb) requires at least two arguments")
	}
	ctx := L.Context()
	var timeout time.Duration
	name := L.CheckString(1)
	message := L.Get(2)
	timeoutOrCb := L.Get(3)
	cbValue := L.Get(4)
	if timeoutOrCb.Type() == lua.LTNumber {
		timeout = time.Millisecond * time.Duration(L.CheckNumber(3))
	} else if timeoutOrCb.Type() == lua.LTFunction {
		cbValue = timeoutOrCb
	}
	body, err := JSONEncode(message)
	if err != nil {
		L.RaiseError(err.Error())
	}

	cb := NewCallback(cbValue)
	go func() {
		res, err := registry.registry.LRPCCall(ctx, timeout, name, body)
		if err != nil {
			L.SetContext(ctx)
			cb.Reject(L, Error(err))
		} else {
			result, err := JSONDecode(L, res)
			if err != nil {
				cb.Reject(L, Error(err))
			} else {
				cb.Resolve(L, result)
			}
		}
	}()

	return 1
}
