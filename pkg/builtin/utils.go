package builtin

import (
	"context"
	"errors"

	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func RunAsyncTestWithError(src string, open func(L *lua.LState), after ...func(L *lua.LState)) error {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})
	L.SetContext(context.Background())
	var stackError error
	done := make(chan struct{}, 1)
	stack := NewAsyncStack(L, 1024, func(err error) {
		stackError = err
		done <- struct{}{}
	})
	defer L.Close()
	L.Panic = func(L *lua.LState) {
		stackError = errors.New(L.Get(-1).String())
		done <- struct{}{}
	}

	stackUD := L.NewUserData()
	stackUD.Value = stack
	stack.Start()
	L.Env.RawSetString("stack", stackUD)

	lua.OpenBase(L)
	lua.OpenPackage(L)
	open(L)

	L.SetGlobal("resolve", L.NewClosure(func(L *lua.LState) int {
		done <- struct{}{}
		return 0
	}))
	err := L.DoString(src)
	println("what???????????")
	<-done

	for _, a := range after {
		a(L)
	}
	stack.Stop()
	if err != nil {
		return err
	}
	return stackError
}

func RunAsyncTest(src string, open func(L *lua.LState), after ...func(L *lua.LState)) {
	err := RunAsyncTestWithError(src, open, after...)
	Expect(err).To(BeNil())
}

func RunSyncTest(src string, open func(L *lua.LState), after ...func(L *lua.LState)) {
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

func EnsureParamTypes(L *lua.LState, types ...lua.LValueType) bool {
	n := L.GetTop()
	if n != len(types) {
		L.RaiseError("require %d arguments, has %d", len(types), n)
		return true
	}

	for i := 0; i < n; i++ {
		val := L.Get(i + 1)
		if val.Type() != types[i] {
			L.TypeError(i+1, types[i])
			return false
		}
	}

	return true
}

func EnsureTableProperties(L *lua.LState, table *lua.LTable, properties map[string]lua.LValueType) bool {
	for key, typ := range properties {
		val := table.RawGetString(key)
		if val.Type() != typ {
			L.RaiseError("key \"%s\" of table should be %s", key, typ.String())
			return false
		}
	}

	return true
}
