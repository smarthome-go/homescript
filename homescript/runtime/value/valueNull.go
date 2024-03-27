package value

type ValueNull struct{}

func (_ ValueNull) Kind() ValueKind { return NullValueKind }

func (self ValueNull) Display() (string, *VmInterrupt) { return "null", nil }

func (self ValueNull) IsEqual(other Value) (bool, *VmInterrupt) {
	if other.Kind() == NullValueKind {
		return true, nil
	}
	return false, nil
}

func (self ValueNull) Fields() (map[string]*Value, *VmInterrupt) {
	return make(map[string]*Value), nil
}

func (self ValueNull) IntoIter() func() (Value, bool) {
	panic("A value of type null cannot be used as an iterator")
}

func (self ValueNull) Clone() *Value {
	return NewValueNull()
}

func NewValueNull() *Value {
	val := Value(ValueNull{})
	return &val
}
