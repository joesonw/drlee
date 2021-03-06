package http

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/object"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type Listen func(network, addr string) (net.Listener, error)

type uvCreateServer struct {
	listen Listen
	ec     *core.ExecutionContext
}

func openServer(L *lua.LState, ec *core.ExecutionContext, listen Listen) map[string]*lua.LFunction {
	ud := L.NewUserData()
	ud.Value = &uvCreateServer{
		ec:     ec,
		listen: listen,
	}
	return map[string]*lua.LFunction{"create_server": L.NewClosure(lCreateServer, ud)}
}

func lCreateServer(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	cserver, ok := ud.Value.(*uvCreateServer)
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
	listen   Listen
	addr     string
	server   *http.Server
	ec       *core.ExecutionContext
	handler  *lua.LFunction
	resource core.Resource
}

func checkServer(L *lua.LState) *lServer {
	f, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return f.(*lServer)
}

func (s *lServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ch := make(chan error, 1)
	resource := core.NewResource("http.ResponseWriter", func() {
		r.Body.Close()
		ch <- nil
	})
	s.ec.Guard(resource)
	s.ec.Call(core.Scoped(func(L *lua.LState) error {
		req := NewRequest(L, r, s.ec, resource)
		res := NewResponseWriter(L, w, ch, s.ec)
		s.ec.Call(core.ProtectedLua(s.handler, func(err error) {
			ch <- err
		}, req.Value(), res.Value()))
		return nil
	}))
	err := <-ch
	r.Body.Close()
	resource.Cancel()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error())) //nolint:errcheck
	}
}

func lServerStart(L *lua.LState) int {
	server := checkServer(L)
	server.server = &http.Server{
		Handler:  server,
		ErrorLog: nil,
	}
	server.resource = core.NewResource("*http.Server", func() {
		server.server.Close()
	})

	core.GoFunctionCallback(server.ec, L.Get(2), func(ctx context.Context) (lua.LValue, error) {
		lis, err := server.listen("tcp", server.addr)
		if err != nil {
			return lua.LNil, err
		}

		go func() {
			if err := server.server.Serve(lis); err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				core.RaiseError(server.ec, err)
			}
		}()

		return lua.LNil, nil
	})

	return 0
}

func lServerStop(L *lua.LState) int {
	server := checkServer(L)
	server.resource.Cancel()
	err := server.server.Close()
	if err != nil {
		L.Push(utils.LError(err))
	} else {
		L.Push(lua.LNil)
	}
	return 1
}
