package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/joesonw/drlee/pkg/commands"
	"github.com/joesonw/drlee/pkg/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	var (
		logger *zap.Logger
		err    error
	)
	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		panic(err)
	}

	root := &cobra.Command{
		Use:   "drlee",
		Short: "Distributed Lua Execution Environment",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	server := commands.NewServerCommand(logger)
	debug := commands.NewDebugCommand()
	root.AddCommand(server.Build(ctx))
	root.AddCommand(debug.Build(ctx))

	if addr := os.Getenv("PPROF_ADDR"); addr != "" {
		utils.EnablePPROF(addr, logger)
	}

	if err := root.Execute(); err != nil {
		panic(fmt.Errorf("unable to run command: %w", err))
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	<-s

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	server.Stop(ctx)
}
