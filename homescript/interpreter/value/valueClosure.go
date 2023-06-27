package value

import "github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"

type ValueClosure struct {
	Scopes []map[string]*Value
	Block  ast.AnalyzedBlock
}

func (_ ValueClosure) Kind() ValueKind { return ClosureValueKind }

func (_ ValueClosure) Display() (string, *Interrupt) {
	return "<closure>", nil
}

func (_ ValueClosure) IsEqual(other Value) (bool, *Interrupt) {
	return false, nil
}

func (_ ValueClosure) Fields() (map[string]*Value, *Interrupt) {
	return make(map[string]*Value), nil
}

func (self ValueClosure) IntoIter() func() (Value, bool) {
	panic("A value of type function cannot be used as an iterator")
}

func NewValueClosure(block ast.AnalyzedBlock, scopes []map[string]*Value) *Value {
	val := Value(ValueClosure{
		Scopes: scopes,
		Block:  block,
	})

	return &val
}
