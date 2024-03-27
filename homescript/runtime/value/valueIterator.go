package value

type ValueIterator struct {
	Func func() (Value, bool)
}

func (_ ValueIterator) Kind() ValueKind { return IteratorValueKind }

func (self ValueIterator) Display() (string, *VmInterrupt) {
	return "<iterator>", nil // use float fallback
}

func (self ValueIterator) IsEqual(other Value) (bool, *VmInterrupt) {
	if other.Kind() != self.Kind() {
		return false, nil
	}
	return false, nil
}

func (self ValueIterator) Fields() (map[string]*Value, *VmInterrupt) {
	panic("A value of type iter does not have any members")
}

func (self ValueIterator) IntoIter() func() (Value, bool) {
	panic("A value of type iter cannot be used as an iterator")
}

func (self ValueIterator) Clone() *Value {
	v := Value(ValueIterator{
		Func: self.Func,
	})
	return &v
}

func NewValueIter(val Value) *Value {
	v := Value(ValueIterator{
		Func: val.IntoIter(),
	})
	return &v
}
