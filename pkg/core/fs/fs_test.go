package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	"github.com/joesonw/drlee/pkg/runtime"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
	lua "github.com/yuin/gopher-lua"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "core/fs")
}

type testLuaFile struct {
	ch chan struct{}
	*os.File
}

func (f *testLuaFile) Close() error {
	err := f.File.Close()
	f.ch <- struct{}{}
	return err
}

func runFSTest(src string, after ...func(L *lua.LState)) {
	test.Async(src, func(L *lua.LState, ec *core.ExecutionContext) {
		Open(L, ec, func(s string, flag, perm int) (File, error) {
			return os.OpenFile(s, flag, os.FileMode(perm))
		}, runtime.New())
	}, after...)
}

var _ = Describe("FS", func() {
	dir := "/tmp/fs-test/" + uuid.NewV4().String() + "/"
	err := os.MkdirAll(dir, os.ModeDir|os.ModePerm)
	if err != nil {
		panic(err)
	}

	Context("FS", func() {
		It("should open", func() {
			name := dir + uuid.NewV4().String()
			err := ioutil.WriteFile(name, nil, os.ModePerm)
			Expect(err).To(BeNil())

			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.open("%s", function(err)
					assert(err == nil, "err")
					resolve()
				end)
				`, name))
		})

		It("should remove", func() {
			name := dir + uuid.NewV4().String()
			err := ioutil.WriteFile(name, nil, os.ModePerm)
			Expect(err).To(BeNil())
			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.remove("%s", function(err)
					assert(err == nil, "err")
					resolve()
				end)
				`, name))
			_, err = os.Stat(name)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("should removeall", func() {
			name := dir + uuid.NewV4().String()

			err := os.MkdirAll(name, os.ModePerm|os.ModeDir)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(name+"/test", nil, os.ModePerm)
			Expect(err).To(BeNil())

			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.remove_all("%s", function(err)
					assert(err == nil, "err")
					resolve()
				end)
				`, name))
			_, err = os.Stat(name)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("should stat", func() {
			id := uuid.NewV4().String()
			name := dir + id

			err := os.MkdirAll(name, os.ModePerm|os.ModeDir)
			Expect(err).To(BeNil())

			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.stat("%s", function(err, info)
					assert(err == nil, "err")
					assert(info.name == "%s", "name")
					assert(info.isdir == true, "isdir")
					assert(info.mode == 2147484141, "mode")
					resolve()
				end)
				`, name, id))
		})

		It("should readdir", func() {
			name := dir + uuid.NewV4().String()

			err := os.MkdirAll(name, os.ModePerm|os.ModeDir)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(name+"/test", nil, os.ModePerm)
			Expect(err).To(BeNil())

			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.read_dir("%s", function(err, list)
					assert(err == nil, "err")
					assert(table.getn(list) == 1, "length")
					local info = list[1]
					assert(info.name == "test", "name")
					assert(info.isdir == false, "isdir")
					assert(info.mode == 493, "mode")
					assert(info.size == 0, "size")
					resolve()
				end)
				`, name))
		})

		It("should mkdir", func() {
			name := dir + uuid.NewV4().String()

			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.mkdir("%s", function(err)
					assert(err == nil, "err")
					resolve()
				end)
				`, name))
			info, err := os.Stat(name)
			Expect(err).To(BeNil())
			Expect(info.IsDir()).To(Equal(true))
		})

		It("should mkdirall", func() {
			name := dir + uuid.NewV4().String() + "/a/b/c/d/e"

			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.mkdir_all("%s", function(err)
					assert(err == nil, "err")
					resolve()
				end)
				`, name))
			info, err := os.Stat(name)
			Expect(err).To(BeNil())
			Expect(info.IsDir()).To(Equal(true))
		})

		It("should readfile", func() {
			name := dir + uuid.NewV4().String()
			err := ioutil.WriteFile(name, []byte("hello world"), os.ModePerm)
			Expect(err).To(BeNil())

			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.readfile("%s", function(err, text)
					assert(err == nil, "read err")
					assert(text == "hello world", "text")
					resolve()
				end)
				`, name))
		})
	})

	Context("File", func() {
		It("should read", func() {
			name := dir + uuid.NewV4().String()
			err := ioutil.WriteFile(name, []byte("hello world"), os.ModePerm)
			Expect(err).To(BeNil())

			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.open("%s", function(err, file)
					assert(err == nil, "open err")
					file:read(5, function(err, text, n)
						assert(err == nil, "read err")
						assert(text == "hello", "text")
						assert(n == 5, "size")
						resolve()
					end)
				end)
				`, name))
		})

		It("should write", func() {
			name := dir + uuid.NewV4().String()

			runFSTest(fmt.Sprintf(`
				local fs = require "fs"
				fs.open("%s", 513, 511, function(err, file)
					assert(err == nil, "open err")
					file:write("hello world", function(err)
						assert(err == nil, "write err")
						resolve()
					end)
				end)
				`, name))
			b, err := ioutil.ReadFile(name)
			Expect(err).To(BeNil())
			Expect(string(b)).To(Equal("hello world"))
		})

		It("should close", func() {
			name := dir + uuid.NewV4().String()

			ch := make(chan struct{}, 1)
			test.Async(fmt.Sprintf(`
				local fs = require "fs"
				fs.open("%s", 513, 511, function(err, file)
					assert(err == nil, "open err")
					file:close(function(err)
						assert(err == nil, "close err")
						resolve()
					end)
				end)
				`, name),
				func(L *lua.LState, ec *core.ExecutionContext) {
					Open(L, ec, func(s string, flag, perm int) (File, error) {
						f, err := os.OpenFile(s, flag, os.FileMode(perm))
						if err != nil {
							return nil, err
						}
						return &testLuaFile{
							File: f,
							ch:   ch,
						}, nil
					}, runtime.New())
				}, func(L *lua.LState) {
					<-ch
				})
		})
	})
})
