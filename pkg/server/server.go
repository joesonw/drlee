package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joesonw/drlee/pkg/plugin"

	"go.uber.org/atomic"

	"github.com/hashicorp/memberlist"
	"github.com/joesonw/drlee/proto"
	diskqueue "github.com/nsqio/go-diskqueue"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func _() {
	s := &Server{}
	var _ memberlist.Delegate = s
	var _ memberlist.EventDelegate = s
	var _ memberlist.ConflictDelegate = s
	var _ memberlist.MergeDelegate = s
	var _ memberlist.PingDelegate = s
	var _ proto.RPCServer = s
}

// Server Dr.LEE server handles lua execution environment, rpc debugging and metrics, etc.
type Server struct {
	config      *Config
	meta        Meta
	members     *memberlist.Memberlist
	broadcasts  *memberlist.TransmitLimitedQueue
	outboxQueue diskqueue.Interface
	logger      *zap.Logger
	plugins     []plugin.Interface

	deferredMembers func() *memberlist.Memberlist
	endpoints       map[string]*grpc.ClientConn
	endpointRPCs    map[string]proto.RPCClient
	endpointMu      *sync.RWMutex
	services        map[string]map[string]float64
	servicesMu      *sync.RWMutex
	localServices   map[string]float64
	localServicesMu *sync.RWMutex

	replybox  *ReplyBox
	inbox     *Inbox
	listeners *ListenerManager

	luaRunWg            *sync.WaitGroup
	luaScript           string
	luaExitChannelGroup []chan time.Duration

	isLuaReloading *atomic.Bool
	isDebug        bool
}

//nolint:gocritic
// New creates an new Server
func New(config *Config, deferredMembers func() *memberlist.Memberlist, inboxQueue diskqueue.Interface, outboxQueue diskqueue.Interface, logger *zap.Logger, plugins []plugin.Interface) *Server {
	if config.Concurrency < 1 {
		config.Concurrency = 1
	}
	return &Server{
		config: config,
		meta: Meta{
			RPCPort: config.RPC.Port,
		},
		outboxQueue: outboxQueue,
		logger:      logger,
		plugins:     plugins,

		deferredMembers: deferredMembers,
		endpoints:       map[string]*grpc.ClientConn{},
		endpointRPCs:    map[string]proto.RPCClient{},
		endpointMu:      &sync.RWMutex{},
		services:        map[string]map[string]float64{},
		servicesMu:      &sync.RWMutex{},
		localServices:   map[string]float64{},
		localServicesMu: &sync.RWMutex{},

		replybox:  newReplyBox(),
		inbox:     newInbox(inboxQueue),
		listeners: newListenerManager(),

		luaRunWg:       &sync.WaitGroup{},
		isLuaReloading: atomic.NewBool(false),
		isDebug:        strings.EqualFold(os.Getenv("DEBUG"), "true"),
	}
}

// Start start the server
func (s *Server) Start(ctx context.Context) error {
	s.members = s.deferredMembers()
	s.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes:       s.members.NumMembers,
		RetransmitMult: 3,
	}

	return nil
}

// Stop stop the server
func (s *Server) Stop(ctx context.Context) error {
	return nil
}

// handleRegistryBroadcast parse registry broadcast from peer
func (s *Server) handleRegistryBroadcast(broadcast *RegistryBroadcast) {
	s.servicesMu.Lock()
	if _, ok := s.services[broadcast.Name]; !ok {
		s.services[broadcast.Name] = map[string]float64{}
	}
	if broadcast.IsDeleted {
		delete(s.services[broadcast.Name], broadcast.NodeName)
		s.logger.Info(fmt.Sprintf("removed service \"%s\" on node %s", broadcast.Name, broadcast.NodeName))
	} else {
		s.services[broadcast.Name][broadcast.NodeName] = broadcast.Weight
		s.logger.Info(fmt.Sprintf("discovered service \"%s\" on node %s with weight %f", broadcast.Name, broadcast.NodeName, broadcast.Weight))
	}
	s.servicesMu.Unlock()
}

func (s *Server) handleNode(node *memberlist.Node) *Endpoint {
	s.endpointMu.Lock()
	defer s.endpointMu.Unlock()
	ep := &Endpoint{
		Name: node.Name,
		Addr: node.Address(),
		Meta: DecodeMeta(node.Meta),
	}
	if cc, ok := s.endpoints[node.Name]; ok {
		if err := cc.Close(); err != nil {
			s.logger.Error("unable to close grpc connection", zap.Error(err))
		}
	}
	cc, err := grpc.Dial(fmt.Sprintf("%s:%d", node.Addr.String(), ep.Meta.RPCPort), grpc.WithInsecure())
	if err != nil {
		s.logger.Fatal("unable to dial remote rpc service", zap.Error(err))
	}
	s.endpoints[node.Name] = cc
	s.endpointRPCs[node.Name] = proto.NewRPCClient(cc)
	return ep
}

func (s *Server) getRemoteRPC(nodeName string) proto.RPCClient {
	s.endpointMu.RLock()
	defer s.endpointMu.RUnlock()
	return s.endpointRPCs[nodeName]
}
