package libs

import (
	"io"
	"io/ioutil"
	"net/http"

	lua "github.com/yuin/gopher-lua"
)

type ServeHTTP func(addr string) (chan *HTTPTuple, error)

type HTTPTuple struct {
	req *HTTPRequest
	res *HTTPResponse
	ch  chan error
}

func NewHTTPTuple(w http.ResponseWriter, r *http.Request) *HTTPTuple {
	headers := map[string]string{}
	for k := range r.Header {
		headers[k] = r.Header.Get(k)
	}
	ch := make(chan error, 1)
	return &HTTPTuple{
		req: &HTTPRequest{
			ReadCloser: r.Body,
			method:     r.Method,
			url:        r.RequestURI,
			headers:    headers,
		},
		res: &HTTPResponse{
			ResponseWriter: w,
			ch:             ch,
		},
		ch: ch,
	}
}

func (tuple *HTTPTuple) Done() <-chan error {
	return tuple.ch
}

type HTTPRequest struct {
	io.ReadCloser
	method  string
	url     string
	headers map[string]string
}

type HTTPResponse struct {
	http.ResponseWriter
	statusCode int
	ch         chan error
}

func (res *HTTPResponse) GoObjectGet(key lua.LValue) (lua.LValue, bool) {
	if key.String() == "statusCode" {
		return lua.LNumber(res.statusCode), true
	}
	return lua.LNil, false
}

func (res *HTTPResponse) GoObjectSet(key, value lua.LValue) bool {
	if key.String() == "statusCode" && value.Type() == lua.LTNumber {
		res.statusCode = int(value.(lua.LNumber))
		res.WriteHeader(res.statusCode)
		return true
	}
	return false
}

type lHTTPServer struct {
	serve   ServeHTTP
	handler *lua.LFunction
}

func upHTTPServer(L *lua.LState) *lHTTPServer {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	if s, ok := ud.Value.(*lHTTPServer); ok {
		return s
	}
	L.RaiseError("expected http server")
	return nil
}

func lStartHTTPServer(L *lua.LState) int {
	server := upHTTPServer(L)
	addr := L.CheckString(1)
	handler := L.CheckFunction(2)

	ch, err := server.serve(addr)
	if err != nil {
		L.RaiseError("unable to start http server " + err.Error())
	} else {
		go lServeHTTPServer(L, handler, ch)
	}

	return 0
}

func lServeHTTPServer(L *lua.LState, handler *lua.LFunction, ch chan *HTTPTuple) {
	for tuple := range ch {
		headers, _ := MarshalLValue(L, tuple.req.headers)
		reqProps := map[string]lua.LValue{
			"method":  lua.LString(tuple.req.method),
			"url":     lua.LString(tuple.req.url),
			"headers": headers,
		}
		req := NewGoObject(L, lHTTPRequestFuncs, reqProps, tuple.req, false)
		res := NewGoObject(L, lHTTPResponseFuncs, nil, tuple.res, false)

		err := CallOnStack(L, handler, req, res)
		if err != nil {
			tuple.ch <- err
		}
	}
}

var lHTTPServerFuncs = map[string]lua.LGFunction{
	"start_http_server": lStartHTTPServer,
}

func OpenHTTPServer(L *lua.LState, env *Env) {
	ud := L.NewUserData()
	ud.Value = &lHTTPServer{
		serve: env.ServeHTTP,
	}
	RegisterGlobalFuncs(L, lHTTPServerFuncs, ud)
}

var lHTTPRequestFuncs = map[string]lua.LGFunction{
	"readall": lHTTPRequestReadAll,
}

func upHTTPRequest(L *lua.LState) *HTTPRequest {
	req, err := GetValueFromGoObject(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return req.(*HTTPRequest)
}

func lHTTPRequestReadAll(L *lua.LState) int {
	req := upHTTPRequest(L)
	cb := NewCallback(L.Get(2))
	go func() {
		b, err := ioutil.ReadAll(req)
		if err := req.Close(); err != nil {
			L.RaiseError(err.Error())
		}
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Resolve(L, lua.LString(b))
		}
	}()
	return 0
}

var lHTTPResponseFuncs = map[string]lua.LGFunction{
	"write": lHTTPResponseWrite,
	"get":   lHTTPResponseGet,
	"set":   lHTTPResponseSet,
}

func upHTTPResponse(L *lua.LState) *HTTPResponse {
	res, err := GetValueFromGoObject(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return res.(*HTTPResponse)
}

func lHTTPResponseWrite(L *lua.LState) int {
	res := upHTTPResponse(L)
	body := L.Get(2)
	cb := NewCallback(L.Get(3))
	var b []byte
	if body.Type() == lua.LTString || body.Type() == lua.LTNumber || body.Type() == lua.LTBool {
		b = []byte(body.String())
	} else {
		var err error
		b, err = JSONEncode(body)
		if err != nil {
			cb.Reject(L, Error(err))
			return 0
		}
	}

	go func() {
		_, err := res.Write(b)
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Finish(L)
		}
		res.ch <- nil
	}()
	return 0
}

func lHTTPResponseGet(L *lua.LState) int {
	res := upHTTPResponse(L)
	key := L.CheckString(2)
	L.Push(lua.LString(res.Header().Get(key)))
	return 1
}

func lHTTPResponseSet(L *lua.LState) int {
	res := upHTTPResponse(L)
	key := L.CheckString(2)
	value := L.CheckString(3)
	res.Header().Set(key, value)
	return 0
}
