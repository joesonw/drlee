package object

import (
	"errors"
	"fmt"
	"strings"

	"github.com/joesonw/drlee/pkg/core/helpers"
	"github.com/joesonw/drlee/pkg/core/json"
	lua "github.com/yuin/gopher-lua"
)

type Object struct {
	methods    map[string]*lua.LFunction
	properties map[string]lua.LValue
	table      *lua.LTable
	goValue    interface{}
	readonly   bool
	protected  bool
	userData   lua.LValue
}

func (obj *Object) Value() lua.LValue {
	return obj.userData
}

func (obj *Object) SetProperty(key string, value lua.LValue) {
	obj.properties[key] = value
}

func (obj *Object) SetFunction(key string, fn *lua.LFunction) {
	obj.methods[key] = fn
}

// AddMethod add method using object upvalue
func (obj *Object) SetMethod(L *lua.LState, key string, fn lua.LGFunction) {
	obj.methods[key] = L.NewClosure(fn, obj.userData)
}

type CanGet interface {
	ObjectGet(key lua.LValue) (lua.LValue, bool)
}
type CanSet interface {
	ObjectSet(key, value lua.LValue) bool
}

func (obj *Object) MarshalJSON() (data []byte, err error) {
	var pairs []string
	if obj.table != nil {
		data, err = json.Encode(obj.table)
		if err != nil {
			return
		}
		pairs = append(pairs, `"customProperties":`+string(data)+``)
	}
	if len(obj.properties) > 0 {
		var propertiesPairs []string
		for k, v := range obj.properties {
			data, err = json.Encode(v)
			if err != nil {
				return
			}
			propertiesPairs = append(propertiesPairs, fmt.Sprintf(`"%s":%s`, k, string(data)))
		}
		pairs = append(pairs, `"properties":{`+strings.Join(propertiesPairs, ",")+`}`)
	}

	return []byte("{" + strings.Join(pairs, ",") + "}"), nil
}

func New(L *lua.LState, funcs map[string]lua.LGFunction, properties map[string]lua.LValue, value interface{}) *Object {
	ud := L.NewUserData()
	ud.Value = value
	methods := map[string]*lua.LFunction{}
	for name, fn := range funcs {
		methods[name] = L.NewClosure(fn, ud)
	}
	return newObject(L, ud, methods, properties, value, L.NewTable(), false, false)
}

func NewReadOnly(L *lua.LState, funcs map[string]lua.LGFunction, properties map[string]lua.LValue, value interface{}) *Object {
	ud := L.NewUserData()
	ud.Value = value
	methods := map[string]*lua.LFunction{}
	for name, fn := range funcs {
		methods[name] = L.NewClosure(fn, ud)
	}
	return newObject(L, ud, methods, properties, value, nil, true, false)
}

func NewProtected(L *lua.LState, funcs map[string]lua.LGFunction, properties map[string]lua.LValue, value interface{}) *Object {
	ud := L.NewUserData()
	ud.Value = value
	methods := map[string]*lua.LFunction{}
	for name, fn := range funcs {
		methods[name] = L.NewClosure(fn, ud)
	}
	return newObject(L, ud, methods, properties, value, L.NewTable(), false, true)
}

func newObject(L *lua.LState, ud *lua.LUserData, methods map[string]*lua.LFunction, properties map[string]lua.LValue, value interface{}, table *lua.LTable, readonly, protected bool) *Object {
	object := &Object{}
	object.goValue = value
	object.methods = methods
	object.properties = properties
	object.readonly = readonly
	object.protected = protected
	object.table = table

	ud.Value = object

	meta := L.NewTable()
	meta.RawSetString("__newindex", L.NewClosure(lSet, ud))
	meta.RawSetString("__index", L.NewClosure(lGet, ud))
	meta.RawSetString("__metadata", lua.LBool(false))
	L.SetMetatable(ud, meta)
	object.userData = ud

	return object
}

func Is(value lua.LValue) bool {
	if value.Type() != lua.LTUserData {
		return false
	}

	ud := value.(*lua.LUserData)
	_, ok := ud.Value.(*Object)
	return ok
}

func Value(value lua.LValue) (interface{}, error) {
	object, err := Get(value)
	if err != nil {
		return nil, err
	}

	return object.goValue, nil
}

func MustValue(value lua.LValue) interface{} {
	object := MustGet(value)
	if object == nil {
		return nil
	}

	return object.goValue
}

func Get(value lua.LValue) (*Object, error) {
	if value.Type() != lua.LTUserData {
		return nil, errors.New("expected user data")
	}

	ud := value.(*lua.LUserData)
	object, ok := ud.Value.(*Object)
	if !ok {
		return nil, errors.New("expected GoObject")
	}

	return object, nil
}

func MustGet(value lua.LValue) *Object {
	if value.Type() != lua.LTUserData {
		return nil
	}

	ud := value.(*lua.LUserData)
	object, ok := ud.Value.(*Object)
	if !ok {
		return nil
	}

	return object
}

func Clone(L *lua.LState, lVal lua.LValue) lua.LValue {
	if lVal.Type() != lua.LTUserData {
		L.RaiseError("expected user data")
	}

	valueUD := lVal.(*lua.LUserData)
	object, ok := valueUD.Value.(*Object)
	if !ok {
		L.RaiseError("expected GoObject")
	}

	value := object.goValue
	ud := L.NewUserData()
	ud.Value = value
	methods := map[string]*lua.LFunction{}
	for name, fn := range object.methods {
		methods[name] = L.NewClosure(fn.GFunction, ud)
	}
	properties := map[string]lua.LValue{}
	for key, value := range object.properties {
		properties[key] = helpers.Clone(L, value)
	}

	var table *lua.LTable
	if !object.readonly {
		table = helpers.Clone(L, object.table).(*lua.LTable)
	}

	newObject := newObject(L, ud, methods, properties, value, table, object.readonly, object.protected)
	return newObject.Value()
}

func lGet(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	object := ud.Value.(*Object)
	rawKey := L.Get(2)
	if getter, ok := object.goValue.(CanGet); ok {
		result, ok := getter.ObjectGet(rawKey)
		if ok {
			L.Push(result)
			return 1
		}
	}
	key := rawKey.String()
	if value, ok := object.properties[key]; ok {
		L.Push(value)
	} else if fn, ok := object.methods[key]; ok {
		L.Push(fn)
	} else {
		L.Push(object.table.RawGet(rawKey))
	}
	return 1
}

func lSet(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	object := ud.Value.(*Object)
	if setter, ok := object.goValue.(CanSet); ok {
		if setter.ObjectSet(L.Get(2), L.Get(3)) {
			return 0
		}
	}
	if object.readonly {
		L.RaiseError("Attempt to modify read-only value")
		return 0
	}
	key := L.Get(2).String()
	if _, ok := object.properties[key]; ok {
		if object.protected {
			L.RaiseError("Attempt to modify protected properties")
		}
		return 0
	}
	if _, ok := object.methods[key]; ok {
		if object.protected {
			L.RaiseError("Attempt to modify protected methods")
		}
		return 0
	}
	object.table.RawSetString(key, L.Get(3))
	return 0
}
