package http

import (
	"net/http"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers"
	"github.com/joesonw/drlee/pkg/core/object"
	"github.com/joesonw/drlee/pkg/core/stream"
	lua "github.com/yuin/gopher-lua"
)

type uvRequest struct {
	*http.Request
	ec *core.ExecutionContext
}

func NewRequest(L *lua.LState, req *http.Request, ec *core.ExecutionContext, resource core.Resource) *object.Object {
	headers := map[string]string{}
	for k := range req.Header {
		headers[k] = req.Header.Get(k)
	}
	properties := map[string]lua.LValue{
		"headers":     helpers.MustMarshal(L, headers),
		"request_uri": lua.LString(req.RequestURI),
		"url":         lua.LString(req.URL.String()),
		"method":      lua.LString(req.Method),
	}

	ud := L.NewUserData()
	ud.Value = &uvRequest{
		Request: req,
		ec:      ec,
	}

	obj := object.NewProtected(L, map[string]lua.LGFunction{}, properties, ud)
	obj.SetFunction("read", stream.NewReader(L, ec, req.Body, true))
	obj.SetFunction("close", stream.NewCloser(L, ec, resource, req.Body, true))
	return obj
}

type lResponse struct {
	*http.Response
	ec *core.ExecutionContext
}

func NewResponse(L *lua.LState, res *http.Response, ec *core.ExecutionContext, resource core.Resource) *object.Object {
	headers := map[string]string{}
	for k := range res.Header {
		headers[k] = res.Header.Get(k)
	}
	properties := map[string]lua.LValue{
		"headers":     helpers.MustMarshal(L, headers),
		"status_code": lua.LNumber(res.StatusCode),
		"status":      lua.LString(res.Status),
	}

	ud := L.NewUserData()
	ud.Value = &lResponse{
		Response: res,
		ec:       ec,
	}

	obj := object.NewProtected(L, map[string]lua.LGFunction{}, properties, ud)
	obj.SetFunction("read", stream.NewReader(L, ec, res.Body, true))
	obj.SetFunction("close", stream.NewCloser(L, ec, resource, res.Body, true))
	return obj
}

type lResponseWriter struct {
	http.ResponseWriter
	finish chan error
	ec     *core.ExecutionContext
}

func NewResponseWriter(L *lua.LState, w http.ResponseWriter, finish chan error, ec *core.ExecutionContext) *object.Object {
	properties := map[string]lua.LValue{}

	uv := &lResponseWriter{
		ResponseWriter: w,
		ec:             ec,
		finish:         finish,
	}

	obj := object.NewProtected(L, responseWriterFuncs, properties, uv)
	obj.SetFunction("write", stream.NewWriter(L, ec, w, true))
	return obj
}

var responseWriterFuncs = map[string]lua.LGFunction{
	"set_status": lResponseWriterSetStatus,
	"set":        lResponseWriterSet,
	"get":        lResponseWriterGet,
	"finish":     lResponseWriterFinish,
}

func checkResponseWriter(L *lua.LState) *lResponseWriter {
	w, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return w.(*lResponseWriter)
}

func lResponseWriterSetStatus(L *lua.LState) int {
	w := checkResponseWriter(L)
	w.WriteHeader(int(L.CheckNumber(2)))
	return 0
}

func lResponseWriterSet(L *lua.LState) int {
	w := checkResponseWriter(L)
	w.Header().Set(L.CheckString(2), L.CheckString(3))
	return 0
}

func lResponseWriterGet(L *lua.LState) int {
	w := checkResponseWriter(L)
	L.Push(lua.LString(w.Header().Get(L.CheckString(2))))
	return 1
}

func lResponseWriterFinish(L *lua.LState) int {
	w := checkResponseWriter(L)
	w.finish <- nil
	return 0
}
