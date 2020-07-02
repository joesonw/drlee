package builtin

import (
	"context"
	"net/url"
	"strconv"

	redis "github.com/go-redis/redis/v8"
	lua "github.com/yuin/gopher-lua"
)

type RedisDoable interface {
	Do(ctx context.Context, args ...interface{}) *redis.Cmd
}

type RedisNewClient func(*redis.Options) RedisDoable

func OpenRedis(L *lua.LState, env *Env) {
	ud := L.NewUserData()
	ud.Value = env.RedisNewClient
	RegisterGlobalFuncs(L, map[string]lua.LGFunction{"redis_new": lRedisNew}, ud)
}

func lRedisNew(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	open := ud.Value.(RedisNewClient)
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
	L.Push(NewGoObject(L, map[string]lua.LGFunction{"call": lRedisCall}, map[string]lua.LValue{}, client, false))
	return 1
}

func lRedisCall(L *lua.LState) int {
	value, err := GetValueFromGoObject(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	client, _ := value.(RedisDoable)

	top := L.GetTop()
	args := make([]interface{}, top-2)
	for i := 2; i < top; i++ {
		args[i-2], err = UnmarshalValue(L, L.Get(i))
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

		val, err := MarshalLValue(L, result)
		if err != nil {
			cb.Reject(L, lua.LString("unable to parse redis result: "+err.Error()))
			return
		}
		cb.Resolve(L, val)
	}()

	return 0
}
