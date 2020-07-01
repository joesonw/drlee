package libs

import (
	lua "github.com/yuin/gopher-lua"
)

// Callback js-style callback
type Callback struct {
	resolved bool
	args     []lua.LValue
	f        *lua.LFunction
}

// Execute called by callback stack handler
func (cb *Callback) Execute(L *lua.LState) {
	SafeCall(L, nil, cb.f, cb.args...)
}

func (cb *Callback) CallP(L *lua.LState, args ...lua.LValue) {
	if cb.resolved {
		return
	}
	cb.resolved = true

	stack := getAsyncStack(L)
	cb.args = args
	stack.Enqueue(cb)
}

func (cb *Callback) Call(L *lua.LState, err lua.LValue, result lua.LValue) {
	cb.CallP(L, err, result)
}

func (cb *Callback) Resolve(L *lua.LState, result lua.LValue) {
	cb.CallP(L, lua.LNil, result)
}

func (cb *Callback) Reject(L *lua.LState, err lua.LValue) {
	cb.CallP(L, err)
}

func (cb *Callback) Finish(L *lua.LState) {
	cb.CallP(L)
}

func NewCallback(cb lua.LValue) *Callback {
	if cb == lua.LNil || cb == nil || cb.Type() != lua.LTFunction {
		return &Callback{}
	}
	return &Callback{f: cb.(*lua.LFunction)}
}
