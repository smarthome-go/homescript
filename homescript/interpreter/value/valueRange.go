package value

import (
	"context"
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type ValueRange struct {
	Start          *Value
	End            *Value
	EndIsInclusive bool
	IterCurrent    *int64
}

func (_ ValueRange) Kind() ValueKind { return RangeValueKind }

func (self ValueRange) Display() (string, *Interrupt) {
	return fmt.Sprintf("%d..%d", *self.Start, *self.End), nil
}

func (self ValueRange) IsEqual(other Value) (bool, *Interrupt) {
	otherRange := other.(ValueRange)
	return *self.Start == *otherRange.Start && *self.End == *otherRange.End, nil
}

func (self ValueRange) Fields() (map[string]*Value, *Interrupt) {
	return map[string]*Value{
		"start": self.Start,
		"end":   self.End,
		"rev": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			start := (*self.Start).(ValueInt).Inner
			end := (*self.End).(ValueInt).Inner
			return NewValueRange(*NewValueInt(end), *NewValueInt(start), self.EndIsInclusive), nil
		}),
		"diff": NewValueBuiltinFunction(func(executor Executor, cancelCtx *context.Context, span errors.Span, args ...Value) (*Value, *Interrupt) {
			start := (*self.Start).(ValueInt).Inner
			end := (*self.End).(ValueInt).Inner

			var diff int64
			if start > end {
				diff = start - end
			} else {
				diff = end - start
			}

			return NewValueInt(diff), nil
		}),
	}, nil
}

func (self ValueRange) iterNext() (Value, bool) {
	if self.IterCurrent == nil {
		self.iterReset()
	}

	start := (*self.Start).(ValueInt).Inner
	end := (*self.End).(ValueInt).Inner

	// TODO: is this OK?

	if start < end {
		if self.EndIsInclusive {
			end++
		}

		old := *self.IterCurrent
		*self.IterCurrent++

		cond := *self.IterCurrent <= end
		if !cond {
			self.iterReset()
		}

		return *NewValueInt(old), cond
	} else {
		if self.EndIsInclusive {
			end--
		}

		old := *self.IterCurrent
		*self.IterCurrent--

		cond := *self.IterCurrent >= end
		if !cond {
			self.iterReset()
		}

		return *NewValueInt(old), cond
	}
}

func (self ValueRange) iterReset() {
	reset := (*self.Start).(ValueInt).Inner
	*self.IterCurrent = reset
}

func (self ValueRange) IntoIter() func() (Value, bool) {
	return self.iterNext
}

func NewValueRange(start Value, end Value, endIsInclusive bool) *Value {
	startInt := start.(ValueInt).Inner
	val := Value(ValueRange{
		Start:          &start,
		End:            &end,
		EndIsInclusive: endIsInclusive,
		IterCurrent:    &startInt,
	})
	return &val
}
