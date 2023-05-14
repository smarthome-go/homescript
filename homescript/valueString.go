package homescript

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

// String value
type ValueString struct {
	Value            string
	Range            errors.Span
	CurrentIterIndex *int
	IsProtected      bool
}

func (self ValueString) Type() ValueType   { return TypeString }
func (self ValueString) Span() errors.Span { return self.Range }
func (self ValueString) Protected() bool   { return self.IsProtected }
func (self ValueString) Fields(_ Executor, _ errors.Span) (map[string]*Value, *errors.Error) {
	return map[string]*Value{
		// Calculates the utf-8 (rune) length of the string
		"len": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("len", span, args); err != nil {
					return nil, nil, err
				}
				return ValueNumber{
					Value: float64(utf8.RuneCountInString(self.Value)),
				}, nil, nil
			},
		}),
		// Replaces the first occurrence of the first argument in `self.Value` with the content of the second argument
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
		// Replaces all occurrences of the first argument in `self.Value` with the content of the second argument
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
				if err := checkArgs("parse_json", span, args); err != nil {
					return nil, nil, err
				}

				if executor.IsAnalyzer() {
					return nil, nil, nil
				}

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
		"split": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
			if err := checkArgs("split", span, args, TypeString); err != nil {
				return nil, nil, err
			}
			typ := TypeString
			pieces := make([]*Value, 0)
			stringPieces := strings.Split(self.Value, args[0].(ValueString).Value)
			for _, piece := range stringPieces {
				pieces = append(pieces, valPtr(ValueString{Value: piece}))
			}
			return ValueList{ValueType: &typ, Values: &pieces}, nil, nil
		}}),
		"contains": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
			if err := checkArgs("contains", span, args, TypeString); err != nil {
				return nil, nil, err
			}
			contains := strings.Contains(self.Value, args[0].(ValueString).Value)
			return ValueBool{Value: contains}, nil, nil
		}}),
		"parse_num": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
			res, err := strconv.ParseFloat(self.Value, 64)
			if err != nil {
				return nil, nil, errors.NewError(span, err.Error(), errors.ValueError)
			}
			return ValueNumber{Value: res, Range: span, IsProtected: true}, nil, nil
		}}),
		"parse_bool": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
			var res bool
			switch self.Value {
			case "true":
			case "false":
			default:
				return nil, nil, errors.NewError(span, fmt.Sprintf("not a boolean value: %s", self.Value), errors.ValueError)
			}
			return ValueBool{Value: res, IsProtected: true}, nil, nil
		}}),
		"to_json":        marshalHelper(self),
		"to_json_indent": marshalIndentHelper(self),
	}, nil
}

func (self ValueString) Index(_ Executor, indexValue Value, span errors.Span) (*Value, bool, *errors.Error) {
	// Check the type
	if indexValue.Type() != TypeNumber {
		return nil, true, errors.NewError(
			span,
			fmt.Sprintf("cannot index value of type '%v' by a value of type '%v'", TypeList, indexValue.Type()),
			errors.TypeError,
		)
	}
	if float64(int(indexValue.(ValueNumber).Value)) != indexValue.(ValueNumber).Value {
		return nil, true, errors.NewError(
			span,
			fmt.Sprintf("cannot index value of type '%v' by a float number", TypeList),
			errors.ValueError,
		)
	}
	// Check the string len
	index := int(indexValue.(ValueNumber).Value)
	valLen := len(self.Value)
	if index < 0 {
		index = index + valLen
	}
	if index < 0 || index >= valLen {
		return nil, true, errors.NewError(
			span,
			fmt.Sprintf("index out of bounds: index is %d, but length is %d", index, valLen),
			errors.OutOfBoundsError,
		)
	}
	return valPtr(ValueString{
		Value: string([]rune(self.Value)[index]),
		Range: self.Range,
	}), true, nil
}
func (self ValueString) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return self.Value, nil
}
func (self ValueString) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%s (type = string; len = %d)", self.Value, utf8.RuneCountInString(self.Value)), nil
}
func (self ValueString) IsTrue(_ Executor, span errors.Span) (bool, *errors.Error) {
	switch self.Value {
	case "true", "on":
		return true, nil
	case "false", "off":
		return false, nil
	default:
		return false, errors.NewError(span, "Cannot cast illegal string to boolean value", errors.ValueError)
	}
}
func (self ValueString) IsEqual(_ Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() == TypeNull {
		return false, nil
	}
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
func (self *ValueString) Next(val *Value, span errors.Span) bool {
	if self.CurrentIterIndex == nil {
		self.IterReset()
	}

	old := *self.CurrentIterIndex
	*self.CurrentIterIndex++

	shouldContinue := *self.CurrentIterIndex <= utf8.RuneCountInString(self.Value)

	if shouldContinue {
		*val = ValueString{Value: string([]rune(self.Value)[old]), Range: span}
		return true
	} else {
		self.IterReset()
		return false
	}
}
func (self *ValueString) IterReset() {
	zero := 0
	self.CurrentIterIndex = &zero
}
