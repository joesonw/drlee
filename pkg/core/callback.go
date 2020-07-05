package core

import (
	"context"

	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type GoCallback func(ctx context.Context) (lua.LValue, error)

func GoFunctionCallback(ec *ExecutionContext, cb lua.LValue, fn GoCallback) {
	ec.Call(Go(func(ctx context.Context) error {
		result, err := fn(ctx)
		if err != nil {
			ec.Call(Lua(cb, utils.LError(err)))
		} else {
			ec.Call(Lua(cb, lua.LNil, result))
		}
		return nil
	}))
}
