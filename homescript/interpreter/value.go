package interpreter

type ValueType uint8

const (
	Void ValueType = iota
	Number
	String
	Boolean
	Function
)

type Value interface {
	Type() ValueType
}

type ValueVoid struct{}

func (self ValueVoid) Type() ValueType { return Void }

type ValueNumber struct{ value int }

func (self ValueNumber) Type() ValueType { return Number }

type ValueString struct{ value string }

func (self ValueString) Type() ValueType { return String }

type ValueBoolean struct{ value bool }

func (self ValueBoolean) Type() ValueType { return Boolean }

type ValueFunction struct{ callback func(args ...Value) Value }

func (self ValueFunction) Type() ValueType { return Function }
