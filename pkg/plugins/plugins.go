package plugins

import (
	"context"

	"github.com/joesonw/drlee/pkg/core"
	lua "github.com/yuin/gopher-lua"
)

type Interface interface {
	Open(L *lua.LState, ec *core.ExecutionContext)
	Close(ctx context.Context) error
}
