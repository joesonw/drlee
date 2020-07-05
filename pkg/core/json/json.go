package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/joesonw/drlee/pkg/utils"
	lua "github.com/yuin/gopher-lua"
)

var (
	ErrNested      = errors.New("unable to encode recursively nested tables to JSON")
	ErrSparseArray = errors.New("unable to encode sparse array")
	ErrInvalidKeys = errors.New("unable to encode mixed or invalid key types")
	funcs          = map[string]lua.LGFunction{
		"json_decode": lJSONDecode,
		"json_encode": lJSONEncode,
	}
)

type invalidTypeError lua.LValueType

func (e invalidTypeError) Error() string {
	return `unable to encode ` + lua.LValueType(e).String() + ` to JSON`
}

func Encode(value lua.LValue) ([]byte, error) {
	if value == nil || value == lua.LNil {
		return nil, nil
	}
	return json.Marshal(Value{
		LValue:  value,
		visited: make(map[*lua.LTable]bool),
	})
}

type Value struct {
	lua.LValue
	visited map[*lua.LTable]bool
}

//nolint:gocyclo,funlen
func (j Value) MarshalJSON() (data []byte, err error) {
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
		switch v := converted.Value.(type) {
		case fmt.Stringer:
			data = []byte(`"` + v.String() + `"`)
		case json.Marshaler:
			data, err = v.MarshalJSON()
		default:
			data = []byte(`"*USERDATA*"`)
		}
	case *lua.LTable:
		if j.visited[converted] {
			return nil, ErrNested
		}
		j.visited[converted] = true

		key, value := converted.Next(lua.LNil)

		switch key.Type() {
		case lua.LTNil: // empty table
			data = []byte(`[]`)
		case lua.LTNumber:
			arr := make([]Value, 0, converted.Len())
			expectedKey := lua.LNumber(1)
			for key != lua.LNil {
				if key.Type() != lua.LTNumber {
					err = ErrInvalidKeys
					return
				}
				if expectedKey != key {
					err = ErrSparseArray
					return
				}
				arr = append(arr, Value{value, j.visited})
				expectedKey++
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(arr)
		case lua.LTString:
			obj := make(map[string]Value)
			for key != lua.LNil {
				if key.Type() != lua.LTString {
					err = ErrInvalidKeys
					return
				}
				obj[key.String()] = Value{value, j.visited}
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(obj)
		default:
			err = ErrInvalidKeys
		}
	default:
		err = invalidTypeError(j.LValue.Type())
	}
	return data, err
}

func Decode(L *lua.LState, data []byte) (lua.LValue, error) {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return DecodeValue(L, value), nil
}

func DecodeValue(L *lua.LState, value interface{}) lua.LValue {
	if value == nil {
		return lua.LNil
	}
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
			arr.Append(DecodeValue(L, item))
		}
		return arr
	case map[string]interface{}:
		tbl := L.CreateTable(0, len(converted))
		for key, item := range converted {
			tbl.RawSetH(lua.LString(key), DecodeValue(L, item))
		}
		return tbl
	case nil:
		return lua.LNil
	}

	return lua.LNil
}

func Open(L *lua.LState) {
	utils.RegisterGlobalFuncs(L, funcs)
}

func lJSONDecode(L *lua.LState) int {
	if L.GetTop() != 1 {
		L.ArgError(1, "json.decode(string): takes one argument")
		return 0
	}

	val, err := Decode(L, []byte(L.Get(1).String()))
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

	bytes, err := Encode(L.Get(1))
	if err != nil {
		L.ArgError(1, err.Error())
		return 0
	}

	L.Push(lua.LString(string(bytes)))
	return 1
}
