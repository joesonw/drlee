package websocket

import (
	"context"
	"fmt"
	"net"

	"github.com/gobwas/ws"
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/object"
	lua "github.com/yuin/gopher-lua"
)

type Listen func(network, addr string) (net.Listener, error)

type uV struct {
	listen Listen
	ec     *core.ExecutionContext
}

func openServer(L *lua.LState, ec *core.ExecutionContext, listen Listen) map[string]*lua.LFunction {
	ud := L.NewUserData()
	ud.Value = &uV{
		ec:     ec,
		listen: listen,
	}
	return map[string]*lua.LFunction{"create_server": L.NewClosure(lCreateServer, ud)}
}

func lCreateServer(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	cserver, ok := ud.Value.(*uV)
	if !ok {
		L.RaiseError("createServer expected")
	}

	addr := L.CheckString(1)
	handler := L.CheckFunction(2)

	s := &lServer{
		handler: handler,
		addr:    addr,
		listen:  cserver.listen,
		ec:      cserver.ec,
	}

	L.Push(object.NewProtected(L, serverFuncs, map[string]lua.LValue{}, s).Value())
	return 1
}

var serverFuncs = map[string]lua.LGFunction{
	"start": lServerStart,
	"stop":  lServerStop,
}

type lServer struct {
	listen  Listen
	addr    string
	ec      *core.ExecutionContext
	handler *lua.LFunction
	exit    chan struct{}
}

func checkServer(L *lua.LState) *lServer {
	f, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return f.(*lServer)
}

func lServerStart(L *lua.LState) int {
	server := checkServer(L)

	upgrader := ws.Upgrader{}
	core.GoFunctionCallback(server.ec, L.Get(2), func(ctx context.Context) (lua.LValue, error) {
		lis, err := server.listen("tcp", server.addr)
		if err != nil {
			return lua.LNil, err
		}
		go func() {
			for {
				select {
				case <-server.exit:
					return
				default:
				}
				conn, err := lis.Accept()
				if err != nil {
					core.RaiseError(server.ec, fmt.Errorf("unable to accept connection: %w", err))
					return
				}

				_, err = upgrader.Upgrade(conn)
				if err != nil {
					core.RaiseError(server.ec, fmt.Errorf("unable to upgrade connection: %w", err))
					return
				}

				c := newConn(L, server.ec, conn, ws.StateServerSide)
				server.ec.Call(core.Lua(server.handler, c.Value()))
			}
		}()

		return lua.LNil, nil
	})

	return 0
}

func lServerStop(L *lua.LState) int {
	server := checkServer(L)
	server.exit <- struct{}{}
	return 0
}
