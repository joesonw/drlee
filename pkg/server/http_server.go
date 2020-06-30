package server

import (
	"net"
	"net/http"

	"github.com/joesonw/drlee/pkg/libs"
	"go.uber.org/zap"
)

type httpServer struct {
	listener net.Listener
	ch       chan *libs.HTTPTuple
}

func (s *Server) RegisterLuaHTTPServer(addr string) (chan *libs.HTTPTuple, error) {
	s.httpServerMappingMu.Lock()
	defer s.httpServerMappingMu.Unlock()
	hs, ok := s.httpServerMapping[addr]
	if ok {
		return hs.ch, nil
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ch := make(chan *libs.HTTPTuple, 128)

	hs = &httpServer{
		listener: lis,
		ch:       ch,
	}

	closed := make(chan struct{}, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.luaRunWg.Add(1)
		defer s.luaRunWg.Done()
		tuple := libs.NewHTTPTuple(w, r)
		ch <- tuple
		var err error
		select {
		case <-closed:
			http.Error(w, "Server Closed", http.StatusInternalServerError)
			return
		case err = <-tuple.Done():
		}
		if err != nil {
			s.logger.Error(err.Error())
			http.Error(w, "Internal Error", http.StatusInternalServerError)
		}
	})

	go func() {
		if err := http.Serve(lis, handler); err != nil && !s.isLuaReloading {
			s.logger.Fatal("unable to start http server", zap.Error(err))
		}
		closed <- struct{}{}
	}()
	s.httpServerMapping[addr] = hs
	return ch, nil
}
