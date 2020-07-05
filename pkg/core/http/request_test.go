package http

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	"github.com/joesonw/drlee/pkg/runtime"
	"github.com/joesonw/drlee/pkg/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

type HTTPRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f HTTPRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

var _ = Describe("Request", func() {

	type testData struct {
		source string

		requestMethod  string
		requestBody    string
		requestHeaders map[string]string
		requestURL     string

		responseBody       string
		responseStatus     string
		responseStatusCode int
		responseHeaders    map[string]string

		error error
	}

	runTest := func(data *testData) {
		test.Async(
			data.source,
			func(L *lua.LState, ec *core.ExecutionContext) {
				client := &http.Client{
					Transport: HTTPRoundTripperFunc(func(req *http.Request) (*http.Response, error) {
						if data.error != nil {
							return nil, data.error
						}
						Expect(data.requestURL).To(Equal(req.URL.String()))
						Expect(data.requestMethod).To(Equal(req.Method))
						if req.Body != nil {
							body, err := ioutil.ReadAll(req.Body)
							Expect(err).To(BeNil())
							Expect(data.requestBody).To(Equal(string(body)))
						}
						for k, v := range data.requestHeaders {
							Expect(v).To(Equal(req.Header.Get(k)))
						}
						resHeader := http.Header{}
						for k, v := range data.responseHeaders {
							resHeader.Set(k, v)
						}
						return &http.Response{
							Body:       ioutil.NopCloser(strings.NewReader(data.responseBody)),
							Status:     data.responseStatus,
							StatusCode: data.responseStatusCode,
							Header:     resHeader,
						}, nil
					}),
				}
				box := runtime.New()
				utils.RegisterLuaModuleFunctions(L, "_http", openClient(L, ec, client))
				src, err := box.FindString("http.lua")
				if err != nil {
					L.RaiseError(err.Error())
				}
				utils.RegisterLuaScriptModule(L, "http", src)

			})
	}

	It("should get", func() {
		runTest(&testData{
			source: `
			local http = require "http"
			http.get("http://example.com", function (err, res)
				assert(err == nil, "error")
				assert(res.statusCode == 200, "statusCode")
				assert(res.status == "200 OK", "status")
				assert(res.headers.Hello == "world", "headers")
				readall(res, function(err, body)
					assert(body == "OK", "body")
					res:close()
					resolve()
				end)
			end)
		`,
			requestMethod:      http.MethodGet,
			requestURL:         "http://example.com",
			responseBody:       "OK",
			responseStatus:     "200 OK",
			responseStatusCode: http.StatusOK,
			responseHeaders: map[string]string{
				"hello": "world",
			},
		})
	})

	It("should catch error", func() {
		runTest(&testData{
			source: `
			local http = require "http"
			http.get("http://example.com", function(err)
				assert(err == "Get \"http://example.com\": test", "response error")
				resolve()
			end)
		`,
			error: errors.New("test"),
		})
	})

	It("should post", func() {
		runTest(&testData{
			source: `
			local http = require "http"
			http.post("http://example.com", {body="test"}, function(err, res)
				assert(err == nil, "error")
				assert(res.status == "200 OK", "status")
				assert(res.statusCode == 200, "statusCode")

				readall(res, function(err, body)
					assert(body == "OK", "body")
					res:close()
					resolve()
				end)
			end)
		`,
			requestMethod:      http.MethodPost,
			requestURL:         "http://example.com",
			requestBody:        "test",
			responseBody:       "OK",
			responseStatus:     "200 OK",
			responseStatusCode: http.StatusOK,
		})
	})
})
