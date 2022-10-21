package homescript

import (
	hmsError "github.com/smarthome-go/homescript/homescript/errors"
)

// Executes the given Homescript code
// The `sigTerm` variable is used to terminate the script at any point in time
// The value passed into the channel is used as an exit-code to terminate the script
func Run(
	executor Executor,
	sigTerm *chan int,
	filename string,
	program string,
	scopeAdditions map[string]Value,
	debug bool,
	stackSize uint,
	moduleStack []string,
	moduleName string,
) (
	returnValue Value,
	exitCode int,
	rootScope map[string]Value,
	hmsErrors []hmsError.Error,
) {
	// Parse the source code
	parser := newParser(filename, program)
	ast, errors := parser.parse()
	if len(errors) > 0 {
		return nil, 1, nil, errors
	}
	// Create the interpreter
	interpreter := NewInterpreter(
		ast,
		executor,
		sigTerm,
		stackSize,
		scopeAdditions,
		debug,
		moduleStack,
		moduleName,
	)
	// Finally, execute the AST
	returnValue, exitCode, runtimeError := interpreter.run()
	if runtimeError != nil {
		return nil, exitCode, interpreter.scopes[0], []hmsError.Error{*runtimeError}
	}
	return returnValue,
		exitCode,
		interpreter.scopes[0], // Return the root scope
		nil
}
