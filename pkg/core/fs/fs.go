package fs

import (
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/gobuffalo/packr"
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers/params"
	"github.com/joesonw/drlee/pkg/core/object"
	"github.com/joesonw/drlee/pkg/core/stream"
	"github.com/joesonw/drlee/pkg/core/time"
	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

type OpenFile func(name string, flag, perm int) (File, error)

type File interface {
	io.Closer
	io.Reader
	io.Writer
}

func Open(L *lua.LState, ec *core.ExecutionContext, open OpenFile, runtime packr.Box) {
	ud := L.NewUserData()
	ud.Value = &lFS{
		open: open,
		ec:   ec,
	}
	utils.RegisterLuaModule(L, "_fs", funcs, ud)
	src, err := runtime.FindString("fs.lua")
	if err != nil {
		L.RaiseError(err.Error())
	}
	if err := utils.RegisterLuaScriptModule(L, "fs", src); err != nil {
		L.RaiseError(err.Error())
	}
}

type lFS struct {
	open OpenFile
	ec   *core.ExecutionContext
}

func checkFS(L *lua.LState) *lFS {
	uv := L.CheckUserData(lua.UpvalueIndex(1))
	if fs, ok := uv.Value.(*lFS); ok {
		return fs
	}

	L.RaiseError("expected fs")
	return nil
}

var funcs = map[string]lua.LGFunction{
	"flags":      lFlags,
	"open":       lOpen,
	"remove":     lRemove,
	"remove_all": lRemoveAll,
	"stat":       lStat,
	"read_dir":   lReadDir,
	"mkdir":      lMkdir,
	"mkdir_all":  lMkdirAll,
}

func lFlags(L *lua.LState) int {
	tb := L.NewTable()
	tb.RawSetString("READ_ONLY", lua.LNumber(os.O_RDONLY))
	tb.RawSetString("WRITE_ONLY", lua.LNumber(os.O_WRONLY))
	tb.RawSetString("READ_WRITE", lua.LNumber(os.O_RDWR))
	tb.RawSetString("APPEND", lua.LNumber(os.O_APPEND))
	tb.RawSetString("CREATE", lua.LNumber(os.O_CREATE))
	tb.RawSetString("EXCL", lua.LNumber(os.O_EXCL))
	tb.RawSetString("SYNC", lua.LNumber(os.O_SYNC))
	tb.RawSetString("TRUNC", lua.LNumber(os.O_TRUNC))
	L.Push(tb)
	return 1
}

func lOpen(L *lua.LState) int {
	fs := checkFS(L)
	path := params.String()
	flag := params.Number()
	perm := params.Number()
	cb := params.Check(L, 1, 1, "fs.open(path, flag?, mode?, cb?)", path, flag, perm)

	core.GoFunctionCallback(fs.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		file, err := fs.open(path.String(), flag.Int(), perm.Int())
		if err != nil {
			return lua.LNil, err
		}

		resource := core.NewResource("*os.File: "+path.String(), func() {
			file.Close()
		})
		fs.ec.Guard(resource)

		f := &uvFile{
			File: file,
			ec:   fs.ec,
		}

		obj := object.NewProtected(L, fileFuncs, map[string]lua.LValue{}, f)
		obj.SetFunction("read", stream.NewReader(L, fs.ec, file, true))
		obj.SetFunction("write", stream.NewWriter(L, fs.ec, file, true))
		obj.SetFunction("close", stream.NewCloser(L, fs.ec, resource, file, true))

		return obj.Value(), nil
	})
	return 0
}

func lRemove(L *lua.LState) int {
	fs := checkFS(L)
	path := params.String()
	cb := params.Check(L, 1, 1, "fs.remove(path, cb?)", path)
	core.GoFunctionCallback(fs.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		err := os.Remove(path.String())
		return lua.LNil, err
	})
	return 0
}

func lRemoveAll(L *lua.LState) int {
	fs := checkFS(L)
	path := params.String()
	cb := params.Check(L, 1, 1, "fs.remove_all(path, cb?)", path)
	core.GoFunctionCallback(fs.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		err := os.RemoveAll(path.String())
		return lua.LNil, err
	})
	return 0
}

func createStat(L *lua.LState, info os.FileInfo) lua.LValue {
	t := L.NewTable()
	t.RawSetString("name", lua.LString(info.Name()))
	t.RawSetString("isdir", lua.LBool(info.IsDir()))
	t.RawSetString("mode", lua.LNumber(info.Mode()))
	t.RawSetString("size", lua.LNumber(info.Size()))
	t.RawSetString("timestamp", time.New(L, info.ModTime()).Value())
	return t
}

func lStat(L *lua.LState) int {
	fs := checkFS(L)
	path := params.String()
	cb := params.Check(L, 1, 1, "fs.stat(path, cb?)", path)
	core.GoFunctionCallback(fs.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		info, err := os.Stat(path.String())
		if err != nil {
			return lua.LNil, err
		}
		return createStat(L, info), nil
	})
	return 0
}

func lReadDir(L *lua.LState) int {
	fs := checkFS(L)
	path := params.String()
	cb := params.Check(L, 1, 1, "fs.read_dir(path, cb?)", path)
	core.GoFunctionCallback(fs.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		list, err := ioutil.ReadDir(path.String())
		if err != nil {
			return lua.LNil, err
		}
		t := L.NewTable()
		for _, info := range list {
			t.Append(createStat(L, info))
		}
		return t, nil
	})
	return 0
}

//nolint:dupl
func lMkdir(L *lua.LState) int {
	fs := checkFS(L)
	path := params.String()
	mode := params.Number(lua.LNumber(int(os.ModeDir | os.ModePerm)))
	cb := params.Check(L, 1, 1, "fs.mkdir(path, mode?, cb?)", path, mode)
	core.GoFunctionCallback(fs.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		err := os.Mkdir(path.String(), os.FileMode(mode.Int()))
		return lua.LNil, err
	})
	return 0
}

//nolint:dupl
func lMkdirAll(L *lua.LState) int {
	fs := checkFS(L)
	path := params.String()
	mode := params.Number(lua.LNumber(int(os.ModeDir | os.ModePerm)))
	cb := params.Check(L, 1, 1, "fs.mkdir_all(path, mode?, cb?)", path, mode)
	core.GoFunctionCallback(fs.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		err := os.MkdirAll(path.String(), os.FileMode(mode.Int()))
		return lua.LNil, err
	})
	return 0
}

type uvFile struct {
	File
	ec *core.ExecutionContext
}

var fileFuncs = map[string]lua.LGFunction{}
