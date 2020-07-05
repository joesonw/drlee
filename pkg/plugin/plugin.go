package plugin

import (
	"github.com/joesonw/drlee/pkg/core"
	lua "github.com/yuin/gopher-lua"
)

type Interface interface {
	Open(L *lua.LState, ec *core.ExecutionContext)
}
