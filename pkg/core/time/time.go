package time

import (
	"context"
	"time"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers/params"
	"github.com/joesonw/drlee/pkg/core/object"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

const (
	Layout = "2006-01-02T15:04:05.000Z07:00"
)

type uV struct {
	ec  *core.ExecutionContext
	now func() time.Time
}

func up(L *lua.LState) *uV {
	uv := L.CheckUserData(lua.UpvalueIndex(1))
	if u, ok := uv.Value.(*uV); ok {
		return u
	}

	L.RaiseError("expected time")
	return nil
}

var funcs = map[string]lua.LGFunction{
	"now":     lNow,
	"tick":    lTick,
	"timeout": lTimeout,
}

func Open(L *lua.LState, ec *core.ExecutionContext, now func() time.Time) {
	ud := L.NewUserData()
	ud.Value = &uV{
		ec:  ec,
		now: now,
	}
	utils.RegisterLuaModule(L, "time", funcs, ud)
}

func lTimeout(L *lua.LState) int {
	uv := up(L)
	ms := params.Number()
	cb := params.Check(L, 1, 1, "time.timeout(ms, cb?)", ms)
	core.GoFunctionCallback(uv.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		time.Sleep(time.Millisecond * time.Duration(ms.Int64()))
		return lua.LNil, nil
	})
	return 0
}

func lTick(L *lua.LState) int {
	uv := up(L)
	ms := L.CheckInt64(1)
	if ms <= 0 {
		L.ArgError(1, "time.tick(ms): repeat period should be larger than 0")
		return 0
	}

	ticker := &lTicker{
		goTicker: time.NewTicker(time.Millisecond * time.Duration(ms)),
		ec:       uv.ec,
	}
	properties := map[string]lua.LValue{
		"period": lua.LNumber(time.Millisecond * time.Duration(ms)),
	}
	obj := object.NewReadOnly(L, tickerFuncs, properties, ticker)
	L.Push(obj.Value())
	return 1
}

type lTicker struct {
	ec       *core.ExecutionContext
	goTicker *time.Ticker
}

func upTicker(L *lua.LState) *lTicker {
	ticker, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return ticker.(*lTicker)
}

var tickerFuncs = map[string]lua.LGFunction{
	"nextTick": tickerNextTick,
	"stop":     tickerStop,
}

func tickerNextTick(L *lua.LState) int {
	ticker := upTicker(L)
	cb := L.Get(2)
	core.GoFunctionCallback(ticker.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		timestamp := <-ticker.goTicker.C
		return New(L, timestamp).Value(), nil
	})
	return 0
}

func tickerStop(L *lua.LState) int {
	ticker := upTicker(L)
	ticker.goTicker.Stop()
	return 0
}

type lTimestamp struct {
	goTime time.Time
}

func upTimestamp(L *lua.LState) *lTimestamp {
	ts, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return ts.(*lTimestamp)
}

func lNow(L *lua.LState) int {
	uv := up(L)
	now := uv.now()
	L.Push(New(L, now).Value())
	return 1
}

func New(L *lua.LState, t time.Time) *object.Object {
	timestamp := &lTimestamp{
		goTime: t,
	}
	properties := map[string]lua.LValue{
		"year":        lua.LNumber(t.Year()),
		"month":       lua.LNumber(t.Month()),
		"day":         lua.LNumber(t.Day()),
		"weekday":     lua.LNumber(t.Weekday()),
		"hour":        lua.LNumber(t.Hour()),
		"minute":      lua.LNumber(t.Minute()),
		"second":      lua.LNumber(t.Second()),
		"millisecond": lua.LNumber(t.Nanosecond() / 1000000),
		"milliunix":   lua.LNumber(t.UnixNano() / 1000000),
	}
	return object.New(L, timestampFuncs, properties, timestamp)
}

var timestampFuncs = map[string]lua.LGFunction{
	"__tostring": timestampToString,
	"format":     timestampFormat,
}

func timestampToString(L *lua.LState) int {
	timestamp := upTimestamp(L)
	L.Push(lua.LString(timestamp.goTime.Format(Layout)))
	return 1
}

func timestampFormat(L *lua.LState) int {
	timestamp := upTimestamp(L)
	L.Push(lua.LString(timestamp.goTime.Format(L.CheckString(2))))
	return 1
}
