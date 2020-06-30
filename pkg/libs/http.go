package libs

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

var httpFuncs = map[string]lua.LGFunction{
	"http_get":    lHTTPGet,
	"http_post":   lHTTPPost,
	"http_put":    lHTTPPut,
	"http_delete": lHTTPDelete,
	"http_patch":  lHTTPPatch,
}

type lHTTP struct {
	client *http.Client
}

func upHTTP(L *lua.LState) *lHTTP {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	if v, ok := ud.Value.(*lHTTP); ok {
		return v
	}
	L.RaiseError("http expected")
	return nil
}

func OpenHTTP(L *lua.LState, env *Env) {
	client := env.HttpClient
	if client == nil {
		client = http.DefaultClient
	}
	ud := L.NewUserData()
	ud.Value = &lHTTP{
		client: client,
	}
	RegisterGlobalFuncs(L, httpFuncs, ud)
}

func lHTTPGet(L *lua.LState) int {
	return httpAux(L, http.MethodGet)
}

func lHTTPPost(L *lua.LState) int {
	return httpAux(L, http.MethodPost)
}

func lHTTPPut(L *lua.LState) int {
	return httpAux(L, http.MethodPut)
}

func lHTTPDelete(L *lua.LState) int {
	return httpAux(L, http.MethodDelete)
}

func lHTTPPatch(L *lua.LState) int {
	return httpAux(L, http.MethodPatch)
}

func httpAux(L *lua.LState, method string) int {
	h := upHTTP(L)
	if h == nil {
		return 0
	}

	if L.GetTop() == 0 {
		L.RaiseError(fmt.Sprintf("http_%s(url, options): takes at least one argument", method))
		return 0
	}

	lPath := L.CheckString(1)

	var options *lua.LTable
	var body io.Reader
	cbValue := L.Get(L.GetTop())

	if L.GetTop() > 1 {
		value := L.Get(2)
		if value.Type() == lua.LTFunction {
			cbValue = value
		} else if value.Type() == lua.LTTable {
			options = value.(*lua.LTable)
			if v := options.RawGetString("body"); v.Type() == lua.LTString {
				body = strings.NewReader(v.String())
			}
		}
	}

	req, err := http.NewRequest(method, lPath, body)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	if options != nil {
		if v := options.RawGetString("headers"); v.Type() == lua.LTTable {
			headers := v.(*lua.LTable)
			headers.ForEach(func(key lua.LValue, value lua.LValue) {
				req.Header.Add(key.String(), value.String())
			})
		}
	}

	cb := NewCallback(cbValue)
	go func() {
		res, err := h.client.Do(req)
		if err != nil {
			cb.Reject(L, Error(err))
			return
		}
		defer func() {
			if err := res.Body.Close(); err != nil {
				L.RaiseError(err.Error())
			}
		}()

		resBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			cb.Reject(L, Error(err))
			return
		}

		lRes := L.NewTable()
		lRes.RawSetString("status", lua.LString(res.Status))
		lRes.RawSetString("statusCode", lua.LNumber(res.StatusCode))
		lHeaders := L.NewTable()
		for k := range res.Header {
			lHeaders.RawSetString(k, lua.LString(res.Header.Get(k)))
		}
		lRes.RawSetString("headers", lHeaders)
		lRes.RawSetString("body", lua.LString(string(resBody)))
		cb.Resolve(L, lRes)
	}()
	return 0
}
