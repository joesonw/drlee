package stream

import (
	"context"
	"io"

	"github.com/joesonw/drlee/pkg/core"
	lua "github.com/yuin/gopher-lua"
)

type uvCloser struct {
	closer     io.Closer
	ec         *core.ExecutionContext
	startIndex int
	guard      core.Guard
}

func NewCloser(L *lua.LState, ec *core.ExecutionContext, guard core.Guard, closer io.Closer, isMethod ...bool) *lua.LFunction {
	ud := L.NewUserData()
	r := &uvCloser{
		closer:     closer,
		startIndex: 1,
		ec:         ec,
		guard:      guard,
	}
	ud.Value = r
	if len(isMethod) > 0 && isMethod[0] {
		r.startIndex = 2
	}
	return L.NewClosure(lClose, ud)
}

func lClose(L *lua.LState) int {
	uv := L.CheckUserData(lua.UpvalueIndex(1))
	closer, ok := uv.Value.(*uvCloser)
	if !ok {
		L.RaiseError("expected io.Closer")
		return 0
	}

	cb := L.Get(L.GetTop())
	core.GoFunctionCallback(closer.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		err := closer.closer.Close()
		closer.guard.Cancel()
		return lua.LNil, err
	})

	return 0
}
