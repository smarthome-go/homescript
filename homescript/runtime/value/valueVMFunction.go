package value

import "fmt"

type ValueVMFunction struct {
	Ident string
}

func (_ ValueVMFunction) Kind() ValueKind { return VmFunctionValueKind }

func (self ValueVMFunction) Display() (string, *VmInterrupt) {
	return fmt.Sprintf("<vm-runtime-function (%s)>", self.Ident), nil
}

func (_ ValueVMFunction) IsEqual(other Value) (bool, *VmInterrupt) {
	return false, nil
}

func (_ ValueVMFunction) Fields() (map[string]*Value, *VmInterrupt) {
	return make(map[string]*Value), nil
}

func (self ValueVMFunction) IntoIter() func() (Value, bool) {
	panic("A value of type function cannot be used as an iterator")
}

func (self ValueVMFunction) Clone() *Value {
	return NewValueVMFunction(self.Ident)
}

func NewValueVMFunction(ident string) *Value {
	val := Value(ValueVMFunction{
		Ident: ident,
	})

	return &val
}
