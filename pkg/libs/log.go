package libs

import (
	"strings"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

var logFuncs = map[string]lua.LGFunction{
	"log_debug": lLogDebug,
	"log_info":  lLogInfo,
	"log_warn":  lLogWarn,
	"log_error": lLogError,
	"log_fatal": lLogFatal,
}

type lLog struct {
	logger *zap.Logger
}

func upLog(L *lua.LState) *lLog {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	if log, ok := ud.Value.(*lLog); ok {
		return log
	}
	L.RaiseError("log expected")
	return nil
}

func OpenLog(L *lua.LState, env *Env) {
	ud := L.NewUserData()
	ud.Value = &lLog{logger: env.Logger}
	RegisterGlobalFuncs(L, logFuncs, ud)
}

func lLogStringify(L *lua.LState) string {
	top := L.GetTop()
	arr := make([]string, top)
	for i := 1; i <= top; i++ {
		arr[i-1] = L.Get(i).String()
	}
	return strings.Join(arr, " ")
}

func lLogDebug(L *lua.LState) int {
	log := upLog(L)
	log.logger.Debug(lLogStringify(L))
	return 0
}

func lLogInfo(L *lua.LState) int {
	log := upLog(L)
	log.logger.Info(lLogStringify(L))
	return 0
}

func lLogWarn(L *lua.LState) int {
	log := upLog(L)
	log.logger.Warn(lLogStringify(L))
	return 0
}

func lLogError(L *lua.LState) int {
	log := upLog(L)
	log.logger.Error(lLogStringify(L))
	return 0
}

func lLogFatal(L *lua.LState) int {
	log := upLog(L)
	log.logger.Fatal(lLogStringify(L))
	return 0
}
