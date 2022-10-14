package interpreter

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/smarthome-go/homescript/homescript/errors"
)

type ValueType uint8

const (
	Null ValueType = iota
	Number
	Boolean
	String
	Pair
	Object
	Function
	BuiltinFunction
	BuiltinVariable
)

func (self ValueType) String() string {
	switch self {
	case Null:
		return "Null"
	case Number:
		return "Number"
	case Boolean:
		return "Boolean"
	case String:
		return "String"
	case Pair:
		return "Pair"
	case Object:
		return "Object"
	case Function | BuiltinFunction:
		return "Function"
	case BuiltinVariable:
		return "BuiltinVariable"
	default:
		// Unreachabel
		panic("BUG: A new type was introduced without updating this code")
	}
}

// Value interfaces
type Value interface {
	Type() ValueType
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

// Null value
type ValueNull struct{}

func (self ValueNull) Type() ValueType { return Null }
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
	Value float64
}

func (self ValueNumber) Type() ValueType { return Number }
func (self ValueNumber) Display(executor Executor, span errors.Span) (string, *errors.Error) {
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
		if other.Type() == BuiltinVariable {
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
	if other.Type() != Number {
		if other.Type() == BuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return false, err
			}
			return self.IsLessThan(executor, span, value)
		}
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare number to %v: ", other.Type()), errors.TypeError)
	}
	return self.Value < other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsLessThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != Number {
		if other.Type() == BuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return false, err
			}
			return self.IsLessThanOrEqual(executor, span, value)
		}
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare number to %v: ", other.Type()), errors.TypeError)
	}
	return self.Value <= other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsGreaterThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != Number {
		if other.Type() == BuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return false, err
			}
			return self.IsGreaterThan(executor, span, value)
		}
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare number to %v: ", other.Type()), errors.TypeError)
	}
	return self.Value > other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsGreaterThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if other.Type() != Number {
		if other.Type() == BuiltinVariable {
			value, err := other.(ValueBuiltinVariable).Callback(executor, span)
			if err != nil {
				return false, err
			}
			return self.IsGreaterThanOrEqual(executor, span, value)
		}
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare number to %v: ", other.Type()), errors.TypeError)
	}
	return self.Value >= other.(ValueNumber).Value, nil
}

// Boolean value
type ValueBool struct {
	Value bool
}

func (self ValueBool) Type() ValueType { return Boolean }
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

// String value
type ValueString struct {
	Value string
}

func (self ValueString) Type() ValueType { return String }
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

// Pair value
type ValuePair struct {
	Key   string
	Value Value
}

func (self ValuePair) Type() ValueType { return Pair }
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

func (self ValueObject) Type() ValueType { return Object }
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
}

func (self ValueFunction) Type() ValueType { return Function }
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
	Identifier string
	Callback   func(executor Executor, span errors.Span, args ...Value) (Value, *errors.Error)
}

func (self ValueBuiltinFunction) Type() ValueType { return BuiltinFunction }
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
	if self.Type() != other.Type() && other.Type() != Function {
		return false, errors.NewError(
			span,
			fmt.Sprintf("Cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	return self.Identifier == other.(ValueFunction).Identifier, nil
}

// Builtin variable value
type ValueBuiltinVariable struct {
	Callback func(executor Executor, span errors.Span) (Value, *errors.Error)
}

func (self ValueBuiltinVariable) Type() ValueType { return BuiltinVariable }
func (self ValueBuiltinVariable) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "<builtin-variable>", nil
}
func (self ValueBuiltinVariable) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "<builtin-variable>", nil
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
	if value.Type() != Number {
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsLessThan(executor, span, other)
}
func (self ValueBuiltinVariable) IsLessThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != Number {
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsLessThanOrEqual(executor, span, other)
}
func (self ValueBuiltinVariable) IsGreaterThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != Number {
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsGreaterThan(executor, span, other)
}
func (self ValueBuiltinVariable) IsGreaterThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != Number {
		return false, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v: ", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsGreaterThanOrEqual(executor, span, other)
}
