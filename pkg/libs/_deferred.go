package libs

import (
	"sync"

	lua "github.com/yuin/gopher-lua"
)

const (
	dPENDING  = 0
	dRESOLVED = 1
	dREJECTED = 2
)

var deferredFuncs = map[string]lua.LGFunction{
	"deferred_new":      deferredNew,
	"deferred_resolved": deferredResolved,
	"deferred_rejected": deferredRejected,
	"deferred_first":    deferredFirst,
	"deferred_all":      deferredAll,
	"deferred_map":      timeSleep,
}

type DeferredCallback func(err, result lua.LValue) (nextResult lua.LValue, nextErr lua.LValue)
type DeferredResolve func(err, result lua.LValue)

func OpenDeferred(L *lua.LState) {
	RegisterGlobalFuncs(L, deferredFuncs)
	L.SetGlobal("dPENDING", lua.LNumber(dPENDING))
	L.SetGlobal("dRESOLVED", lua.LNumber(dRESOLVED))
	L.SetGlobal("dREJECTED", lua.LNumber(dREJECTED))
}

func deferredNew(L *lua.LState) int {
	d := NewDeferred(L)
	L.Push(d.Value())
	return 1
}

func deferredResolved(L *lua.LState) int {
	d := NewResolvedDeferred(L, L.Get(1))
	L.Push(d.Value())
	return 1
}

func deferredRejected(L *lua.LState) int {
	d := NewRejectedDeferred(L, L.Get(1))
	L.Push(d.Value())
	return 1
}

func deferredAll(L *lua.LState) int {
	table := L.CheckTable(1)
	d := NewDeferred(L)
	L.Push(d.Value())
	var defers []*lDeferred
	n := table.Len()
	for i := 1; i <= n; i++ {
		value := table.RawGetInt(i)
		in, err := GetValueFromGoObject(value)
		if err != nil {
			d.Reject(L, Error(err))
			return 1
		}
		ld, ok := in.(*lDeferred)
		if !ok {
			d.Reject(L, lua.LString("expected deferred"))
			return 1
		}
		defers = append(defers, ld)
	}
	var resultList []lua.LValue
	wg := &sync.WaitGroup{}
	for _, ld := range defers {
		wg.Add(1)
		go func(ld *lDeferred) {
			ld.next(L, func(err, result lua.LValue) (nextResult lua.LValue, nextErr lua.LValue) {
				wg.Done()
				if err != lua.LNil {
					d.Reject(L, err)
					return
				}
				resultList = append(resultList, result)
				return
			})
		}(ld)
	}

	go func(d *lDeferred) {
		wg.Wait()
		list := L.NewTable()
		for _, r := range resultList {
			list.Append(r)
		}
		d.Resolve(L, list)
	}(d)

	return 1
}

func deferredFirst(L *lua.LState) int {
	table := L.CheckTable(1)
	d := NewDeferred(L)
	L.Push(d.Value())
	var defers []*lDeferred
	n := table.Len()
	for i := 1; i <= n; i++ {
		value := table.RawGetInt(i)
		in, err := GetValueFromGoObject(value)
		if err != nil {
			d.Reject(L, Error(err))
			return 1
		}
		ld, ok := in.(*lDeferred)
		if !ok {
			d.Reject(L, lua.LString("expected deferred"))
			return 1
		}
		defers = append(defers, ld)
	}
	for _, ld := range defers {
		go func(ld *lDeferred) {
			ld.next(L, func(err, result lua.LValue) (nextResult lua.LValue, nextErr lua.LValue) {
				if err != lua.LNil {
					d.Reject(L, err)
				} else {
					d.Resolve(L, result)
				}
				return
			})
		}(ld)
	}

	return 1
}

func resolveDefer(L *lua.LState, next *lDeferred, value lua.LValue, cb DeferredResolve) {
	if v, ok := MustGetValueFromGoObject(value).(*lDeferred); ok && v == next {
		cb(lua.LString("chained self call"), lua.LNil)
		return
	}

	if v, ok := MustGetValueFromGoObject(value).(*lDeferred); ok {
		v.next(L, func(err, result lua.LValue) (nextResult lua.LValue, nextErr lua.LValue) {
			if err != lua.LNil {
				cb(err, lua.LNil)
			} else {
				resolveDefer(L, next, result, cb)
			}
			return
		})
	} else {
		cb(lua.LNil, value)
	}
}

