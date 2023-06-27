package value

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

type ValueFunction struct {
	Module string
	Block  ast.AnalyzedBlock
}

func (_ ValueFunction) Kind() ValueKind { return FunctionValueKind }

func (_ ValueFunction) Display() (string, *Interrupt) {
	return "<function>", nil
}

func (_ ValueFunction) IsEqual(other Value) (bool, *Interrupt) {
	return false, nil
}

func (_ ValueFunction) Fields() (map[string]*Value, *Interrupt) {
	return make(map[string]*Value), nil
}

func (self ValueFunction) IntoIter() func() (Value, bool) {
	panic("A value of type function cannot be used as an iterator")
}

func NewValueFunction(module string, block ast.AnalyzedBlock) *Value {
	val := Value(ValueFunction{
		Module: module,
		Block:  block,
	})

	return &val
}
