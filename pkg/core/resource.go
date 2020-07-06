package core

import (
	"go.uber.org/atomic"
)

type resourceNode struct {
	next     *resourceNode
	prev     *resourceNode
	resource Resource
}

type ResourcePool struct {
	exit      chan struct{}
	cache     chan Resource
	exclusive chan struct{}
	tail      *resourceNode
}

func NewResourcePool(size int) *ResourcePool {
	p := &ResourcePool{
		exit:      make(chan struct{}),
		cache:     make(chan Resource, size),
		exclusive: make(chan struct{}, 1),
	}

	go func() {
		for {
			select {
			case <-p.exit:
				return
			case res := <-p.cache:
				p.exclusive <- struct{}{}
				node := &resourceNode{
					resource: res,
				}
				res.setNode(node)
				res.setPool(p)
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

func (p *ResourcePool) Insert(call Resource) {
	p.cache <- call
}

func (p *ResourcePool) Remove(node *resourceNode) {
	if node == nil {
		return
	}
	p.exclusive <- struct{}{}
	node.resource.setNode(nil)
	prev := node.prev
	next := node.next

	if prev != nil {
		prev.next = next
	}

	if next != nil {
		next.prev = prev
	}

	if node == p.tail {
		p.tail = nil
	}
	<-p.exclusive
}

func (p *ResourcePool) ForEach(iter func(Resource)) {
	if p.tail == nil {
		return
	}
	p.exclusive <- struct{}{}
	cur := p.tail
	for ; cur.prev != nil; cur = cur.prev {
	}
	for ; cur != nil; cur = cur.next {
		iter(cur.resource)
	}
	<-p.exclusive
}

func (p *ResourcePool) Close() {
	p.exit <- struct{}{}
}

type Resource interface {
	Release()
	Cancel()
	Name() string
	setNode(*resourceNode)
	setPool(*ResourcePool)
}

// resource a resource with release function, will be called only once
type resource struct {
	name         string
	released     *atomic.Bool
	release      func()
	node         *resourceNode
	pool         *ResourcePool
	afterRelease func()
}

func NewResource(name string, release func(), callback ...func()) Resource {
	var afterRelease func()
	if len(callback) > 0 {
		afterRelease = callback[0]
	}
	return &resource{
		name:         name,
		released:     atomic.NewBool(false),
		release:      release,
		afterRelease: afterRelease,
	}
}

func (g *resource) Release() {
	if g.released.CAS(false, true) {
		g.release()
		if g.afterRelease != nil {
			g.afterRelease()
		}
	}
}

func (g *resource) Name() string {
	return g.name
}

func (g *resource) setNode(node *resourceNode) {
	g.node = node
}

func (g *resource) setPool(pool *ResourcePool) {
	g.pool = pool
}

func (g *resource) Cancel() {
	if g.node == nil || g.pool == nil {
		return
	}

	g.pool.Remove(g.node)
}
