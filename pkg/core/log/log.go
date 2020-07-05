package log

import (
	"strings"

	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

var funcs = map[string]lua.LGFunction{
	"debug": lDebug,
	"info":  lInfo,
	"warn":  lWarn,
	"error": lError,
	"fatal": lFatal,
}

type userValue struct {
	logger *zap.Logger
}

func up(L *lua.LState) *userValue {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	if log, ok := ud.Value.(*userValue); ok {
		return log
	}
	L.RaiseError("log expected")
	return nil
}

func Open(L *lua.LState, logger *zap.Logger) {
	ud := L.NewUserData()
	ud.Value = &userValue{logger: logger}
	utils.RegisterLuaModule(L, "log", funcs, ud)
}

func lLogStringify(L *lua.LState) string {
	top := L.GetTop()
	arr := make([]string, top)
	for i := 1; i <= top; i++ {
		arr[i-1] = L.Get(i).String()
	}
	return strings.Join(arr, " ")
}

func lDebug(L *lua.LState) int {
	log := up(L)
	log.logger.Debug(lLogStringify(L))
	return 0
}

func lInfo(L *lua.LState) int {
	log := up(L)
	log.logger.Info(lLogStringify(L))
	return 0
}

func lWarn(L *lua.LState) int {
	log := up(L)
	log.logger.Warn(lLogStringify(L))
	return 0
}

func lError(L *lua.LState) int {
	log := up(L)
	log.logger.Error(lLogStringify(L))
	return 0
}

func lFatal(L *lua.LState) int {
	log := up(L)
	log.logger.Fatal(lLogStringify(L))
	return 0
}
