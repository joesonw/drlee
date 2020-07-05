package params

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

type Type interface {
	Type() lua.LValueType
	setValue(value lua.LValue)
}

type numberType struct {
	value lua.LNumber
}

func (numberType) Type() lua.LValueType {
	return lua.LTNumber
}

func (t *numberType) setValue(value lua.LValue) {
	t.value = value.(lua.LNumber)
}

func (t numberType) Int() int {
	return int(t.value)
}

func (t numberType) Int64() int64 {
	return int64(t.value)
}

func (t numberType) Float64() float64 {
	return float64(t.value)
}

func Number(value ...lua.LNumber) *numberType {
	if len(value) > 0 {
		return &numberType{value[0]}
	}
	return &numberType{lua.LNumber(0)}
}

type stringType struct {
	value lua.LString
}

func (stringType) Type() lua.LValueType {
	return lua.LTString
}

func (t *stringType) setValue(value lua.LValue) {
	t.value = value.(lua.LString)
}

func (t stringType) String() string {
	return t.value.String()
}

func String(value ...string) *stringType {
	if len(value) > 0 {
		return &stringType{lua.LString(value[0])}
	}
	return &stringType{lua.LString("")}
}

type boolType struct {
	value lua.LBool
}

func (boolType) Type() lua.LValueType {
	return lua.LTBool
}

func (t *boolType) setValue(value lua.LValue) {
	t.value = value.(lua.LBool)
}

func (t boolType) Bool() bool {
	return bool(t.value)
}

func Bool(value ...bool) *boolType {
	if len(value) > 0 {
		return &boolType{lua.LBool(value[0])}
	}
	return &boolType{lua.LBool(false)}
}

type functionType struct {
	value *lua.LFunction
}

func (functionType) Type() lua.LValueType {
	return lua.LTFunction
}

func (t *functionType) setValue(value lua.LValue) {
	t.value = value.(*lua.LFunction)
}

func (t *functionType) Value() *lua.LFunction {
	return t.value
}

func Function(value ...*lua.LFunction) *functionType {
	if len(value) > 0 {
		return &functionType{value[0]}
	}
	return &functionType{nil}
}

type tableType struct {
	value *lua.LTable
}

func (tableType) Type() lua.LValueType {
	return lua.LTTable
}

func (t *tableType) setValue(value lua.LValue) {
	t.value = value.(*lua.LTable)
}

func (t tableType) Table() *lua.LTable {
	return t.value
}

func Table(value ...*lua.LTable) *tableType {
	if len(value) > 0 {
		return &tableType{value[0]}
	}
	return &tableType{nil}
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
		n = n - 1
	}
	n = n - (startIndex - 1)
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
