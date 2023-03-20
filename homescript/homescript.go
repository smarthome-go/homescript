package homescript

import (
	hmsError "github.com/smarthome-go/homescript/v2/homescript/errors"
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
	filename string,
) (
	returnValue Value,
	exitCode int,
	rootScope map[string]*Value,
	hmsErrors []hmsError.Error,
) {
	// Parse the source code
	parser := newParser(program, filename)
	ast, errors, _ := parser.parse()
	if len(errors) > 0 {
		return nil, 1, nil, errors
	}
	argsTemp := make(map[string]*Value)
	for key, val := range args {
		temp := val
		argsTemp[key] = &temp
	}
	// Create the interpreter
	interpreter := NewInterpreter(
		ast,
		executor,
		sigTerm,
		stackSize,
		scopeAdditions,
		argsTemp,
		debug,
		moduleStack,
		moduleName,
		filename,
	)
	// Finally, execute the AST
	returnValue, exitCode, runtimeError := interpreter.run()
	if runtimeError != nil {
		return nil, exitCode, interpreter.scopes[0], []hmsError.Error{*runtimeError}
	}
	return returnValue,
		exitCode,
		interpreter.scopes[0], // Return the root scope
		make([]hmsError.Error, 0) // Easier for the user
}

// Analyzes the given Homescript code
func Analyze(
	executor Executor,
	program string,
	scopeAdditions map[string]Value,
	moduleStack []string,
	moduleName string,
	filename string,
) (
	diagnostics []Diagnostic,
	symbols []symbol,
	rootScope map[string]*Value,
) {
	// Parse the source code
	parser := newParser(program, filename)
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
			return diagnostics, nil, nil
		}
	}
	// Append the current script to the module stack
	moduleStack = append(moduleStack, moduleName)
	// Create the analyzer
	analyzer := NewAnalyzer(
		ast,
		executor,
		scopeAdditions,
		moduleStack,
		filename,
	)
	// Finally, analyze the AST
	semanticDiagnostics, rootScope := analyzer.analyze()
	diagnostics = append(diagnostics, semanticDiagnostics...)
	return diagnostics, analyzer.symbols, rootScope
}
