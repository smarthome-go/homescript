package homescript

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/smarthome-go/homescript/homescript/errors"
)

type ValueType uint8

const (
	TypeNull ValueType = iota
	TypeNumber
	TypeBoolean
	TypeString
	TypePair
	TypeObject
	TypeFunction
	TypeBuiltinFunction
	TypeBuiltinVariable
)

func (self ValueType) String() string {
	switch self {
	case TypeNull:
		return "Null"
	case TypeNumber:
		return "Number"
	case TypeBoolean:
		return "Boolean"
	case TypeString:
		return "String"
	case TypePair:
		return "Pair"
	case TypeObject:
		return "Object"
	case TypeFunction, TypeBuiltinFunction:
		return "Function"
	case TypeBuiltinVariable:
		return "BuiltinVariable"
	default:
		// Unreachable
		panic("BUG: A new type was introduced without updating this code")
	}
}

// Value interfaces
type Value interface {
	Type() ValueType
	Ident() *string
	// Is also used for `as str` and printing
	Display(executor Executor, span errors.Span) (string, *errors.Error)
	Debug(executor Executor, span errors.Span) (string, *errors.Error)
	IsTrue(executor Executor, span errors.Span) (bool, *errors.Error)
	IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error)
}

type ValueRelational interface {
	IsLessThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error)
	IsLessThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error)
	IsGreaterThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error)
	IsGreaterThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error)
}

type ValueAlg interface {
	Add(executor Executor, span errors.Span, other Value) (Value, *errors.Error)
	Sub(executor Executor, span errors.Span, other Value) (Value, *errors.Error)
	Mul(executor Executor, span errors.Span, other Value) (Value, *errors.Error)
	Div(executor Executor, span errors.Span, other Value) (Value, *errors.Error)
	Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error)
	Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error)
}

// Null value
type ValueNull struct {
	Identifier *string
}

func (self ValueNull) Type() ValueType { return TypeNull }
func (self ValueNull) Ident() *string  { return self.Identifier }
func (self ValueNull) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "null", nil
}
func (self ValueNull) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "null", nil
}
func (self ValueNull) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	return false, nil
}
func (self ValueNull) IsEqual(_ Executor, _ errors.Span, other Value) (bool, *errors.Error) {
	return false, nil
}

// Number value
type ValueNumber struct {
	Value      float64
	Identifier *string
}

func (self ValueNumber) Type() ValueType { return TypeNumber }
func (self ValueNumber) Ident() *string  { return self.Identifier }
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
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return false, err
			}
			return self.IsEqual(executor, span, value)
		}
		return false, errors.NewError(
			span,
			fmt.Sprintf("Cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	return self.Value == other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsLessThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != TypeNumber {
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return false, err
			}
			return self.IsLessThan(executor, span, value)
		}
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", TypeNumber, other.Type()), errors.TypeError)
	}
	return self.Value < other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsLessThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != TypeNumber {
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return false, err
			}
			return self.IsLessThanOrEqual(executor, span, value)
		}
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", TypeNumber, other.Type()), errors.TypeError)
	}
	return self.Value <= other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsGreaterThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != TypeNumber {
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return false, err
			}
			return self.IsGreaterThan(executor, span, value)
		}
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", TypeNumber, other.Type()), errors.TypeError)
	}
	return self.Value > other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsGreaterThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != TypeNumber {
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return false, err
			}
			return self.IsGreaterThanOrEqual(executor, span, value)
		}
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", TypeNumber, other.Type()), errors.TypeError)
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
	case TypeBuiltinVariable:
		otherCallback, err := other.(ValueBuiltinVariable).Callback(executor, span)
		if err != nil {
			return nil, err
		}
		self.Add(executor, span, otherCallback)
	}
	return nil, errors.NewError(span, fmt.Sprintf("Cannot add %v to %v: ", other.Type(), TypeString), errors.TypeError)
}

