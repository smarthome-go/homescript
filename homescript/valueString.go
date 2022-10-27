package homescript

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/smarthome-go/homescript/homescript/errors"
)

// String value
type ValueString struct {
	Value       string
	Range       errors.Span
	IsProtected bool
}

func (self ValueString) Type() ValueType   { return TypeString }
func (self ValueString) Span() errors.Span { return self.Range }
func (self ValueString) Protected() bool   { return self.IsProtected }
func (self ValueString) Fields() map[string]*Value {
	return map[string]*Value{
		// Replaces the first occurence of the first argument in `self.Value` with the content of the second argument
		"replace": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("replace", span, args, TypeString, TypeString); err != nil {
					return nil, nil, err
				}
				return ValueString{
					Value: strings.Replace(self.Value, args[0].(ValueString).Value, args[1].(ValueString).Value, 1),
				}, nil, nil
			},
		}),
		// Replaces all occurences of the first argument in `self.Value` with the content of the second argument
		"replace_all": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("replace_all", span, args, TypeString, TypeString); err != nil {
					return nil, nil, err
				}
				return ValueString{
					Value: strings.ReplaceAll(self.Value, args[0].(ValueString).Value, args[1].(ValueString).Value),
				}, nil, nil
			},
		}),
		// Repeats `self.Value` n times, where n is the first argument
		"repeat": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("replace_all", span, args, TypeNumber); err != nil {
					return nil, nil, err
				}
				num := args[0].(ValueNumber)
				if num.Value != float64(int(num.Value)) {
					return nil, nil, errors.NewError(
						span,
						"first argument of string.replace_all mut be an integer number, found float",
						errors.ValueError,
					)
				}
				return ValueString{
					Value: strings.Repeat(self.Value, int(num.Value)),
				}, nil, nil
			},
		}),
		"parse_json": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				var raw interface{}
				if err := json.Unmarshal([]byte(self.Value), &raw); err != nil {
					return nil, nil, errors.NewError(span, err.Error(), errors.ValueError)
				}
				value, err := unmarshalValue(span, raw)
				if err != nil {
					return nil, nil, err
				}
				return value, nil, nil
			},
		}),
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
		output := ValueObject{ObjFields: make(map[string]*Value)}
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
		return ValueList{
			Values:    &values,
			ValueType: &valueType,
		}, nil
	default:
		return nil, errors.NewError(
			span,
			fmt.Sprintf("cannot parse unknown JSON to HMS value: %v", self),
			errors.RuntimeError,
		)
	}
}

func (self ValueString) Index(_ Executor, index int, span errors.Span) (*Value, *errors.Error) {
	// Check the string len
	valLen := len(self.Value)
	if index < 0 {
		index = index + valLen
	}
	if index < 0 || index >= valLen {
		return nil, errors.NewError(
			span,
			fmt.Sprintf("index out of bounds: index is %d, but length is %d", index, valLen),
			errors.OutOfBoundsError,
		)
	}
	return valPtr(ValueString{
		Value: string([]rune(self.Value)[index]),
		Range: self.Range,
	}), nil
}
func (self ValueString) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return self.Value, nil
}
func (self ValueString) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%s (len = %d)", self.Value, utf8.RuneCountInString(self.Value)), nil
}
func (self ValueString) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	return self.Value != "", nil
}
func (self ValueString) IsEqual(_ Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if self.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	return self.Value == other.(ValueString).Value, nil
}

func (self ValueString) Add(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	switch other.Type() {
	case TypeString, TypeBoolean, TypeNumber:
		display, err := other.Display(executor, span)
		if err != nil {
			return nil, err
		}
		return ValueString{Value: self.Value + display}, nil
	}
	return nil, errors.NewError(span, fmt.Sprintf("cannot add %v to %v", other.Type(), self.Type()), errors.TypeError)
}
func (self ValueString) Sub(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueString) Mul(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueString) Div(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueString) IntDiv(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueString) Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueString) Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
