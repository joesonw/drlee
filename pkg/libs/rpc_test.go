package libs

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

type testRPC struct {
	methods map[string]bool
	ch      chan *RPCRequest
	call    func(name string, body []byte) ([]byte, error)
}

func (r *testRPC) LRPCRegister(ctx context.Context, name string) (chan *RPCRequest, error) {
	r.methods[name] = true
	return r.ch, nil
}

func (r *testRPC) LRPCCall(ctx context.Context, timeout time.Duration, name string, body []byte) ([]byte, error) {
	return r.call(name, body)
}

var _ = Describe("RPC", func() {
	It("should register", func() {
		r := &testRPC{
			methods: map[string]bool{},
			ch:      make(chan *RPCRequest, 1),
		}
		req := NewRPCRequest("test", []byte("\"hello world\""))
		r.ch <- req
		runAsyncLuaTest(`
			rpc_register("test", function(message, reply)
				assert(message == "hello world", "message")
				reply(nil, "success")
				resolve()
			end)
			`,
			func(L *lua.LState) {
				OpenRPC(L, &Env{
					RPC: r,
				})
			})

		<-req.Done()
		Expect(r.methods["test"]).To(BeTrue())
		Expect(req.Err()).To(BeNil())
		Expect(string(req.Result())).To(Equal("\"success\""))
	})

	It("should call", func() {
		type callTuple struct {
			name string
			body []byte
		}
		callCh := make(chan callTuple, 1)
		r := &testRPC{
			call: func(name string, body []byte) ([]byte, error) {
				callCh <- callTuple{name, body}
				return []byte("\"ok\""), nil
			},
		}

		runAsyncLuaTest(`
			rpc_call("test", "body", function(err, message)
				assert(err == nil, "err")
				assert(message == "ok", "message")
				resolve()
			end)
			`,
			func(L *lua.LState) {
				OpenRPC(L, &Env{
					RPC: r,
				})
			})
		call := <-callCh
		Expect(call.name).To(Equal("test"))
		Expect(string(call.body)).To(Equal("\"body\""))
	})
})
