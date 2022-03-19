package homescript

import "github.com/MikMuellerDev/homescript-dev/homescript/interpreter"

type Interpreter struct {
	StartNode Expressions
	Executor  interpreter.Executor
	Scope     map[string]interpreter.Value
}

func NewInterpreter(startNode Expressions, executor interpreter.Executor) Interpreter {
	scope := map[string]interpreter.Value{}
	return Interpreter{
		StartNode: startNode,
		Executor:  executor,
		Scope:     scope,
	}
}
