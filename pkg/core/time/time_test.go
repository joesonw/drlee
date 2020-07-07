package time

import (
	"fmt"
	"testing"
	"time"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "core/time")
}

var _ = Describe("time", func() {
	It("should get current time", func() {
		now := time.Date(2006, 01, 02, 15, 04, 05, 30000000, time.Local)

		test.Sync(
			fmt.Sprintf(`
					local time = require "time"
					now = time.now()
					assert(now:__tostring() == "%s", "*:__tostring")
					assert(now:format("%s") == "%s", "TIMESTAMP*:format")
					assert(now.year == 2006, "TIMESTAMP*:year")
					assert(now.month == 1, "TIMESTAMP*:month")
					assert(now.day == 2, "TIMESTAMP*:day")
					assert(now.weekday == 1, "TIMESTAMP*:weekday")
					assert(now.hour == 15, "TIMESTAMP*:hour")
					assert(now.minute == 04, "TIMESTAMP*:minute")
					assert(now.second == 05, "TIMESTAMP*:second")
					assert(now.millisecond == 30, "TIMESTAMP*:millisecond")
					assert(now.milliunix == %d, "TIMESTAMP*:milliunix")
				`, now.Format(Layout), time.RFC850, now.Format(time.RFC850), now.UnixNano()/1000000),
			func(L *lua.LState) {
				Open(L, nil, func() time.Time {
					return now
				})
			})
	})

	It("should sleep", func() {
		now := time.Date(2006, 01, 02, 15, 04, 05, 30000000, time.Local)
		var callCount int64
		start := time.Now()

		test.Async(fmt.Sprintf(`
				local time = require "time"
				local now = time.now()
				assert(now:__tostring() == "%s")
				time.timeout(1000, function()
					local now = time.now()
					assert(now:__tostring() == "%s")
					resolve()
				end)
			`, now.Format(Layout), now.Add(time.Second).Format(Layout)),
			func(L *lua.LState, ec *core.ExecutionContext) {
				Open(L, ec, func() time.Time {
					if callCount != 0 {
						now2 := time.Now()
						Expect(now2.Sub(start).Milliseconds()).Should(BeNumerically(">=", int64(1000)))
						Expect(now2.Sub(start).Milliseconds()).Should(BeNumerically("<", int64(1010)))
					}
					t := now.Add(time.Duration(callCount) * time.Second)
					callCount++
					return t
				})
			})
	})

	It("should tick", func() {
		test.Async(`
				local time = require "time"
				ticker = time.tick(1000)
				start = time.now()
				ticker:nextTick(function (now)
					assert((now.milliunix - start.milliunix) >= 1000)
					assert((now.milliunix - start.milliunix) < 1010)
					ticker:nextTick(function (now)
						assert((now.milliunix - start.milliunix) >= 2000)
						assert((now.milliunix - start.milliunix) < 2020)
						ticker:stop()
						resolve()
					end)
				end)
			`, func(L *lua.LState, ec *core.ExecutionContext) {
			Open(L, ec, time.Now)
		})
	})
})
