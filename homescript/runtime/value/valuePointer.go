package value

import (
	"fmt"
)

type ValuePointer struct {
	Inner *Value
}

func (_ ValuePointer) Kind() ValueKind { return PointerValueKind }

func (self ValuePointer) Display() (string, *VmInterrupt) {
	return fmt.Sprint(self.Inner), nil // use float fallback
}

func (self ValuePointer) IsEqual(other Value) (bool, *VmInterrupt) {
	if other.Kind() != self.Kind() {
		return false, nil
	}
	return self.Inner == other.(ValuePointer).Inner, nil
}

func (self ValuePointer) Fields() (map[string]*Value, *VmInterrupt) {
	panic("A value of type pointer does not have any members")
}

func (self ValuePointer) IntoIter() func() (Value, bool) {
	panic("A value of type int cannot be used as an iterator")
}

func NewValuePointer(inner *Value) *Value {
	val := Value(ValuePointer{Inner: inner})
	return &val
}
