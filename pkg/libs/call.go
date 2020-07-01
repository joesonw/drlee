package libs

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

func EnqueueExecutable(L *lua.LState, recovery func(err error), f *lua.LFunction, args ...lua.LValue) {
	stack := getAsyncStack(L)
	stack.Enqueue(AsyncFunction(func(L *lua.LState) {
		SafeCall(L, recovery, f, args...)
	}))
}
