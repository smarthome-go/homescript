package homescript

import (
	"fmt"
	"math"

	"github.com/smarthome-go/homescript/homescript/errors"
)

// Number value
type ValueNumber struct {
	Value      float64
	Identifier *string
	Range      errors.Span
}

func (self ValueNumber) Type() ValueType   { return TypeNumber }
func (self ValueNumber) Span() errors.Span { return self.Range }
func (self ValueNumber) Fields() map[string]Value {
	return map[string]Value{
		// Specifies whether this number can be represented as an integer without loss of information
		// For example, 42.00 can be represented as 42 whilst 3.14159264 cannot be easily represented as an integer
		"is_int": ValueBuiltinFunction{
			Callback: func(_ Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("is_int", span, args); err != nil {
					return nil, nil, err
				}
				return ValueBool{
					Value: float64(int(self.Value)) == self.Value,
				}, nil, nil
			},
		},
		// Returns the integer value of `self.Value`
		"trunc": ValueBuiltinFunction{
			Callback: func(_ Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("trunc", span, args); err != nil {
					return nil, nil, err
				}
				return ValueNumber{
					Value: math.Trunc(self.Value),
				}, nil, nil
			},
		},
		// Rounds the float to the neares integer
		"round": ValueBuiltinFunction{
			Callback: func(_ Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("round", span, args); err != nil {
					return nil, nil, err
				}
				return ValueNumber{
					Value: math.Round(self.Value),
				}, nil, nil
			},
		},
	}
}
func (self ValueNumber) Ident() *string { return self.Identifier }
func (self ValueNumber) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	// Check if the value is actually an integer
	if float64(int(self.Value)) == self.Value {
		return fmt.Sprintf("%d", int(self.Value)), nil
	}
	return fmt.Sprintf("%f", self.Value), nil
}
func (self ValueNumber) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%f", self.Value), nil
}
func (self ValueNumber) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	return self.Value != 0, nil
}
func (self ValueNumber) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
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
		Value: math.Remainder(self.Value, other.(ValueNumber).Value),
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
