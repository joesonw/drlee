package network

import (
	"net"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/object"
	"github.com/joesonw/drlee/pkg/core/stream"
	lua "github.com/yuin/gopher-lua"
)

type uvConn struct {
	conn net.Conn
	ec   *core.ExecutionContext
}

func newConn(L *lua.LState, ec *core.ExecutionContext, conn net.Conn) *object.Object {
	c := &uvConn{
		conn: conn,
		ec:   ec,
	}
	properties := map[string]lua.LValue{
		"remote_addr": lua.LString(conn.RemoteAddr().String()),
	}
	obj := object.NewProtected(L, connFuncs, properties, c)
	guard := core.NewGuard("net.Conn", func() {
		conn.Close()
	})
	ec.Defer(guard)
	obj.SetFunction("write", stream.NewWriter(L, ec, conn, true))
	obj.SetFunction("read", stream.NewReader(L, ec, conn, true))
	obj.SetFunction("close", stream.NewCloser(L, ec, guard, conn, true))
	return obj
}

var connFuncs = map[string]lua.LGFunction{}
