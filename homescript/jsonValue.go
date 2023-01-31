package homescript

import (
	"encoding/json"
	"fmt"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

func marshalValue(self Value, span errors.Span, isInner bool) (interface{}, bool, *errors.Error) {
	switch self := self.(type) {
	case ValueString:
		return self.Value, false, nil
	case ValueNumber:
		if float64(int(self.Value)) == self.Value {
			return self.Value, false, nil
		}
		return self.Value, false, nil
	case ValueBool:
		return self.Value, false, nil
	case ValueObject:
		output := make(map[string]interface{}, 0)
		for key, value := range self.Fields() {
			if value == nil {
				return nil, false, nil
			}
			marshaled, skipNull, err := marshalValue(*value, span, true)
			if err != nil {
				return nil, false, err
			}
			// Skip builtin functions
			if marshaled != nil && !skipNull {
				output[key] = marshaled
			}
		}
		return output, false, nil
	case ValueList:
		output := make([]interface{}, 0)
		for _, value := range *self.Values {
			marshaled, _, err := marshalValue(*value, span, true)
			if err != nil {
				return nil, false, err
			}
			output = append(output, marshaled)
		}
		return output, false, nil
	case ValueBuiltinFunction:
		// Skip builtin functions (on objects)
		return nil, true, nil
	case ValuePair:
		output := make(map[string]interface{})
		marshaledValue, _, err := marshalValue(*self.Value, span, true)
		if err != nil {
			return nil, false, err
		}
		if (*self.Key).Type() != TypeString {
			return nil, false, errors.NewError(
				span,
				fmt.Sprintf("cannot encode pair key of type '%v' to JSON", (*self.Key).Type()),
				errors.RuntimeError,
			)
		}
		output[(*self.Key).(ValueString).Value] = marshaledValue
		return output, false, nil
	case ValueNull, nil:
		return nil, false, nil
	default:
		inner := ""
		if isInner {
			inner = "inner"
		}
		return nil, false, errors.NewError(
			span,
			fmt.Sprintf("cannot encode %s value of type '%v' to JSON", inner, self.Type()),
			errors.ValueError,
		)
	}
}

func unmarshalValue(span errors.Span, self interface{}) (Value, *errors.Error) {
	switch self := self.(type) {
	case string:
		return ValueString{Value: self}, nil
	case float64:
		return ValueNumber{Value: self}, nil
	case int:
		return ValueNumber{Value: float64(self)}, nil
	case bool:
		return ValueBool{Value: self}, nil
	case map[string]interface{}:
		zero := 0
		output := ValueObject{ObjFields: make(map[string]*Value), CurrentIterIndex: &zero}
		for key, field := range self {
			value, err := unmarshalValue(span, field)
			if err != nil {
				return nil, err
			}
			output.ObjFields[key] = &value
		}
		return output, nil
	case []interface{}:
		values := make([]*Value, 0)
		valueType := TypeUnknown
		for _, item := range self {
			value, err := unmarshalValue(span, item)
			if err != nil {
				return nil, err
			}
			// Check type equality
			if valueType != TypeUnknown {
				if valueType != value.Type() {
					return nil, errors.NewError(
						span,
						fmt.Sprintf("type inconsistency in list: cannot insert value of type '%v' into %v<%v>", value.Type(), TypeList, valueType),
						errors.RuntimeError,
					)
				}
			} else {
				valueType = value.Type()
			}
			values = append(values, &value)
		}
		zero := 0
		return ValueList{
			Values:           &values,
			ValueType:        &valueType,
			CurrentIterIndex: &zero,
		}, nil
	case nil:
		return ValueNull{}, nil
	default:
		return nil, errors.NewError(
			span,
			fmt.Sprintf("cannot parse unknown JSON value: `%v` to HMS value", self),
			errors.RuntimeError,
		)
	}
}

func marshalHelper(self Value) *Value {
	return valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
		marshaled, _, err := marshalValue(self, span, false)
		if err != nil {
			return nil, nil, err
		}
		output, jsonErr := json.Marshal(marshaled)
		if jsonErr != nil {
			return nil, nil, errors.NewError(
				span,
				jsonErr.Error(),
				errors.RuntimeError,
			)
		}
		return ValueString{Value: string(output)}, nil, nil
	}})
}

func marshalIndentHelper(self Value) *Value {
	return valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
		marshaled, _, err := marshalValue(self, span, false)
		if err != nil {
			return nil, nil, err
		}
		output, jsonErr := json.MarshalIndent(marshaled, "", "    ")
		if jsonErr != nil {
			return nil, nil, errors.NewError(
				span,
				jsonErr.Error(),
				errors.RuntimeError,
			)
		}
		return ValueString{Value: string(output)}, nil, nil
	}})
}
