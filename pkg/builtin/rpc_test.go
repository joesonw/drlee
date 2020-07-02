package builtin

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

type testRPC struct {
	methods   map[string]bool
	ch        chan *RPCRequest
	call      func(name string, body []byte) ([]byte, error)
	broadcast func(name string, body []byte) []RPCBroadcastResult
}

func (r *testRPC) LRPCRegister(ctx context.Context, name string) (chan *RPCRequest, error) {
	r.methods[name] = true
	return r.ch, nil
}

func (r *testRPC) LRPCCall(ctx context.Context, timeout time.Duration, name string, body []byte) ([]byte, error) {
	return r.call(name, body)
}

func (r *testRPC) LRPCBroadcast(ctx context.Context, timeout time.Duration, name string, body []byte) []RPCBroadcastResult {
	return r.broadcast(name, body)
}

var _ = Describe("RPC", func() {
	It("should register", func() {
		r := &testRPC{
			methods: map[string]bool{},
			ch:      make(chan *RPCRequest, 1),
		}
		req := NewRPCRequest("test", []byte("\"hello world\""))
		r.ch <- req
		RunAsyncTest(`
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

		RunAsyncTest(`
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

	It("should broadcast", func() {
		type callTuple struct {
			name string
			body []byte
		}
		callCh := make(chan callTuple, 1)
		r := &testRPC{
			broadcast: func(name string, body []byte) []RPCBroadcastResult {
				callCh <- callTuple{name, body}
				return []RPCBroadcastResult{
					{
						Body: []byte("\"ok1\""),
					},
					{
						Body: []byte("\"ok2\""),
					},
					{
						Error: errors.New("error"),
					},
				}
			},
		}

		RunAsyncTest(`
			rpc_broadcast("test", "body", function(res)
				assert(table.getn(res) == 3, "length")
				assert(res[1].body == "ok1", "1")
				assert(res[1].error == nil, "1")
				assert(res[2].body == "ok2", "2")
				assert(res[2].error == nil, "2")
				assert(res[3].error == "error", "3")
				resolve()
			end)
			`,
			func(L *lua.LState) {
				lua.OpenTable(L)
				OpenRPC(L, &Env{
					RPC: r,
				})
			})
		call := <-callCh
		Expect(call.name).To(Equal("test"))
		Expect(string(call.body)).To(Equal("\"body\""))
	})
})
