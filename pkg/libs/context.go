package libs

import (
	"context"
	"sync"
)

type (
	contextKeyLock           struct{}
	contextKeyParentCallback struct{}
	contextKeyCallbackStack  struct{}
)

type CallbackStack chan *Callback

func NewContext(parent context.Context, callbackStack CallbackStack) context.Context {
	l := &sync.RWMutex{}
	return context.WithValue(context.WithValue(parent, contextKeyLock{}, l), contextKeyCallbackStack{}, callbackStack)
}

func GetContextCallbackStack(ctx context.Context) CallbackStack {
	s, _ := ctx.Value(contextKeyCallbackStack{}).(CallbackStack)
	return s
}

func GetContextLock(ctx context.Context) *sync.RWMutex {
	l, _ := ctx.Value(contextKeyLock{}).(*sync.RWMutex)
	return l
}

func FromContext(lua, parent context.Context) context.Context {
	lock := GetContextLock(lua)
	return context.WithValue(parent, contextKeyLock{}, lock)
}

func WithRecover(parent context.Context, cb func(error)) context.Context {
	return context.WithValue(parent, contextKeyParentCallback{}, cb)
}

func GetContextRecovery(ctx context.Context) func(error) {
	f, _ := ctx.Value(contextKeyParentCallback{}).(func(error))
	return f
}
