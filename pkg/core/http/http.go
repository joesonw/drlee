package http

import (
	"net/http"

	"github.com/gobuffalo/packr"
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

func Open(L *lua.LState, ec *core.ExecutionContext, runtime packr.Box, client *http.Client, listen Listen) {
	funcs := map[string]*lua.LFunction{}
	requestFuncs := openClient(L, ec, client)
	for k, v := range requestFuncs {
		funcs[k] = v
	}

	serverFuncs := openServer(L, ec, listen)
	for k, v := range serverFuncs {
		funcs[k] = v
	}

	utils.RegisterLuaModuleFunctions(L, "_http", funcs)
	src, err := runtime.FindString("http.lua")
	if err != nil {
		L.RaiseError(err.Error())
	}
	if err := utils.RegisterLuaScriptModule(L, "http", src); err != nil {
		L.RaiseError(err.Error())
	}
}
