package websocket

import (
	"context"
	"net"

	"github.com/gobwas/ws"
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/object"
	"github.com/joesonw/drlee/pkg/core/stream"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

func newConn(L *lua.LState, ec *core.ExecutionContext, conn net.Conn, state ws.State) *object.Object {
	c := &lConn{
		conn:  conn,
		ec:    ec,
		state: state,
	}
	properties := map[string]lua.LValue{
		"remote_addr": lua.LString(conn.RemoteAddr().String()),
	}
	obj := object.NewProtected(L, connFuncs, properties, c)
	resource := core.NewResource("net.Conn", func() {
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	})
	ec.Guard(resource)
	obj.SetFunction("close", stream.NewCloser(L, ec, resource, conn, true))
	return obj
}

type lConn struct {
	ec    *core.ExecutionContext
	conn  net.Conn
	state ws.State
}

func checkConn(L *lua.LState) *lConn {
	conn, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return conn.(*lConn)
}

var connFuncs = map[string]lua.LGFunction{
	"read_frame":  lConnReadFrame,
	"write_frame": lConnWriteFrame,
}

func lConnReadFrame(L *lua.LState) int {
	conn := checkConn(L)
	cb := L.Get(2)
	go func() {
		var (
			frame ws.Frame
			err   error
		)
		for {
			frame, err = ws.ReadFrame(conn.conn)
			if err != nil {
				conn.ec.Call(core.Lua(cb, utils.LError(err)))
				return
			}
			if frame.Header.OpCode == ws.OpBinary {
				_ = ws.WriteFrame(conn.conn, ws.NewTextFrame([]byte("binary frame is not supported")))
				continue
			}
			if frame.Header.OpCode != ws.OpText {
				continue
			}
			break
		}

		if frame.Header.Masked {
			ws.Cipher(frame.Payload, frame.Header.Mask, 0)
		}
		conn.ec.Call(core.Lua(cb, lua.LNil, lua.LString(frame.Payload)))
	}()
	return 0
}

func lConnWriteFrame(L *lua.LState) int {
	conn := checkConn(L)
	body := L.CheckString(2)
	cb := L.Get(3)
	core.GoFunctionCallback(conn.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		err := ws.WriteFrame(conn.conn, ws.NewTextFrame([]byte(body)))
		return lua.LNil, err
	})
	return 0
}
