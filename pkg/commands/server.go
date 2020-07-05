package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	goPlugin "plugin"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/joesonw/drlee/pkg/plugin"
	"github.com/joesonw/drlee/pkg/server"
	"github.com/joesonw/drlee/proto"
	diskqueue "github.com/nsqio/go-diskqueue"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	yaml "gopkg.in/yaml.v2"
)

type ServerCommand struct {
	logger       *zap.Logger
	grpcListener net.Listener
	server       *server.Server
}

func NewServerCommand(logger *zap.Logger) *ServerCommand {
	return &ServerCommand{
		logger: logger,
	}
}

func (s *ServerCommand) Build(ctx context.Context) *cobra.Command {
	logger := s.logger
	var pJoin *[]string
	cmd := &cobra.Command{
		Use:   "server",
		Short: "run server",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				println("usage: drlee server <config file> <lua file>")
				os.Exit(1)
			}
			bytes, err := ioutil.ReadFile(args[0])
			if err != nil {
				logger.Fatal("unable to read config", zap.Error(err))
			}
			config := &server.Config{}
			if err := yaml.Unmarshal(bytes, config); err != nil {
				logger.Fatal("unable to parse config", zap.Error(err))
			}

			var plugins []plugin.Interface
			for _, pluginConfig := range config.Plugins {
				p, err := goPlugin.Open(pluginConfig.Path)
				if err != nil {
					logger.Fatal("unable to load plugin: "+pluginConfig.Path, zap.Error(err))
				}

				symbol, err := p.Lookup(pluginConfig.Symbol)
				if err != nil {
					logger.Fatal("unable to load plugin: "+pluginConfig.Path, zap.Error(err))
				}

				plg, ok := symbol.(plugin.Interface)
				if !ok {
					logger.Fatal(fmt.Sprintf("plugin %s@%s is not a plugin.Interface", pluginConfig.Symbol, pluginConfig.Path))
				}
				plugins = append(plugins, plg)
			}

			if config.Queue.MaxBytesPerFile <= 0 {
				config.Queue.MaxBytesPerFile = 100 * 1024 * 1024
			}

			if config.Queue.SyncEvery <= 0 {
				config.Queue.SyncEvery = 2500
			}

			if config.Queue.SyncTimeout <= 0 {
				config.Queue.SyncTimeout = time.Second * 2
			}

			if config.Queue.MaxMsgSize <= 0 {
				config.Queue.MaxMsgSize = 1024 * 1024
			}

			diskqueueLogger := logger.Sugar()

			diskqueueLeveledLoggers := map[diskqueue.LogLevel]func(string, ...interface{}){
				diskqueue.DEBUG: diskqueueLogger.Debugf,
				diskqueue.INFO:  diskqueueLogger.Infof,
				diskqueue.WARN:  diskqueueLogger.Warnf,
				diskqueue.ERROR: diskqueueLogger.Errorf,
				diskqueue.FATAL: diskqueueLogger.Fatalf,
			}
			diskqueuLogFunc := func(lvl diskqueue.LogLevel, f string, args ...interface{}) {
				diskqueueLeveledLoggers[lvl](f, args...)
			}

			if err := os.MkdirAll(config.Queue.Dir, os.ModePerm|os.ModeDir); err != nil {
				logger.Fatal("unable to create dir: "+config.Queue.Dir+" for queue", zap.Error(err))
			}

			inbox := diskqueue.New("inbox", config.Queue.Dir, config.Queue.MaxBytesPerFile, 1, config.Queue.MaxMsgSize, config.Queue.SyncEvery, config.Queue.SyncTimeout, diskqueuLogFunc)
			outbox := diskqueue.New("outbox", config.Queue.Dir, config.Queue.MaxBytesPerFile, 1, config.Queue.MaxMsgSize, config.Queue.SyncEvery, config.Queue.SyncTimeout, diskqueuLogFunc)

			var members *memberlist.Memberlist

			srv := server.New(config, func() *memberlist.Memberlist { return members }, inbox, outbox, logger, plugins)
			memberlistConfig := memberlist.DefaultLANConfig()
			memberlistConfig.Name = config.NodeName
			memberlistConfig.BindAddr = config.Gossip.Addr
			memberlistConfig.BindPort = config.Gossip.Port
			memberlistConfig.AdvertisePort = config.Gossip.Port
			memberlistConfig.Delegate = srv
			memberlistConfig.Events = srv
			//memberlistConfig.Conflict = srv
			memberlistConfig.Merge = srv
			//memberlistConfig.Ping = srv
			//memberlistConfig.Alive = srv
			memberlistConfig.Logger = zap.NewStdLog(logger)

			grpcServer := grpc.NewServer()
			proto.RegisterRPCServer(grpcServer, srv)
			lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.RPC.Addr, config.RPC.Port))
			if err != nil {
				logger.Fatal(fmt.Sprintf("unable to listen grpc on %s:%d", config.RPC.Addr, config.RPC.Port), zap.Error(err))
			}

			if config.Gossip.SecretKey != "" {
				memberlistConfig.SecretKey = []byte(config.Gossip.SecretKey)
			}
			members, err = memberlist.Create(memberlistConfig)
			if err != nil {
				logger.Fatal("unable to start membership", zap.Error(err))
			}
			logger.Info("server started")

			if err := srv.Start(ctx); err != nil {
				logger.Fatal("unable to start server", zap.Error(err))
			}

			go func() {
				if err := grpcServer.Serve(lis); err != nil {
					logger.Error("unable to serve grpc server")
				}
			}()

			if len(*pJoin) > 0 {
				n, err := members.Join(*pJoin)
				if err != nil {
					logger.Fatal("unable to join existing cluster")
				}
				logger.Info(fmt.Sprintf("joined a cluster with %d nodes", n))
			}
			s.server = srv
			if err := srv.LoadLua(ctx, args[1]); err != nil {
				logger.Fatal("unable to run lua", zap.Error(err))
			}
			logger.Info("lua script loaded")

			srv.StartReplyWorkers()
			logger.Info("services registered")
		},
	}
	pJoin = cmd.Flags().StringArray("join", nil, "peer nodes to join")

	return cmd
}

func (s *ServerCommand) Stop(ctx context.Context) {
	if s.grpcListener != nil {
		if err := s.grpcListener.Close(); err != nil {
			s.logger.Error("unable to stop grpc server", zap.Error(err))
		}
	}
	if s.server == nil {
		return
	}
	if err := s.server.Stop(ctx); err != nil {
		s.logger.Error("unable to stop server", zap.Error(err))
	}
}
