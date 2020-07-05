package rpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/joesonw/drlee/pkg/core"
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
}

type Env struct {
	Register  func(name string)
	Call      func(ctx context.Context, req Request, cb func(Response))
	Broadcast func(ctx context.Context, req Request, cb func([]Response))
	Reply     func(id, nodeName string, isLoopBack bool, res Response)
	ReadChan  func() <-chan Request
	Start     func()
}

type uV struct {
	env      *Env
	ec       *core.ExecutionContext
	handlers map[string]*lua.LFunction
}

func Open(L *lua.LState, ec *core.ExecutionContext, env *Env) {
	ud := L.NewUserData()
	ud.Value = &uV{
		env:      env,
		ec:       ec,
		handlers: map[string]*lua.LFunction{},
	}
	utils.RegisterLuaModule(L, "rpc", funcs, ud)
}

func up(L *lua.LState) *uV {
	uv := L.CheckUserData(lua.UpvalueIndex(1))
	if fs, ok := uv.Value.(*uV); ok {
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
	uv := up(L)
	ch := uv.env.ReadChan()
	uv.env.Start()
	go func() {
		for req := range ch {
			handler, ok := uv.handlers[req.Name]
			if !ok {
				uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, Response{
					Error: fmt.Errorf("method \"%s\" is not found", req.Name),
				})
				continue
			}
			uv.ec.Call(core.Scoped(func(L *lua.LState) error {
				v, err := json.Decode(L, req.Body)
				if err != nil {
					uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, Response{
						Error: fmt.Errorf("method \"%s\" is not found", req.Name),
					})
					return nil
				}

				err = utils.CallLuaFunction(L, handler, v, L.NewFunction(func(L *lua.LState) int {
					err := L.Get(1)
					if err == nil || err == lua.LNil {
					} else {
						uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, Response{Error: errors.New(err.String())})
						return 0
					}
					val := L.Get(2)
					b, e := json.Encode(val)
					if e != nil {
						L.RaiseError(e.Error())
						uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, Response{Error: e})
					} else {
						uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, Response{Body: b})
					}
					return 0
				}))
				if err != nil {
					uv.env.Reply(req.ID, req.NodeName, req.IsLoopBack, Response{
						Error: err,
					})
					return nil
				}
				return nil
			}))
		}
	}()
	return 0
}

func lRegister(L *lua.LState) int {
	uv := up(L)
	name := L.CheckString(1)
	handler := L.CheckFunction(2)
	uv.env.Register(name)
	uv.handlers[name] = handler
	return 0
}

func lCall(L *lua.LState) int {
	uv := up(L)
	name := L.CheckString(1)
	message := L.CheckAny(2)
	reply := L.CheckFunction(3)

	body, err := json.Encode(message)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	uv.ec.Call(core.Go(func(ctx context.Context) error {
		uv.env.Call(ctx, Request{
			Name: name,
			Body: body,
		}, func(res Response) {
			uv.ec.Call(core.Scoped(func(L *lua.LState) error {
				if res.Error != nil {
					return utils.CallLuaFunction(L, reply, utils.LError(res.Error))
				}

				val, err := json.Decode(L, res.Body)
				if err != nil {
					return utils.CallLuaFunction(L, reply, utils.LError(err))
				}

				return utils.CallLuaFunction(L, reply, lua.LNil, val)
			}))
		})
		return nil
	}))
	return 0
}

func lBroadcast(L *lua.LState) int {
	uv := up(L)
	name := L.CheckString(1)
	message := L.CheckAny(2)
	reply := L.CheckFunction(3)

	body, err := json.Encode(message)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	uv.ec.Call(core.Go(func(ctx context.Context) error {
		uv.env.Broadcast(ctx, Request{
			Name: name,
			Body: body,
		}, func(list []Response) {
			uv.ec.Call(core.Scoped(func(L *lua.LState) error {
				result := L.NewTable()
				for _, res := range list {
					tb := L.NewTable()
					if res.Error != nil {
						tb.RawSetString("error", utils.LError(res.Error))
					} else {
						val, err := json.Decode(L, res.Body)
						if err != nil {
							return utils.CallLuaFunction(L, reply, utils.LError(err))
						}
						tb.RawSetString("body", val)
					}
					result.Append(tb)
				}
				return utils.CallLuaFunction(L, reply, lua.LNil, result)
			}))
		})
		return nil
	}))

	return 0
}
