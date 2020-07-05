package network

import (
	"context"
	"net"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers/params"
	lua "github.com/yuin/gopher-lua"
)

type Dial func(network, addr string) (net.Conn, error)

type uvClient struct {
	dial Dial
	ec   *core.ExecutionContext
}

func openClient(L *lua.LState, ec *core.ExecutionContext, dial Dial) map[string]*lua.LFunction {
	ud := L.NewUserData()
	ud.Value = &uvClient{
		dial: dial,
		ec:   ec,
	}
	return map[string]*lua.LFunction{"dial": L.NewClosure(lDial, ud)}
}

func lDial(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	client, ok := ud.Value.(*uvClient)
	if !ok {
		L.RaiseError("tcp client expected")
	}

	network := params.String()
	addr := params.String()
	options := params.Table()
	cb := params.Check(L, 1, 2, "tcp.dial(network, addr, options?, cb?)", network, addr, options)

	core.GoFunctionCallback(client.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		conn, err := client.dial(network.String(), addr.String())
		if err != nil {
			return lua.LNil, err
		}
		obj := newConn(L, client.ec, conn)
		return obj.Value(), nil
	})
	return 0
}
