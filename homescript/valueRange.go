package homescript

import (
	"fmt"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

// Range value
type ValueRange struct {
	Start       *Value
	End         *Value
	Current     *float64
	Range       errors.Span
	IsProtected bool
}

func (self ValueRange) Type() ValueType   { return TypeRange }
func (self ValueRange) Span() errors.Span { return self.Range }
func (self ValueRange) Protected() bool   { return self.IsProtected }
func (self ValueRange) Fields(_ Executor, _ errors.Span) (map[string]*Value, *errors.Error) {
	return map[string]*Value{
		// Returns the difference of the start and end value
		"diff": valPtr(ValueBuiltinFunction{
			Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				return ValueNumber{Value: float64((*self.Start).(ValueNumber).Value - (*self.End).(ValueNumber).Value)}, nil, nil
			},
		}),
		"start": self.Start,
		"end":   self.End,
	}, nil
}
func (self ValueRange) Index(_ Executor, _ Value, span errors.Span) (*Value, bool, *errors.Error) {
	return nil, false, errors.NewError(span, fmt.Sprintf("cannot index a value of type %v", self.Type()), errors.TypeError)
}
func (self ValueRange) Display(executor Executor, span errors.Span) (string, *errors.Error) {
	return fmt.Sprintf("%d..%d", int((*self.Start).(ValueNumber).Value), int((*self.End).(ValueNumber).Value)), nil
}
func (self ValueRange) Debug(executor Executor, span errors.Span) (string, *errors.Error) {
	start := int((*self.Start).(ValueNumber).Value)
	end := int((*self.End).(ValueNumber).Value)
	return fmt.Sprintf("(%d..%d; start = %d; end = %d)", start, end, start, end), nil
}
func (self ValueRange) IsTrue(_ Executor, _ errors.Span) (bool, *errors.Error) {
	start := (*self.Start).(ValueNumber).Value
	end := (*self.End).(ValueNumber).Value
	return start == 0.0 && end == 0.0, nil
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
func (self *ValueRange) Next(val *Value, span errors.Span) bool {
	if self.Current == nil {
		self.IterReset()
	}

	start := (*self.Start).(ValueNumber).Value
	end := (*self.End).(ValueNumber).Value

	if start < end {
		old := *self.Current
		*self.Current += 1.0

		cond := *self.Current <= end
		if !cond {
			self.IterReset()
		}

		*val = ValueNumber{Value: old, Range: span}
		return cond
	} else {
		old := *self.Current
		*self.Current -= 1.0

		cond := *self.Current >= start
		if !cond {
			self.IterReset()
		}

		*val = ValueNumber{Value: old, Range: span}
		return cond
	}
}
func (self *ValueRange) IterReset() {
	*self.Current = (*self.Start).(ValueNumber).Value
}
