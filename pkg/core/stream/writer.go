package stream

import (
	"context"
	"io"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers/params"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type uvWriter struct {
	startIndex int
	writer     io.Writer
	ec         *core.ExecutionContext
}

func NewWriter(L *lua.LState, ec *core.ExecutionContext, writer io.Writer, isMethod ...bool) *lua.LFunction {
	ud := L.NewUserData()
	r := &uvWriter{
		writer:     writer,
		ec:         ec,
		startIndex: 1,
	}
	ud.Value = r
	if len(isMethod) > 0 && isMethod[0] {
		r.startIndex = 2
	}
	return L.NewClosure(lWrite, ud)
}

func lWrite(L *lua.LState) int {
	uv := L.CheckUserData(lua.UpvalueIndex(1))
	writer, ok := uv.Value.(*uvWriter)
	if !ok {
		L.RaiseError("expected io.Writer")
		return 0
	}

	buf := params.String()
	cb := params.Check(L, writer.startIndex, 1, "write(buf, cb?)", buf)

	writer.ec.Call(core.Go(func(ctx context.Context) error {
		n, err := writer.writer.Write([]byte(buf.String()))
		if err != nil {
			writer.ec.Call(core.Lua(cb, utils.LError(err)))
			return nil
		}
		writer.ec.Call(core.Lua(cb, lua.LNil, lua.LNumber(n)))
		return nil
	}))

	return 0
}
