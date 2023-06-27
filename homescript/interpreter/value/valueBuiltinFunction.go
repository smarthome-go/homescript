package value

import (
	"context"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueBuiltinFunction struct {
	Callback func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt)
}

func (_ ValueBuiltinFunction) Kind() ValueKind { return BuiltinFunctionValueKind }

func (self ValueBuiltinFunction) Display() (string, *Interrupt) {
	return "<builtin-function>", nil
}

func (self ValueBuiltinFunction) IsEqual(other Value) (bool, *Interrupt) {
	return false, nil
}

func (self ValueBuiltinFunction) Fields() (map[string]*Value, *Interrupt) {
	return make(map[string]*Value), nil
}

func (self ValueBuiltinFunction) IntoIter() func() (Value, bool) {
	panic("A value of type builtin-function cannot be used as an iterator")
}

func NewValueBuiltinFunction(callback func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt)) *Value {
	val := Value(ValueBuiltinFunction{Callback: callback})
	return &val
}
