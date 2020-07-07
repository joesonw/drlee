package global

import (
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/utils"
	uuid "github.com/satori/go.uuid"
	lua "github.com/yuin/gopher-lua"
)

func Open(L *lua.LState, ec *core.ExecutionContext, dirname string) {
	utils.RegisterGlobalFuncs(L, map[string]lua.LGFunction{
		"uuid":    lUUID,
		"bit_or":  lBitOr,
		"bit_and": lBitAnd,
		"bit_xor": lBitXor,
	})
	L.SetGlobal("__dirname__", lua.LString(dirname))
}

func lUUID(L *lua.LState) int {
	L.Push(lua.LString(uuid.NewV4().String()))
	return 1
}

func lBitOr(L *lua.LState) int {
	a := int(L.CheckNumber(1))
	b := int(L.CheckNumber(2))
	L.Push(lua.LNumber(a | b))
	return 1
}

func lBitAnd(L *lua.LState) int {
	a := int(L.CheckNumber(1))
	b := int(L.CheckNumber(2))
	L.Push(lua.LNumber(a & b))
	return 1
}

func lBitXor(L *lua.LState) int {
	a := int(L.CheckNumber(1))
	b := int(L.CheckNumber(2))
	L.Push(lua.LNumber(a ^ b))
	return 1
}
