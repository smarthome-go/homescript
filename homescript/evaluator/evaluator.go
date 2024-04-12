package evaluator

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

type RuntimeModule struct {
	functions    map[string]ast.AnalyzedFunctionDefinition
	scopes       []map[string]*value.Value
	currentScope uint // Indexes the field above.
	filename     string
}

type Interpreter struct {
	modules    map[string]RuntimeModule
	currModule *RuntimeModule
}

func NewInterpreter(program map[string]ast.AnalyzedProgram, entrypoint string) Interpreter {
	return Interpreter{}
}
