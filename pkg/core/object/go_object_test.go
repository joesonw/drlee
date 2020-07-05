package object

import (
	"context"
	"testing"

	"github.com/joesonw/drlee/pkg/core/json"
	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "core/object")
}

var _ = Describe("GoObject", func() {
	Context("New", func() {
		It("should allow extra data", func() {
			test.Sync(`
				obj.test = 123
				result = json_encode(obj)
				`,
				func(L *lua.LState) {
					obj := New(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil)
					L.SetGlobal("obj", obj.Value())
					json.Open(L)
				},
				func(L *lua.LState) {
					val := L.GetGlobal("result")
					Expect(val.Type()).To(Equal(lua.LTString))
					Expect(val.String()).To(Equal(`{"customProperties":{"test":123},"properties":{"hello":"world"}}`))
				})
		})

		It("should allow extra data in protected", func() {
			test.Sync(`
				obj.test = 123
				result = json_encode(obj)
				`,
				func(L *lua.LState) {
					obj := NewProtected(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil)
					L.SetGlobal("obj", obj.Value())
					json.Open(L)
				},
				func(L *lua.LState) {
					val := L.GetGlobal("result")
					Expect(val.Type()).To(Equal(lua.LTString))
					Expect(val.String()).To(Equal(`{"customProperties":{"test":123},"properties":{"hello":"world"}}`))
				})
		})

		It("should allow protected key", func() {
			err := test.SyncWithError(`
				obj.hello = 123
				`,
				func(L *lua.LState) {
					obj := NewProtected(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil)
					L.SetGlobal("obj", obj.Value())
					json.Open(L)
				})
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("<string>:2: Attempt to modify protected properties\nstack traceback:\n\t[G]: in function (anonymous)\n\t<string>:2: in main chunk\n\t[G]: ?"))
		})

		It("should not allow extra data", func() {
			L := lua.NewState(lua.Options{})
			L.SetContext(context.Background())
			defer L.Close()
			lua.OpenBase(L)
			lua.OpenPackage(L)
			obj := NewReadOnly(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil)
			L.SetGlobal("obj", obj.Value())
			err := L.DoString(`obj.test = 123`)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(Equal("<string>:1: Attempt to modify read-only value\nstack traceback:\n\t[G]: in function (anonymous)\n\t<string>:1: in main chunk\n\t[G]: ?"))
		})

		It("should not change properties", func() {
			Skip("")
			test.Sync(`
				obj.hello = 123
				result = json_encode(obj)
				`,
				func(L *lua.LState) {
					obj := New(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil)
					L.SetGlobal("obj", obj.Value())
					json.Open(L)
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
			test.Sync(`
				assert(obj.hello == "world", "lGoObjectGet")
				result = json_encode(obj)
				`,
				func(L *lua.LState) {
					obj := NewReadOnly(L, nil, map[string]lua.LValue{"hello": lua.LString("world")}, nil)
					L.SetGlobal("obj", obj.Value())
					json.Open(L)
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
			L := lua.NewState(lua.Options{})
			L.SetContext(context.Background())
			defer L.Close()
			lua.OpenBase(L)
			lua.OpenPackage(L)
			val := "test"
			obj := NewReadOnly(L, nil, nil, val)
			v, err := Value(obj.Value())
			Expect(err).To(BeNil())
			Expect(v).To(Equal(val))
		})
	})

	Context("CloneGoObject", func() {

		It("should have equal values", func() {
			L := lua.NewState(lua.Options{})
			L.SetContext(context.Background())
			defer L.Close()
			lua.OpenBase(L)
			lua.OpenPackage(L)
			val := "test"
			obj := New(L, nil, nil, val)
			L.SetGlobal("obj", obj.Value())
			err := L.DoString(`obj.test = 123`)
			Expect(err).To(BeNil())

			L2 := lua.NewState(lua.Options{})
			L2.SetContext(context.Background())
			defer L2.Close()
			L2.SetContext(context.Background())
			lua.OpenBase(L2)
			lua.OpenPackage(L2)

			v := Clone(L2, obj.Value())
			L2.SetGlobal("obj", v)
			vv, err := Value(v)
			Expect(err).To(BeNil())
			Expect(vv).To(Equal(val))

			err = L.DoString(`obj.test = 456`)
			Expect(err).To(BeNil())

			err = L2.DoString(`assert(obj.test == 123, "cloned custom properties")`)
			Expect(err).To(BeNil())
		})
	})
})
