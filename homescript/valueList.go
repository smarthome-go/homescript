package homescript

import (
	"fmt"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

// List value
type ValueList struct {
	Values *[]*Value
	// Is set to a value type the first time the list contains at least 1 element
	ValueType   *ValueType
	Range       errors.Span
	IsProtected bool
}

func (self ValueList) Type() ValueType   { return TypeList }
func (self ValueList) Span() errors.Span { return self.Range }
func (self ValueList) Protected() bool   { return self.IsProtected }
func (self ValueList) Fields() map[string]*Value {
	return map[string]*Value{
		// Returns the length of the list as a number value
		"len": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				return ValueNumber{Value: float64(len(*self.Values))}, nil, nil
			},
		}),
		// Checks if the given element is included in the list
		"contains": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
			if len(args) != 1 {
				return nil, nil, errors.NewError(
					span,
					fmt.Sprintf("function 'contains' requires exactly 1 argument but %d were given", len(args)),
					errors.TypeError,
				)
			}
			for _, value := range *self.Values {
				equal, err := (*value).IsEqual(executor, span, args[0])
				if err != nil {
					return nil, nil, err
				}
				if equal {
					return ValueBool{Value: true}, nil, nil
				}
			}
			return ValueBool{Value: false}, nil, nil
		}}),
		// Appends a list to the end of another list
		"concat": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("concat", span, args, TypeList); err != nil {
					return nil, nil, err
				}
				other := args[0].(ValueList)
				// Check that the types overlap
				if *self.ValueType != TypeUnknown && *other.ValueType != TypeUnknown && *self.ValueType != *other.ValueType {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("cannot concatenate %v<%v> to %v<%v>", TypeList, other.ValueType, TypeList, self.ValueType),
						errors.TypeError,
					)
				}
				*self.Values = append(*self.Values, *other.Values...)
				return ValueNull{}, nil, nil
			},
		}),
		// Joins the list together into a string using the specified seperator
		"join": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("join", span, args, TypeString); err != nil {
					return nil, nil, err
				}
				seperator := args[0].(ValueString).Value
				var output string

				for idx, value := range *self.Values {
					display, err := (*value).Display(executor, span)
					if err != nil {
						return nil, nil, err
					}
					if idx == 0 {
						output = display
					} else {
						output += (seperator + display)
					}
				}
				return ValueString{Value: output}, nil, nil
			},
		}),
		// Adds an element to the end of the list
		"push": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if len(args) != 1 {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("function 'push' takes exactly 1 argument but %d were given", len(args)),
						errors.TypeError,
					)
				}
				if *self.ValueType != TypeUnknown {
					if args[0].Type() != *self.ValueType {
						return nil, nil, errors.NewError(
							span,
							fmt.Sprintf("cannot push value of type %v into %v<%v>", args[0].Type(), TypeList, self.ValueType),
							errors.TypeError,
						)
					}
				} else {
					*self.ValueType = args[0].Type()
				}
				(*self.Values) = append((*self.Values), &args[0])
				return ValueNull{}, nil, nil
			},
		}),
		// Removes the last element of the list and return it
		"pop": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if len(args) != 0 {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("function 'pop' takes no arguments but %d were given", len(args)),
						errors.TypeError,
					)
				}
				length := len(*self.Values)
				// If the list is already empty, do not pop any values
				if length == 0 {
					return ValueNull{}, nil, nil
				}
				// Remove the last slice element
				var last Value
				last, *self.Values = *(*self.Values)[length-1], (*self.Values)[:length-1]
				// Return the recently popped value
				return last, nil, nil
			},
		}),
		// Adds an alement to the front of the list
		"push_front": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if len(args) != 1 {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("function 'push_front' takes exactly 1 arguments but %d were given", len(args)),
						errors.TypeError,
					)
				}
				*self.Values = append([]*Value{&args[0]}, *self.Values...)
				return ValueNull{}, nil, nil
			},
		}),
		// Removes the first element of the list and returns it
		"pop_front": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if len(args) != 0 {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("function 'pop_front' takes no arguments but %d were given", len(args)),
						errors.TypeError,
					)
				}
				length := len(*self.Values)
				// If the list is already empty, do not pop any values
				if length == 0 {
					return ValueNull{}, nil, nil
				}
				// Remove the first slice element
				var first Value
				first, *self.Values = *(*self.Values)[0], (*self.Values)[1:]
				// Return the recently popped value
				return first, nil, nil
			},
		}),
		// Inserts a value at the location before a given index
		// Consider following list: `let a = [0, 1, 2, 3]`
		// If the function is invoked like `a.insert(2, 9)`
		// following result will occur [`0, 1, 9, 2, 3`]
		// This is because the 9 is inserted before index 2, meaning before the number 2
		"insert": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if len(args) != 2 {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("function 'insert' takes exactly 2 arguments but %d were given", len(args)),
						errors.TypeError,
					)
				}
				// Check that the first argument if a whole number
				if args[0].Type() != TypeNumber {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("type %v<%v> cannot be indexed by %v", TypeList, self.ValueType, args[0].Type()),
						errors.TypeError,
					)
				}
				num := args[0].(ValueNumber).Value
				if float64(int(num)) != num {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("type %v<%v> must be indexed by integer numbers, found float", TypeList, self.ValueType),
						errors.ValueError,
					)
				}
				// Check the index bounds
				index := int(num)
				// Enable index wrapping (-1 = len-1)
				length := len(*self.Values)
				if index < 0 {
					index = index + length
				}
				if index < 0 || index > length {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("index out of bounds: index is %d, but length is %d", index, length),
						errors.OutOfBoundsError,
					)
				}
				if len(*self.Values) == index {
					*self.Values = append(*self.Values, &args[1])
					return ValueNull{}, nil, nil
				}
				*self.Values = append((*self.Values)[:index+1], (*self.Values)[index:]...)
				(*self.Values)[index] = &args[1]
				return ValueNull{}, nil, nil
			},
		}),
		"remove": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if len(args) != 1 {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("function 'remove' takes exactly 1 argument but %d were given", len(args)),
						errors.TypeError,
					)
				}
				// Check that the first argument if a whole number
				if args[0].Type() != TypeNumber {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("type %v<%v> cannot be indexed by %v", TypeList, self.ValueType, args[0].Type()),
						errors.TypeError,
					)
				}
				num := args[0].(ValueNumber).Value
				if float64(int(num)) != num {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("type %v<%v> must be indexed by integer numbers, found float", TypeList, self.ValueType),
						errors.ValueError,
					)
				}
				// Check the index bounds
				index := int(num)
				// Enable index wrapping (-1 = len-1)
				length := len(*self.Values)
				if index < 0 {
					index = index + length
				}
				if index < 0 || index >= length {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("index out of bounds: index is %d, but length is %d", index, length),
						errors.OutOfBoundsError,
					)
				}
				*self.Values = append((*self.Values)[:index], (*self.Values)[index+1:]...)
				return ValueNull{}, nil, nil
			},
		}),
		"to_json":        marshalHelper(self),
		"to_json_indent": marshalIndentHelper(self),
	}
}
func (self ValueList) Index(executor Executor, indexValue Value, span errors.Span) (*Value, bool, *errors.Error) {
	// Check the type
	if indexValue.Type() != TypeNumber {
		return nil, false, errors.NewError(
			span,
			fmt.Sprintf("cannot index value of type '%v' by a value of type '%v'", TypeList, indexValue.Type()),
			errors.TypeError,
		)
	}
	if float64(int(indexValue.(ValueNumber).Value)) != indexValue.(ValueNumber).Value {
		return nil, false, errors.NewError(
			span,
			fmt.Sprintf("cannot index value of type '%v' by a float number", TypeList),
			errors.ValueError,
		)
	}
	// Check the length
	index := int(indexValue.(ValueNumber).Value)
	length := len(*self.Values)
	if index < 0 {
		index = index + length
	}
	if index < 0 || index >= length {
		return nil, false, errors.NewError(
			span,
			fmt.Sprintf("index out of bounds: index is %d, but length is %d", index, length),
			errors.OutOfBoundsError,
		)
	}
	return (*self.Values)[index], true, nil
}
func (self ValueList) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	length := len(*self.Values)
	output := "["
	for idx, value := range *self.Values {
		display, err := (*value).Display(executor, span)
		if err != nil {
			return "", err
		}
		output += display
		if idx < length-1 {
			output += ", "
		}
	}
	return output + "]", nil
}
func (self ValueList) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	length := len(*self.Values)
	output := fmt.Sprintf("(\n    len = %d\n    type = list<%s>\n    values = [", length, self.ValueType.String())
	values := make([]string, 0)
	multiline := false
	for _, value := range *self.Values {
		display, err := (*value).Debug(executor, span)
		if err != nil {
			return "", err
		}
		values = append(values, display)
		if len(display) > 10 {
			multiline = true
		}
	}
	if multiline {
		output += "\n"
	}
	for idx, value := range values {
		if multiline {
			output += "        " + value
		} else {
			output += value
		}
		if idx < length-1 {
			output += ","
			if multiline {
				output += "\n"
			} else {
				output += " "
			}
		}
	}
	if multiline {
		output += "\n    "
	}
	return output + "]\n)", nil
}
func (self ValueList) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	return false, nil
}
func (self ValueList) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
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
	if len(*self.Values) != len(*otherList.Values) {
		return false, nil
	}
	// Check for equality of every index
	for idx, left := range *self.Values {
		isEqual, err := (*left).IsEqual(
			executor,
			span,
			*(*otherList.Values)[idx],
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