type lDeferred struct {
	state  int
	queue  []func()
	value  lua.LValue
	lValue lua.LValue
	object *GoObject
}

func NewDeferred(L *lua.LState) *lDeferred {
	d := &lDeferred{
		state: dPENDING,
	}
	d.lValue = NewGoObject(L, map[string]lua.LGFunction{
		"next":    deferredDeferredNext,
		"resolve": deferredDeferredResolve,
		"reject":  deferredDeferredReject,
	}, d.properties(), d, false)
	d.object, _ = GetGoObject(d.lValue)
	return d
}

func (d *lDeferred) Resolve(L *lua.LState, result lua.LValue) {
	if d.state == dPENDING {
		d.state = dRESOLVED
		d.value = result
		d.object.properties = d.properties()
		l := GetContextLock(L.Context())
		l.Lock()
		defer l.Unlock()
		for _, cb := range d.queue {
			cb()
		}
	}
}

func (d *lDeferred) Reject(L *lua.LState, err lua.LValue) {
	if d.state == dPENDING {
		d.state = dREJECTED
		d.value = err
		d.object.properties = d.properties()
		l := GetContextLock(L.Context())
		l.Lock()
		defer l.Unlock()
		for _, cb := range d.queue {
			cb()
		}
	}
}

func (d *lDeferred) properties() map[string]lua.LValue {
	return map[string]lua.LValue{"state": lua.LNumber(d.state), "value": d.value}
}

func (d *lDeferred) Value() lua.LValue {
	return d.lValue
}

func (d *lDeferred) next(L *lua.LState, cb DeferredCallback) int {
	next := NewDeferred(L)
	if next.state == dRESOLVED {
		result, err := cb(lua.LNil, d.value)
		if err != lua.LNil {
			next.Reject(L, err)
		} else {
			resolveDefer(L, next, result, func(err, result lua.LValue) {
				if err != lua.LNil {
					next.Reject(L, err)
				} else {
					next.Resolve(L, result)
				}
			})
		}
	} else if next.state == dREJECTED {
		result, err := cb(d.value, lua.LNil)
		if err != lua.LNil {
			next.Reject(L, err)
		} else {
			resolveDefer(L, next, result, func(err, result lua.LValue) {
				if err != lua.LNil {
					next.Reject(L, err)
				} else {
					next.Resolve(L, result)
				}
			})
		}
	} else {
		d.queue = append(d.queue, func() {
			if d.state == dREJECTED {
				result, err := cb(d.value, lua.LNil)
				if err != lua.LNil {
					next.Reject(L, err)
				} else {
					resolveDefer(L, next, result, func(err, result lua.LValue) {
						if err != lua.LNil {
							next.Reject(L, err)
						} else {
							next.Resolve(L, result)
						}
					})
				}
			} else {
				result, err := cb(lua.LNil, d.value)
				if err != lua.LNil {
					next.Reject(L, err)
				} else {
					resolveDefer(L, next, result, func(err, result lua.LValue) {
						if err != lua.LNil {
							next.Reject(L, err)
						} else {
							next.Resolve(L, result)
						}
					})
				}
			}
			return
		})
	}
	L.Push(next.Value())
	return 1
}

func NewResolvedDeferred(L *lua.LState, val lua.LValue) *lDeferred {
	d := NewDeferred(L)
	d.Resolve(L, val)
	return d
}

func NewRejectedDeferred(L *lua.LState, err lua.LValue) *lDeferred {
	d := NewDeferred(L)
	d.Reject(L, err)
	return d
}

func upDeferred(L *lua.LState) *lDeferred {
	deferred, err := GetValueFromGoObject(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return deferred.(*lDeferred)
}

func deferredDeferredNext(L *lua.LState) int {
	d := upDeferred(L)
	cb := L.CheckFunction(2)
	return d.next(L, func(err, result lua.LValue) (returnVal lua.LValue, pError lua.LValue) {
		e := L.CallByParam(
			lua.P{
				Fn:   cb,
				NRet: 1,
			}, err, result,
		)
		if e != nil {
			pError = lua.LString(e.Error())
		} else {
			returnVal = L.Get(L.GetTop())
			pError = lua.LNil
		}
		return
	})
}

func deferredDeferredResolve(L *lua.LState) int {
	d := upDeferred(L)
	d.Resolve(L, L.Get(2))
	return 0
}

func deferredDeferredReject(L *lua.LState) int {
	d := upDeferred(L)
	d.Reject(L, L.Get(2))
	return 0
}
