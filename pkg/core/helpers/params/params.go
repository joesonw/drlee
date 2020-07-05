package params

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

type Type interface {
	Type() lua.LValueType
	setValue(value lua.LValue)
}

type NumberType struct {
	value lua.LNumber
}

func (NumberType) Type() lua.LValueType {
	return lua.LTNumber
}

func (t *NumberType) setValue(value lua.LValue) {
	t.value = value.(lua.LNumber)
}

func (t NumberType) Int() int {
	return int(t.value)
}

func (t NumberType) Int64() int64 {
	return int64(t.value)
}

func (t NumberType) Float64() float64 {
	return float64(t.value)
}

func Number(value ...lua.LNumber) *NumberType {
	if len(value) > 0 {
		return &NumberType{value[0]}
	}
	return &NumberType{lua.LNumber(0)}
}

type StringType struct {
	value lua.LString
}

func (StringType) Type() lua.LValueType {
	return lua.LTString
}

func (t *StringType) setValue(value lua.LValue) {
	t.value = value.(lua.LString)
}

func (t StringType) String() string {
	return t.value.String()
}

func String(value ...string) *StringType {
	if len(value) > 0 {
		return &StringType{lua.LString(value[0])}
	}
	return &StringType{lua.LString("")}
}

type BoolType struct {
	value lua.LBool
}

func (BoolType) Type() lua.LValueType {
	return lua.LTBool
}

func (t *BoolType) setValue(value lua.LValue) {
	t.value = value.(lua.LBool)
}

func (t BoolType) Bool() bool {
	return bool(t.value)
}

func Bool(value ...bool) *BoolType {
	if len(value) > 0 {
		return &BoolType{lua.LBool(value[0])}
	}
	return &BoolType{lua.LBool(false)}
}

type FunctionType struct {
	value *lua.LFunction
}

func (FunctionType) Type() lua.LValueType {
	return lua.LTFunction
}

func (t *FunctionType) setValue(value lua.LValue) {
	t.value = value.(*lua.LFunction)
}

func (t *FunctionType) Value() *lua.LFunction {
	return t.value
}

func Function(value ...*lua.LFunction) *FunctionType {
	if len(value) > 0 {
		return &FunctionType{value[0]}
	}
	return &FunctionType{nil}
}

type TableType struct {
	value *lua.LTable
}

func (TableType) Type() lua.LValueType {
	return lua.LTTable
}

func (t *TableType) setValue(value lua.LValue) {
	t.value = value.(*lua.LTable)
}

func (t TableType) Table() *lua.LTable {
	return t.value
}

func Table(value ...*lua.LTable) *TableType {
	if len(value) > 0 {
		return &TableType{value[0]}
	}
	return &TableType{nil}
}

type UserDataType struct {
	value *lua.LUserData
}

func (UserDataType) Type() lua.LValueType {
	return lua.LTUserData
}

func (t *UserDataType) setValue(value lua.LValue) {
	t.value = value.(*lua.LUserData)
}

func (t UserDataType) Value() *lua.LUserData {
	return t.value
}

func UserData() *UserDataType {
	return &UserDataType{nil}
}

func Check(L *lua.LState, startIndex, requiredLength int, msg string, types ...Type) lua.LValue {
	n := L.GetTop()
	cb := lua.LNil
	if L.Get(n).Type() == lua.LTFunction { // last parameter is callback
		cb = L.Get(n)
		n--
	}
	n -= startIndex - 1
	if n < requiredLength {
		L.RaiseError(fmt.Sprintf("%s requires at least %d arguments", msg, requiredLength))
	}

	for i := 0; i < n; i++ {
		idx := startIndex + i
		val := L.Get(idx)
		if val.Type() != types[i].Type() {
			L.RaiseError(fmt.Sprintf("bad arguments for %s: argument %d should be of type %s, but have %s", msg, idx, types[i].Type().String(), val.Type().String()))
		} else {
			types[i].setValue(val)
		}
	}

	return cb
}
