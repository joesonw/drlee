package utils

import (
	"net/http"
	_ "net/http/pprof"

	"go.uber.org/zap"
)

func EnablePPROF(addr string, logger *zap.Logger) {
	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			logger.Fatal("unable to server pprof", zap.Error(err))
		}
	}()
}
