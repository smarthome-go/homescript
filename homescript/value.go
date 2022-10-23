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
		return "null"
	case TypeNumber:
		return "number"
	case TypeBoolean:
		return "boolean"
	case TypeString:
		return "string"
	case TypePair:
		return "pair"
	case TypeObject:
		return "object"
	case TypeFunction, TypeBuiltinFunction:
		return "function"
	case TypeBuiltinVariable:
		return "builtinVariable"
	default:
		// Unreachable
		panic("BUG: A new type was introduced without updating this code")
	}
}

// Value interfaces
type Value interface {
	Type() ValueType
	Ident() *string
	Span() errors.Span
	Fields() map[string]Value
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
	IntDiv(executor Executor, span errors.Span, other Value) (Value, *errors.Error)
	Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error)
	Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error)
}

// Null value
type ValueNull struct {
	Identifier *string
	Range      errors.Span
}

func (self ValueNull) Type() ValueType          { return TypeNull }
func (self ValueNull) Span() errors.Span        { return self.Range }
func (self ValueNull) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueNull) Ident() *string           { return self.Identifier }
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
	Range      errors.Span
}

func (self ValueNumber) Type() ValueType          { return TypeNumber }
func (self ValueNumber) Span() errors.Span        { return self.Range }
func (self ValueNumber) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueNumber) Ident() *string           { return self.Identifier }
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

// Boolean value
type ValueBool struct {
	Value      bool
	Identifier *string
	Range      errors.Span
}

func (self ValueBool) Type() ValueType          { return TypeBoolean }
func (self ValueBool) Span() errors.Span        { return self.Range }
func (self ValueBool) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueBool) Ident() *string           { return self.Identifier }
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
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
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
	return nil, errors.NewError(span, fmt.Sprintf("cannot add %v to %v", other.Type(), self.Type()), errors.TypeError)
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
func (self ValueBool) IntDiv(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
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
	Range      errors.Span
}

func (self ValueString) Type() ValueType          { return TypeString }
func (self ValueString) Span() errors.Span        { return self.Range }
func (self ValueString) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueString) Ident() *string           { return self.Identifier }
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

// Pair value
type ValuePair struct {
	Key        string
	Value      Value
	Identifier *string
	Range      errors.Span
}

func (self ValuePair) Type() ValueType          { return TypePair }
func (self ValuePair) Span() errors.Span        { return self.Range }
func (self ValuePair) Fields() map[string]Value { return make(map[string]Value) }
func (self ValuePair) Ident() *string           { return self.Identifier }
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
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
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
	// Can be used if a builtin function only accepts objects of a certain type
	DataType string
	// The fields of the object
	ObjFields map[string]Value
}

func (self ValueObject) Type() ValueType          { return TypeObject }
func (self ValueObject) Span() errors.Span        { return errors.Span{} }
func (self ValueObject) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueObject) Ident() *string           { return nil }
func (self ValueObject) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	fields := make([]string, 0)
	for key, value := range self.ObjFields {
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
	for key, value := range self.ObjFields {
		valueDisplay, err := value.Display(executor, span)
		if err != nil {
			return "", err
		}
		fields = append(fields, fmt.Sprintf("\t%s: %s", key, valueDisplay))
	}
	return fmt.Sprintf("{\n%s\n}", strings.Join(fields, "\n")), nil
}
func (self ValueObject) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	for _, value := range self.ObjFields {
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
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	if len(self.ObjFields) != len(other.(ValueObject).ObjFields) {
		return false, nil
	}
	for key, value := range self.ObjFields {
		eq, err := other.(ValueObject).ObjFields[key].IsEqual(executor, span, value)
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
	Identifier *string
	Args       []struct {
		Identifier string
		Span       errors.Span
	}
	Body  []Statement
	Range errors.Span
}

func (self ValueFunction) Type() ValueType          { return TypeFunction }
func (self ValueFunction) Span() errors.Span        { return self.Range }
func (self ValueFunction) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueFunction) Ident() *string           { return nil }
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
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	if self.Identifier != nil && other.(ValueFunction).Identifier != nil {
		return *self.Identifier == *(other.(ValueFunction)).Identifier, nil
	}
	return false, nil
}

// Builtin function value
type ValueBuiltinFunction struct {
	Callback func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error)
}

func (self ValueBuiltinFunction) Type() ValueType          { return TypeBuiltinFunction }
func (self ValueBuiltinFunction) Span() errors.Span        { return errors.Span{} }
func (self ValueBuiltinFunction) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueBuiltinFunction) Ident() *string           { return nil }
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
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	return false, nil
}

// Builtin variable value
type ValueBuiltinVariable struct {
	Callback func(executor Executor, span errors.Span) (Value, *errors.Error)
}

func (self ValueBuiltinVariable) Type() ValueType          { return TypeBuiltinVariable }
func (self ValueBuiltinVariable) Span() errors.Span        { return errors.Span{} }
func (self ValueBuiltinVariable) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueBuiltinVariable) Ident() *string           { return nil }
func (self ValueBuiltinVariable) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	panic("A bare builtin variable should not exist")
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
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
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
		return false, errors.NewError(span, fmt.Sprintf("cannot compare %v to %v", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsLessThan(executor, span, other)
}
func (self ValueBuiltinVariable) IsLessThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("cannot compare %v to %v", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsLessThanOrEqual(executor, span, other)
}
func (self ValueBuiltinVariable) IsGreaterThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("cannot compare %v to %v", value.Type(), other.Type()), errors.TypeError)
	}
	return value.(ValueNumber).IsGreaterThan(executor, span, other)
}
func (self ValueBuiltinVariable) IsGreaterThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	if value.Type() != TypeNumber {
		return false, errors.NewError(span, fmt.Sprintf("cannot compare %v to %v", value.Type(), other.Type()), errors.TypeError)
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
func (self ValueBuiltinVariable) IntDiv(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	value, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}
	switch value.Type() {
	case TypeNumber:
		return value.(ValueNumber).IntDiv(executor, span, other)
	case TypeString:
		return value.(ValueString).IntDiv(executor, span, other)
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
func getField(executor Executor, span errors.Span, self Value, fieldKey string) (Value, *errors.Error) {
	value, exists := self.Fields()[fieldKey]
	if !exists {
		return nil, errors.NewError(span, fmt.Sprintf("%v has no member named %s", self.Type(), fieldKey), errors.TypeError)
	}
	return value, nil
}

// Helper factory functions
func makeNull(span errors.Span) Value {
	return ValueNull{Range: span}
}

func makeNullResult(span errors.Span) Result {
	null := makeNull(span)
	return Result{Value: &null}
}

func makeBool(span errors.Span, value bool) Value {
	return ValueBool{Value: value, Range: span}
}

func makeBoolResult(span errors.Span, value bool) Result {
	bool := makeBool(span, value)
	return Result{Value: &bool}
}

func makeNum(span errors.Span, value float64) Value {
	return ValueNumber{Value: value, Range: span}
}

func makeStr(span errors.Span, value string) Value {
	return ValueString{Value: value, Range: span}
}

func makePair(span errors.Span, key string, value Value) Value {
	return ValuePair{Key: key, Value: value, Range: span}
}

func makeFn(identifier *string, span errors.Span) Value {
	return ValueFunction{Identifier: identifier, Range: span}
}
