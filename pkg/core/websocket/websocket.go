package websocket

import (
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

func Open(L *lua.LState, ec *core.ExecutionContext, listen Listen, dial Dial) {
	funcs := map[string]*lua.LFunction{}
	clientFuncs := openClient(L, ec, dial)
	for k, v := range clientFuncs {
		funcs[k] = v
	}

	serverFuncs := openServer(L, ec, listen)
	for k, v := range serverFuncs {
		funcs[k] = v
	}

	utils.RegisterLuaModuleFunctions(L, "websocket", funcs)
}
