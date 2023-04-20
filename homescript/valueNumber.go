package homescript

import (
	"fmt"
	"math"
	"strconv"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

// Number value
type ValueNumber struct {
	Value       float64
	Range       errors.Span
	IsProtected bool
}

func (self ValueNumber) Type() ValueType   { return TypeNumber }
func (self ValueNumber) Span() errors.Span { return self.Range }
func (self ValueNumber) Protected() bool   { return self.IsProtected }
func (self ValueNumber) Fields(_ Executor, _ errors.Span) (map[string]*Value, *errors.Error) {
	return map[string]*Value{
		// Specifies whether this number can be represented as an integer without loss of information
		// For example, 42.00 can be represented as 42 whilst 3.14159264 cannot be easily represented as an integer
		"is_int": valPtr(ValueBuiltinFunction{
			Callback: func(_ Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("is_int", span, args); err != nil {
					return nil, nil, err
				}
				return ValueBool{
					Value:       float64(int(self.Value)) == self.Value,
					Range:       span,
					IsProtected: true,
				}, nil, nil
			},
		}),
		// Returns the integer value of `self.Value`
		"trunc": valPtr(ValueBuiltinFunction{
			Callback: func(_ Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("trunc", span, args); err != nil {
					return nil, nil, err
				}
				return ValueNumber{
					Value:       math.Trunc(self.Value),
					Range:       span,
					IsProtected: true,
				}, nil, nil
			},
		}),
		// Rounds the float to the neares integer
		"round": valPtr(ValueBuiltinFunction{
			Callback: func(_ Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("round", span, args); err != nil {
					return nil, nil, err
				}
				return ValueNumber{
					Value:       math.Round(self.Value),
					Range:       span,
					IsProtected: true,
				}, nil, nil
			},
		}),
		// Convert the number into its binary representation
		"bin": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
			if err := checkArgs("bin", span, args); err != nil {
				return nil, nil, err
			}
			if float64(int(self.Value)) != self.Value {
				return nil, nil, errors.NewError(
					span,
					"can only create binary representation on integer numbers, found float",
					errors.TypeError,
				)
			}
			return ValueString{Value: strconv.FormatInt(int64(self.Value), 2), Range: span, IsProtected: true}, nil, nil
		}}),
		// Create a range from 0 to the number (0..num)
		"range": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
			if float64(int(self.Value)) != self.Value {
				return nil, nil, errors.NewError(span, "can only create a range from intgeger numbers, found float", errors.ValueError)
			}

			zero := 0.0
			start := valPtr(ValueNumber{Value: zero})
			end := valPtr(ValueNumber{Value: self.Value})
			return ValueRange{Start: start, End: end, Current: &zero}, nil, nil
		},
		}),
		// Convert the number into a string
		"to_string": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
			if err := checkArgs("to_string", span, args); err != nil {
				return nil, nil, err
			}
			display, err := self.Display(executor, span)
			if err != nil {
				return nil, nil, err
			}
			return ValueString{
				Value:       display,
				Range:       span,
				IsProtected: true,
			}, nil, nil
		}}),
		"to_json":        marshalHelper(self),
		"to_json_indent": marshalIndentHelper(self),
	}, nil
}
func (self ValueNumber) Index(_ Executor, _ Value, span errors.Span) (*Value, bool, *errors.Error) {
	return nil, false, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueNumber) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	// Check if the value is actually an integer
	if float64(int(self.Value)) == self.Value {
		return fmt.Sprintf("%d", int(self.Value)), nil
	}
	return fmt.Sprintf("%f", self.Value), nil
}
func (self ValueNumber) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	// Check if the value is actually an integer
	if float64(int(self.Value)) == self.Value {
		return fmt.Sprintf("%d (type = int)", int(self.Value)), nil
	}
	return fmt.Sprintf("%f (type = float)", self.Value), nil
}
func (self ValueNumber) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	return self.Value != 0, nil
}
func (self ValueNumber) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
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
	return self.Value == other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsLessThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()), errors.TypeError)
	}
	return self.Value < other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsLessThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()), errors.TypeError)
	}
	return self.Value <= other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsGreaterThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()), errors.TypeError)
	}
	return self.Value > other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsGreaterThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()), errors.TypeError)
	}
	return self.Value >= other.(ValueNumber).Value, nil
}

func (self ValueNumber) Add(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	switch other.Type() {
	case TypeNumber:
		return ValueNumber{Value: self.Value + other.(ValueNumber).Value}, nil
	case TypeString:
		// Convert the number to a display representation
		display, err := self.Display(executor, span)
		if err != nil {
			return nil, err
		}
		// Return a string concatonation
		return ValueString{Value: display + other.(ValueString).Value}, nil
	}
	return nil, errors.NewError(span, fmt.Sprintf("cannot add %v to %v", other.Type(), self.Type()), errors.TypeError)
}

func (self ValueNumber) Sub(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		return nil, errors.NewError(span, fmt.Sprintf("cannot subtract %v from %v", other.Type(), self.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: self.Value - other.(ValueNumber).Value,
	}, nil
}
func (self ValueNumber) Mul(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return nil, err
			}
			return self.Mul(executor, span, value)
		}
		return nil, errors.NewError(span, fmt.Sprintf("cannot multiply %v by %v", self.Type(), other.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: self.Value * other.(ValueNumber).Value,
	}, nil
}
func (self ValueNumber) Div(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		return nil, errors.NewError(span, fmt.Sprintf("cannot divide %v by %v", self.Type(), other.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: self.Value / other.(ValueNumber).Value,
	}, nil
}
func (self ValueNumber) IntDiv(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		return nil, errors.NewError(span, fmt.Sprintf("cannot divide %v by %v", self.Type(), other.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: math.Floor(self.Value / other.(ValueNumber).Value),
	}, nil
}
func (self ValueNumber) Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		return nil, errors.NewError(span, fmt.Sprintf("cannot calculate reminder of %v / %v", self.Type(), other.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: math.Mod(self.Value, other.(ValueNumber).Value),
	}, nil
}
func (self ValueNumber) Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		return nil, errors.NewError(span, fmt.Sprintf("cannot calculate power of %v and %v", self.Type(), other.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: math.Pow(self.Value, other.(ValueNumber).Value),
	}, nil
}
