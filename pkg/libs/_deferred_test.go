package libs

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("Deferred", func() {
	It("should resolve", func() {
		L := lua.NewState(lua.Options{
			SkipOpenLibs: true,
		})
		L.SetContext(context.Background())
		defer L.Close()
		lua.OpenBase(L)
		lua.OpenPackage(L)
		OpenDeferred(L)

		d := NewDeferred(L)
		L.SetGlobal("d", d.Value())
		go func() {
			time.Sleep(time.Second)
			d.Resolve(L, lua.LNumber(2))
			fmt.Println("resolved")
		}()
		done := make(chan struct{}, 1)
		L.SetGlobal("resolve", L.NewClosure(func(L *lua.LState) int {
			done <- struct{}{}
			return 0
		}))
		err := L.DoString(`
		print(d.state)
		assert(d.state == dPENDING, "state")
		d:next(function(err, res) 
			assert(err == nil, "error")
			assert(res == 2, "result")
			resolve()
		end)
		`)
		Expect(err).To(BeNil())
		<-done
	})

	It("should reject", func() {
		L := lua.NewState(lua.Options{
			SkipOpenLibs: true,
		})
		L.SetContext(context.Background())
		defer L.Close()
		lua.OpenBase(L)
		lua.OpenPackage(L)
		OpenDeferred(L)

		d := NewDeferred(L)
		L.SetGlobal("d", d.Value())
		go func() {
			time.Sleep(time.Second)
			d.Reject(L, lua.LString("error"))
		}()
		done := make(chan struct{}, 1)
		L.SetGlobal("resolve", L.NewClosure(func(L *lua.LState) int {
			done <- struct{}{}
			return 0
		}))
		err := L.DoString(`
		d:next(function(err, res)
			assert(err == "error", "error")
			assert(res == nil, "result")
			resolve()
		end)
		`)
		Expect(err).To(BeNil())
		<-done
	})
})
