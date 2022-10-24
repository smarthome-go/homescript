package homescript

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/smarthome-go/homescript/homescript/errors"
)

// String value
type ValueString struct {
	Value      string
	Identifier *string
	Range      errors.Span
}

func (self ValueString) Type() ValueType   { return TypeString }
func (self ValueString) Span() errors.Span { return self.Range }
func (self ValueString) Fields() map[string]Value {
	return map[string]Value{
		// Replaces the first occurence of the first argument in `self.Value` with the content of the second argument
		"replace": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("replace", span, args, TypeString, TypeString); err != nil {
					return nil, nil, err
				}
				return ValueString{
					Value: strings.Replace(self.Value, args[0].(ValueString).Value, args[1].(ValueString).Value, 1),
				}, nil, nil
			},
		},
		// Replaces all occurences of the first argument in `self.Value` with the content of the second argument
		"replace_all": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("replace_all", span, args, TypeString, TypeString); err != nil {
					return nil, nil, err
				}
				return ValueString{
					Value: strings.ReplaceAll(self.Value, args[0].(ValueString).Value, args[1].(ValueString).Value),
				}, nil, nil
			},
		},
		// Repeats `self.Value` n times, where n is the first argument
		"repeat": ValueBuiltinFunction{
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
		},
	}
}
func (self ValueString) Ident() *string { return self.Identifier }
func (self ValueString) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return self.Value, nil
}
func (self ValueString) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%s (len %d)", self.Value, utf8.RuneCountInString(self.Value)), nil
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