func (self ValueNumber) Sub(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return nil, err
			}
			return self.Sub(executor, span, value)
		}
		return nil, errors.NewError(span, fmt.Sprintf("Cannot subtract %v from %v: ", other.Type(), TypeNumber), errors.TypeError)
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
		return nil, errors.NewError(span, fmt.Sprintf("Cannot multiply %v by %v: ", TypeNumber, other.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: self.Value * other.(ValueNumber).Value,
	}, nil
}
func (self ValueNumber) Div(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return nil, err
			}
			return self.Div(executor, span, value)
		}
		return nil, errors.NewError(span, fmt.Sprintf("Cannot divide %v by %v: ", TypeNumber, other.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: self.Value / other.(ValueNumber).Value,
	}, nil
}
func (self ValueNumber) Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return nil, err
			}
			return self.Rem(executor, span, value)
		}
		return nil, errors.NewError(span, fmt.Sprintf("Cannot calculate reminder of %v / %v: ", TypeNumber, other.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: math.Remainder(self.Value, other.(ValueNumber).Value),
	}, nil
}

func (self ValueNumber) Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	if other.Type() != TypeNumber {
		if other.Type() == TypeBuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return nil, err
			}
			return self.Pow(executor, span, value)
		}
		return nil, errors.NewError(span, fmt.Sprintf("Cannot calculate power of %v and %v: ", TypeNumber, other.Type()), errors.TypeError)
	}
	return ValueNumber{
		Value: math.Pow(self.Value, other.(ValueNumber).Value),
	}, nil
}

// Boolean value
type ValueBool struct {
	Value      bool
	Identifier *string
}

func (self ValueBool) Type() ValueType { return TypeBoolean }
func (self ValueBool) Ident() *string  { return self.Identifier }
func (self ValueBool) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%t", self.Value), nil
}
func (self ValueBool) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%t", self.Value), nil
}
func (self ValueBool) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	return self.Value, nil
}
func (self ValueBool) IsEqual(_ Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if self.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("Cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	return self.Value == other.(ValueBool).Value, nil
}

func (self ValueBool) Add(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	switch other.Type() {
	case TypeString:
		// Convert the boolean to a display representation
		display, err := self.Display(executor, span)
		if err != nil {
			return nil, err
		}
		// Return a string concatonation
		return ValueString{Value: display + other.(ValueString).Value}, nil
	case TypeBuiltinVariable:
		otherCallback, err := other.(ValueBuiltinVariable).Callback(executor, span)
		if err != nil {
			return nil, err
		}
		self.Add(executor, span, otherCallback)
	}
	return nil, errors.NewError(span, fmt.Sprintf("Cannot add %v to %v: ", other.Type(), TypeString), errors.TypeError)
}
func (self ValueBool) Sub(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueBool) Mul(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueBool) Div(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueBool) Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueBool) Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}

// String value
type ValueString struct {
	Value      string
	Identifier *string
}

func (self ValueString) Type() ValueType { return TypeString }
func (self ValueString) Ident() *string  { return self.Identifier }
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
			fmt.Sprintf("Cannot compare %v to %v", self.Type(), other.Type()),
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
	case TypeBuiltinVariable:
		// This is required so that string cannot be added to builtin-var of type object
		otherCallback, err := other.(ValueBuiltinVariable).Callback(executor, span)
		if err != nil {
			return nil, err
		}
		self.Add(executor, span, otherCallback)
	}
	return nil, errors.NewError(span, fmt.Sprintf("Cannot add %v to %v: ", other.Type(), TypeString), errors.TypeError)
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
func (self ValueString) Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}
func (self ValueString) Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("Unsupported operation on type %v", self.Type()), errors.TypeError)
}

// Pair value
type ValuePair struct {
	Key        string
	Value      Value
	Identifier *string
}

