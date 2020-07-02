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
