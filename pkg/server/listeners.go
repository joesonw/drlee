package server

import (
	"fmt"
	"net"
	"sync"

	"go.uber.org/zap"
)

type ListenerManager struct {
	mu        *sync.Mutex
	logger    *zap.Logger
	listeners map[string]map[string]net.Listener
}

func newListenerManager(logger *zap.Logger) *ListenerManager {
	return &ListenerManager{
		mu:        &sync.Mutex{},
		logger:    logger,
		listeners: map[string]map[string]net.Listener{},
	}
}

func (m *ListenerManager) Listen(network, addr string) (net.Listener, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.listeners[network]; !ok {
		m.listeners[network] = map[string]net.Listener{}
	}
	if lis, ok := m.listeners[network][addr]; ok {
		return lis, nil
	}
	lis, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	m.listeners[network][addr] = lis
	return lis, nil
}

func (m *ListenerManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for network, group := range m.listeners {
		for addr, lis := range group {
			if err := lis.Close(); err != nil {
				m.logger.Error(fmt.Sprintf("unable to close listener %s@%s", network, addr))
				continue
			}
			m.logger.Info(fmt.Sprintf("closed listener %s@%s", network, addr))
		}
	}

	m.listeners = map[string]map[string]net.Listener{}
}
