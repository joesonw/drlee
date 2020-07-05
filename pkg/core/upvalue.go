package core

import lua "github.com/yuin/gopher-lua"

func UpValue(L *lua.LState, ec *ExecutionContext) lua.LValue {
	ud := L.NewUserData()
	ud.Value = ec
	return ud
}

func Up(L *lua.LState) *ExecutionContext {
	uv := L.Get(lua.UpvalueIndex(1)).(*lua.LUserData)
	if ec, ok := uv.Value.(*ExecutionContext); ok {
		return ec
	}

	L.RaiseError("expected stacked as upvalue")
	return nil
}
