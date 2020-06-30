package server

import "github.com/hashicorp/memberlist"

// AliveDelegate is used to involve a client in processing
// a node "alive" message. When a node joins, either through
// a UDP gossip or TCP push/pull, we update the state of
// that node via an alive message. This can be used to filter
// a node out and prevent it from being considered a peer
// using application specific logic.

// NotifyAlive is invoked when a message about a live
// node is received from the network.  Returning a non-nil
// error prevents the node from being considered a peer.
func (s *Server) NotifyAlive(peer *memberlist.Node) error {
	return nil
}
