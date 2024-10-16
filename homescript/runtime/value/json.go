package value

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	herrors "github.com/smarthome-go/homescript/v3/homescript/errors"
)

// HACK: this is necessary as Go has a really crappy JSON convention.
// refer to: https://github.com/golang/go/issues/42246
type jsonFloat float64

func (f jsonFloat) MarshalJSON() ([]byte, error) {
	const floatSize = 64

	n := float64(f)
	if math.IsInf(n, 0) || math.IsNaN(n) {
		return nil, errors.New("unsupported float64")
	}
	prec := -1
	if math.Trunc(n) == n {
		prec = 1 // Force ".0" for integers.
	}
	return strconv.AppendFloat(nil, n, 'f', prec, floatSize), nil
}

// NOTE: `skipNull` is required so that builtin functions do not show up as `null` in the marshaled output.
func MarshalValue(self Value, isInner bool) (out interface{}, skipNull bool) {
	switch self := self.(type) {
	case ValueString:
		return self.Inner, false
	case ValueInt:
		return self.Inner, false
	case ValueFloat:
		return jsonFloat(self.Inner), false
	case ValueBool:
		return self.Inner, false
	case ValueAnyObject:
		output := make(map[string]interface{}, 0)
		for key, value := range self.FieldsInternal {
			if value == nil {
				return nil, false
			}
			marshaled, skipNull := MarshalValue(*value, true)
			// skip builtin functions
			if marshaled != nil && !skipNull {
				output[key] = marshaled
			}
		}
		return output, false
	case ValueObject:
		output := make(map[string]interface{}, 0)
		for key, value := range self.FieldsInternal {
			if value == nil {
				return nil, false
			}
			marshaled, skipNull := MarshalValue(*value, true)
			// skip builtin functions
			if marshaled != nil && !skipNull {
				output[key] = marshaled
			}
		}
		return output, false
	case ValueList:
		output := make([]interface{}, 0)
		for _, value := range *self.Values {
			marshaled, skipNull := MarshalValue(*value, true)

			// skip builtin functions
			if marshaled != nil && !skipNull {
				output = append(output, marshaled)
			}
		}
		return output, false
	case ValueBuiltinFunction:
		// skip builtin functions
		return nil, true
	case ValueNull, nil:
		return nil, false
	case ValueOption:
		if self.IsSome() {
			return MarshalValue(*self.Inner, true)
		} else {
			return nil, false
		}
	default:
		panic(fmt.Sprintf("Cannot encode value of type '%v' to JSON", self.Kind()))
	}
}

func TypeAwareUnmarshalValue(self interface{}, typ ast.Type) *Value {
	if typ.Kind() == ast.OptionTypeKind {
		opt := typ.(ast.OptionType)

		switch self.(type) {
		case nil:
			return NewNoneOption()
		default:
			return NewValueOption(TypeAwareUnmarshalValue(self, opt.Inner))
		}
	}

	switch self := self.(type) {
	case string:
		return NewValueString(self)
	case float64:
		if typ.Kind() == ast.IntTypeKind {
			return NewValueInt(int64(self))
		}
		return NewValueFloat(self)
	case jsonFloat:
		if typ.Kind() == ast.IntTypeKind {
			return NewValueInt(int64(self))
		}
		return NewValueFloat(float64(self))
	case int:
		if typ.Kind() == ast.FloatTypeKind {
			return NewValueFloat(float64(self))
		}
		return NewValueInt(int64(self))
	case int64:
		if typ.Kind() == ast.FloatTypeKind {
			return NewValueFloat(float64(self))
		}
		return NewValueInt(self)
	case bool:
		return NewValueBool(self)
	case map[string]interface{}:
		typeFields := typ.(ast.ObjectType).ObjFields

		fields := make(map[string]*Value)
		for _, typeField := range typeFields {
			field := self[typeField.FieldName.Ident()]
			fields[typeField.FieldName.Ident()] = TypeAwareUnmarshalValue(field, typeField.Type)
		}

		return NewValueObject(fields)
	case []interface{}:
		innerType := typ.(ast.ListType).Inner
		values := make([]*Value, 0)
		for _, item := range self {
			values = append(values, TypeAwareUnmarshalValue(item, innerType))
		}
		return NewValueList(values)
	case nil:
		return NewNoneOption()
	default:
		panic(fmt.Sprintf("Cannot parse unknown JSON value: `%v` (%v) to HMS value", self, reflect.TypeOf(self)))
	}
}

// TODO: write docs why this is public
func UnmarshalValue(span herrors.Span, self interface{}) (*Value, *VmInterrupt) {
	// TODO: do this
	switch self := self.(type) {
	case string:
		return NewValueString(self), nil
	case float64:
		if float64(int64(self)) == self {
			return NewValueInt(int64(self)), nil
		}
		return NewValueFloat(self), nil
	case int:
		return NewValueInt(int64(self)), nil
	case int64:
		return NewValueInt(self), nil
	case bool:
		return NewValueBool(self), nil
	case map[string]interface{}:
		fields := make(map[string]*Value)
		for key, field := range self {
			value, err := UnmarshalValue(span, field)
			if err != nil {
				return nil, err
			}
			fields[key] = value
		}
		return NewValueObject(fields), nil
	case []interface{}:
		values := make([]*Value, 0)
		for _, item := range self {
			value, err := UnmarshalValue(span, item)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return NewValueList(values), nil
	case nil:
		return NewNoneOption(), nil
	default:
		return nil, NewVMFatalException(fmt.Sprintf("Cannot parse unknown JSON value: `%v` (%v) to HMS value", self, reflect.TypeOf(self)), Vm_JsonErrorKind, span)
	}
}

func MarshalToString(self Value) *Value {
	return NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span herrors.Span, args ...Value) (*Value, *VmInterrupt) {
		// TODO: fail if skipNull is true?
		marshaled, _ := MarshalValue(self, false)
		output, jsonErr := json.Marshal(marshaled)
		if jsonErr != nil {
			return nil, NewVMFatalException(jsonErr.Error(), Vm_JsonErrorKind, span)
		}
		return NewValueString(string(output)), nil
	})
}

func MarshalIndentToString(self Value) *Value {
	return NewValueBuiltinFunction(func(_ Executor, cancelCtx *context.Context, span herrors.Span, args ...Value) (*Value, *VmInterrupt) {
		// TODO: fail if skipNull is true?
		marshaled, _ := MarshalValue(self, false)
		output, jsonErr := json.MarshalIndent(marshaled, "", "    ")
		if jsonErr != nil {
			return nil, NewVMFatalException(jsonErr.Error(), Vm_JsonErrorKind, span)
		}
		return NewValueString(string(output)), nil
	})
}