func (self ValuePair) Type() ValueType { return TypePair }
func (self ValuePair) Ident() *string  { return self.Identifier }
func (self ValuePair) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	value, err := self.Value.Display(executor, span)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s => %s", self.Key, value), nil
}
func (self ValuePair) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	value, err := self.Value.Debug(executor, span)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(Key: %s | Value: %s)", self.Key, value), nil
}
func (self ValuePair) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	value, err := self.Value.IsTrue(executor, span)
	if err != nil {
		return false, err
	}
	return self.Key != "" && value, nil
}
func (self ValuePair) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if self.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("Cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	value, err := self.Value.IsEqual(executor, span, other.(ValuePair).Value)
	if err != nil {
		return false, err
	}
	return self.Key == other.(ValuePair).Key && value, nil
}

// Object value
type ValueObject struct {
	Fields map[string]Value
}

func (self ValueObject) Type() ValueType { return TypeObject }
func (self ValueObject) Ident() *string  { return nil }
func (self ValueObject) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	fields := make([]string, 0)
	for key, value := range self.Fields {
		valueDisplay, err := value.Display(executor, span)
		if err != nil {
			return "", err
		}
		fields = append(fields, fmt.Sprintf("%s: %s", key, valueDisplay))
	}
	return fmt.Sprintf("{%s}", strings.Join(fields, "; ")), nil
}
func (self ValueObject) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	fields := make([]string, 0)
	for key, value := range self.Fields {
		valueDisplay, err := value.Display(executor, span)
		if err != nil {
			return "", err
		}
		fields = append(fields, fmt.Sprintf("\t%s: %s", key, valueDisplay))
	}
	return fmt.Sprintf("{\n%s\n}", strings.Join(fields, "\n")), nil
}
func (self ValueObject) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	for _, value := range self.Fields {
		valueTrue, err := value.IsTrue(executor, span)
		if err != nil {
			return false, err
		}
		if !valueTrue {
			return false, nil
		}
	}
	return true, nil
}
func (self ValueObject) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if self.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("Cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	if len(self.Fields) != len(other.(ValueObject).Fields) {
		return false, nil
	}
	for key, value := range self.Fields {
		eq, err := other.(ValueObject).Fields[key].IsEqual(executor, span, value)
		if err != nil {
			return false, err
		}
		if !eq {
			return false, nil
		}
	}
	return true, nil
}

// Function value
type ValueFunction struct {
	Identifier string
	Args       []string
	Body       []Statement
}

func (self ValueFunction) Type() ValueType { return TypeFunction }
func (self ValueFunction) Ident() *string  { return nil }
func (self ValueFunction) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "<function>", nil
}
func (self ValueFunction) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "<function>", nil
}
func (self ValueFunction) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	return true, nil
}
func (self ValueFunction) IsEqual(_ Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if self.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("Cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	return self.Identifier == other.(ValueFunction).Identifier, nil
}

// Builtin function value
type ValueBuiltinFunction struct {
	Callback func(executor Executor, span errors.Span, args ...Value) (Value, *errors.Error)
}

func (self ValueBuiltinFunction) Type() ValueType { return TypeBuiltinFunction }
func (self ValueBuiltinFunction) Ident() *string  { return nil }
func (self ValueBuiltinFunction) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "<builtin-function>", nil
}
func (self ValueBuiltinFunction) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "<builtin-function>", nil
}
func (self ValueBuiltinFunction) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	return true, nil
}
func (self ValueBuiltinFunction) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if self.Type() != other.Type() && other.Type() != TypeFunction {
		return false, errors.NewError(
			span,
			fmt.Sprintf("Cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	return false, nil
}

// Builtin variable value
type ValueBuiltinVariable struct {
	Callback func(executor Executor, span errors.Span) (Value, *errors.Error)
}

func (self ValueBuiltinVariable) Type() ValueType { return TypeBuiltinVariable }
func (self ValueBuiltinVariable) Ident() *string  { return nil }
func (self ValueBuiltinVariable) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return "", err
	}
	// Invoke display on the callback result
	return value.Display(executor, span)
}
func (self ValueBuiltinVariable) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return "", err
	}
	// Invoke debug on the callback result
	return value.Debug(executor, span)
}
func (self ValueBuiltinVariable) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	valueIsTrue, err := value.IsTrue(executor, span)
	if err != nil {
		return false, err
	}
	return valueIsTrue, nil
}
func (self ValueBuiltinVariable) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("Cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	return value.IsEqual(executor, span, other)
}
func (self ValueBuiltinVariable) IsLessThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsLessThan(executor, span, other)
}
func (self ValueBuiltinVariable) IsLessThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsLessThanOrEqual(executor, span, other)
}
func (self ValueBuiltinVariable) IsGreaterThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsGreaterThan(executor, span, other)
}
func (self ValueBuiltinVariable) IsGreaterThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsGreaterThanOrEqual(executor, span, other)
}

