package json

import (
	"testing"

	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "core/json")
}

var _ = Describe("JSON", func() {
	It("should encode and decode", func() {
		test.Sync(
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
				Open(L)
			})
	})
})
