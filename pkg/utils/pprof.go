package utils

import (
	"net/http"
	"net/http/pprof" // go pprof

	"go.uber.org/zap"
)

func EnablePPROF(addr string, logger *zap.Logger) {
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		err := http.ListenAndServe(addr, mux)
		if err != nil {
			logger.Fatal("unable to server pprof", zap.Error(err))
		}
	}()
}
