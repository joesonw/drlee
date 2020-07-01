package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/memberlist"
	"github.com/joesonw/drlee/pkg/libs"
	"github.com/joesonw/drlee/proto"
	diskqueue "github.com/nsqio/go-diskqueue"
	lua "github.com/yuin/gopher-lua"
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

type Server struct {
	config      *Config
	meta        Meta
	members     *memberlist.Memberlist
	broadcasts  *memberlist.TransmitLimitedQueue
	inboxQueue  diskqueue.Interface
	outboxQueue diskqueue.Interface
	queueCond   *sync.Cond
	logger      *zap.Logger

	deferredMembers func() *memberlist.Memberlist
	endpoints       map[string]*grpc.ClientConn
	endpointRPCs    map[string]proto.RPCClient
	endpointMu      *sync.RWMutex
	services        map[string]map[string]float64
	servicesMu      *sync.RWMutex
	localServices   map[string]float64
	localServicesMu *sync.RWMutex
	replyInbox      map[string]chan *RPCResponse
	replyInboxMu    *sync.RWMutex

	servicesRequestCh   chan *libs.RPCRequest
	httpServerMappingMu *sync.RWMutex
	httpServerMapping   map[string]*httpServer

	reloadMu            *sync.Mutex
	luaRunWg            *sync.WaitGroup
	luaScript           string
	luaExitChannelGroup []chan struct{}
	luaStates           map[int]*lua.LState
	isLuaReloading      bool

	luaOpenedFileMu *sync.Mutex
	luaOpenedFiles  map[string]libs.File
}

func New(config *Config, deferredMembers func() *memberlist.Memberlist, inboxQueue diskqueue.Interface, outboxQueue diskqueue.Interface, logger *zap.Logger) *Server {
	if config.Concurrency < 1 {
		config.Concurrency = 1
	}
	return &Server{
		config: config,
		meta: Meta{
			RPCPort: config.RPC.Port,
		},
		inboxQueue:  inboxQueue,
		outboxQueue: outboxQueue,
		logger:      logger,

		deferredMembers: deferredMembers,
		endpoints:       map[string]*grpc.ClientConn{},
		endpointRPCs:    map[string]proto.RPCClient{},
		endpointMu:      &sync.RWMutex{},
		services:        map[string]map[string]float64{},
		servicesMu:      &sync.RWMutex{},
		localServices:   map[string]float64{},
		localServicesMu: &sync.RWMutex{},
		replyInbox:      map[string]chan *RPCResponse{},
		replyInboxMu:    &sync.RWMutex{},

		servicesRequestCh:   make(chan *libs.RPCRequest, 1024),
		httpServerMappingMu: &sync.RWMutex{},
		httpServerMapping:   map[string]*httpServer{},

		reloadMu:  &sync.Mutex{},
		luaRunWg:  &sync.WaitGroup{},
		luaStates: map[int]*lua.LState{},

		luaOpenedFileMu: &sync.Mutex{},
		luaOpenedFiles:  map[string]libs.File{},
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.members = s.deferredMembers()
	s.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes:       s.members.NumMembers,
		RetransmitMult: 3,
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return nil
}

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
