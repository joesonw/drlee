package libs

import (
	"net/url"
	"strconv"

	"go.uber.org/zap"

	redis "github.com/go-redis/redis/v8"
	lua "github.com/yuin/gopher-lua"
)

type lRedis struct {
	client *redis.Client
	logger *zap.Logger
}

type lRedisOpenFunction func(*redis.Options) *redis.Client

func OpenRedis(L *lua.LState, open func(*redis.Options) *redis.Client) {
	ud := L.NewUserData()
	ud.Value = lRedisOpenFunction(open)
	L.SetGlobal("redis_open", L.NewClosure(lRedisOpen, ud))
}

func lRedisOpen(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	open := ud.Value.(lRedisOpenFunction)
	uriString := L.CheckString(1)
	uri, err := url.Parse(uriString)
	if err != nil {
		L.RaiseError("unable to parse \"%s\": %s", uriString, err.Error())
		return 0
	}
	if uri.Scheme != "redis" {
		L.RaiseError("uri schema \"%s\" is not supported", uri.Scheme)
	}
	options := &redis.Options{}
	options.Addr = uri.Host
	if uri.Path != "" {
		db, err := strconv.ParseInt(uri.Path[1:], 10, 32)
		if err != nil {
			L.RaiseError("unable to parse db \"%s\": %s", uri.Path[1:], err.Error())
			return 0
		}
		options.DB = int(db)
	}
	if u := uri.User; u != nil {
		pwd, _ := u.Password()
		options.Password = pwd
	}

	client := open(options)

	{
		ud := L.NewUserData()
		ud.Value = &lRedis{
			client: client,
		}
		proxy := L.NewTable()
		meta := L.NewTable()
		meta.RawSetString("__newindex", L.NewClosure(LRaiseReadOnly))
		meta.RawSetString("__index", L.NewClosure(lRedisDo, ud))
		L.SetMetatable(proxy, meta)
		L.Push(proxy)
	}
	return 1
}

func lRedisDo(L *lua.LState) int {
	ud := L.CheckUserData(1)
	r := ud.Value.(*lRedis)
	client := r.client

	top := L.GetTop()
	args := make([]interface{}, top)
	var err error
	for i := 1; i < top; i++ {
		args[i-1], err = AnyUnmarshal(L, L.Get(i))
		if err != nil {
			L.RaiseError(err.Error())
			return 0
		}
	}

	cb := NewCallback(L.Get(top))

	go func() {
		result, err := client.Do(L.Context(), args...).Result()
		if err != nil {
			cb.Reject(L, lua.LString("redis error: "+err.Error()))
			return
		}

		val, err := MarshalTable(L, result)
		if err != nil {
			cb.Reject(L, lua.LString("unable to parse redis result: "+err.Error()))
			return
		}
		cb.Resolve(L, val)
	}()

	return 0
}
