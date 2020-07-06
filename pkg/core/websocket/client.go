package websocket

import (
	"context"
	"net"
	"net/url"

	"github.com/gobwas/ws"
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
		L.RaiseError("websocket client expected")
	}

	addr := params.String()
	options := params.Table()
	cb := params.Check(L, 1, 1, "websocket.dial(addr, options?, cb?)", addr, options)

	u, err := url.Parse(addr.String())
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	if u.Scheme != "ws" {
		L.RaiseError("only ws:// protocol is supported")
		return 0
	}

	core.GoFunctionCallback(client.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		conn, err := client.dial("tcp", u.Host)
		if err != nil {
			return lua.LNil, err
		}
		_, _, err = ws.DefaultDialer.Upgrade(conn, u)
		if err != nil {
			return lua.LNil, err
		}
		obj := newConn(L, client.ec, conn, ws.StateClientSide)
		return obj.Value(), nil
	})
	return 0
}
