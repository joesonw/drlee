package libs

import (
	lua "github.com/yuin/gopher-lua"
)

// SafeCall L.CallByParam safe wrapper with exclusive lock
func SafeCall(L *lua.LState, f *lua.LFunction, args ...lua.LValue) error {
	lock := GetContextLock(L.Context())
	lock.Lock()
	defer lock.Unlock()
	return L.CallByParam(lua.P{
		Fn:      f,
		Protect: true,
	}, args...)
}
