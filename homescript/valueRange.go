package homescript

import (
	"fmt"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

// Range value
type ValueRange struct {
	Start       *float64
	End         *float64
	Current     *float64
	Range       errors.Span
	IsProtected bool
}

func (self ValueRange) Type() ValueType   { return TypeRange }
func (self ValueRange) Span() errors.Span { return self.Range }
func (self ValueRange) Protected() bool   { return self.IsProtected }
func (self ValueRange) Fields() map[string]*Value {
	return map[string]*Value{
		// Returns the difference of the start and end value
		"diff": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				return ValueNumber{Value: float64(*self.Start - *self.End)}, nil, nil
			},
		}),
		"start": valPtr(ValueNumber{Value: *self.Start}),
		"end":   valPtr(ValueNumber{Value: *self.End}),
	}
}
func (self ValueRange) Index(_ Executor, _ Value, span errors.Span) (*Value, bool, *errors.Error) {
	return nil, false, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueRange) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%d..%d", int(*self.Start), int(*self.End)), nil
}
func (self ValueRange) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("(%d..%d; start = %f; end = %f)", int(*self.Start), int(*self.End), *self.Start, *self.End), nil
}
func (self ValueRange) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	return *self.Start == 0.0 && *self.End == 0.0, nil
}
func (self ValueRange) IsEqual(executor Executor, span errors.Span, other Value) (bool, *errors.Error) {
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
	otherRange := other.(ValueRange)
	return otherRange.Start == self.Start && otherRange.End == self.End, nil
}
func (self *ValueRange) Next() (Value, bool) {
	if *self.Start < *self.End {
		old := *self.Current
		*self.Current += 1.0
		return ValueNumber{Value: old}, *self.Current <= *self.End
	} else {
		old := *self.Current
		*self.Current -= 1.0
		return ValueNumber{Value: old}, *self.Current >= *self.Start
	}
}
