package value

type ValueNull struct{}

func (_ ValueNull) Kind() ValueKind { return NullValueKind }

func (self ValueNull) Display() (string, *Interrupt) { return "null", nil }

func (self ValueNull) IsEqual(other Value) (bool, *Interrupt) {
	if other.Kind() == NullValueKind {
		return true, nil
	}
	return false, nil
}

func (self ValueNull) Fields() (map[string]*Value, *Interrupt) {
	return make(map[string]*Value), nil
}

func (self ValueNull) IntoIter() func() (Value, bool) {
	panic("A value of type null cannot be used as an iterator")
}

func NewValueNull() *Value {
	val := Value(ValueNull{})
	return &val
}
