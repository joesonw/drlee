package libs

import lua "github.com/yuin/gopher-lua"

func Error(err error) lua.LValue {
	return lua.LString(err.Error())
}
