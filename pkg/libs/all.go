package libs

import (
	lua "github.com/yuin/gopher-lua"
)

func getAsyncStack(L *lua.LState) *AsyncStack {
	v, _ := L.Env.RawGetString("stack").(*lua.LUserData).Value.(*AsyncStack)
	return v
}

// OpenAll open all libraries
func OpenAll(
	L *lua.LState,
	env *Env,
) {
	stackUD := L.NewUserData()
	stackUD.Value = env.AsyncStack
	L.Env.RawSetString("stack", stackUD)
	L.SetGlobal("start_server", L.NewFunction(func(state *lua.LState) int {
		env.ServerStartMU.Unlock()
		return 0
	}))

	OpenJSON(L)
	OpenLog(L, env)
	OpenTime(L)
	OpenSQL(L, env)
	OpenEnv(L)
	OpenRPC(L, env)
	OpenHTTPServer(L, env)
	OpenFS(L, env)
	OpenRedis(L, env)
	lua.OpenBase(L)
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenIo(L)
	lua.OpenMath(L)
	lua.OpenOs(L)
	lua.OpenPackage(L)
}
