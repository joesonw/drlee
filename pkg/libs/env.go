package libs

import (
	"database/sql"
	"net/http"
	"os"
	"sync"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

func OpenEnv(L *lua.LState) {
	L.SetGlobal("getenv", L.NewClosure(lGetenv))
}

func lGetenv(L *lua.LState) int {
	name := L.CheckString(1)
	L.Push(lua.LString(os.Getenv(name)))
	return 1
}

type GlobalFunc struct {
	Context  interface{}
	Function lua.LGFunction
}

type OpenFile func(name string, flag, perm int) (File, error)

type Env struct {
	Logger        *zap.Logger
	ServerStartMU sync.Locker
	OpenSQL       func(driverName, dataSourceName string) (*sql.DB, error)
	Dir           string
	HttpClient    *http.Client
	GlobalFuncs   map[string]*GlobalFunc
	Globals       map[string]lua.LValue
	RPC           RPC
	ServeHTTP     ServeHTTP
	OpenFile      OpenFile
	AsyncStack    *AsyncStack
}

func (e *Env) Clone(L *lua.LState) *Env {

	globalFuncs := map[string]*GlobalFunc{}
	for k, v := range e.GlobalFuncs {
		globalFuncs[k] = v
	}

	for name, f := range e.GlobalFuncs {
		ud := L.NewUserData()
		ud.Value = f.Context
		L.SetGlobal(name, L.NewClosure(f.Function, ud))
	}

	globals := map[string]lua.LValue{}
	for key, value := range e.Globals {
		globals[key] = CloneValue(L, value)
	}

	return &Env{
		Logger:      e.Logger,
		OpenSQL:     e.OpenSQL,
		HttpClient:  e.HttpClient,
		Globals:     globals,
		GlobalFuncs: globalFuncs,
		RPC:         e.RPC,
	}
}

func (e *Env) ApplyGlobals(L *lua.LState) {
	for name, f := range e.GlobalFuncs {
		ud := L.NewUserData()
		ud.Value = f.Context
		L.SetGlobal(name, L.NewClosure(f.Function, ud))
	}
}
