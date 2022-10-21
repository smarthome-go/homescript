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
	parser := newParser(program)
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

// Analyzes the given Homescript code
func Analyze(
	executor Executor,
	program string,
	scopeAdditions map[string]Value,
) (
	diagnostics []Diagnostic,
) {
	// Parse the source code
	parser := newParser(program)
	ast, errors := parser.parse()
	if len(errors) > 0 {
		for _, err := range errors {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: Error,
				Kind:     err.Kind,
				Message:  err.Message,
				Span:     err.Span,
			})

		}
		return diagnostics
	}
	// Create the analyzer
	analyzer := NewAnalyzer(
		ast,
		executor,
		scopeAdditions,
	)
	// Finally, analyze the AST
	return analyzer.analyze()
}
