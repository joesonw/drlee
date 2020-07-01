package libs

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("GoObject", func() {
	Context("NewGoObject", func() {
		It("should allow extra data", func() {
			runSyncLuaTest(`
				obj.test = 123
				result = json_encode(obj)
			`,
				func(L *lua.LState) {
					obj := NewGoObject(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil, true)
					L.SetGlobal("obj", obj)
					OpenJSON(L)
				},
				func(L *lua.LState) {

					val := L.GetGlobal("result")
					Expect(val.Type()).To(Equal(lua.LTString))
					Expect(val.String()).To(Equal(`{"customProperties":{"test":123},"properties":{"hello":"world"}}`))
				})

		})

		It("should not allow extra data", func() {
			L := lua.NewState(lua.Options{
				SkipOpenLibs: true,
			})
			defer L.Close()
			L.SetContext(NewContext(context.Background(), nil))
			lua.OpenBase(L)
			lua.OpenPackage(L)
			obj := NewGoObject(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil, false)
			L.SetGlobal("obj", obj)
			err := L.DoString(`obj.test = 123`)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("<string>:1: Attempt to modify read-only table\nstack traceback:\n\t[G]: in function (anonymous)\n\t<string>:1: in main chunk\n\t[G]: ?"))
		})

		It("should not change properties", func() {

			runSyncLuaTest(`
				obj.hello = 123
				result = json_encode(obj)
				`,
				func(L *lua.LState) {
					obj := NewGoObject(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil, true)
					L.SetGlobal("obj", obj)
					OpenJSON(L)
				},
				func(L *lua.LState) {
					val := L.GetGlobal("result")
					Expect(val.Type()).To(Equal(lua.LTString))
					Expect(val.String()).To(Equal(`{"customProperties":[],"properties":{"hello":"world"}}`))
				})
		})
	})

	Context("MarshalJSON", func() {
		It("should have proper data", func() {
			runSyncLuaTest(`
				assert(obj.hello == "world", "lGoObjectGet")
				result = json_encode(obj)
				`,
				func(L *lua.LState) {
					obj := NewGoObject(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil, false)
					L.SetGlobal("obj", obj)
					OpenJSON(L)
				},
				func(L *lua.LState) {
					val := L.GetGlobal("result")
					Expect(val.Type()).To(Equal(lua.LTString))
					Expect(val.String()).To(Equal(`{"properties":{"hello":"world"}}`))
				})

		})
	})

	Context("GetValueFromGoObject", func() {
		It("should have equal values", func() {
			L := lua.NewState(lua.Options{
				SkipOpenLibs: true,
			})
			defer L.Close()
			L.SetContext(NewContext(context.Background(), nil))
			lua.OpenBase(L)
			lua.OpenPackage(L)
			val := "test"
			obj := NewGoObject(L, nil, nil, val, false)
			v, err := GetValueFromGoObject(obj)
			Expect(err).To(BeNil())
			Expect(v).To(Equal(val))
		})
	})

	Context("CloneGoObject", func() {
		It("should have equal values", func() {
			L := lua.NewState(lua.Options{
				SkipOpenLibs: true,
			})
			defer L.Close()
			L.SetContext(NewContext(context.Background(), nil))
			lua.OpenBase(L)
			lua.OpenPackage(L)
			val := "test"
			obj := NewGoObject(L, nil, nil, val, true)
			L.SetGlobal("obj", obj)
			err := L.DoString(`obj.test = 123`)
			Expect(err).To(BeNil())

			L2 := lua.NewState(lua.Options{
				SkipOpenLibs: true,
			})
			defer L2.Close()
			L2.SetContext(context.Background())
			lua.OpenBase(L2)
			lua.OpenPackage(L2)

			v := CloneGoObject(L2, obj)
			L2.SetGlobal("obj", v)
			vv, err := GetValueFromGoObject(v)
			Expect(err).To(BeNil())
			Expect(vv).To(Equal(val))

			err = L.DoString(`obj.test = 456`)
			Expect(err).To(BeNil())

			err = L2.DoString(`assert(obj.test == 123, "cloned custom properties")`)
			Expect(err).To(BeNil())
		})
	})
})
