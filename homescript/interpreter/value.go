package interpreter

import "fmt"

type ValueType uint8

const (
	Void ValueType = iota
	Number
	String
	Boolean
	Function
	Variable
)

type Value interface {
	Type() ValueType
	ToString() string
}

type ValueVoid struct{}

func (self ValueVoid) Type() ValueType  { return Void }
func (self ValueVoid) ToString() string { return "void" }

type ValueNumber struct{ Value int }

func (self ValueNumber) Type() ValueType  { return Number }
func (self ValueNumber) ToString() string { return fmt.Sprint(self.Value) }

type ValueString struct{ Value string }

func (self ValueString) Type() ValueType  { return String }
func (self ValueString) ToString() string { return self.Value }

type ValueBoolean struct{ Value bool }

func (self ValueBoolean) Type() ValueType  { return Boolean }
func (self ValueBoolean) ToString() string { return fmt.Sprintf("%t", self.Value) }

type ValueFunction struct {
	Callback func(executor Executor, args ...Value) (Value, error)
}

func (self ValueFunction) Type() ValueType  { return Function }
func (self ValueFunction) ToString() string { return "<function>" }

type ValueVariable struct {
	Callback func(executor Executor) (Value, error)
}

func (self ValueVariable) Type() ValueType  { return Variable }
func (self ValueVariable) ToString() string { return "<variable>" }
