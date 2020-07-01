package libs

import lua "github.com/yuin/gopher-lua"

type AsyncTask interface {
	Execute(L *lua.LState)
}

type AsyncFunction func(L *lua.LState)

func (f AsyncFunction) Execute(L *lua.LState) {
	f(L)
}

type AsyncStack struct {
	L    *lua.LState
	ch   chan AsyncTask
	exit chan struct{}
}

func NewAsyncStack(L *lua.LState, size int) *AsyncStack {
	return &AsyncStack{
		L:    L,
		ch:   make(chan AsyncTask, size),
		exit: make(chan struct{}, 1),
	}
}

func (s *AsyncStack) Start() {
	go func() {
		for {
			select {
			case <-s.exit:
				return
			case task := <-s.ch:
				task.Execute(s.L)
			}
		}
	}()
}

func (s *AsyncStack) Tick(L *lua.LState) {
	task := <-s.ch
	task.Execute(L)
}

func (s *AsyncStack) Stop() {
	s.exit <- struct{}{}
}

func (s *AsyncStack) Enqueue(task AsyncTask) {
	s.ch <- task
}