func (self ValueBuiltinVariable) Add(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}
	switch value.Type() {
	case TypeNumber:
		return value.(ValueNumber).Add(executor, span, other)
	case TypeString:
		return value.(ValueString).Add(executor, span, other)
	default:
		return nil, errors.NewError(span, fmt.Sprintf("Invalid operation on type %v", self.Type()), errors.TypeError)
	}
}

func (self ValueBuiltinVariable) Sub(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}
	switch value.Type() {
	case TypeNumber:
		return value.(ValueNumber).Sub(executor, span, other)
	case TypeString:
		return value.(ValueString).Sub(executor, span, other)
	default:
		return nil, errors.NewError(span, fmt.Sprintf("Invalid operation on type %v", self.Type()), errors.TypeError)
	}
}
func (self ValueBuiltinVariable) Mul(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}
	switch value.Type() {
	case TypeNumber:
		return value.(ValueNumber).Mul(executor, span, other)
	case TypeString:
		return value.(ValueString).Mul(executor, span, other)
	default:
		return nil, errors.NewError(span, fmt.Sprintf("Invalid operation on type %v", self.Type()), errors.TypeError)
	}
}
func (self ValueBuiltinVariable) Div(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}
	switch value.Type() {
	case TypeNumber:
		return value.(ValueNumber).Div(executor, span, other)
	case TypeString:
		return value.(ValueString).Div(executor, span, other)
	default:
		return nil, errors.NewError(span, fmt.Sprintf("Invalid operation on type %v", self.Type()), errors.TypeError)
	}
}
func (self ValueBuiltinVariable) Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}
	switch value.Type() {
	case TypeNumber:
		return value.(ValueNumber).Rem(executor, span, other)
	case TypeString:
		return value.(ValueString).Rem(executor, span, other)
	default:
		return nil, errors.NewError(span, fmt.Sprintf("Invalid operation on type %v", self.Type()), errors.TypeError)
	}
}
func (self ValueBuiltinVariable) Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}
	switch value.Type() {
	case TypeNumber:
		return value.(ValueNumber).Pow(executor, span, other)
	case TypeString:
		return value.(ValueString).Pow(executor, span, other)
	default:
		return nil, errors.NewError(span, fmt.Sprintf("Invalid operation on type %v", self.Type()), errors.TypeError)
	}
}

// Helper functions for values
func getField(span errors.Span, self Value, fieldKey string) (Value, *errors.Error) {
	if self.Type() != TypeObject {
		return nil, errors.NewError(span, fmt.Sprintf("Cannot access fields of type %v", self.Type()), errors.TypeError)
	}
	fieldValue, exists := self.(ValueObject).Fields[fieldKey]
	if !exists {
		return nil, errors.NewError(span, fmt.Sprintf("Value has no member named %s", fieldKey), errors.TypeError)
	}
	return fieldValue, nil
}

// Helper factory functions
func makeNull() Value {
	return ValueNull{}
}

func makeNullResult() Result {
	null := makeNull()
	return Result{Value: &null}
}

func makeBool(value bool) Value {
	return ValueBool{Value: value}
}

func makeBoolResult(value bool) Result {
	bool := makeBool(value)
	return Result{Value: &bool}
}

func makeNum(value float64) Value {
	return ValueNumber{Value: value}
}

func makeStr(value string) Value {
	return ValueString{Value: value}
}

func makePair(key string, value Value) Value {
	return ValuePair{Key: key, Value: value}
}
