package value

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

// NOTE: `skipNull` is required so that builtin functions do not show up as `null` in the marshaled output.
func MarshalValue(self Value, span errors.Span, isInner bool) (out interface{}, skipNull bool, interrupt *VmInterrupt) {
	switch self := self.(type) {
	case ValueString:
		return self.Inner, false, nil
	case ValueInt:
		return self.Inner, false, nil
	case ValueFloat:
		return self.Inner, false, nil
	case ValueBool:
		return self.Inner, false, nil
	case ValueAnyObject:
		output := make(map[string]interface{}, 0)

		for key, value := range self.FieldsInternal {
			if value == nil {
				return nil, false, nil
			}
			marshaled, skipNull, err := MarshalValue(*value, span, true)
			if err != nil {
				return nil, false, err
			}
			// skip builtin functions
			if marshaled != nil && !skipNull {
				output[key] = marshaled
			}
		}
		return output, false, nil
	case ValueObject:
		output := make(map[string]interface{}, 0)

		for key, value := range self.FieldsInternal {
			if value == nil {
				return nil, false, nil
			}
			marshaled, skipNull, err := MarshalValue(*value, span, true)
			if err != nil {
				return nil, false, err
			}
			// skip builtin functions
			if marshaled != nil && !skipNull {
				output[key] = marshaled
			}
		}
		return output, false, nil
	case ValueList:
		output := make([]interface{}, 0)
		for _, value := range *self.Values {
			marshaled, _, err := MarshalValue(*value, span, true)
			if err != nil {
				return nil, false, err
			}
			output = append(output, marshaled)
		}
		return output, false, nil
	case ValueBuiltinFunction:
		// skip builtin functions
		return nil, true, nil
	case ValueNull, nil:
		return nil, false, nil
	case ValueOption:
		if self.IsSome() {
			return MarshalValue(*self.Inner, span, true)
		} else {
			return nil, false, nil
		}
	default:
		inner := ""
		if isInner {
			inner = "inner"
		}
		return nil, false, NewVMFatalException(fmt.Sprintf("Cannot encode %s value of type '%v' to JSON", inner, self.Kind()), Vm_JsonErrorKind, span)
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
		marshaled, _, err := MarshalValue(self, span, false)
		if err != nil {
			return nil, err
		}
		output, jsonErr := json.Marshal(marshaled)
		if jsonErr != nil {
			return nil, NewVMFatalException(jsonErr.Error(), Vm_JsonErrorKind, span)
		}
		return NewValueString(string(output)), nil
	})
}

func MarshalIndentToString(self Value) *Value {
	return NewValueBuiltinFunction(func(_ Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
		marshaled, _, i := MarshalValue(self, span, false)
		if i != nil {
			return nil, i
		}
		output, jsonErr := json.MarshalIndent(marshaled, "", "    ")
		if jsonErr != nil {
			return nil, NewVMFatalException(jsonErr.Error(), Vm_JsonErrorKind, span)
		}
		return NewValueString(string(output)), nil
	})
}
