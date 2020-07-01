package libs

import (
	"context"

	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func runAsyncLuaTest(src string, open func(L *lua.LState), after ...func(L *lua.LState)) {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})

	ch := make(chan *Callback, 1024)
	exit := make(chan struct{}, 1)
	go func() {
		for c := range ch {
			c.Execute(L)
		}
	}()
	L.SetContext(NewContext(context.Background(), ch))
	defer L.Close()

	lua.OpenBase(L)
	lua.OpenPackage(L)
	open(L)

	done := make(chan struct{}, 1)
	L.SetGlobal("resolve", L.NewClosure(func(L *lua.LState) int {
		done <- struct{}{}
		return 0
	}))
	err := L.DoString(src)
	Expect(err).To(BeNil())
	<-done

	for _, a := range after {
		a(L)
	}

	exit <- struct{}{}
}

func runSyncLuaTest(src string, open func(L *lua.LState), after ...func(L *lua.LState)) {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})
	L.SetContext(NewContext(context.Background(), nil))
	defer L.Close()

	lua.OpenBase(L)
	lua.OpenPackage(L)

	open(L)

	err := L.DoString(src)
	Expect(err).To(BeNil())

	for _, a := range after {
		a(L)
	}
}
