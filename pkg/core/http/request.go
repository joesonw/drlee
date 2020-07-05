package http

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers/params"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type uvClient struct {
	client *http.Client
	ec     *core.ExecutionContext
}

func openClient(L *lua.LState, ec *core.ExecutionContext, client *http.Client) map[string]*lua.LFunction {
	ud := L.NewUserData()
	ud.Value = &uvClient{
		client: client,
		ec:     ec,
	}
	return map[string]*lua.LFunction{"request": L.NewClosure(lRequest, ud)}
}

func lRequest(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	client, ok := ud.Value.(*uvClient)
	if !ok {
		L.RaiseError("http client expected")
	}

	method := params.String()
	url := params.String()
	options := params.Table()
	cb := params.Check(L, 1, 2, "http.request(method, url, options?, cb?)", method, url, options)

	var body io.Reader
	reqHeaders := http.Header{}

	if tb := options.Table(); tb != nil && tb != lua.LNil {
		if v := tb.RawGetString("body"); v.Type() == lua.LTString {
			body = strings.NewReader(v.String())
		}
		if v := tb.RawGetString("headers"); v.Type() == lua.LTTable {
			headers := v.(*lua.LTable)
			headers.ForEach(func(key lua.LValue, value lua.LValue) {
				reqHeaders.Add(key.String(), value.String())
			})
		}
	}

	req, err := http.NewRequest(method.String(), url.String(), body)
	if err != nil {
		client.ec.Call(core.Lua(cb, utils.LError(err)))
		return 0
	}

	req.Header = reqHeaders

	core.GoFunctionCallback(client.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		res, err := client.client.Do(req)
		if err != nil {
			return lua.LNil, err
		}

		guard := core.NewGuard("*http.Response", func() {
			res.Body.Close()
		})
		client.ec.Defer(guard)

		return NewResponse(L, res, client.ec, guard).Value(), nil
	})
	return 0
}
