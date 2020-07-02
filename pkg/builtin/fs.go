package builtin

import (
	"io"
	"io/ioutil"
	"os"

	lua "github.com/yuin/gopher-lua"
)

type File interface {
	io.Closer
	io.Reader
	io.Writer
}

func OpenFS(L *lua.LState, env *Env) {
	ud := L.NewUserData()
	ud.Value = &lFS{
		open: env.OpenFile,
	}
	RegisterGlobalFuncs(L, fsFuncs, ud)
}

type lFS struct {
	open OpenFile
}

func upFS(L *lua.LState) *lFS {
	uv := L.Get(lua.UpvalueIndex(1)).(*lua.LUserData)
	if fs, ok := uv.Value.(*lFS); ok {
		return fs
	}

	L.RaiseError("expected fs")
	return nil
}

var fsFuncs = map[string]lua.LGFunction{
	"fs_open":      lFSOpen,
	"fs_remove":    lFSRemove,
	"fs_removeall": lFSRemoveAll,
	"fs_stat":      lFSStat,
	"fs_readdir":   lFSReaddir,
	"fs_mkdir":     lFSMkdir,
	"fs_mkdirall":  lFSMkdirAll,
}

func lFSOpen(L *lua.LState) int {
	fs := upFS(L)
	name := L.CheckString(1)
	cbValue := L.Get(2)
	flag := 0
	perm := 0
	if cbValue.Type() == lua.LTNumber {
		flag = int(L.CheckNumber(2))
		cbValue = L.Get(3)

		if cbValue.Type() == lua.LTNumber {
			perm = int(L.CheckNumber(3))
			cbValue = L.Get(4)
		}
	}
	cb := NewCallback(cbValue)
	go func() {
		file, err := fs.open(name, flag, perm)
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Resolve(L, NewGoObject(L, fileFuncs, map[string]lua.LValue{}, &lFile{File: file}, false))
		}
	}()
	return 0
}

func lFSRemove(L *lua.LState) int {
	name := L.CheckString(1)
	cb := NewCallback(L.Get(2))
	go func() {
		err := os.Remove(name)
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Finish(L)
		}
	}()
	return 0
}

func lFSRemoveAll(L *lua.LState) int {
	name := L.CheckString(1)
	cb := NewCallback(L.Get(2))
	go func() {
		err := os.RemoveAll(name)
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Finish(L)
		}
	}()
	return 0
}

func lOSFileInfoToTable(L *lua.LState, info os.FileInfo) lua.LValue {
	t := L.NewTable()
	t.RawSetString("name", lua.LString(info.Name()))
	t.RawSetString("isdir", lua.LBool(info.IsDir()))
	t.RawSetString("mode", lua.LNumber(info.Mode()))
	t.RawSetString("size", lua.LNumber(info.Size()))
	t.RawSetString("timestamp", NewTimestamp(L, info.ModTime()))
	return t
}

func lFSStat(L *lua.LState) int {
	name := L.CheckString(1)
	cb := NewCallback(L.Get(2))
	go func() {
		info, err := os.Stat(name)
		if err != nil {
			cb.Reject(L, Error(err))
			return
		}

		cb.Resolve(L, lOSFileInfoToTable(L, info))
	}()
	return 0
}

func lFSReaddir(L *lua.LState) int {
	name := L.CheckString(1)
	cb := NewCallback(L.Get(2))
	go func() {
		list, err := ioutil.ReadDir(name)
		if err != nil {
			cb.Reject(L, Error(err))
			return
		}
		t := L.NewTable()
		for _, info := range list {
			t.Append(lOSFileInfoToTable(L, info))
		}
		cb.Resolve(L, t)
	}()
	return 0
}

func lFSMkdir(L *lua.LState) int {
	name := L.CheckString(1)
	cbValue := L.Get(2)
	fileMode := os.ModeDir | os.ModePerm
	if cbValue.Type() == lua.LTNumber {
		cbValue = L.Get(3)
		fileMode = os.FileMode(L.CheckNumber(2))
	}
	cb := NewCallback(cbValue)
	go func() {
		err := os.Mkdir(name, fileMode)
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Finish(L)
		}
	}()
	return 0
}

func lFSMkdirAll(L *lua.LState) int {
	name := L.CheckString(1)
	cbValue := L.Get(2)
	fileMode := os.ModeDir | os.ModePerm
	if cbValue.Type() == lua.LTNumber {
		cbValue = L.Get(3)
		fileMode = os.FileMode(L.CheckNumber(2))
	}
	cb := NewCallback(cbValue)
	go func() {
		err := os.MkdirAll(name, fileMode)
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Finish(L)
		}
	}()
	return 0
}

func upFile(L *lua.LState) *lFile {
	f, err := GetValueFromGoObject(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return f.(*lFile)
}

type lFile struct {
	File
}

var fileFuncs = map[string]lua.LGFunction{
	"read":    lFileRead,
	"readall": lFileReadAll,
	"write":   lFileWrite,
	"close":   lFileClose,
}

func lFileRead(L *lua.LState) int {
	file := upFile(L)
	size := L.CheckNumber(2)
	cb := NewCallback(L.Get(3))
	go func() {
		b := make([]byte, int(size))
		_, err := file.Read(b)
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Resolve(L, lua.LString(b))
		}
	}()
	return 0
}

func lFileReadAll(L *lua.LState) int {
	file := upFile(L)
	cb := NewCallback(L.Get(2))
	go func() {
		b, err := ioutil.ReadAll(file)
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Resolve(L, lua.LString(b))
		}
	}()
	return 0
}

func lFileWrite(L *lua.LState) int {
	file := upFile(L)
	content := L.CheckString(2)
	cb := NewCallback(L.Get(3))
	go func() {
		_, err := file.Write([]byte(content))
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Finish(L)
		}
	}()
	return 0
}

func lFileClose(L *lua.LState) int {
	file := upFile(L)
	cb := NewCallback(L.Get(2))
	go func() {
		err := file.Close()
		if err != nil {
			cb.Reject(L, Error(err))
		} else {
			cb.Finish(L)
		}
	}()
	return 0
}
