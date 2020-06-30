package libs

import (
	lua "github.com/yuin/gopher-lua"
)

type Callback struct {
	f *lua.LFunction
}

func (cb *Callback) CallP(L *lua.LState, args ...lua.LValue) {
	if cb.f == nil {
		return
	}

	parent := GetContextRecovery(L.Context())
	if L.IsClosed() {
		return
	}

	if err := SafeCall(L, cb.f, args...); err != nil {
		if parent != nil {
			parent(err)
		} else {
			L.RaiseError(err.Error())
		}
	}
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
