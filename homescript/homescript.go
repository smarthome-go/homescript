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
) (Value, int, *hmsError.Error) {
	// Parse the source code
	parser := newParser(filename, program)
	ast, err := parser.parse()
	if err != nil {
		return nil, 1, err
	}
	// Create the interpreter
	interpreter := NewInterpreter(
		ast,
		executor,
		sigTerm,
		stackSize,
		scopeAdditions,
		debug,
	)
	// Finally, execute the AST
	value, exitCode, err := interpreter.run()
	return value, exitCode, err
}
