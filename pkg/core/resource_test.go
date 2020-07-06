package core

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Guard", func() {
	It("should call release", func() {
		pool := NewResourcePool(10)
		ch1 := make(chan struct{}, 1)
		ch2 := make(chan struct{}, 1)
		r1 := NewResource("test", func() {
			ch1 <- struct{}{}
		}).(*resource)
		r2 := NewResource("test", func() {
			ch2 <- struct{}{}
		}).(*resource)
		pool.Insert(r1)
		pool.Insert(r2)
		time.Sleep(time.Millisecond * 10)

		Expect(pool.tail).To(Equal(r2.node))
		Expect(r2.node.prev).To(Equal(r1.node))
		Expect(r2.node.next).To(BeNil())

		Expect(r1.node.prev).To(BeNil())
		Expect(r1.node.next).To(Equal(r2.node))

		g1Found := false
		g2Found := false
		pool.ForEach(func(g Resource) {
			if g.(*resource) == r1 {
				g1Found = true
			}
			if g.(*resource) == r2 {
				g2Found = true
			}
		})
		Expect(g1Found).To(BeTrue())
		Expect(g2Found).To(BeTrue())

		r1.Cancel()
		g1Found = false
		g2Found = false
		pool.ForEach(func(g Resource) {
			if g.(*resource) == r1 {
				g1Found = true
			}
			if g.(*resource) == r2 {
				g2Found = true
			}
		})
		Expect(g1Found).To(BeFalse())
		Expect(g2Found).To(BeTrue())
	})
})
