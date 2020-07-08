package http

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/joesonw/drlee/pkg/core"
	coreHTTP "github.com/joesonw/drlee/pkg/core/http"
	coreTime "github.com/joesonw/drlee/pkg/core/time"
	"github.com/joesonw/drlee/pkg/runtime"
	lua "github.com/yuin/gopher-lua"
)

func newState(b *testing.B, listen coreHTTP.Listen, timeout bool) *lua.LState {
	L := lua.NewState()
	ec := core.NewExecutionContext(L, core.Config{
		OnError: func(err error) {
			b.Error(err)
		},
		LuaStackSize:      64,
		GoStackSize:       64,
		GoCallConcurrency: 4,
	})
	ec.Start()
	box := runtime.New()
	globalSrc, _ := box.FindString("global.lua")
	err := L.DoString(globalSrc)
	if err != nil {
		b.Error(err)
	}
	coreHTTP.Open(L, ec, box, http.DefaultClient, listen)
	coreTime.Open(L, ec, time.Now)
	if timeout {
		err = L.DoString(`
		local http = require "http"
		local time = require "time"
		
		http.create_server("", function(req, res)
			time.timeout(100, function()
				res:write("hello", function()
					res:finish()
				end)
			end)
		end):start()
		`)
	} else {
		err = L.DoString(`
		local http = require "http"
		
		http.create_server("", function(req, res)
			res:write("hello", function()
				res:finish()
			end)
		end):start()
		`)
	}
	return L
}

func runLuaTest(b *testing.B, size int, timeout bool, concurrency int) {
	var (
		addr *net.TCPAddr
		lis  net.Listener
		err  error
	)
	listen := func(network, _ string) (net.Listener, error) {
		lis, err = net.Listen("tcp", "localhost:0")
		if err != nil {
			b.Error(err)
		}
		addr = lis.Addr().(*net.TCPAddr)
		return lis, err
	}
	for i := 0; i < size; i++ {
		newState(b, listen, timeout)
	}

	time.Sleep(time.Millisecond * 100)
	defer lis.Close()
	runClient(b, addr.Port, concurrency)
}

func BenchmarkLua(b *testing.B) {
	runLuaTest(b, 1, false, 1)
}

func BenchmarkLuaParallel4(b *testing.B) {
	runLuaTest(b, 4, false, 1)
}

func BenchmarkLuaSleep(b *testing.B) {
	runLuaTest(b, 1, true, 1)
}

func BenchmarkLuaSleepParallel4(b *testing.B) {
	runLuaTest(b, 4, true, 1)
}

func BenchmarkLuaConcurrent4(b *testing.B) {
	runLuaTest(b, 1, false, 4)
}

func BenchmarkLuaParallel4Concurrent4(b *testing.B) {
	runLuaTest(b, 4, false, 4)
}

func BenchmarkLuaSleepConcurrent4(b *testing.B) {
	runLuaTest(b, 1, true, 4)
}

func BenchmarkLuaSleepParallel4Concurrent4(b *testing.B) {
	runLuaTest(b, 4, true, 4)
}
