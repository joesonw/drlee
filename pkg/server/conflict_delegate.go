package server

import (
	"fmt"

	"github.com/hashicorp/memberlist"
)

// ConflictDelegate is a used to inform a client that
// a node has attempted to join which would result in a
// name conflict. This happens if two clients are configured
// with the same name but different addresses.

// NotifyConflict is invoked when a name conflict is detected
func (s *Server) NotifyConflict(existing, other *memberlist.Node) {
	s.logger.Error(fmt.Sprintf("cluster naming conflict, both existing (%s) and newly (%s) joined as name %s", existing.Address(), other.Address(), existing.Name))
}
