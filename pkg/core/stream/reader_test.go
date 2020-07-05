package stream

import (
	"bytes"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("Reader", func() {
	It("should read from reader", func() {
		buf := bytes.NewBuffer(nil)
		buf.WriteString("hello world")
		test.Async(`
			read(5, function(err, res)
				assert(err == nil, "err")
				assert(res == "hello", "res")
				resolve()
			end)
			`,
			func(L *lua.LState, ec *core.ExecutionContext) {
				L.SetGlobal("read", NewReader(L, ec, buf))
			})
		b := make([]byte, 6)
		n, err := buf.Read(b)
		Expect(err).To(BeNil())
		Expect(n).To(Equal(6))
		Expect(string(b)).To(Equal(" world"))
	})
})
