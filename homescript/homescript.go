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
	args map[string]Value,
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
	ast, errors, _ := parser.parse()
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
		args,
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
	symbols []symbol,
) {
	// Parse the source code
	parser := newParser(program)
	ast, errors, critical := parser.parse()
	if len(errors) > 0 {
		for _, err := range errors {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: Error,
				Kind:     err.Kind,
				Message:  err.Message,
				Span:     err.Span,
			})

		}
		// If there was a critical error, return only the syntax errors
		if critical {
			return diagnostics, nil
		}
	}
	// Create the analyzer
	analyzer := NewAnalyzer(
		ast,
		executor,
		scopeAdditions,
	)
	// Finally, analyze the AST
	semanticDiagnostics := analyzer.analyze()
	diagnostics = append(diagnostics, semanticDiagnostics...)
	return diagnostics, analyzer.symbols
}
