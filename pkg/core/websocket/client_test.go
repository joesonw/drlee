package websocket

import (
	"fmt"
	"net"

	"github.com/gobwas/ws"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("Client", func() {
	It("should read", func() {
		lis, err := net.Listen("tcp", "localhost:0")
		Expect(err).To(BeNil())
		a := lis.Addr().(*net.TCPAddr)
		go func() {
			defer GinkgoRecover()
			var conn net.Conn
			conn, err = lis.Accept()
			Expect(err).To(BeNil())

			_, err = ws.DefaultUpgrader.Upgrade(conn)
			Expect(err).To(BeNil())

			var frame ws.Frame
			frame, err = ws.ReadFrame(conn)
			Expect(err).To(BeNil())
			Expect(string(frame.Payload)).To(Equal("hello"))

			err = ws.WriteFrame(conn, ws.NewFrame(ws.OpText, false, []byte("world")))
			Expect(err).To(BeNil())
		}()

		test.Async(`
			local websocket = require "websocket"
			websocket.dial("ws://localhost:80", function(err, conn)
				assert(err == nil, "err")
				conn:write_frame("hello", function(err)
					assert(err == nil, "err")
					conn:read_frame(function(err, body)
						assert(err == nil, "err")
						assert(body == "world", "body")
						assert(err == nil, "err")
						conn:close(function(err)
							assert(err == nil, "err")
							resolve()
						end)
					end)
				end)
			end)
			`,
			func(L *lua.LState, ec *core.ExecutionContext) {
				Open(L, ec, nil, func(network, addr string) (net.Conn, error) {
					Expect(addr).To(Equal("localhost:80"))
					return net.Dial(network, fmt.Sprintf("localhost:%d", a.Port))
				})
			})
	})
})
