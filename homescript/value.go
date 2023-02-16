package homescript

import (
	"fmt"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

type ValueType uint8

const (
	TypeUnknown ValueType = iota
	TypeNull
	TypeNumber
	TypeBoolean
	TypeString
	TypePair
	TypeObject
	TypeList
	TypeRange
	TypeFunction
	TypeBuiltinFunction
	TypeBuiltinVariable
)

func (self ValueType) String() string {
	switch self {
	case TypeUnknown:
		return "unknown"
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
	case TypeRange:
		return "range"
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
	Fields() map[string]*Value
	Index(executor Executor, index Value, span errors.Span) (*Value, bool, *errors.Error)
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

func (self ValueNull) Type() ValueType   { return TypeNull }
func (self ValueNull) Span() errors.Span { return self.Range }
func (self ValueNull) Fields() map[string]*Value {
	return map[string]*Value{
		"to_json":        marshalHelper(self),
		"to_json_indent": marshalIndentHelper(self),
	}
}
func (self ValueNull) Index(_ Executor, _ Value, span errors.Span) (*Value, bool, *errors.Error) {
	return nil, false, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueNull) Protected() bool { return self.IsProtected }
func (self ValueNull) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "null", nil
}
func (self ValueNull) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return "null (type = null)", nil
}
func (self ValueNull) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	return false, nil
}
func (self ValueNull) IsEqual(_ Executor, span errors.Span, other Value) (bool, *errors.Error) {
	return self.Type() == other.Type(), nil
}

// Boolean value
type ValueBool struct {
	Value       bool
	Range       errors.Span
	IsProtected bool
}

func (self ValueBool) Type() ValueType   { return TypeBoolean }
func (self ValueBool) Span() errors.Span { return self.Range }

// Fields also return a pointer so that member assignments are possible
func (self ValueBool) Fields() map[string]*Value {
	return map[string]*Value{
		"to_json":        marshalHelper(self),
		"to_json_indent": marshalHelper(self),
	}
}

// Index returns a pointer so that index assignments are possible
func (self ValueBool) Index(_ Executor, _ Value, span errors.Span) (*Value, bool, *errors.Error) {
	return nil, false, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueBool) Protected() bool { return self.IsProtected }
func (self ValueBool) Display(_ Executor, _ errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%t", self.Value), nil
}
func (self ValueBool) Debug(_ Executor, _ errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%t (type = bool)", self.Value), nil
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
	if other.Type() != TypeString {
		return nil, errors.NewError(span, fmt.Sprintf("cannot add %v to %v", other.Type(), self.Type()), errors.TypeError)
	}
	// Convert the boolean to a display representation
	display, err := self.Display(executor, span)
	if err != nil {
		return nil, err
	}
	// Return a string concatonation
	return ValueString{Value: display + other.(ValueString).Value}, nil
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

// Function value
type ValueFunction struct {
	Identifier *string
	Args       []struct {
		Identifier string
		Span       errors.Span
	}
	Body        Block
	Range       errors.Span
	IsProtected bool
}

func (self ValueFunction) Type() ValueType           { return TypeFunction }
func (self ValueFunction) Span() errors.Span         { return self.Range }
func (self ValueFunction) Fields() map[string]*Value { return make(map[string]*Value) }
func (self ValueFunction) Index(_ Executor, _ Value, span errors.Span) (*Value, bool, *errors.Error) {
	return nil, false, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
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

func (self ValueBuiltinFunction) Type() ValueType           { return TypeBuiltinFunction }
func (self ValueBuiltinFunction) Span() errors.Span         { return errors.Span{} }
func (self ValueBuiltinFunction) Fields() map[string]*Value { return make(map[string]*Value) }
func (self ValueBuiltinFunction) Index(_ Executor, _ Value, span errors.Span) (*Value, bool, *errors.Error) {
	return nil, false, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
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

func (self ValueBuiltinVariable) Type() ValueType           { return TypeBuiltinVariable }
func (self ValueBuiltinVariable) Span() errors.Span         { return errors.Span{} }
func (self ValueBuiltinVariable) Fields() map[string]*Value { return make(map[string]*Value) }
func (self ValueBuiltinVariable) Index(_ Executor, _ Value, span errors.Span) (*Value, bool, *errors.Error) {
	panic("A bare builtin variable should not exist")
}
func (self ValueBuiltinVariable) Protected() bool {
	return true
}
func (self ValueBuiltinVariable) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return "", err
	}
	return val.Display(executor, span)
}
func (self ValueBuiltinVariable) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return "", err
	}
	return val.Debug(executor, span)
}
func (self ValueBuiltinVariable) IsTrue(executor Executor, span errors.Span) (bool, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	return val.IsTrue(executor, span)
}
func (self ValueBuiltinVariable) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}
	return val.IsEqual(executor, span, other)
}
func (self ValueBuiltinVariable) IsLessThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}

	base, err := compPrep(val, other, span)
	if err != nil {
		return false, err
	}

	return base.IsLessThan(executor, span, other)
}
func (self ValueBuiltinVariable) IsLessThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}

	base, err := compPrep(val, other, span)
	if err != nil {
		return false, err
	}

	return base.IsLessThanOrEqual(executor, span, other)
}
func (self ValueBuiltinVariable) IsGreaterThan(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}

	base, err := compPrep(val, other, span)
	if err != nil {
		return false, err
	}

	return base.IsGreaterThan(executor, span, other)
}
func (self ValueBuiltinVariable) IsGreaterThanOrEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return false, err
	}

	base, err := compPrep(val, other, span)
	if err != nil {
		return false, err
	}

	return base.IsGreaterThanOrEqual(executor, span, other)
}

