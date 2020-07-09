package http

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	"github.com/joesonw/drlee/pkg/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("HTTP Server", func() {
	It("should serve", func() {
		ch := make(chan *http.Response, 1)
		test.Async(`
			local http = require "_http"
			local server = http.create_server("localhost:0", function (req, res)
				assert(req.method == "POST", "method")
				assert(req.url == "/test", "method")
				assert(req.headers.Hello == "world", "headers")
				readall(req, function(err, body)
					req:close()
					assert(err == nil, "readall err")
					assert(body == "hello world", "body")
					res:set("result", "yes")
					assert(res:get("result") == "yes", "get header")
					res:set_status(300)
					res:write("ok", function(err)
						assert(err == nil, "write err")
						res:finish()
						resolve()
					end)
				end)
			end)
			server:start(function (err)
				assert(err == nil, "err")
			end)
			`,
			func(L *lua.LState, ec *core.ExecutionContext) {
				utils.RegisterLuaModuleFunctions(L, "_http", openServer(L, ec, func(network, addr string) (net.Listener, error) {
					lis, err := net.Listen(network, addr)
					go func() {
						defer GinkgoRecover()
						time.Sleep(time.Second)
						addr := lis.Addr().(*net.TCPAddr)
						var req *http.Request
						req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d/test", addr.Port), strings.NewReader("hello world"))
						Expect(err).To(BeNil())
						req.Header.Set("Hello", "world")
						var res *http.Response
						res, err = http.DefaultClient.Do(req)
						Expect(err).To(BeNil())
						ch <- res
					}()
					return lis, err
				}))
			})

		res := <-ch
		defer res.Body.Close()
		Expect(res.StatusCode).To(Equal(300))
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).To(BeNil())
		Expect(string(body)).To(Equal("ok"))
		Expect(res.Header.Get("result")).To(Equal("yes"))
	})

	It("should catch error", func() {
		L := lua.NewState(lua.Options{})
		L.OpenLibs()

		ec := core.NewExecutionContext(L, core.Config{
			OnError: func(err error) {
				panic(err)
			},
			LuaStackSize:      1024,
			GoStackSize:       1024,
			GoCallConcurrency: 4,
		})
		ec.Start()
		defer ec.Close()

		ch := make(chan *http.Response, 1)
		utils.RegisterLuaModuleFunctions(L, "_http", openServer(L, ec, func(network, addr string) (net.Listener, error) {
			lis, err := net.Listen(network, addr)
			go func() {
				defer GinkgoRecover()
				time.Sleep(time.Second)
				addr := lis.Addr().(*net.TCPAddr)
				var req *http.Request
				req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d", addr.Port), nil)
				Expect(err).To(BeNil())
				var res *http.Response
				res, err = http.DefaultClient.Do(req)
				Expect(err).To(BeNil())
				ch <- res
			}()
			return lis, err
		}))

		err := L.DoString(`
			local http = require "_http"
			local server = http.create_server("localhost:0", function (req, res)
				assert(req.method == "POST", "method")
			end)
			server:start(function (err)
				assert(err == nil, "err")
			end)
			`)
		Expect(err).To(BeNil())
		res := <-ch
		defer res.Body.Close()
		Expect(res.StatusCode).To(Equal(http.StatusInternalServerError))
		body, err := ioutil.ReadAll(res.Body)
		Expect(err).To(BeNil())
		Expect(string(body)).To(Equal("<string>:4: method\nstack traceback:\n\t[G]: in function 'assert'\n\t<string>:4: in main chunk\n\t[G]: ?"))
	})
})
