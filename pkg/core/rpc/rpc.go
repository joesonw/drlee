package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers/params"
	"github.com/joesonw/drlee/pkg/core/json"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type Response struct {
	Body  []byte
	Error error
}

type Request struct {
	ID         string
	NodeName   string
	IsLoopBack bool
	Name       string
	Body       []byte
	ExpiresAt  time.Time
}

type Env struct {
	Register  func(name string)
	Call      func(ctx context.Context, req *Request, cb func(*Response))
	Broadcast func(ctx context.Context, req *Request, cb func([]*Response))
	Reply     func(id, nodeName string, isLoopBack bool, res *Response)
	ReadChan  func() <-chan *Request
	Start     func()
}

type lRPC struct {
	env      *Env
	ec       *core.ExecutionContext
	handlers map[string]*lua.LFunction
}

func (uv *lRPC) handle(req *Request) {
	if req == nil {
		return
	}
	handler, ok := uv.handlers[req.Name]
	if !ok {
		uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, &Response{
			Error: fmt.Errorf("method \"%s\" is not found", req.Name),
		})
		return
	}
	uv.ec.Call(core.Scoped(func(L *lua.LState) error {
		v, err := json.Decode(L, req.Body)
		if err != nil {
			uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, &Response{
				Error: fmt.Errorf("method \"%s\" is not found", req.Name),
			})
			return nil
		}

		err = utils.CallLuaFunction(L, handler, v, L.NewFunction(func(L *lua.LState) int {
			if exp := req.ExpiresAt; !exp.IsZero() && exp.Before(time.Now()) {
				L.RaiseError(fmt.Sprintf("req \"%s\" is already timedout", req.ID))
				return 0
			}
			err := L.Get(1)
			if err != nil && err != lua.LNil {
				uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, &Response{Error: errors.New(err.String())})
				return 0
			}
			val := L.Get(2)
			b, e := json.Encode(val)
			if e != nil {
				L.RaiseError(e.Error())
				uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, &Response{Error: e})
			} else {
				uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, &Response{Body: b})
			}
			return 0
		}))
		if err != nil {
			uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, &Response{
				Error: err,
			})
			return nil
		}
		return nil
	}))
}

func Open(L *lua.LState, ec *core.ExecutionContext, env *Env) {
	ud := L.NewUserData()
	ud.Value = &lRPC{
		env:      env,
		ec:       ec,
		handlers: map[string]*lua.LFunction{},
	}
	utils.RegisterLuaModule(L, "rpc", funcs, ud)
}

func checkRPC(L *lua.LState) *lRPC {
	uv := L.CheckUserData(lua.UpvalueIndex(1))
	if fs, ok := uv.Value.(*lRPC); ok {
		return fs
	}

	L.RaiseError("expected rpc")
	return nil
}

var funcs = map[string]lua.LGFunction{
	"start":     lStart,
	"register":  lRegister,
	"call":      lCall,
	"broadcast": lBroadcast,
}

func lStart(L *lua.LState) int {
	uv := checkRPC(L)
	ch := uv.env.ReadChan()
	uv.env.Start()
	go func() {
		for req := range ch {
			uv.handle(req)
		}
	}()
	return 0
}

func lRegister(L *lua.LState) int {
	uv := checkRPC(L)
	name := L.CheckString(1)
	handler := L.CheckFunction(2)
	uv.env.Register(name)
	uv.handlers[name] = handler
	return 0
}

func lCall(L *lua.LState) int {
	uv := checkRPC(L)

	return auxCall(L, "rpc.call(name, message, options?, cb?)", func(ctx context.Context, cancel context.CancelFunc, req *Request, cb lua.LValue) {
		uv.env.Call(ctx, req, func(res *Response) {
			cancel()
			uv.ec.Call(core.Scoped(func(L *lua.LState) error {
				if res.Error != nil {
					return utils.CallLuaFunction(L, cb, utils.LError(res.Error))
				}

				val, err := json.Decode(L, res.Body)
				if err != nil {
					return utils.CallLuaFunction(L, cb, utils.LError(err))
				}

				return utils.CallLuaFunction(L, cb, lua.LNil, val)
			}))
		})
	})
}

func lBroadcast(L *lua.LState) int {
	uv := checkRPC(L)

	return auxCall(L, "rpc.broadcast(name, message, options?, cb?)", func(ctx context.Context, cancel context.CancelFunc, req *Request, cb lua.LValue) {
		uv.env.Broadcast(ctx, req, func(list []*Response) {
			cancel()
			uv.ec.Call(core.Scoped(func(L *lua.LState) error {
				result := L.NewTable()
				for _, res := range list {
					tb := L.NewTable()
					if res.Error != nil {
						tb.RawSetString("error", utils.LError(res.Error))
					} else {
						val, err := json.Decode(L, res.Body)
						if err != nil {
							return utils.CallLuaFunction(L, cb, utils.LError(err))
						}
						tb.RawSetString("body", val)
					}
					result.Append(tb)
				}
				return utils.CallLuaFunction(L, cb, lua.LNil, result)
			}))
		})
	})
}

func auxCall(L *lua.LState, funcName string, f func(ctx context.Context, cancel context.CancelFunc, req *Request, cb lua.LValue)) int {
	uv := checkRPC(L)
	name := params.String()
	message := params.Any()
	options := params.Table()
	cb := params.Check(L, 1, 2, funcName, name, message, options)

	body, err := json.Encode(message.Value())
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	uv.ec.Call(core.Go(func(ctx context.Context) error {
		var expiresAt time.Time
		var cancel context.CancelFunc = func() {}
		if tb := options.Table(); tb != nil {
			val := tb.RawGetString("timeout")
			if val.Type() == lua.LTNumber {
				timeout := time.Duration(lua.LVAsNumber(val)) * time.Millisecond
				expiresAt = time.Now().Add(timeout)
				ctx, cancel = context.WithTimeout(ctx, timeout)
			}
		}
		f(ctx, cancel, &Request{
			Name:      name.String(),
			Body:      body,
			ExpiresAt: expiresAt,
		}, cb)
		return nil
	}))

	return 0
}
