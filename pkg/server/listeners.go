package server

import (
	"net"
	"sync"
)

type ListenerManager struct {
	mu        *sync.Mutex
	listeners map[string]map[string]net.Listener
}

func newListenerManager() *ListenerManager {
	return &ListenerManager{
		mu:        &sync.Mutex{},
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
	for _, group := range m.listeners {
		for _, lis := range group {
			lis.Close()
		}
	}

	m.listeners = map[string]map[string]net.Listener{}
}
