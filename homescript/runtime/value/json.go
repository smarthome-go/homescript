package value

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

// NOTE: `skipNull` is required so that builtin functions do not show up as `null` in the marshaled output.
func MarshalValue(self Value, span errors.Span, isInner bool) (out interface{}, skipNull bool) {
	switch self := self.(type) {
	case ValueString:
		return self.Inner, false
	case ValueInt:
		return self.Inner, false
	case ValueFloat:
		return self.Inner, false
	case ValueBool:
		return self.Inner, false
	case ValueAnyObject:
		output := make(map[string]interface{}, 0)
		for key, value := range self.FieldsInternal {
			if value == nil {
				return nil, false
			}
			marshaled, skipNull := MarshalValue(*value, span, true)
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
			marshaled, skipNull := MarshalValue(*value, span, true)
			// skip builtin functions
			if marshaled != nil && !skipNull {
				output[key] = marshaled
			}
		}
		return output, false
	case ValueList:
		output := make([]interface{}, 0)
		for _, value := range *self.Values {
			marshaled, skipNull := MarshalValue(*value, span, true)

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
			return MarshalValue(*self.Inner, span, true)
		} else {
			return nil, false
		}
	default:
		panic(fmt.Sprintf("Cannot encode value of type '%v' to JSON", self.Kind()))
	}
}

// TODO: write docs why this is public
func UnmarshalValue(span errors.Span, self interface{}) (*Value, *VmInterrupt) {
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
		return nil, NewVMFatalException(fmt.Sprintf("Cannot parse unknown JSON value: `%v` to HMS value", self), Vm_JsonErrorKind, span)
	}
}

func MarshalToString(self Value) *Value {
	return NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
		// TODO: fail if skipNull is true?
		marshaled, _ := MarshalValue(self, span, false)
		output, jsonErr := json.Marshal(marshaled)
		if jsonErr != nil {
			return nil, NewVMFatalException(jsonErr.Error(), Vm_JsonErrorKind, span)
		}
		return NewValueString(string(output)), nil
	})
}

func MarshalIndentToString(self Value) *Value {
	return NewValueBuiltinFunction(func(_ Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
		// TODO: fail if skipNull is true?
		marshaled, _ := MarshalValue(self, span, false)
		output, jsonErr := json.MarshalIndent(marshaled, "", "    ")
		if jsonErr != nil {
			return nil, NewVMFatalException(jsonErr.Error(), Vm_JsonErrorKind, span)
		}
		return NewValueString(string(output)), nil
	})
}
