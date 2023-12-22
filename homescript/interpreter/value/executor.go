package value

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

type Executor interface {
	// if it exists, returns a value which is part of the host builtin modules
	GetBuiltinImport(moduleName string, toImport string) (val Value, found bool)
	// returns the Homescript code of the requested module
	ResolveModuleCode(moduleName string) (code string, found bool, err error)
	// Writes the given string (produced by a print function for instance) to any arbitrary source
	WriteStringTo(input string) error
	// Returns the username of the user who is executing the current script
	GetUser() string
	// TODO: load singleton (singleton value, providesSingleton, err)
	LoadSingleton(ident string, typ ast.Type) (*Value, bool, *Interrupt)
}
