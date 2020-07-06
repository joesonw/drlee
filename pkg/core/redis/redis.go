package redis

import (
	"context"
	"net/url"
	"strconv"

	redis "github.com/go-redis/redis/v8"
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers"
	"github.com/joesonw/drlee/pkg/core/object"
	"github.com/joesonw/drlee/pkg/core/stream"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type Doable interface {
	Do(ctx context.Context, args ...interface{}) *redis.Cmd
	Close() error
}

type NewClient func(*redis.Options) Doable

type uV struct {
	ec        *core.ExecutionContext
	newClient NewClient
}

func Open(L *lua.LState, ec *core.ExecutionContext, newClient NewClient) {
	ud := L.NewUserData()
	ud.Value = &uV{
		ec:        ec,
		newClient: newClient,
	}
	utils.RegisterLuaModule(L, "redis", map[string]lua.LGFunction{"new": lRedisNew}, ud)
}

func lRedisNew(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	uv := ud.Value.(*uV)
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

	client := uv.newClient(options)
	obj := object.NewProtected(L, map[string]lua.LGFunction{"call": lCall}, map[string]lua.LValue{}, &uvClient{doable: client, ec: uv.ec})
	resource := core.NewResource("*redis.Client", func() {
		client.Close()
	})
	obj.SetFunction("close", stream.NewCloser(L, uv.ec, resource, client, true))
	L.Push(obj.Value())
	return 1
}

type uvClient struct {
	doable Doable
	ec     *core.ExecutionContext
}

func lCall(L *lua.LState) int {
	value, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	client, _ := value.(*uvClient)

	top := L.GetTop()
	args := make([]interface{}, top-2)
	for i := 2; i < top; i++ {
		args[i-2] = helpers.MustUnmarshalToMap(L, L.Get(i))
	}

	cb := L.Get(L.GetTop())
	client.ec.Call(core.Go(func(ctx context.Context) error {
		result, err := client.doable.Do(L.Context(), args...).Result()
		if err != nil {
			client.ec.Call(core.Lua(cb, utils.LError(err)))
		} else {
			client.ec.Call(core.Scoped(func(L *lua.LState) error {
				val, err := helpers.Marshal(L, result)
				if err != nil {
					return utils.CallLuaFunction(L, cb, utils.LError(err))
				}
				return utils.CallLuaFunction(L, cb, lua.LNil, val)
			}))
		}
		return nil
	}))

	return 0
}
