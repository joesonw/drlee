package libs

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

type HTTPRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f HTTPRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

var _ = Describe("HTTP", func() {

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

	runTest := func(data *testData) error {
		L := lua.NewState(lua.Options{
			SkipOpenLibs: true,
		})
		defer L.Close()
		L.SetContext(NewContext(context.Background()))
		lua.OpenBase(L)
		lua.OpenPackage(L)
		L.SetContext(NewContext(context.Background()))
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
		OpenHTTP(L, &Env{
			HttpClient: client,
		})
		done := make(chan struct{}, 1)
		L.SetGlobal("resolve", L.NewClosure(func(L *lua.LState) int {
			done <- struct{}{}
			return 0
		}))
		err := L.DoString(data.source)
		<-done
		return err
	}

	Context("Get", func() {
		It("should succeed", func() {
			err := runTest(&testData{
				source: `
			http_get("http://example.com", function (err, res)
				assert(err == nil, "error")
				assert(res.body == "OK", "response.body")
				assert(res.status == "200 OK", "response.status")
				assert(res.statusCode == 200, "response.statusCode")
				resolve()
			end)
		`,
				requestMethod:      http.MethodGet,
				requestURL:         "http://example.com",
				responseBody:       "OK",
				responseStatus:     "200 OK",
				responseStatusCode: http.StatusOK,
			})
			Expect(err).To(BeNil())
		})
	})

	Context("Get with error", func() {
		It("should catch error", func() {
			err := runTest(&testData{
				source: `
			http_get("http://example.com", function(err)
				assert(err == "Get \"http://example.com\": test", "response error")
				resolve()
			end)
		`,
				error: errors.New("test"),
			})
			Expect(err).To(BeNil())
		})
	})

	Context("Post", func() {
		It("should succeed", func() {
			err := runTest(&testData{
				source: `
			http_post("http://example.com", {body="test"}, function(err, res)
				assert(err == nil, "error")
				assert(res.body == "OK", "response.body")
				assert(res.status == "200 OK", "response.status")
				assert(res.statusCode == 200, "response.statusCode")
				resolve()
			end)
		`,
				requestMethod:      http.MethodPost,
				requestURL:         "http://example.com",
				requestBody:        "test",
				responseBody:       "OK",
				responseStatus:     "200 OK",
				responseStatusCode: http.StatusOK,
			})
			Expect(err).To(BeNil())
		})
	})
})
