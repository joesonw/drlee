package core

import (
	"go.uber.org/atomic"
)

type guardNode struct {
	next  *guardNode
	prev  *guardNode
	guard Guard
}

type GuardPool struct {
	exit      chan struct{}
	cache     chan Guard
	exclusive chan struct{}
	tail      *guardNode
}

func NewGuardPool(size int) *GuardPool {
	p := &GuardPool{
		exit:      make(chan struct{}),
		cache:     make(chan Guard, size),
		exclusive: make(chan struct{}, 1),
	}

	go func() {
		for {
			select {
			case <-p.exit:
				return
			case call := <-p.cache:
				p.exclusive <- struct{}{}
				node := &guardNode{
					guard: call,
				}
				call.setNode(node)
				call.setPool(p)
				node.prev = p.tail
				if p.tail != nil {
					p.tail.next = node
				}
				p.tail = node
				<-p.exclusive
			}
		}
	}()

	return p
}

func (s *GuardPool) Insert(call Guard) {
	s.cache <- call
}

func (s *GuardPool) Remove(node *guardNode) {
	if node == nil {
		return
	}
	s.exclusive <- struct{}{}
	node.guard.setNode(nil)
	prev := node.prev
	next := node.next

	if prev != nil {
		prev.next = next
	}

	if next != nil {
		next.prev = prev
	}
	<-s.exclusive
}

func (s *GuardPool) ForEach(iter func(Guard)) {
	if s.tail == nil {
		return
	}
	s.exclusive <- struct{}{}
	cur := s.tail
	for ; cur.prev != nil; cur = cur.prev {
	}
	for ; cur != nil; cur = cur.next {
		iter(cur.guard)
	}
	<-s.exclusive
}

func (s *GuardPool) Close() {
	s.exit <- struct{}{}
}

type Guard interface {
	Release()
	Cancel()
	setNode(*guardNode)
	setPool(*GuardPool)
}

// guard guard a resource release function, will be called only once TODO: should we ref count? or timeout to delete leaked resources?
type guard struct {
	name     string
	released *atomic.Bool
	release  func()
	node     *guardNode
	pool     *GuardPool
}

func NewGuard(name string, release func()) Guard {
	return &guard{
		name:     name,
		released: atomic.NewBool(false),
		release:  release,
	}
}

func (g *guard) Release() {
	if g.released.CAS(false, true) {
		g.release()
	}
}

func (g *guard) setNode(node *guardNode) {
	g.node = node
}

func (g *guard) setPool(pool *GuardPool) {
	g.pool = pool
}

func (g *guard) Cancel() {
	if g.node == nil || g.pool == nil {
		return
	}

	g.pool.Remove(g.node)
}
