package websocket

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gobwas/ws"
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("Server", func() {
	It("should serve", func() {
		done := make(chan struct{}, 1)
		test.Async(`
			local websocket = require "websocket"
			local server = websocket.create_server("localhost:0", function (conn)
				conn:read_frame(function(err, body)
					assert(err == nil, "err")
					assert(body == "hello", "body")
					conn:write_frame("world", function(err)
						assert(err == nil, "err")
						assert(err == nil, "err")
						conn:close(function(err)
							assert(err == nil, "err")
							resolve()
						end)
					end)
				end)
			end)
			server:start(function (err)
				assert(err == nil, "err")
			end)
			`,
			func(L *lua.LState, ec *core.ExecutionContext) {
				Open(L, ec, func(network, addr string) (net.Listener, error) {
					lis, err := net.Listen(network, addr)
					go func() {
						defer GinkgoRecover()
						time.Sleep(time.Second)
						addr := lis.Addr().(*net.TCPAddr)
						var conn net.Conn
						conn, _, _, err = ws.Dial(context.TODO(), fmt.Sprintf("ws://localhost:%d", addr.Port))
						Expect(err).To(BeNil())
						err = ws.WriteFrame(conn, ws.NewFrame(ws.OpText, false, []byte("hello")))
						Expect(err).To(BeNil())
						var frame ws.Frame
						frame, err = ws.ReadFrame(conn)
						Expect(err).To(BeNil())
						Expect(string(frame.Payload)).To(Equal("world"))
						done <- struct{}{}
					}()
					return lis, err
				}, nil)
			})
		<-done
	})
})
