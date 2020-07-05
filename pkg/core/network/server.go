package network

import (
	"context"
	"fmt"
	"net"

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

	network := L.CheckString(1)
	addr := L.CheckString(2)
	handler := L.CheckFunction(3)

	s := &uvServer{
		handler: handler,
		network: network,
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

type uvServer struct {
	listen  Listen
	network string
	addr    string
	ec      *core.ExecutionContext
	handler *lua.LFunction
	exit    chan struct{}
}

func upServer(L *lua.LState) *uvServer {
	f, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return f.(*uvServer)
}

func lServerStart(L *lua.LState) int {
	server := upServer(L)

	core.GoFunctionCallback(server.ec, L.Get(2), func(ctx context.Context) (lua.LValue, error) {
		lis, err := server.listen(server.network, server.addr)
		if err != nil {
			return lua.LNil, err
		}

		server.exit = make(chan struct{}, 1)
		go func() {
			for {
				select {
				case <-server.exit:
					return
				default:
				}
				conn, err := lis.Accept()
				if err != nil {
					server.ec.Call(core.Scoped(func(L *lua.LState) error {
						return fmt.Errorf("unable to accept connection: %w", err)
					}))
					return
				}
				c := newConn(L, server.ec, conn)
				server.ec.Call(core.Lua(server.handler, c.Value()))
			}
		}()

		return lua.LNil, nil
	})

	return 0
}

func lServerStop(L *lua.LState) int {
	server := upServer(L)
	server.exit <- struct{}{}
	return 0
}
