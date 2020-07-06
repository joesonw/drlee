package env

import (
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/object"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type Env struct {
	NodeName string
	WorkerID int
	WorkDir  string
	Args     []string
}

func Open(L *lua.LState, ec *core.ExecutionContext, env Env) {
	args := L.NewTable()
	for _, arg := range env.Args {
		args.Append(lua.LString(arg))
	}
	obj := object.NewReadOnly(L, map[string]lua.LGFunction{}, map[string]lua.LValue{
		"node":       lua.LString(env.NodeName),
		"worker_id":  lua.LNumber(env.WorkerID),
		"worker_dir": lua.LString(env.WorkDir),
		"args":       args,
	}, &env)
	utils.RegisterLuaModuleObject(L, "env", obj.Value())
}
