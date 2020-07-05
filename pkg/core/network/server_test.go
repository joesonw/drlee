package network

import (
	"fmt"
	"net"
	"time"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("TCP Server", func() {
	It("should serve", func() {
		done := make(chan struct{}, 1)
		test.Async(`
			local network = require "network"
			local server = network.create_server("tcp", "localhost:0", function (conn)
				conn:read(5, function(err, body)
					assert(err == nil, "err")
					assert(body == "hello", "body")
					conn:write("world", function(err)
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
						conn, err := net.Dial(network, fmt.Sprintf("localhost:%d", addr.Port))
						Expect(err).To(BeNil())
						_, err = conn.Write([]byte("hello"))
						Expect(err).To(BeNil())
						b := make([]byte, 5)
						_, err = conn.Read(b)
						Expect(err).To(BeNil())
						Expect(string(b)).To(Equal("world"))
						done <- struct{}{}
					}()
					return lis, err
				}, nil)
			})
		<-done
	})
})
