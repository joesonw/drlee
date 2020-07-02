package builtin

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

type testHTTPResponseWriter struct {
	io.Writer
	header     http.Header
	statusCode int
}

func (w *testHTTPResponseWriter) Header() http.Header {
	return w.header
}

func (w *testHTTPResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

var _ = Describe("HTTP Server", func() {
	It("should serve", func() {
		tuples := make(chan *HTTPTuple, 1)
		ch := make(chan error, 1)
		output := bytes.NewBuffer(nil)

		res := &testHTTPResponseWriter{
			Writer: output,
			header: http.Header{},
		}

		tuples <- &HTTPTuple{
			req: &HTTPRequest{
				ReadCloser: ioutil.NopCloser(strings.NewReader("hello body")),
				method:     "POST",
				url:        "/test",
				headers:    map[string]string{"hello": "world"},
			},
			res: &HTTPResponse{
				ResponseWriter: res,
				ch:             ch,
			},
			ch: ch,
		}

		RunAsyncTest(`
			start_http_server("0.0.0.0:80", function (req, res)
 				assert(req.method == "POST", "method")
				assert(req.url == "/test", "method")
				assert(req.headers.hello == "world", "headers")
				req:readall(function(err, body)
					assert(err == nil, "readall err")
					assert(body == "hello body", "body")
					res:set("result", "yes")
					assert(res:get("result") == "yes", "get header")
					res.statusCode = 300
					assert(res.statusCode == 300, "status")
					res:write("ok", function(err)
						assert(err == nil, "write err")
						resolve()
					end)
				end)
			end)
			`,
			func(L *lua.LState) {
				OpenHTTPServer(L, &Env{
					ServeHTTP: func(addr string) (chan *HTTPTuple, error) {
						Expect(addr).To(Equal("0.0.0.0:80"))
						return tuples, nil
					},
				})
			})
		err := <-ch
		Expect(err).To(BeNil())
		Expect(res.header.Get("result")).To(Equal("yes"))
		Expect(res.statusCode).To(Equal(300))
		Expect(output.String()).To(Equal("ok"))
	})

	It("should error", func() {
		tuples := make(chan *HTTPTuple, 1)
		ch := make(chan error, 1)
		output := bytes.NewBuffer(nil)

		res := &testHTTPResponseWriter{
			Writer: output,
			header: http.Header{},
		}

		tuples <- &HTTPTuple{
			req: &HTTPRequest{
				method: "GET",
			},
			res: &HTTPResponse{
				ResponseWriter: res,
				ch:             ch,
			},
			ch: ch,
		}

		L := lua.NewState(lua.Options{
			SkipOpenLibs: true,
		})
		L.SetContext(context.Background())
		stack := NewAsyncStack(L, 1024, nil)
		defer L.Close()

		stackUD := L.NewUserData()
		stackUD.Value = stack
		stack.Start()
		defer stack.Stop()
		L.Env.RawSetString("stack", stackUD)

		lua.OpenBase(L)
		lua.OpenPackage(L)
		OpenHTTPServer(L, &Env{
			ServeHTTP: func(addr string) (chan *HTTPTuple, error) {
				Expect(addr).To(Equal("0.0.0.0:80"))
				return tuples, nil
			},
		})

		err := L.DoString(`
			start_http_server("0.0.0.0:80", function (req, res)
				assert(req.method == "POST", "method")
			end)
			`)
		Expect(err).To(BeNil())
		err = <-ch
		Expect(err).To(Not(BeNil()))
		Expect(err.Error()).To(Equal("<string>:3: method\nstack traceback:\n\t[G]: in function 'assert'\n\t<string>:3: in main chunk\n\t[G]: ?"))
	})
})
