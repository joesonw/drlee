package global

import (
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/utils"
	uuid "github.com/satori/go.uuid"
	lua "github.com/yuin/gopher-lua"
)

func Open(L *lua.LState, ec *core.ExecutionContext, dirname string) {
	utils.RegisterGlobalFuncs(L, map[string]lua.LGFunction{
		"uuid": lUUID,
	})
	L.SetGlobal("__dirname__", lua.LString(dirname))
}

func lUUID(L *lua.LState) int {
	L.Push(lua.LString(uuid.NewV4().String()))
	return 1
}
