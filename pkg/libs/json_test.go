package libs

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("JSON", func() {
	It("should encode and decode", func() {
		L := lua.NewState(lua.Options{
			SkipOpenLibs: true,
		})
		defer L.Close()

		L.SetContext(NewContext(context.Background()))
		lua.OpenBase(L)
		lua.OpenPackage(L)
		OpenJSON(L)

		err := L.DoString(`
	data = {}
	data["key"] = "value"
	str = json_encode(data)
	data = nil
	assert(str == '{"key":"value"}', "json encode")
	data = json_decode('{"key1":"value1"}')
	assert(data["key1"] == "value1", "json decode")
	`)

		Expect(err).To(BeNil())
	})
})
