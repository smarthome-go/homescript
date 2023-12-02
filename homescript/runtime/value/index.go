package value

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

func IndexValue(base *Value, index *Value, span func() errors.Span) (*Value, *VmInterrupt) {
	switch (*base).Kind() {
	case ObjectValueKind, AnyObjectValueKind:
		idx := (*index).(ValueString)
		fields, i := (*base).Fields()
		if i != nil {
			return nil, i
		}
		val, found := fields[idx.Inner]
		if !found {
			return nil, NewVMFatalException(
				fmt.Sprintf("Value of type '%s' has no field named '%s'", (*base).Kind(), idx.Inner),
				Vm_IndexOutOfBoundsErrorKind,
				span(),
			)
		}
		return val, nil
	case ListValueKind:
		list := (*base).(ValueList)
		index := (*index).(ValueInt).Inner

		// handle index wrapping (-1 = len - 1)
		length := int64(len(*list.Values))
		if index < 0 {
			index = index + length
		}

		if index < 0 || index >= length {
			return nil, NewVMFatalException(
				fmt.Sprintf("Index out of bounds: cannot index a list of length %d with %d", len(*list.Values), index),
				Vm_IndexOutOfBoundsErrorKind,
				span(),
			)
		}
		return (*list.Values)[int(index)], nil
	case StringValueKind:
		str := (*base).(ValueString).Inner
		index := (*index).(ValueInt).Inner

		// handle index wrapping (-1 = len - 1)
		length := int64(len(str))
		if index < 0 {
			index = index + length
		}

		if index < 0 || index >= length {
			return nil, NewVMFatalException(
				fmt.Sprintf("Index out of bounds: cannot index a string of length %d with %d", len(str), index),
				Vm_IndexOutOfBoundsErrorKind,
				span(),
			)
		}
		return NewValueString(string((str)[int(index)])), nil
	}

	panic(fmt.Sprintf("A new type which can be indexed was added without updating this code: %s", (*base).Kind()))
}
