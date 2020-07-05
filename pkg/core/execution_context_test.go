package core

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("ExecutionContext", func() {
	It("should call", func() {
		L := lua.NewState()
		defer L.Close()
		errCh := make(chan error, 1)
		ec := NewExecutionContext(L, Config{
			OnError: func(err error) {
				errCh <- err
			},
			LuaStackSize:      64,
			GoStackSize:       64,
			GoCallConcurrency: 1,
		})
		defer ec.Close()
		ec.Start()

		ch := make(chan struct{}, 1)
		ec.Call(Go(func(ctx context.Context) error {
			ch <- struct{}{}
			return nil
		}))
		<-ch

		ec.Call(Go(func(ctx context.Context) error {
			return errors.New("from go")
		}))
		err := <-errCh
		Expect(err).To(Not(BeNil()))
		Expect(err.Error()).To(Equal("from go"))

		ch = make(chan struct{}, 1)
		fn := L.NewClosure(func(L *lua.LState) int {
			shouldError := L.CheckBool(1)
			if shouldError {
				L.RaiseError("from lua")
			}
			ch <- struct{}{}
			return 0
		})
		ec.Call(Lua(fn, lua.LBool(false)))
		<-ch

		ec.Call(Lua(fn, lua.LBool(true)))
		err = <-errCh
		Expect(err).To(Not(BeNil()))
		Expect(err.Error()).To(Equal(" from lua\nstack traceback:\n\t[G]: in main chunk\n\t[G]: ?"))

		caughtErr := make(chan error, 1)
		ec.Call(LuaCatch(fn, func(err error) {
			caughtErr <- err
		}, lua.LBool(true)))
		err = <-caughtErr
		Expect(err).To(Not(BeNil()))
		Expect(err.Error()).To(Equal(" from lua\nstack traceback:\n\t[G]: in main chunk\n\t[G]: ?"))

	})
})
