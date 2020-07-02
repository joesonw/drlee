package builtin

import (
	lua "github.com/yuin/gopher-lua"
)

// SafeCall L.CallByParam safe wrapper with exclusive lock
func SafeCall(L *lua.LState, recovery func(err error), f *lua.LFunction, args ...lua.LValue) {
	if f == nil {
		return
	}

	if err := L.CallByParam(lua.P{
		Fn:      f,
		Protect: true,
	}, args...); err != nil {
		if recovery != nil {
			recovery(err)
		} else {
			L.RaiseError(err.Error())
		}
	}
}

func CallOnStack(L *lua.LState, f *lua.LFunction, args ...lua.LValue) error {
	stack := getAsyncStack(L)
	ch := make(chan error, 1)
	done := false
	stack.Enqueue(AsyncFunction(func(L *lua.LState, _ func(err error)) {
		SafeCall(L, func(err error) {
			done = true
			ch <- err
		}, f, args...)
		if !done {
			ch <- nil
		}
	}))
	return <-ch
}

// Callback js-style callback
type Callback struct {
	resolved bool
	args     []lua.LValue
	f        *lua.LFunction
}

// Execute called by callback stack handler
func (cb *Callback) Execute(L *lua.LState, recovery func(err error)) {
	SafeCall(L, recovery, cb.f, cb.args...)
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
