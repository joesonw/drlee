package core

import (
	"context"

	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type IsCall interface {
	IsCall()
}

type LuaCall interface {
	IsCall
	Call(L *lua.LState) error
}

type GoCall interface {
	IsCall
	Call(ctx context.Context) error
}

type OnError interface {
	OnError(error)
}

type callScoped struct {
	fn func(L *lua.LState) error
}

func (call *callScoped) Call(L *lua.LState) error {
	return call.fn(L)
}

func (callScoped) IsCall() {}

func Scoped(fn func(L *lua.LState) error) LuaCall {
	return &callScoped{
		fn: fn,
	}
}

type callLua struct {
	fn   lua.LValue
	args []lua.LValue
}

func (call *callLua) Call(L *lua.LState) error {
	return utils.CallLuaFunction(L, call.fn, call.args...)
}

func (callLua) IsCall() {}

func Lua(fn lua.LValue, args ...lua.LValue) LuaCall {
	return &callLua{
		fn:   fn,
		args: args,
	}
}

type callLuaRecoverable struct {
	fn      *lua.LFunction
	args    []lua.LValue
	onError func(error)
}

func (call *callLuaRecoverable) Call(L *lua.LState) error {
	return L.CallByParam(lua.P{
		Fn:      call.fn,
		Protect: true,
	}, call.args...)
}

func (call *callLuaRecoverable) OnError(err error) {
	call.onError(err)
}

func (callLuaRecoverable) IsCall() {}

func ProtectedLua(fn *lua.LFunction, onError func(error), args ...lua.LValue) LuaCall {
	return &callLuaRecoverable{
		fn:      fn,
		args:    args,
		onError: onError,
	}
}

type callGo struct {
	fn func(context.Context) error
}

func (callGo) IsCall() {}

func Go(fn func(context.Context) error) GoCall {
	return &callGo{
		fn: fn,
	}
}

func (call *callGo) Call(ctx context.Context) error {
	return call.fn(ctx)
}
