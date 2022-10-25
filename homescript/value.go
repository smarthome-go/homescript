package homescript

import (
	"fmt"
	"strings"

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
	TypeList
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
	case TypeList:
		return "list"
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
	Protected() bool
	Span() errors.Span
	Fields() map[string]Value
	Index(executor Executor, index int, span errors.Span) (Value, *errors.Error)
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
	Range       errors.Span
	IsProtected bool
}

func (self ValueNull) Type() ValueType          { return TypeNull }
func (self ValueNull) Span() errors.Span        { return self.Range }
func (self ValueNull) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueNull) Index(_ Executor, _ int, span errors.Span) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueNull) Protected() bool { return self.IsProtected }
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

// Boolean value
type ValueBool struct {
	Value       bool
	Range       errors.Span
	IsProtected bool
}

func (self ValueBool) Type() ValueType          { return TypeBoolean }
func (self ValueBool) Span() errors.Span        { return self.Range }
func (self ValueBool) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueBool) Index(_ Executor, _ int, span errors.Span) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueBool) Protected() bool { return self.IsProtected }
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

// Pair value
type ValuePair struct {
	Key         string
	Value       Value
	Range       errors.Span
	IsProtected bool
}

func (self ValuePair) Type() ValueType          { return TypePair }
func (self ValuePair) Span() errors.Span        { return self.Range }
func (self ValuePair) Fields() map[string]Value { return make(map[string]Value) }
func (self ValuePair) Index(_ Executor, _ int, span errors.Span) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValuePair) Protected() bool { return self.IsProtected }
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
	// Specifies whether this object is dynamic
	// If it is dynamic, the analyzer will not run its field checks
	// Such a dynamic object could be the global `ARGS` object
	IsDynamic bool
	// The fields of the object
	ObjFields   map[string]Value
	Range       errors.Span
	IsProtected bool
}

func (self ValueObject) Type() ValueType          { return TypeObject }
func (self ValueObject) Span() errors.Span        { return self.Range }
func (self ValueObject) Fields() map[string]Value { return self.ObjFields }
func (self ValueObject) Index(_ Executor, _ int, span errors.Span) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueObject) Protected() bool { return self.IsProtected }
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
	Body        []Statement
	Range       errors.Span
	IsProtected bool
}

func (self ValueFunction) Type() ValueType          { return TypeFunction }
func (self ValueFunction) Span() errors.Span        { return self.Range }
func (self ValueFunction) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueFunction) Index(_ Executor, _ int, span errors.Span) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueFunction) Protected() bool { return self.IsProtected }
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
	return false, nil
}

// Builtin function value
type ValueBuiltinFunction struct {
	Callback func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error)
}

func (self ValueBuiltinFunction) Type() ValueType          { return TypeBuiltinFunction }
func (self ValueBuiltinFunction) Span() errors.Span        { return errors.Span{} }
func (self ValueBuiltinFunction) Fields() map[string]Value { return make(map[string]Value) }
func (self ValueBuiltinFunction) Index(_ Executor, _ int, span errors.Span) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueBuiltinFunction) Protected() bool { return true }
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
func (self ValueBuiltinVariable) Index(_ Executor, _ int, span errors.Span) (Value, *errors.Error) {
	return nil, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueBuiltinVariable) Protected() bool {
	return true
}
func (self ValueBuiltinVariable) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) IsLessThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) IsLessThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) IsGreaterThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) IsGreaterThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	panic("A bare builtin variable should not exist")
}

func (self ValueBuiltinVariable) Add(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	panic("A bare builtin variable should not exist")
}

func (self ValueBuiltinVariable) Sub(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) Mul(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) Div(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) IntDiv(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	panic("A bare builtin variable should not exist")
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

func valPtr(input Value) *Value {
	return &input
}

func setValueSpan(value Value, span errors.Span) Value {
	switch value.Type() {
	case TypeNull:
		return ValueNull{
			Range: span,
		}
	case TypeNumber:
		return ValueNumber{
			Value: value.(ValueNumber).Value,
			Range: span,
		}
	case TypeBoolean:
		return ValueBool{
			Value: value.(ValueBool).Value,
			Range: span,
		}
	case TypeString:
		value = ValueString{
			Value: value.(ValueString).Value,
			Range: span,
		}
	case TypePair:
		value = ValuePair{
			Key:   value.(ValuePair).Key,
			Value: value.(ValuePair).Value,
			Range: span,
		}
	case TypeObject:
		value = ValueObject{
			DataType:  value.(ValueObject).DataType,
			IsDynamic: value.(ValueObject).IsDynamic,
			ObjFields: value.(ValueObject).ObjFields,
			Range:     span,
		}
	}
	// For other types, it is not possible to insert an identifier, so just return it as is
	return value
}
