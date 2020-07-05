package test

import (
	"context"
	"errors"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/runtime"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func AsyncWithError(src string, open func(L *lua.LState, ec *core.ExecutionContext), after ...func(L *lua.LState)) error {
	L := lua.NewState(lua.Options{})
	L.SetContext(context.Background())
	var executionError error
	done := make(chan struct{}, 1)
	ec := core.NewExecutionContext(L, core.Config{
		OnError: func(err error) {
			executionError = err
			done <- struct{}{}
		},
		LuaStackSize:      1024,
		GoStackSize:       1024,
		GoCallConcurrency: 4,
	})
	defer L.Close()
	L.Panic = func(L *lua.LState) {
		executionError = errors.New(L.Get(-1).String())
		done <- struct{}{}
	}

	ecUD := L.NewUserData()
	ecUD.Value = ec
	ec.Start()
	L.OpenLibs()

	box := runtime.New()
	globalSrc, err := box.FindString("global.lua")
	if err != nil {
		return err
	}
	err = L.DoString(globalSrc)
	if err != nil {
		return err
	}

	open(L, ec)

	L.SetGlobal("resolve", L.NewClosure(func(L *lua.LState) int {
		done <- struct{}{}
		return 0
	}))
	err = L.DoString(src)
	if err == nil {
		<-done
	}

	for _, a := range after {
		a(L)
	}
	ec.Close()
	if err != nil {
		return err
	}
	return executionError
}

func Async(src string, open func(L *lua.LState, ec *core.ExecutionContext), after ...func(L *lua.LState)) {
	err := AsyncWithError(src, open, after...)
	Expect(err).To(BeNil())
}

func SyncWithError(src string, open func(L *lua.LState), after ...func(L *lua.LState)) error {
	L := lua.NewState(lua.Options{})
	L.SetContext(context.Background())
	defer L.Close()

	L.OpenLibs()
	box := runtime.New()
	globalSrc, err := box.FindString("global.lua")
	if err != nil {
		return err
	}
	err = L.DoString(globalSrc)
	if err != nil {
		return err
	}
	open(L)

	err = L.DoString(src)
	if err != nil {
		return err
	}

	for _, a := range after {
		a(L)
	}

	return nil
}

func Sync(src string, open func(L *lua.LState), after ...func(L *lua.LState)) {
	err := SyncWithError(src, open, after...)
	Expect(err).To(BeNil())
}