func (self ValueBuiltinVariable) Add(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}

	base, err := algPrep(val, span)
	if err != nil {
		return nil, err
	}

	return base.Add(executor, span, other)
}

func (self ValueBuiltinVariable) Sub(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}

	base, err := algPrep(val, span)
	if err != nil {
		return nil, err
	}

	return base.Sub(executor, span, other)
}
func (self ValueBuiltinVariable) Mul(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}

	base, err := algPrep(val, span)
	if err != nil {
		return nil, err
	}

	return base.Mul(executor, span, other)
}
func (self ValueBuiltinVariable) Div(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}

	base, err := algPrep(val, span)
	if err != nil {
		return nil, err
	}

	return base.Div(executor, span, other)
}
func (self ValueBuiltinVariable) IntDiv(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}

	base, err := algPrep(val, span)
	if err != nil {
		return nil, err
	}

	return base.IntDiv(executor, span, other)
}
func (self ValueBuiltinVariable) Rem(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}

	base, err := algPrep(val, span)
	if err != nil {
		return nil, err
	}

	return base.Rem(executor, span, other)
}
func (self ValueBuiltinVariable) Pow(executor Executor, span errors.Span, other Value) (Value, *errors.Error) {
	val, err := self.Callback(executor, span)
	if err != nil {
		return nil, err
	}

	base, err := algPrep(val, span)
	if err != nil {
		return nil, err
	}

	return base.Pow(executor, span, other)
}

// Helper functions for values
func getField(executor Executor, span errors.Span, self Value, fieldKey string, isAssignmentLHS bool) (*Value, *errors.Error) {
	value, exists := self.Fields()[fieldKey]
	if !exists {
		if isAssignmentLHS && self.Type() == TypeObject && self.(ValueObject).IsDynamic && !self.(ValueObject).IsProtected {
			ptr := valPtr(ValueNull{})
			self.Fields()[fieldKey] = ptr
			return ptr, nil
		}
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

func makePair(span errors.Span, key Value, value Value) Value {
	return ValuePair{Key: &key, Value: &value, Range: span}
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
		pair := value.(ValuePair)
		value = ValuePair{
			Key:   pair.Key,
			Value: pair.Value,
			Range: span,
		}
	case TypeObject:
		object := value.(ValueObject)
		value = ValueObject{
			DataType:         object.DataType,
			IsDynamic:        object.IsDynamic,
			ObjFields:        object.ObjFields,
			Range:            span,
			CurrentIterIndex: object.CurrentIterIndex,
		}
	case TypeList:
		list := value.(ValueList)
		value = ValueList{
			Values:           list.Values,
			ValueType:        list.ValueType,
			Range:            span,
			CurrentIterIndex: list.CurrentIterIndex,
		}
	case TypeRange:
		rng := value.(ValueRange)
		value = ValueRange{
			Start:   rng.Start,
			End:     rng.End,
			Current: rng.Current,
			Range:   rng.Range,
		}
	}
	// For other types, it is not possible to insert the span, so just return it as is
	return value
}

func compPrep(val Value, other Value, span errors.Span) (ValueRelational, *errors.Error) {
	var baseVal Value
	switch val.Type() {
	case TypeNumber:
		baseVal = val.(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = val.(ValueBuiltinVariable)
	default:
		return nil, errors.NewError(span, fmt.Sprintf("Cannot compare %v to %v", val.Type(), other.Type()), errors.TypeError)
	}

	baseComp := baseVal.(ValueRelational)
	return baseComp, nil
}

func algPrep(val Value, span errors.Span) (ValueAlg, *errors.Error) {
	// Check that the base holds a valid type to perform the requested operations
	var baseVal Value
	switch val.Type() {
	case TypeNumber:
		baseVal = val.(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = val.(ValueBuiltinVariable)
	case TypeString:
		baseVal = val.(ValueString)
	case TypeBoolean:
		baseVal = val.(ValueBool)
	default:
		return nil, errors.NewError(span, fmt.Sprintf("Cannot apply operation on type %v", val.Type()), errors.TypeError)
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(ValueAlg)
	return baseAlg, nil
}
