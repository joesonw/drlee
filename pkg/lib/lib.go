package lib

import lua "github.com/yuin/gopher-lua"

func Open(L *lua.LState) error {
	scripts := []string{
		lAsyncParallel,
		lAsyncSeries,
	}

	for _, src := range scripts {
		if err := L.DoString(src); err != nil {
			return err
		}
	}

	return nil
}
