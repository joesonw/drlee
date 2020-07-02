package lib

import (
	"github.com/joesonw/drlee/pkg/builtin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("async", func() {
	It("should parallel", func() {
		builtin.RunAsyncTest(
			`
			local aCount = 0
			function a(cb)
				aCount = aCount + 1
				cb(nil, aCount)
			end
			function b(cb)
				cb("error")
			end

			local count = 0
			function done()
				count = count + 1			
				if count == 2 then
					resolve()
				end
			end
			async_parallel({a,a,a}, function(err, res)
				assert(err == nil, "error")
				assert(table.getn(res) == 3, "length")
				assert(res[1] == 1, "1")
				assert(res[2] == 2, "2")
				assert(res[3] == 3, "3")
				done()
			end)
			async_parallel({b, a, a}, function(err, res)
				assert(err == "error", "error")
				assert(aCount == 3)
				done()
			end)
			`,
			func(L *lua.LState) {
				lua.OpenTable(L)
				Expect(L.DoString(lAsyncParallel)).To(BeNil())
			})
	})

	It("should series", func() {
		builtin.RunAsyncTest(
			`
			local aCount = 0
			function a(cb)
				aCount = aCount + 1
				cb(nil, aCount)
			end
			function b(cb)
				cb("error")
			end

			local count = 0
			function done()
				count = count + 1			
				if count == 2 then
					resolve()
				end
			end
			async_series({a,a,a}, function(err, res)
				assert(err == nil, "error")
				assert(res == 3, "1")
				done()
			end)
			async_series({b, a, a}, function(err, res)
				assert(err == "error", "error")
				assert(aCount == 3)
				done()
			end)
			`,
			func(L *lua.LState) {
				lua.OpenTable(L)
				Expect(L.DoString(lAsyncSeries)).To(BeNil())
			})
	})
})
