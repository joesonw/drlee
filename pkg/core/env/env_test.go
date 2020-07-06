package env

import (
	"testing"

	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "core/env")
}

var _ = Describe("Env", func() {
	It("should work", func() {
		test.Sync(`
			local env = require "env"
			assert(env.node == "node", "node")
			assert(env.worker_id == 2, "worker_id")
			assert(env.worker_dir == "/", "worker_dir")
			`, func(L *lua.LState) {
			Open(L, nil, Env{
				NodeName: "node",
				WorkerID: 2,
				WorkDir:  "/",
			})
		})
	})

})
