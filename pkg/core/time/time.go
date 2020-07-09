package time

import (
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

type lTime struct {
	ec  *core.ExecutionContext
	now func() time.Time
}

func checkTime(L *lua.LState) *lTime {
	uv := L.CheckUserData(lua.UpvalueIndex(1))
	if u, ok := uv.Value.(*lTime); ok {
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
	ud.Value = &lTime{
		ec:  ec,
		now: now,
	}
	utils.RegisterLuaModule(L, "time", funcs, ud)
}

func lTimeout(L *lua.LState) int {
	uv := checkTime(L)
	ms := params.Number()
	cb := params.Check(L, 1, 1, "time.timeout(ms, cb?)", ms)
	go func() {
		time.Sleep(time.Millisecond * time.Duration(ms.Int64()))
		uv.ec.Call(core.Lua(cb))
	}()
	return 0
}

func lTick(L *lua.LState) int {
	uv := checkTime(L)
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

func checkTicker(L *lua.LState) *lTicker {
	ticker, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return ticker.(*lTicker)
}

var tickerFuncs = map[string]lua.LGFunction{
	"next_tick": lTickerNextTick,
	"stop":      lTickerStop,
}

func lTickerNextTick(L *lua.LState) int {
	ticker := checkTicker(L)
	cb := L.Get(2)
	go func() {
		timestamp := <-ticker.goTicker.C
		ticker.ec.Call(core.Lua(cb, New(L, timestamp).Value()))
	}()
	return 0
}

func lTickerStop(L *lua.LState) int {
	ticker := checkTicker(L)
	ticker.goTicker.Stop()
	return 0
}

type lTimestamp struct {
	goTime time.Time
}

func checkTimestamp(L *lua.LState) *lTimestamp {
	ts, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return ts.(*lTimestamp)
}

func lNow(L *lua.LState) int {
	uv := checkTime(L)
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
	timestamp := checkTimestamp(L)
	L.Push(lua.LString(timestamp.goTime.Format(Layout)))
	return 1
}

func timestampFormat(L *lua.LState) int {
	timestamp := checkTimestamp(L)
	L.Push(lua.LString(timestamp.goTime.Format(L.CheckString(2))))
	return 1
}
