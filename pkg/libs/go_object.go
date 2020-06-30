package libs

import (
	"errors"
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type GoObject struct {
	methods    map[string]*lua.LFunction
	properties map[string]lua.LValue
	table      *lua.LTable
	value      interface{}
	allowExtra bool
}

type GoObjectGet interface {
	GoObjectGet(key lua.LValue) (lua.LValue, bool)
}
type GoObjectSet interface {
	GoObjectSet(key, value lua.LValue) bool
}

func (obj *GoObject) MarshalJSON() (data []byte, err error) {
	var pairs []string
	if obj.table != nil {
		data, err = JSONEncode(obj.table)
		if err != nil {
			return
		}
		pairs = append(pairs, `"customProperties":`+string(data)+``)
	}
	if len(obj.properties) > 0 {
		var propertiesPairs []string
		for k, v := range obj.properties {
			data, err = JSONEncode(v)
			if err != nil {
				return
			}
			propertiesPairs = append(propertiesPairs, fmt.Sprintf(`"%s":%s`, k, string(data)))
		}
		pairs = append(pairs, `"properties":{`+strings.Join(propertiesPairs, ",")+`}`)
	}

	return []byte("{" + strings.Join(pairs, ",") + "}"), nil
}

func NewGoObject(L *lua.LState, funcs map[string]lua.LGFunction, properties map[string]lua.LValue, value interface{}, allowExtra bool) lua.LValue {
	ud := L.NewUserData()
	ud.Value = value
	methods := map[string]*lua.LFunction{}
	for name, fn := range funcs {
		methods[name] = L.NewClosure(fn, ud)
	}
	var table *lua.LTable
	if allowExtra {
		table = L.NewTable()
	}
	return newGoObject(L, ud, methods, properties, value, table)
}

func newGoObject(L *lua.LState, ud *lua.LUserData, methods map[string]*lua.LFunction, properties map[string]lua.LValue, value interface{}, table *lua.LTable) lua.LValue {
	object := &GoObject{}
	object.value = value
	object.methods = methods
	object.properties = properties
	object.allowExtra = table != nil
	object.table = table

	ud.Value = object

	meta := L.NewTable()
	meta.RawSetString("__newindex", L.NewClosure(lGoObjectSet, ud))
	meta.RawSetString("__index", L.NewClosure(lGoObjectGet, ud))
	meta.RawSetString("__metadata", lua.LBool(false))
	L.SetMetatable(ud, meta)
	return ud
}

func IsGoObject(value lua.LValue) bool {
	if value.Type() != lua.LTUserData {
		return false
	}

	ud := value.(*lua.LUserData)
	_, ok := ud.Value.(*GoObject)
	return ok
}

func GetValueFromGoObject(value lua.LValue) (interface{}, error) {
	object, err := GetGoObject(value)
	if err != nil {
		return nil, err
	}

	return object.value, nil
}

func MustGetValueFromGoObject(value lua.LValue) interface{} {
	object := MustGetGoObject(value)
	if object == nil {
		return nil
	}

	return object.value
}

func GetGoObject(value lua.LValue) (*GoObject, error) {
	if value.Type() != lua.LTUserData {
		return nil, errors.New("expected user data")
	}

	ud := value.(*lua.LUserData)
	object, ok := ud.Value.(*GoObject)
	if !ok {
		return nil, errors.New("expected GoObject")
	}

	return object, nil
}

func MustGetGoObject(value lua.LValue) *GoObject {
	if value.Type() != lua.LTUserData {
		return nil
	}

	ud := value.(*lua.LUserData)
	object, ok := ud.Value.(*GoObject)
	if !ok {
		return nil
	}

	return object
}

func CloneGoObject(L *lua.LState, lVal lua.LValue) lua.LValue {
	if lVal.Type() != lua.LTUserData {
		L.RaiseError("expected user data")
	}

	valueUD := lVal.(*lua.LUserData)
	object, ok := valueUD.Value.(*GoObject)
	if !ok {
		L.RaiseError("expected GoObject")
	}

	value := object.value
	ud := L.NewUserData()
	ud.Value = value
	methods := map[string]*lua.LFunction{}
	for name, fn := range object.methods {
		methods[name] = L.NewClosure(fn.GFunction, ud)
	}
	properties := map[string]lua.LValue{}
	for key, value := range object.properties {
		properties[key] = CloneValue(L, value)
	}
	var table *lua.LTable
	if object.allowExtra {
		table = CloneTable(L, object.table)
	}

	return newGoObject(L, ud, methods, properties, value, table)
}

func lGoObjectGet(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	object := ud.Value.(*GoObject)
	rawKey := L.Get(2)
	if getter, ok := object.value.(GoObjectGet); ok {
		result, ok := getter.GoObjectGet(rawKey)
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

func lGoObjectSet(L *lua.LState) int {
	ud := L.CheckUserData(lua.UpvalueIndex(1))
	object := ud.Value.(*GoObject)
	if setter, ok := object.value.(GoObjectSet); ok {
		if setter.GoObjectSet(L.Get(2), L.Get(3)) {
			return 0
		}
	}
	if !object.allowExtra {
		L.RaiseError("Attempt to modify read-only table")
		return 0
	}
	key := L.Get(2).String()
	if _, ok := object.properties[key]; ok {
		return 0
	}
	if _, ok := object.methods[key]; ok {
		return 0
	}
	object.table.RawSetString(key, L.Get(3))
	return 0
}
