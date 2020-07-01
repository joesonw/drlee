package libs

import (
	"time"

	lua "github.com/yuin/gopher-lua"
)

const (
	lTickerClass    = "TICKER*"
	lTimestampClass = "TIMESTAMP*"
	TimeFormat      = "2006-01-02T15:04:05.000Z07:00"
)

var timeFuncs = map[string]lua.LGFunction{
	"time_now":   timeNow,
	"time_tick":  timeTick,
	"time_sleep": timeSleep,
}

func OpenTime(L *lua.LState) {
	RegisterGlobalFuncs(L, timeFuncs)
}

type lTicker struct {
	goTicker *time.Ticker
}

func upTicker(L *lua.LState) *lTicker {
	ticker, err := GetValueFromGoObject(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return ticker.(*lTicker)
}

type lTimestamp struct {
	goTime time.Time
}

func upTimestamp(L *lua.LState) *lTimestamp {
	ts, err := GetValueFromGoObject(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return ts.(*lTimestamp)
}

func timeNow(L *lua.LState) int {
	now := time.Now()
	L.Push(NewTimestamp(L, now))
	return 1
}

func timeTick(L *lua.LState) int {
	ms := L.CheckInt64(1)
	if ms <= 0 {
		L.ArgError(1, "time.tick(milliseconds): repeat period should be larger than 0")
		return 0
	}

	ticker := &lTicker{
		goTicker: time.NewTicker(time.Millisecond * time.Duration(ms)),
	}
	properties := map[string]lua.LValue{
		"period": lua.LNumber(time.Millisecond * time.Duration(ms)),
	}
	L.Push(NewGoObject(L, tickerFuncs, properties, ticker, false))
	return 1
}

func timeSleep(L *lua.LState) int {
	ms := L.CheckInt64(1)
	cb := NewCallback(L.Get(2))
	go func() {
		if ms > 0 {
			time.Sleep(time.Millisecond * time.Duration(ms))
		}
		cb.Finish(L)
	}()
	return 0
}

var tickerFuncs = map[string]lua.LGFunction{
	"__tostring": tickerToString,
	"nextTick":   tickerNextTick,
	"stop":       tickerStop,
}

func tickerToString(L *lua.LState) int {
	L.Push(lua.LString(lTickerClass))
	return 1
}

func tickerNextTick(L *lua.LState) int {
	ticker := upTicker(L)
	cb := NewCallback(L.Get(2))
	go func() {
		timestamp := <-ticker.goTicker.C
		cb.CallP(L, NewTimestamp(L, timestamp))
	}()
	return 0
}

func tickerStop(L *lua.LState) int {
	ticker := upTicker(L)
	ticker.goTicker.Stop()
	return 0
}

func NewTimestamp(L *lua.LState, t time.Time) lua.LValue {
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
	return NewGoObject(L, timestampFuncs, properties, timestamp, false)
}

var timestampFuncs = map[string]lua.LGFunction{
	"__tostring": timestampToString,
	"format":     timestampFormat,
}

func timestampToString(L *lua.LState) int {
	timestamp := upTimestamp(L)
	L.Push(lua.LString(timestamp.goTime.Format(TimeFormat)))
	return 1
}

func timestampFormat(L *lua.LState) int {
	timestamp := upTimestamp(L)
	L.Push(lua.LString(timestamp.goTime.Format(L.CheckString(2))))
	return 1
}
