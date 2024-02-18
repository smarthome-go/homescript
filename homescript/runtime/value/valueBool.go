package value

import (
	"context"
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueBool struct {
	Inner bool
}

func (_ ValueBool) Kind() ValueKind { return BoolValueKind }

func (self ValueBool) Display() (string, *VmInterrupt) { return fmt.Sprint(self.Inner), nil }

func (self ValueBool) IsEqual(other Value) (bool, *VmInterrupt) {
	return self.Inner == other.(ValueBool).Inner, nil
}

func (self ValueBool) Fields() (map[string]*Value, *VmInterrupt) {
	return map[string]*Value{
		"to_string": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *VmInterrupt) {
			display, i := self.Display()
			if i != nil {
				return nil, i
			}
			return NewValueString(display), nil
		}),
	}, nil
}

func (self ValueBool) IntoIter() func() (Value, bool) {
	panic("A value of type bool cannot be used as an iterator")
}

func NewValueBool(inner bool) *Value {
	val := Value(ValueBool{Inner: inner})
	return &val
}
