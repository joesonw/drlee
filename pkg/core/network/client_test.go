package network

import (
	"fmt"
	"net"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("TCP Client", func() {
	It("should read", func() {
		lis, err := net.Listen("tcp", "localhost:0")
		Expect(err).To(BeNil())
		a := lis.Addr().(*net.TCPAddr)
		go func() {
			defer GinkgoRecover()
			conn, err := lis.Accept()
			Expect(err).To(BeNil())

			b := make([]byte, 5)
			_, err = conn.Read(b)
			Expect(err).To(BeNil())
			Expect(string(b)).To(Equal("hello"))
			conn.Write([]byte("world"))
		}()

		test.Async(`
			local network = require "network"
			network.dial("tcp", "localhost:80", function(err, conn)
				conn:write("hello", function(err)
					conn:read(5, function(err, body)
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
