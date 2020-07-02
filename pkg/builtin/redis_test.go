package builtin

import (
	"context"

	redis "github.com/go-redis/redis/v8"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

type testRedisDoable func(ctx context.Context, args ...interface{}) *redis.Cmd

func (do testRedisDoable) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	return do(ctx, args...)
}

var _ = Describe("Redis", func() {
	It("should call", func() {
		optionsCh := make(chan *redis.Options, 1)
		argsCh := make(chan []interface{}, 1)
		RunAsyncTest(`
			local c = redis_new("redis://u:password@localhost:6379/12")
			c:call("get", "test", function(err, res)
				assert(err == nil, "err")
				assert(res == 123, "res")
				resolve()
			end)
			`,
			func(L *lua.LState) {
				OpenRedis(L, &Env{
					RedisNewClient: func(options *redis.Options) RedisDoable {
						optionsCh <- options
						return testRedisDoable(func(ctx context.Context, args ...interface{}) *redis.Cmd {
							argsCh <- args
							return redis.NewCmdResult(123, nil)
						})
					},
				})
			})

		options := <-optionsCh
		Expect(options.Addr).To(Equal("localhost:6379"))
		Expect(options.DB).To(Equal(12))
		Expect(options.Password).To(Equal("password"))

		args := <-argsCh
		Expect(args[0]).To(Equal(lua.LString("get")))
		Expect(args[1]).To(Equal(lua.LString("test")))
	})
})
