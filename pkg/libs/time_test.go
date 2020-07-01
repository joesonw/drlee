package libs

import (
	"fmt"
	"time"

	"bou.ke/monkey"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("time", func() {
	Describe("Timestamp", func() {
		It("shuold get current time", func() {
			now := time.Date(2006, 01, 02, 15, 04, 05, 30000000, time.Local)
			guard := monkey.Patch(time.Now, func() time.Time {
				return now
			})
			defer guard.Unpatch()

			runSyncLuaTest(
				fmt.Sprintf(`
					now = time_now()
					assert(now:__tostring() == "%s", "TIMESTAMP*:__tostring")
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
				`, now.Format(TimeFormat), time.RFC850, now.Format(time.RFC850), now.UnixNano()/1000000),
				func(L *lua.LState) {
					OpenTime(L)
				})
		})
	})

	Describe("Sleep", func() {
		It("should sleep", func() {
			now := time.Date(2006, 01, 02, 15, 04, 05, 30000000, time.Local)
			var callCount int64
			start := time.Now()
			var guard *monkey.PatchGuard
			guard = monkey.Patch(time.Now, func() time.Time {
				if callCount != 0 {
					guard.Unpatch()
					now := time.Now()
					guard.Restore()
					Expect(now.Sub(start).Milliseconds()).Should(BeNumerically(">=", int64(1000)))
					Expect(now.Sub(start).Milliseconds()).Should(BeNumerically("<", int64(1010)))
				}
				t := now.Add(time.Duration(callCount) * time.Second)
				callCount++
				return t
			})
			defer guard.Unpatch()

			runAsyncLuaTest(fmt.Sprintf(`
				local now = time_now()
				assert(now:__tostring() == "%s")
				time_sleep(1000, function()
					local now = time_now()
					assert(now:__tostring() == "%s")
					resolve()
				end)
			`, now.Format(TimeFormat), now.Add(time.Second).Format(TimeFormat)),
				func(L *lua.LState) {
					OpenTime(L)
				})
		})
	})

	Describe("Ticker", func() {
		It("should tick", func() {
			runAsyncLuaTest(`
				ticker = time_tick(1000)
				start = time_now()
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
			`, func(L *lua.LState) {
				OpenTime(L)
			})
		})
	})
})
