package stream

import (
	"context"
	"io"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers/params"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type uvReader struct {
	startIndex int
	reader     io.Reader
	ec         *core.ExecutionContext
}

func NewReader(L *lua.LState, ec *core.ExecutionContext, reader io.Reader, isMethod ...bool) *lua.LFunction {
	ud := L.NewUserData()
	r := &uvReader{
		reader:     reader,
		ec:         ec,
		startIndex: 1,
	}
	ud.Value = r
	if len(isMethod) > 0 && isMethod[0] {
		r.startIndex = 2
	}
	return L.NewClosure(lRead, ud)
}

func lRead(L *lua.LState) int {
	uv := L.CheckUserData(lua.UpvalueIndex(1))
	reader, ok := uv.Value.(*uvReader)
	if !ok {
		L.RaiseError("expected io.Reader")
		return 0
	}

	size := params.Number()
	cb := params.Check(L, reader.startIndex, 1, "read(size, cb?)", size)

	reader.ec.Call(core.Go(func(ctx context.Context) error {
		buf := make([]byte, size.Int())
		n, err := reader.reader.Read(buf)
		if err != nil && err != io.EOF {
			reader.ec.Call(core.Lua(cb, utils.LError(err)))
			return nil
		}
		reader.ec.Call(core.Lua(cb, lua.LNil, lua.LString(buf), lua.LNumber(n)))
		return nil
	}))

	return 0
}
