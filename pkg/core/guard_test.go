package core

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Guard", func() {
	It("should call release", func() {
		pool := NewGuardPool(10)
		ch1 := make(chan struct{}, 1)
		ch2 := make(chan struct{}, 1)
		g1 := NewGuard("test", func() {
			ch1 <- struct{}{}
		}).(*guard)
		g2 := NewGuard("test", func() {
			ch2 <- struct{}{}
		}).(*guard)
		pool.Insert(g1)
		pool.Insert(g2)
		time.Sleep(time.Millisecond * 10)

		Expect(pool.tail).To(Equal(g2.node))
		Expect(g2.node.prev).To(Equal(g1.node))
		Expect(g2.node.next).To(BeNil())

		Expect(g1.node.prev).To(BeNil())
		Expect(g1.node.next).To(Equal(g2.node))

		g1Found := false
		g2Found := false
		pool.ForEach(func(g Guard) {
			if g.(*guard) == g1 {
				g1Found = true
			}
			if g.(*guard) == g2 {
				g2Found = true
			}
		})
		Expect(g1Found).To(BeTrue())
		Expect(g2Found).To(BeTrue())

		g1.Cancel()
		g1Found = false
		g2Found = false
		pool.ForEach(func(g Guard) {
			if g.(*guard) == g1 {
				g1Found = true
			}
			if g.(*guard) == g2 {
				g2Found = true
			}
		})
		Expect(g1Found).To(BeFalse())
		Expect(g2Found).To(BeTrue())
	})
})
