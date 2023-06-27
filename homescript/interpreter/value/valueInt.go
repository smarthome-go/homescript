package value

import (
	"context"
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueInt struct {
	Inner int64
}

func (_ ValueInt) Kind() ValueKind { return IntValueKind }

func (self ValueInt) Display() (string, *Interrupt) {
	return fmt.Sprint(self.Inner), nil // use float fallback
}

func (self ValueInt) IsEqual(other Value) (bool, *Interrupt) {
	if other.Kind() != self.Kind() {
		return false, nil
	}
	return self.Inner == other.(ValueInt).Inner, nil
}

func (self ValueInt) Fields() (map[string]*Value, *Interrupt) {
	return map[string]*Value{
		"to_string": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			disp, i := self.Display()
			if i != nil {
				return nil, i
			}
			return NewValueString(disp), nil
		}),
		"to_range": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			return NewValueRange(
				*NewValueInt(0),
				*NewValueInt(self.Inner),
			), nil
		}),
	}, nil
}

func (self ValueInt) IntoIter() func() (Value, bool) {
	panic("A value of type int cannot be used as an iterator")
}

func NewValueInt(inner int64) *Value {
	val := Value(ValueInt{Inner: inner})
	return &val
}
