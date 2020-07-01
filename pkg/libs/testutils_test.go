package libs

import (
	"context"

	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func runAsyncLuaTestWithError(src string, open func(L *lua.LState), after ...func(L *lua.LState)) error {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})
	L.SetContext(context.Background())
	stack := NewAsyncStack(L, 1024)
	defer L.Close()

	stackUD := L.NewUserData()
	stackUD.Value = stack
	stack.Start()
	L.Env.RawSetString("stack", stackUD)

	lua.OpenBase(L)
	lua.OpenPackage(L)
	open(L)

	done := make(chan struct{}, 1)
	L.SetGlobal("resolve", L.NewClosure(func(L *lua.LState) int {
		done <- struct{}{}
		return 0
	}))
	err := L.DoString(src)
	<-done

	for _, a := range after {
		a(L)
	}
	stack.Stop()
	return err
}

func runAsyncLuaTest(src string, open func(L *lua.LState), after ...func(L *lua.LState)) {
	err := runAsyncLuaTestWithError(src, open, after...)
	Expect(err).To(BeNil())
}

func runSyncLuaTest(src string, open func(L *lua.LState), after ...func(L *lua.LState)) {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})
	L.SetContext(context.Background())
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
