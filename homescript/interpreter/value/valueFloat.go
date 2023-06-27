package value

import (
	"context"
	"fmt"
	"math"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueFloat struct {
	Inner float64
}

func (_ ValueFloat) Kind() ValueKind { return FloatValueKind }

func (self ValueFloat) Display() (string, *Interrupt) {
	return fmt.Sprint(self.Inner), nil // use float fallback
}

func (self ValueFloat) IsEqual(other Value) (bool, *Interrupt) {
	if other.Kind() != self.Kind() {
		return false, nil
	}
	return self.Inner == other.(ValueFloat).Inner, nil
}

func (self ValueFloat) Fields() (map[string]*Value, *Interrupt) {
	return map[string]*Value{
		"is_int": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			isInt := float64(int64(self.Inner)) == self.Inner
			return NewValueBool(isInt), nil
		}),
		"trunc": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			return NewValueInt(int64(math.Trunc(self.Inner))), nil
		}),
		"round": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			return NewValueInt(int64(math.Round(self.Inner))), nil
		}),
		"to_string": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			disp, i := self.Display()
			if i != nil {
				return nil, i
			}
			return NewValueString(disp), nil
		}),
	}, nil
}

func (self ValueFloat) IntoIter() func() (Value, bool) {
	panic("A value of type float cannot be used as an iterator")
}

func NewValueFloat(inner float64) *Value {
	val := Value(ValueFloat{Inner: inner})
	return &val
}
