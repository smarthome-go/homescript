package value

import (
	"context"
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueList struct {
	// both the sclice and its contents are pointers so that interior mutability can be leveraged
	Values *[]*Value
	// saves the index of the element of the current iteration
	currIterIdx *int
}

func (_ ValueList) Kind() ValueKind { return ListValueKind }

func (self ValueList) Display() (string, *Interrupt) {
	values := make([]string, 0)
	for _, val := range *self.Values {
		disp, err := (*val).Display()
		if err != nil {
			return "", err
		}

		values = append(values, disp)
	}
	return fmt.Sprintf("[%s]", strings.Join(values, ", ")), nil
}

func (self ValueList) IsEqual(other Value) (bool, *Interrupt) {
	otherList := other.(ValueList)
	// check length
	if len(*otherList.Values) != len(*self.Values) {
		return false, nil
	}

	for idx := 0; idx < len(*self.Values); idx++ {
		this := *(*self.Values)[idx]
		other := *(*otherList.Values)[idx]

		if this.Kind() != other.Kind() {
			return false, nil
		}

		equal, i := this.IsEqual(other)
		if i != nil || !equal {
			return equal, i
		}
	}

	return true, nil
}

func (self ValueList) Fields() (map[string]*Value, *Interrupt) {
	return map[string]*Value{
		"to_string": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			dispay, i := self.Display()
			if i != nil {
				return nil, i
			}
			return NewValueString(dispay), nil
		}),
		"len": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			return NewValueInt(int64(len(*self.Values))), nil
		}),
		"contains": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			compareTo := args[0]

			// check equality for each element
			for _, item := range *self.Values {
				equal, i := compareTo.IsEqual(*item)
				if i != nil {
					return nil, i
				}

				if equal {
					return NewValueBool(true), nil
				}
			}

			return NewValueBool(false), nil
		}),
		"concat": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			other := args[0].(ValueList)
			*self.Values = append(*self.Values, *other.Values...)
			return NewValueNull(), nil
		}),
		"join": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			separator := args[0].(ValueString).Inner
			var output string
			for idx, value := range *self.Values {
				display, i := (*value).Display()
				if i != nil {
					return nil, i
				}
				if idx == 0 {
					output = display
				} else {
					output += (separator + display)
				}
			}
			return NewValueString(output), nil
		}),
		"push": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			*self.Values = append(*self.Values, &args[0])
			return NewValueNull(), nil
		}),
		"pop": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			length := len(*self.Values)
			// if the list is already empty, do not pop any values
			if length == 0 {
				return NewValueNull(), nil
			}
			// remove the last slice element
			var last Value
			last, *self.Values = *(*self.Values)[length-1], (*self.Values)[:length-1]
			// return the recently popped value
			return &last, nil
		}),
		"push_front": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			*self.Values = append([]*Value{&args[0]}, *self.Values...)
			return NewValueNull(), nil
		}),
		"pop_front": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			length := len(*self.Values)
			// if the list is already empty, do not pop any values
			if length == 0 {
				return NewValueNull(), nil
			}
			// remove the first slice element
			var first Value
			first, *self.Values = *(*self.Values)[0], (*self.Values)[1:]
			// return the recently popped value
			return &first, nil
		}),
		"insert": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			// check the index bounds
			index := int(args[0].(ValueInt).Inner)
			// enable index wrapping (-1 = len-1)
			length := len(*self.Values)
			if index < 0 {
				index = index + length
			}
			if index < 0 || index > length {
				return nil, NewRuntimeErr(
					fmt.Sprintf("Index out of bounds: the index is %d, the but length is %d", index, length),
					IndexOutOfBoundsErrorKind,
					span,
				)
			}
			if len(*self.Values) == index {
				*self.Values = append(*self.Values, &args[1])
				return NewValueNull(), nil
			}
			*self.Values = append((*self.Values)[:index+1], (*self.Values)[index:]...)
			(*self.Values)[index] = &args[1]
			return NewValueNull(), nil
		}),
		"remove": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			// check the index bounds
			index := int(args[0].(ValueInt).Inner)
			// enable index wrapping (-1 = len-1)
			length := len(*self.Values)
			if index < 0 {
				index = index + length
			}
			if index < 0 || index >= length {
				return nil, NewRuntimeErr(
					fmt.Sprintf("Index out of bounds: the index is %d, the but length is %d", index, length),
					IndexOutOfBoundsErrorKind,
					span,
				)
			}
			*self.Values = append((*self.Values)[:index], (*self.Values)[index+1:]...)
			return NewValueNull(), nil
		}),
		"sort": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			if len(*self.Values) == 0 {
				return NewValueNull(), nil
			}

			switch (*(*self.Values)[0]).Kind() {
			case IntValueKind:
				self.insertionSortInt()
				return NewValueNull(), nil
			case FloatValueKind:
				self.insertionSortFloat()
				return NewValueNull(), nil
			case StringValueKind:
				self.insertionSortString()
				return NewValueNull(), nil
			}

			panic("Unreachable, this inner type is not supported")
		}),
		"last": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			length := len(*self.Values)
			if length == 0 {
				return NewNoneOption(), nil
			}
			return NewValueOption((*self.Values)[length-1]), nil
		}),
		"to_json":        marshalHelper(self),
		"to_json_indent": marshalIndentHelper(self),
	}, nil
}

func (self ValueList) iterNext() (Value, bool) {
	if self.currIterIdx == nil {
		self.iterReset()
	}

	old := *self.currIterIdx
	*self.currIterIdx++

	shouldContinue := *self.currIterIdx <= len(*self.Values)

	if shouldContinue {
		return *(*self.Values)[old], true
	} else {
		self.iterReset()
		return nil, false
	}
}

func (self ValueList) iterReset() {
	*self.currIterIdx = 0
}

func (self ValueList) IntoIter() func() (Value, bool) {
	return self.iterNext
}

func NewValueList(values []*Value) *Value {
	zero := 0
	val := Value(ValueList{Values: &values, currIterIdx: &zero})
	return &val
}

//
// Insertion sort for ValueInt
//

func (self ValueList) insertionSortInt() {
	for currentIndex := 1; currentIndex < len(*self.Values); currentIndex++ {
		temp := (*self.Values)[currentIndex]
		iterator := currentIndex
		for ; iterator > 0 && (*(*self.Values)[iterator-1]).(ValueInt).Inner > (*temp).(ValueInt).Inner; iterator-- {
			(*self.Values)[iterator] = (*self.Values)[iterator-1]
		}
		(*self.Values)[iterator] = temp
	}
}

//
// Insertion sort for ValueFloat
//

func (self ValueList) insertionSortFloat() {
	for currentIndex := 1; currentIndex < len(*self.Values); currentIndex++ {
		temp := (*self.Values)[currentIndex]
		iterator := currentIndex
		for ; iterator > 0 && (*(*self.Values)[iterator-1]).(ValueFloat).Inner > (*temp).(ValueFloat).Inner; iterator-- {
			(*self.Values)[iterator] = (*self.Values)[iterator-1]
		}
		(*self.Values)[iterator] = temp
	}
}

func (self ValueList) insertionSortString() {
	for currentIndex := 1; currentIndex < len(*self.Values); currentIndex++ {
		temp := (*self.Values)[currentIndex]
		iterator := currentIndex
		for ; iterator > 0 && strings.Compare((*(*self.Values)[iterator-1]).(ValueString).Inner, (*temp).(ValueString).Inner) == 1; iterator-- {
			(*self.Values)[iterator] = (*self.Values)[iterator-1]
		}
		(*self.Values)[iterator] = temp
	}
}
