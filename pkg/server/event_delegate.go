package server

import (
	"fmt"

	"github.com/hashicorp/memberlist"
)

// EventDelegate is a simpler delegate that is used only to receive
// notifications about members joining and leaving. The methods in this
// delegate may be called by multiple goroutines, but never concurrently.
// This allows you to reason about ordering.

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (s *Server) NotifyJoin(node *memberlist.Node) {
	ep := s.handleNode(node)
	s.logger.Info(fmt.Sprintf("peer %s(%s) joined, rpc-port: %d", ep.Name, ep.Addr, ep.Meta.RPCPort))
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (s *Server) NotifyLeave(node *memberlist.Node) {
	s.logger.Info(fmt.Sprintf("peer %s(%s) left", node.Name, node.Addr))
	s.servicesMu.Lock()
	defer s.servicesMu.Unlock()
	s.endpointMu.Lock()
	defer s.endpointMu.Unlock()

	delete(s.endpointRPCs, node.Name)
	delete(s.endpoints, node.Name)
	for _, group := range s.services {
		delete(group, node.Name)
	}
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (s *Server) NotifyUpdate(node *memberlist.Node) {
	s.endpointMu.Lock()
	_, ok := s.endpoints[node.Name]
	s.endpointMu.Unlock()
	if !ok {
		s.logger.Warn(fmt.Sprintf("memberlist NotifyUpdate non-exist node \"%s\"", node.Name))
		return
	}
	ep := s.handleNode(node)
	s.logger.Info(fmt.Sprintf("peer %s(%s) updateda, rpc-port: %d", ep.Name, ep.Addr, ep.Meta.RPCPort))
}
