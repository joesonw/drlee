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

type lLogger struct {
	logger *zap.Logger
}

func checkLogger(L *lua.LState) *lLogger {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	if log, ok := ud.Value.(*lLogger); ok {
		return log
	}
	L.RaiseError("log expected")
	return nil
}

func Open(L *lua.LState, logger *zap.Logger) {
	ud := L.NewUserData()
	ud.Value = &lLogger{logger: logger}
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
	log := checkLogger(L)
	log.logger.Debug(lLogStringify(L))
	return 0
}

func lInfo(L *lua.LState) int {
	log := checkLogger(L)
	log.logger.Info(lLogStringify(L))
	return 0
}

func lWarn(L *lua.LState) int {
	log := checkLogger(L)
	log.logger.Warn(lLogStringify(L))
	return 0
}

func lError(L *lua.LState) int {
	log := checkLogger(L)
	log.logger.Error(lLogStringify(L))
	return 0
}

func lFatal(L *lua.LState) int {
	log := checkLogger(L)
	log.logger.Fatal(lLogStringify(L))
	return 0
}
