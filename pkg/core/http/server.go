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

type Listen func(addr string) (net.Listener, error)

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

	s := &uvServer{
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

type uvServer struct {
	listen  Listen
	addr    string
	server  *http.Server
	ec      *core.ExecutionContext
	handler *lua.LFunction
	guard   core.Guard
}

func upServer(L *lua.LState) *uvServer {
	f, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return f.(*uvServer)
}

func (s *uvServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ch := make(chan error, 1)
	guard := core.NewGuard("http.ResponseWriter", func() {
		r.Body.Close()
		ch <- nil
	})
	s.ec.Defer(guard)
	s.ec.Call(core.Scoped(func(L *lua.LState) error {
		s.ec.Defer(guard)
		req := NewRequest(L, r, s.ec, guard)
		res := NewResponseWriter(L, w, ch, s.ec)
		s.ec.Call(core.LuaCatch(s.handler, func(err error) {
			ch <- err
		}, req.Value(), res.Value()))
		return nil
	}))
	err := <-ch
	r.Body.Close()
	guard.Cancel()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func lServerStart(L *lua.LState) int {
	server := upServer(L)
	server.server = &http.Server{
		Handler:  server,
		ErrorLog: nil,
	}
	server.guard = core.NewGuard("*http.Server", func() {
		server.server.Close()
	})

	core.GoFunctionCallback(server.ec, L.Get(2), func(ctx context.Context) (lua.LValue, error) {
		lis, err := server.listen(server.addr)
		if err != nil {
			return lua.LNil, err
		}

		go func() {
			if err := server.server.Serve(lis); err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				server.ec.Call(core.Scoped(func(L *lua.LState) error {
					return err
				}))
			}
		}()

		return lua.LNil, nil
	})

	return 0
}

func lServerStop(L *lua.LState) int {
	server := upServer(L)
	server.guard.Cancel()
	err := server.server.Close()
	if err != nil {
		L.Push(utils.LError(err))
	} else {
		L.Push(lua.LNil)
	}
	return 1
}
