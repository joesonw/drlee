package libs

import lua "github.com/yuin/gopher-lua"

type AsyncTask interface {
	Execute(L *lua.LState, recovery func(err error))
}

type AsyncFunction func(L *lua.LState, recovery func(err error))

func (f AsyncFunction) Execute(L *lua.LState, recovery func(err error)) {
	f(L, recovery)
}

type AsyncStack struct {
	L        *lua.LState
	ch       chan AsyncTask
	exit     chan struct{}
	recovery func(err error)
}

func NewAsyncStack(L *lua.LState, size int, recovery func(err error)) *AsyncStack {
	return &AsyncStack{
		L:        L,
		ch:       make(chan AsyncTask, size),
		exit:     make(chan struct{}, 1),
		recovery: recovery,
	}
}

func (s *AsyncStack) Start() {
	go func() {
		for {
			select {
			case <-s.exit:
				return
			case task := <-s.ch:
				task.Execute(s.L, s.recovery)
			}
		}
	}()
}

func (s *AsyncStack) Tick(L *lua.LState, recovery func(err error)) {
	task := <-s.ch
	task.Execute(L, recovery)
}

func (s *AsyncStack) Stop() {
	s.exit <- struct{}{}
}

func (s *AsyncStack) Enqueue(task AsyncTask) {
	s.ch <- task
}
