package rpc

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/joesonw/drlee/pkg/core"

	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "core/rpc")
}

var _ = Describe("RPC", func() {
	It("should register", func() {
		serviceName := ""
		test.Sync(`
			local rpc = require "rpc"
			rpc.register("hello", function() end)
			`,
			func(L *lua.LState) {
				Open(L, nil, &Env{
					Register: func(name string) {
						serviceName = name
					},
				})
			})
		Expect(serviceName).To(Equal("hello"))
	})

	It("should call", func() {
		var r *Request
		test.Async(`
			local rpc = require "rpc"
			rpc.call("hello", "world", function(err, body)
				assert(err == nil, "err")
				assert(body == "ok", "body")
				resolve()
			end)
			`,
			func(L *lua.LState, ec *core.ExecutionContext) {
				Open(L, ec, &Env{
					Call: func(ctx context.Context, req Request, cb func(Response)) {
						r = &req
						cb(Response{
							Body: []byte(strconv.Quote("ok")),
						})
					},
				})
			})
		Expect(string(r.Body)).To(Equal(strconv.Quote("world")))
		Expect(r.Name).To(Equal("hello"))
	})

	It("should broadcast", func() {
		var r *Request
		test.Async(`
			local rpc = require "rpc"
			rpc.broadcast("hello", "world", function (err, list)
				assert(err == nil, "err")
				assert(table.getn(list) == 2, "list length")
				assert(list[1].body == "ok", "body")
				assert(list[2].error == "error", "error")
				resolve()
			end)
			`,
			func(L *lua.LState, ec *core.ExecutionContext) {
				Open(L, ec, &Env{
					Broadcast: func(ctx context.Context, req Request, cb func([]Response)) {
						r = &req
						cb([]Response{{
							Body: []byte(strconv.Quote("ok")),
						}, {
							Error: errors.New("error"),
						}})
					},
				})
			})
		Expect(string(r.Body)).To(Equal(strconv.Quote("world")))
		Expect(r.Name).To(Equal("hello"))
	})

	It("should start", func() {
		read := make(chan Request, 1)
		read <- Request{
			ID:   "123",
			Name: "hello",
			Body: []byte(strconv.Quote("world")),
		}
		response := make(chan Response, 1)
		test.Async(`
			local rpc = require "rpc"
			rpc.register("hello", function (message, reply)
				assert(message == "world", "message")
				reply(nil, "ok")
				resolve()
			end)
			rpc.start()
			`,
			func(L *lua.LState, ec *core.ExecutionContext) {
				Open(L, ec, &Env{
					Register: func(name string) {},
					Start: func() {
					},
					Reply: func(id, nodeName string, isLoopBack bool, res Response) {
						if id == "123" {
							response <- res
						}
					},
					ReadChan: func() <-chan Request {
						return read
					},
				})
			})
		res := <-response
		Expect(string(res.Body)).To(Equal(strconv.Quote("ok")))
	})
})
