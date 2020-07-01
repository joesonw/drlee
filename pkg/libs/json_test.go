package libs

import (
	. "github.com/onsi/ginkgo"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("JSON", func() {
	It("should encode and decode", func() {
		runSyncLuaTest(
			`
			data = {}
			data["key"] = "value"
			str = json_encode(data)
			data = nil
			assert(str == '{"key":"value"}', "json encode")
			data = json_decode('{"key1":"value1"}')
			assert(data["key1"] == "value1", "json decode")
			`,
			func(L *lua.LState) {
				OpenJSON(L)
			})
	})
})
