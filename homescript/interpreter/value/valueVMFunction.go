package value

type ValueVMFunction struct {
	Ident string
}

func (_ ValueVMFunction) Kind() ValueKind { return VmFunctionValueKind }

func (_ ValueVMFunction) Display() (string, *Interrupt) {
	return "<vm-runtime-function>", nil
}

func (_ ValueVMFunction) IsEqual(other Value) (bool, *Interrupt) {
	return false, nil
}

func (_ ValueVMFunction) Fields() (map[string]*Value, *Interrupt) {
	return make(map[string]*Value), nil
}

func (self ValueVMFunction) IntoIter() func() (Value, bool) {
	panic("A value of type function cannot be used as an iterator")
}

func NewValueVMFunction(ident string) *Value {
	val := Value(ValueVMFunction{
		Ident: ident,
	})

	return &val
}
