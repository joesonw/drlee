package stream

import (
	"bytes"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("Writer", func() {
	It("should write from writer", func() {
		buf := bytes.NewBuffer(nil)
		test.Async(`
			write("hello world", function(err)
				assert(err == nil, "err")
				resolve()
			end)
			`,
			func(L *lua.LState, ec *core.ExecutionContext) {
				L.SetGlobal("write", NewWriter(L, ec, buf))
			})
		Expect(buf.String()).To(Equal("hello world"))
	})
})
