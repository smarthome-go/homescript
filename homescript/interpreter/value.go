package interpreter

import (
	"fmt"

	"github.com/smarthome-go/homescript/homescript/error"
)

type ValueType uint8

const (
	Void ValueType = iota
	Number
	String
	Boolean
	Function
	Variable
)

func (self ValueType) Name() string {
	switch self {
	case Void:
		return "Void"
	case Number:
		return "Number"
	case String:
		return "String"
	case Boolean:
		return "Boolean"
	case Function:
		return "Function"
	case Variable:
		return "Variable"
	default:
		panic(0)
	}
}

type Value interface {
	Type() ValueType
	ToString(executor Executor, location error.Location) (string, *error.Error)
	IsTrue(executor Executor, location error.Location) (bool, *error.Error)
	IsEqual(executor Executor, location error.Location, other Value) (bool, *error.Error)
}

type ValueRelational interface {
	IsLessThan(executor Executor, other Value, location error.Location) (bool, *error.Error)
	IsLessThanOrEqual(executor Executor, other Value, location error.Location) (bool, *error.Error)
	IsGreaterThan(executor Executor, other Value, location error.Location) (bool, *error.Error)
	IsGreaterThanOrEqual(executor Executor, other Value, location error.Location) (bool, *error.Error)
}

type ValueVoid struct{}

func (self ValueVoid) Type() ValueType { return Void }
func (self ValueVoid) ToString(_ Executor, _ error.Location) (string, *error.Error) {
	return "void", nil
}
func (self ValueVoid) IsTrue(_ Executor, _ error.Location) (bool, *error.Error) { return false, nil }
func (self ValueVoid) IsEqual(_ Executor, _ error.Location, other Value) (bool, *error.Error) {
	return other.Type() == Void, nil
}

type ValueNumber struct{ Value float64 }

func (self ValueNumber) Type() ValueType { return Number }
func (self ValueNumber) ToString(_ Executor, _ error.Location) (string, *error.Error) {
	return fmt.Sprint(self.Value), nil
}
func (self ValueNumber) IsTrue(_ Executor, _ error.Location) (bool, *error.Error) {
	return self.Value != 0, nil
}
func (self ValueNumber) IsEqual(_ Executor, _ error.Location, other Value) (bool, *error.Error) {
	return other.Type() == Number && self.Value == other.(ValueNumber).Value, nil
}
func (self ValueNumber) IsLessThan(executor Executor, other Value, location error.Location) (bool, *error.Error) {
	var val Value
	if other.Type() == Variable {
		temp, err := other.(ValueVariable).Callback(executor, location)
		if err != nil {
			return false, err
		}
		val = temp
	} else {
		val = other
	}
	if val.Type() != Number {
		return false, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Cannot compare %s type with %s type", self.Type().Name(), val.Type().Name()),
		)
	}
	return self.Value < val.(ValueNumber).Value, nil
}
func (self ValueNumber) IsLessThanOrEqual(executor Executor, other Value, location error.Location) (bool, *error.Error) {
	var val Value
	if other.Type() == Variable {
		temp, err := other.(ValueVariable).Callback(executor, location)
		if err != nil {
			return false, err
		}
		val = temp
	} else {
		val = other
	}
	if val.Type() != Number {
		return false, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Cannot compare %s type with %s type", self.Type().Name(), val.Type().Name()),
		)
	}
	return self.Value <= val.(ValueNumber).Value, nil
}
func (self ValueNumber) IsGreaterThan(executor Executor, other Value, location error.Location) (bool, *error.Error) {
	var val Value
	if other.Type() == Variable {
		temp, err := other.(ValueVariable).Callback(executor, location)
		if err != nil {
			return false, err
		}
		val = temp
	} else {
		val = other
	}
	if val.Type() != Number {
		return false, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Cannot compare %s type with %s type", self.Type().Name(), val.Type().Name()),
		)
	}
	return self.Value > val.(ValueNumber).Value, nil
}
func (self ValueNumber) IsGreaterThanOrEqual(executor Executor, other Value, location error.Location) (bool, *error.Error) {
	var val Value
	if other.Type() == Variable {
		temp, err := other.(ValueVariable).Callback(executor, location)
		if err != nil {
			return false, err
		}
		val = temp
	} else {
		val = other
	}
	if val.Type() != Number {
		return false, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Cannot compare %s type with %s type", self.Type().Name(), val.Type().Name()),
		)
	}
	return self.Value >= val.(ValueNumber).Value, nil
}

