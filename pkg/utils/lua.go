package utils

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

func RegisterLuaModuleObject(L *lua.LState, name string, object lua.LValue) {
	tb := L.FindTable(L.Get(lua.RegistryIndex).(*lua.LTable), "_LOADED", 1)
	mod := L.GetField(tb, name)
	if mod == lua.LNil {
		L.SetField(tb, name, object)
	}
}

func RegisterLuaModule(L *lua.LState, name string, funcs map[string]lua.LGFunction, upvalues ...lua.LValue) {
	tb := L.FindTable(L.Get(lua.RegistryIndex).(*lua.LTable), "_LOADED", 1)
	mod := L.GetField(tb, name)
	if mod.Type() != lua.LTTable {
		newmod := L.FindTable(tb.(*lua.LTable), name, len(funcs))
		if newmodtb, ok := newmod.(*lua.LTable); !ok {
			L.RaiseError("name conflict for module(%v)", name)
		} else {
			for fname, fn := range funcs {
				newmodtb.RawSetString(fname, L.NewClosure(fn, upvalues...))
			}
			L.SetField(tb, name, newmodtb)
		}
	}
}

func RegisterLuaModuleFunctions(L *lua.LState, name string, funcs map[string]*lua.LFunction) {
	tb := L.FindTable(L.Get(lua.RegistryIndex).(*lua.LTable), "_LOADED", 1)
	mod := L.GetField(tb, name)
	if mod.Type() != lua.LTTable {
		newmod := L.FindTable(tb.(*lua.LTable), name, len(funcs))
		if newmodtb, ok := newmod.(*lua.LTable); !ok {
			L.RaiseError("name conflict for module(%v)", name)
		} else {
			for fname, fn := range funcs {
				newmodtb.RawSetString(fname, fn)
			}
			L.SetField(tb, name, newmodtb)
		}
	}
}

func RegisterLuaScriptModule(L *lua.LState, name, src string) error {
	tb := L.FindTable(L.Get(lua.RegistryIndex).(*lua.LTable), "_LOADED", 1)
	mod := L.GetField(tb, name)
	if mod == lua.LNil {
		L.SetTop(0)
		if err := L.DoString(src); err != nil {
			println(err.Error())
			return fmt.Errorf("unable to register module: " + err.Error())
		}
		L.SetField(tb, name, L.Get(1))
		L.SetTop(0)
	}

	return nil
}

func RegisterGlobalFuncs(L *lua.LState, funcs map[string]lua.LGFunction, upvalues ...lua.LValue) {
	for name, f := range funcs {
		L.SetGlobal(name, L.NewClosure(f, upvalues...))
	}
}

func LError(err error) lua.LValue {
	return lua.LString(err.Error())
}

func CallLuaFunction(L *lua.LState, fn lua.LValue, args ...lua.LValue) error {
	if fn == lua.LNil || fn == nil || fn.Type() != lua.LTFunction {
		return nil
	}

	return L.CallByParam(lua.P{
		Fn:      fn,
		Protect: true,
	}, args...)
}
