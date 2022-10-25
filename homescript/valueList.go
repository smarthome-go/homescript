package homescript

import (
	"fmt"

	"github.com/smarthome-go/homescript/homescript/errors"
)

// List value
type ValueList struct {
	Values []Value
	// Is set to a value type the first time the list contains at least 1 element
	ValueType   *ValueType
	Range       errors.Span
	IsProtected bool
}

func (self ValueList) Type() ValueType   { return TypeList }
func (self ValueList) Span() errors.Span { return self.Range }
func (self ValueList) Protected() bool   { return self.IsProtected }
func (self ValueList) Fields() map[string]Value {
	return map[string]Value{
		"len": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				panic("Not yet implemented")
			},
		},
		"concat": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				panic("Not yet implemented")
			},
		},
		"push": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				panic("Not yet implemented")
			},
		},
		"pop": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				panic("Not yet implemented")
			},
		},
		"push_front": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				panic("Not yet implemented")
			},
		},
		"pop_front": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				panic("Not yet implemented")
			},
		},
		"insert": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				panic("Not yet implemented")
			},
		},
		"remove": ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				panic("Not yet implemented")
			},
		},
	}
}
func (self ValueList) Index(executor Executor, index int, span errors.Span) (Value, *errors.Error) {
	// Check the length
	length := len(self.Values)
	if index < 0 {
		index = index + length
	}
	if index < 0 || index >= length {
		return nil, errors.NewError(
			span,
			fmt.Sprintf("index out of bounds: index is %d, but length is %d", index, length, self.Type()),
			errors.OutOfBoundsError,
		)
	}
	return self.Values[index], nil
}
func (self ValueList) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	length := len(self.Values)
	output := "["
	for idx, value := range self.Values {
		display, err := value.Display(executor, span)
		if err != nil {
			return "", err
		}
		output += display
		if idx < length {
			output += ", "
		}
	}
	return output + "]", nil
}
func (self ValueList) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	length := len(self.Values)
	output := "["
	for idx, value := range self.Values {
		display, err := value.Display(executor, span)
		if err != nil {
			return "", err
		}
		output += display
		if idx < length {
			output += ", "
			if len(display) > 10 {
				output += "\n"
			}
		}
	}
	return output + "]", nil
}
func (self ValueList) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	return false, nil
}
func (self ValueList) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
	if self.Type() != other.Type() {
		return false, errors.NewError(
			span,
			fmt.Sprintf("cannot compare %v to %v", self.Type(), other.Type()),
			errors.TypeError,
		)
	}
	otherList := other.(ValueList)
	// Check that list types are identical
	if self.ValueType == nil && otherList.ValueType == nil {
		return true, nil
	} else if *self.ValueType != *otherList.ValueType {
		return false, errors.NewError(
			span,
			fmt.Sprintf("cannot compare %v<%v> to %v<%v>", TypeList, self.ValueType, TypeList, otherList.ValueType),
			errors.TypeError,
		)
	}
	// Check that length is identical
	if len(self.Values) != len(otherList.Values) {
		return false, nil
	}
	// Check for equality of every index
	for idx, left := range self.Values {
		isEqual, err := left.IsEqual(
			executor,
			span,
			otherList.Values[idx],
		)
		if err != nil {
			return false, err
		}
		if !isEqual {
			return false, nil
		}
	}
	// If every item so far was equal, return here
	return true, nil
}