type ValueString struct{ Value string }

func (self ValueString) Type() ValueType { return String }
func (self ValueString) ToString(_ Executor, _ error.Location) (string, *error.Error) {
	return self.Value, nil
}
func (self ValueString) IsTrue(_ Executor, _ error.Location) (bool, *error.Error) {
	return self.Value != "", nil
}
func (self ValueString) IsEqual(_ Executor, _ error.Location, other Value) (bool, *error.Error) {
	return other.Type() == String && self.Value == other.(ValueString).Value, nil
}

type ValueBoolean struct{ Value bool }

func (self ValueBoolean) Type() ValueType { return Boolean }
func (self ValueBoolean) ToString(_ Executor, _ error.Location) (string, *error.Error) {
	return fmt.Sprintf("%t", self.Value), nil
}
func (self ValueBoolean) IsTrue(_ Executor, _ error.Location) (bool, *error.Error) {
	return self.Value, nil
}
func (self ValueBoolean) IsEqual(_ Executor, _ error.Location, other Value) (bool, *error.Error) {
	return other.Type() == Boolean && self.Value == other.(ValueBoolean).Value, nil
}

type ValueFunction struct {
	Callback func(executor Executor, location error.Location, args ...Value) (Value, *error.Error)
}

func (self ValueFunction) Type() ValueType { return Function }
func (self ValueFunction) ToString(_ Executor, _ error.Location) (string, *error.Error) {
	return "<function>", nil
}
func (self ValueFunction) IsTrue(_ Executor, _ error.Location) (bool, *error.Error) {
	return false, nil
}
func (self ValueFunction) IsEqual(_ Executor, _ error.Location, other Value) (bool, *error.Error) {
	return other.Type() == Function && fmt.Sprintf("%p", self.Callback) == fmt.Sprintf("%p", other.(ValueFunction).Callback), nil
}

type ValueVariable struct {
	Callback func(executor Executor, location error.Location) (Value, *error.Error)
}

func (self ValueVariable) Type() ValueType { return Variable }
func (self ValueVariable) ToString(executor Executor, location error.Location) (string, *error.Error) {
	val, err := self.Callback(executor, location)
	if err != nil {
		return "", err
	}
	str, _ := val.ToString(executor, location)
	return str, nil
}
func (self ValueVariable) IsTrue(executor Executor, location error.Location) (bool, *error.Error) {
	val, err := self.Callback(executor, location)
	if err != nil {
		return false, err
	}
	res, _ := val.IsTrue(executor, location)
	return res, nil
}
func (self ValueVariable) IsEqual(executor Executor, location error.Location, other Value) (bool, *error.Error) {
	val, err := self.Callback(executor, location)
	if err != nil {
		return false, err
	}
	res, _ := val.IsEqual(executor, location, other)
	return res, nil
}
func (self ValueVariable) IsLessThan(executor Executor, other Value, location error.Location) (bool, *error.Error) {
	val, err := self.Callback(executor, location)
	if err != nil {
		return false, err
	}
	if val.Type() != Number {
		return false, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Cannot compare %s type with %s type", val.Type().Name(), other.Type().Name()),
		)
	}
	return val.(ValueNumber).IsLessThan(executor, other, location)
}
func (self ValueVariable) IsLessThanOrEqual(executor Executor, other Value, location error.Location) (bool, *error.Error) {
	val, err := self.Callback(executor, location)
	if err != nil {
		return false, err
	}
	if val.Type() != Number {
		return false, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Cannot compare %s type with %s type", val.Type().Name(), other.Type().Name()),
		)
	}
	return val.(ValueNumber).IsLessThanOrEqual(executor, other, location)
}
func (self ValueVariable) IsGreaterThan(executor Executor, other Value, location error.Location) (bool, *error.Error) {
	val, err := self.Callback(executor, location)
	if err != nil {
		return false, err
	}
	if val.Type() != Number {
		return false, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Cannot compare %s type with %s type", val.Type().Name(), other.Type().Name()),
		)
	}
	return val.(ValueNumber).IsGreaterThan(executor, other, location)
}
func (self ValueVariable) IsGreaterThanOrEqual(executor Executor, other Value, location error.Location) (bool, *error.Error) {
	val, err := self.Callback(executor, location)
	if err != nil {
		return false, err
	}
	if val.Type() != Number {
		return false, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Cannot compare %s type with %s type", val.Type().Name(), other.Type().Name()),
		)
	}
	return val.(ValueNumber).IsGreaterThanOrEqual(executor, other, location)
}
