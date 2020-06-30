package server

import (
	"encoding/json"
	"time"

	"go.uber.org/zap"
)

// Delegate is the interface that clients must implement if they want to hook
// into the gossip layer of Memberlist. All the methods must be thread-safe,
// as they can and generally will be called concurrently.

// NodeMeta is used to retrieve meta-data about the current node
// when broadcasting an alive message. It's length is limited to
// the given byte size. This metadata is available in the Node structure.
func (s *Server) NodeMeta(limit int) []byte {
	return s.meta.Encode()
}

// NotifyMsg is called when a user-data message is received.
// Care should be taken that this method does not block, since doing
// so would block the entire UDP packet receive loop. Additionally, the byte
// slice may be modified after the call returns, so it should be copied if needed
func (s *Server) NotifyMsg(b []byte) {
	switch MessageType(b[0]) {
	case TypeRegistryBroadcast:
		{
			broadcast := &RegistryBroadcast{}
			if err := unmarshalMessage(b, broadcast); err != nil {
				s.logger.Error("unable to unmarshal RegistryBroadcast message", zap.Error(err))
				return
			}
			s.handleRegistryBroadcast(broadcast)
		}
	}
}

// GetBroadcasts is called when user data messages can be broadcast. // It can return a list of buffers to send. Each buffer should assume an // overhead as provided with a limit on the total byte size allowed.
// The total byte size of the resulting data to send must not exceed
// the limit. Care should be taken that this method does not block,
// since doing so would block the entire UDP packet receive loop.
func (s *Server) GetBroadcasts(overhead, limit int) [][]byte {
	s.broadcasts.GetBroadcasts(overhead, limit)
	return nil
}

// LocalState is used for a TCP Push/Pull. This is sent to
// the remote side in addition to the membership information. Any
// data can be sent here. See MergeRemoteState as well. The `join`
// boolean indicates this is for a join instead of a push/pull.
func (s *Server) LocalState(join bool) []byte {
	services := make([]*RegistryBroadcast, len(s.localServices))
	i := 0
	nodeName := s.members.LocalNode().Name
	s.localServicesMu.RLock()
	defer s.localServicesMu.RUnlock()
	for name, weight := range s.localServices {
		services[i] = &RegistryBroadcast{
			NodeName:  nodeName,
			Timestamp: time.Now(),
			Name:      name,
			Weight:    weight,
		}
		i++
	}
	b, _ := json.Marshal(services)
	return b
}

// MergeRemoteState is invoked after a TCP Push/Pull. This is the
// state received from the remote side and is the result of the
// remote side's LocalState call. The 'join'
// boolean indicates this is for a join instead of a push/pull.
func (s *Server) MergeRemoteState(buf []byte, join bool) {
	var services []*RegistryBroadcast
	err := json.Unmarshal(buf, &services)
	if err != nil {
		s.logger.Error("unable to merge remote state", zap.Error(err))
		return
	}
	for _, svc := range services {
		s.handleRegistryBroadcast(svc)
	}
	s.logger.Info("merged remote state")
}
