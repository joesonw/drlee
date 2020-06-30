package libs

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	lua "github.com/yuin/gopher-lua"
)

var (
	errNested      = errors.New("unable to encode recursively nested tables to JSON")
	errSparseArray = errors.New("unable to encode sparse array")
	errInvalidKeys = errors.New("unable to encode mixed or invalid key types")
	jsonFuncs      = map[string]lua.LGFunction{
		"json_decode": lJSONDecode,
		"json_encode": lJSONEncode,
	}
)

type jsonInvalidTypeError lua.LValueType

func (e jsonInvalidTypeError) Error() string {
	return `unable to encode ` + lua.LValueType(e).String() + ` to JSON`
}

func JSONEncode(value lua.LValue) ([]byte, error) {
	return json.Marshal(jsonValue{
		LValue:  value,
		visited: make(map[*lua.LTable]bool),
	})
}

type jsonValue struct {
	lua.LValue
	visited map[*lua.LTable]bool
}

func (j jsonValue) MarshalJSON() (data []byte, err error) {
	switch converted := j.LValue.(type) {
	case lua.LBool:
		data, err = json.Marshal(bool(converted))
	case lua.LNumber:
		data, err = json.Marshal(float64(converted))
	case *lua.LNilType:
		data = []byte(`null`)
	case lua.LString:
		data, err = json.Marshal(string(converted))
	case *lua.LUserData:
		if stringer, ok := converted.Value.(fmt.Stringer); ok {
			data = []byte(`"` + stringer.String() + `"`)
		} else if marshaller, ok := converted.Value.(json.Marshaler); ok {
			data, err = marshaller.MarshalJSON()
		} else {
			data = []byte(`"*USERDATA*"`)
		}
	case *lua.LTable:
		if j.visited[converted] {
			return nil, errNested
		}
		j.visited[converted] = true

		key, value := converted.Next(lua.LNil)

		switch key.Type() {
		case lua.LTNil: // empty table
			data = []byte(`[]`)
		case lua.LTNumber:
			arr := make([]jsonValue, 0, converted.Len())
			expectedKey := lua.LNumber(1)
			for key != lua.LNil {
				if key.Type() != lua.LTNumber {
					err = errInvalidKeys
					return
				}
				if expectedKey != key {
					err = errSparseArray
					return
				}
				arr = append(arr, jsonValue{value, j.visited})
				expectedKey++
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(arr)
		case lua.LTString:
			obj := make(map[string]jsonValue)
			for key != lua.LNil {
				if key.Type() != lua.LTString {
					err = errInvalidKeys
					return
				}
				obj[key.String()] = jsonValue{value, j.visited}
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(obj)
		default:
			err = errInvalidKeys
		}
	default:
		err = jsonInvalidTypeError(j.LValue.Type())
	}
	return
}

func JSONDecode(L *lua.LState, data []byte) (lua.LValue, error) {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return JSONDecodeValue(L, value), nil
}

func JSONDecodeValue(L *lua.LState, value interface{}) lua.LValue {
	switch converted := value.(type) {
	case bool:
		return lua.LBool(converted)
	case float64:
		return lua.LNumber(converted)
	case int64:
		return lua.LNumber(converted)
	case time.Time:
		return lua.LNumber(converted.UnixNano() / 1000000)
	case string:
		return lua.LString(converted)
	case json.Number:
		return lua.LString(converted)
	case []interface{}:
		arr := L.CreateTable(len(converted), 0)
		for _, item := range converted {
			arr.Append(JSONDecodeValue(L, item))
		}
		return arr
	case map[string]interface{}:
		tbl := L.CreateTable(0, len(converted))
		for key, item := range converted {
			tbl.RawSetH(lua.LString(key), JSONDecodeValue(L, item))
		}
		return tbl
	case nil:
		return lua.LNil
	}

	return lua.LNil
}

func OpenJSON(L *lua.LState) {
	RegisterGlobalFuncs(L, jsonFuncs)
}

func lJSONDecode(L *lua.LState) int {
	if L.GetTop() != 1 {
		L.ArgError(1, "json.decode(string): takes one argument")
		return 0
	}

	val, err := JSONDecode(L, []byte(L.Get(1).String()))
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}

	L.Push(val)
	return 1
}

func lJSONEncode(L *lua.LState) int {
	if L.GetTop() != 1 {
		L.ArgError(1, "json.encode(table): takes one argument")
		return 0
	}

	bytes, err := JSONEncode(L.Get(1))
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}

	L.Push(lua.LString(string(bytes)))
	return 1
}
