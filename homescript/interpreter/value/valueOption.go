package value

import (
	"context"
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueOption struct {
	Inner *Value
}

func (self ValueOption) IsSome() bool {
	return self.Inner != nil
}

func (_ ValueOption) Kind() ValueKind { return OptionValueKind }

func (self ValueOption) Display() (string, *Interrupt) {
	switch self.IsSome() {
	case true:
		disp, i := (*self.Inner).Display()
		if i != nil {
			return "", i
		}
		return fmt.Sprintf("Some(%s)", disp), nil
	case false:
		return "null", nil
	}
	panic("A boolean is binary")
}

func (self ValueOption) IsEqual(other Value) (bool, *Interrupt) {
	if other.Kind() != OptionValueKind {
		return false, nil
	}

	otherOpt := other.(ValueOption)
	selfIsSome := self.IsSome()
	otherIsSome := otherOpt.IsSome()

	if selfIsSome && otherIsSome {
		return (*self.Inner).IsEqual(*otherOpt.Inner)
	} else if !selfIsSome && !otherIsSome {
		return true, nil
	} else {
		return false, nil
	}
}

func (self ValueOption) Fields() (map[string]*Value, *Interrupt) {
	return map[string]*Value{
		"is_some": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			return NewValueBool(self.IsSome()), nil
		}),
		"is_none": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			return NewValueBool(!self.IsSome()), nil
		}),
		"unwrap": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			if !self.IsSome() {
				return nil, NewRuntimeErr(
					"Called 'unwrap' on a 'null' option value",
					ValueErrorKind,
					span,
				)
			}
			return self.Inner, nil
		}),
		"expect": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			if !self.IsSome() {
				return nil, NewRuntimeErr(
					args[0].(ValueString).Inner,
					ValueErrorKind,
					span,
				)
			}
			return self.Inner, nil
		}),
	}, nil
}

func (self ValueOption) IntoIter() func() (Value, bool) {
	panic("A value of type option cannot be used as an iterator")
}

func NewNoneOption() *Value {
	return NewValueOption(nil)
}

func NewValueOption(inner *Value) *Value {
	val := Value(ValueOption{
		Inner: inner,
	})
	return &val
}
