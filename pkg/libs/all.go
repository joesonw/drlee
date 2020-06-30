package libs

import (
	lua "github.com/yuin/gopher-lua"
)

func OpenAll(
	L *lua.LState,
	env *Env,
) {
	L.SetGlobal("start_server", L.NewFunction(func(state *lua.LState) int {
		env.ServerStartMU.Unlock()
		return 0
	}))

	OpenJSON(L)
	OpenLog(L, env)
	OpenTime(L)
	OpenSQL(L, env)
	OpenEnv(L)
	OpenRegistry(L, env)
	OpenHTTPServer(L, env)
	lua.OpenBase(L)
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenIo(L)
	lua.OpenMath(L)
	lua.OpenOs(L)
	lua.OpenPackage(L)
}
