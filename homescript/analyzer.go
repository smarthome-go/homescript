package homescript

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

type Analyzer struct {
	program  []StatementOrExpr
	executor Executor

	// Scope stack: manages scopes (is searched in to (p -> down order)
	// The last element is the top whilst the first element is the bottom of the stack
	scopes []scope

	// Contains all the symbols contained in the file
	symbols []symbol

	// Holds the analyzer's diagnostics
	diagnostics []Diagnostic

	// Holds the modules visited so far (by import statements)
	// in order to prevent a circular import
	moduleStack []string

	// Will allow assignment to invalid object members
	// For exampls, `a.foo = 1;` will still be valid even though a has no member named `foo`
	isAssignLHSCount uint

	filename string
}

type symbol struct {
	Span       errors.Span
	Type       symbolType
	Value      Value
	InFunction bool
	InLoop     bool
}

type symbolType string

const (
	SymbolTypeUnknown         symbolType = "unknown"
	SymbolTypeNull            symbolType = "null"
	SymbolTypeNumber          symbolType = "number"
	SymbolTypeBoolean         symbolType = "boolean"
	SymbolTypeString          symbolType = "string"
	SymbolTypeList            symbolType = "list"
	SymbolTypePair            symbolType = "pair"
	SymbolTypeObject          symbolType = "object"
	SymbolTypeRange           symbolType = "range"
	SymbolDynamicMember       symbolType = "dynamic member"
	SymbolTypeFunction        symbolType = "function"
	SymbolTypeBuiltinFunction symbolType = "builtin function"
	SymbolTypeBuiltinVariable symbolType = "builtin variable"
	SymbolTypeEnum            symbolType = "enum"
	SymbolTypeEnumVariant     symbolType = "enum variant"
)

func (self ValueType) toSymbolType() symbolType {
	switch self {
	case TypeNull:
		return SymbolTypeNull
	case TypeNumber:
		return SymbolTypeNumber
	case TypeBoolean:
		return SymbolTypeBoolean
	case TypeString:
		return SymbolTypeString
	case TypeList:
		return SymbolTypeList
	case TypePair:
		return SymbolTypePair
	case TypeObject:
		return SymbolTypeObject
	case TypeRange:
		return SymbolTypeRange
	case TypeFunction:
		return SymbolTypeFunction
	case TypeBuiltinFunction:
		return SymbolTypeBuiltinFunction
	case TypeBuiltinVariable:
		return SymbolTypeBuiltinVariable
	case TypeEnum:
		return SymbolTypeEnum
	case TypeEnumVariant:
		return SymbolTypeEnumVariant
	default:
		// Unreachable
		panic("BUG: A new type was introduced without updating this code")
	}
}

func (self Diagnostic) Display(program string) string {
	lines := strings.Split(program, "\n")

	line1 := ""
	if self.Span.Start.Line > 1 {
		line1 = fmt.Sprintf("\n \x1b[90m%- 3d | \x1b[0m%s", self.Span.Start.Line-1, lines[self.Span.Start.Line-2])
	}
	line2 := fmt.Sprintf(" \x1b[90m%- 3d | \x1b[0m%s", self.Span.Start.Line, lines[self.Span.Start.Line-1])
	line3 := ""
	if int(self.Span.Start.Line) < len(lines) {
		line3 = fmt.Sprintf("\n \x1b[90m%- 3d | \x1b[0m%s", self.Span.Start.Line+1, lines[self.Span.Start.Line])
	}

	var color int
	switch self.Severity {
	case Info:
		color = 6
	case Warning:
		color = 3
	case Error:
		color = 1
	default:
		panic("New severity was introduced without updating this code")
	}

	var markers string
	switch self.Severity {
	case Info:
		markers = "~"
	case Warning:
		markers = "~"
	case Error:
		markers = "^"
	default:
		panic("New severity was introduced without updating this code")
	}
	if self.Span.Start.Line == self.Span.End.Line {
		// This is required because token spans are inclusive
		width := int(self.Span.End.Column-self.Span.Start.Column) + 1
		// If the span is just 1 character, use a more readable symbol
		if width == 1 {
			markers = "^"
		}
		// Repeat the markers for the span
		markers = strings.Repeat(markers, width)
	} else {
		// If the span is over multiple lines, use the more readable symbol
		markers = "^"
	}
	marker := fmt.Sprintf("%s\x1b[1;3%dm%s\x1b[0m", strings.Repeat(" ", int(self.Span.Start.Column+6)), color, markers)

	return fmt.Sprintf(
		"\x1b[1;3%dm%v\x1b[39m at %s:%d:%d\x1b[0m\n%s\n%s\n%s%s\n\n\x1b[1;3%dm%s\x1b[0m\n",
		color,
		self.Kind,
		self.Span.Filename,
		self.Span.Start.Line,
		self.Span.Start.Column,
		line1,
		line2,
		marker,
		line3,
		color,
		self.Message,
	)
}

type Diagnostic struct {
	Severity DiagnosticSeverity
	Kind     errors.ErrorKind
	Message  string
	Span     errors.Span
}

type DiagnosticSeverity uint8

const (
	Info DiagnosticSeverity = iota
	Warning
	Error
)

type scope struct {
	// Holds the actual scope
	this map[string]*Value
	// If this scope belongs to a function its identifier is put here
	identifier *string
	// Where the scope was pushed
	span errors.Span
	// Holds the args which were passed into this scope
	args []string
	// Saves which functions have been called in this scope
	// Used for preventing duplicate analysis of a function
	// Also serves to inform about unused functions
	functionCalls []string
	// Saves which variable names have been accessed
	// Used for issuing warnings about unused varirables
	variableAccesses []string
	// Saves which functions were imported
	// Because import is currently not implemented, the function should not be analyzed
	importedFunctions []string
	// If the analyzer is currently in a function
	inFunction bool
	// If the analyzer is currently in a loopp
	inLoop bool
}

func throwDummy(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if len(args) != 1 {
		return nil, nil, errors.NewError(span, fmt.Sprintf("function 'throw' requires exactly 1 argument but %d were given", len(args)), errors.TypeError)
	}
	for _, arg := range args {
		_, err := arg.Display(executor, span)
		if err != nil {
			return nil, nil, err
		}
	}
	return nil, nil, nil
}

func NewAnalyzer(
	program []StatementOrExpr,
	executor Executor,
	scopeAdditions map[string]Value, // Allows the user to add more entries to the scope
	moduleStack []string,
	filename string,
) Analyzer {
	scopes := make([]scope, 0)
	// Adds the root scope
	scopes = append(scopes, scope{
		this: map[string]*Value{
			// Builtin functions implemented by Homescript
			"exit":       valPtr(ValueBuiltinFunction{Callback: Exit}),
			"throw":      valPtr(ValueBuiltinFunction{Callback: throwDummy}),
			"assert":     valPtr(ValueBuiltinFunction{Callback: AnalyzerAssert}),
			"debug":      valPtr(ValueBuiltinFunction{Callback: Debug}),
			"print":      valPtr(ValueBuiltinFunction{Callback: Print}),
			"println":    valPtr(ValueBuiltinFunction{Callback: Print}),
			"switch":     valPtr(ValueBuiltinFunction{Callback: Switch}),
			"get_switch": valPtr(ValueBuiltinFunction{Callback: GetSwitch}),
			"notify":     valPtr(ValueBuiltinFunction{Callback: Notify}),
			"remind":     valPtr(ValueBuiltinFunction{Callback: Remind}),
			"log":        valPtr(ValueBuiltinFunction{Callback: Log}),
			"exec":       valPtr(ValueBuiltinFunction{Callback: Exec}),
			"get":        valPtr(ValueBuiltinFunction{Callback: Get}),
			"http":       valPtr(ValueBuiltinFunction{Callback: Http}),
			"ping":       valPtr(ValueBuiltinFunction{Callback: Ping}),
			"user":       valPtr(ValueBuiltinVariable{Callback: GetUser}),
			"weather":    valPtr(ValueBuiltinVariable{Callback: GetWeather}),
			"time":       valPtr(ValueBuiltinVariable{Callback: Time}),
			"fmt":        valPtr(ValueBuiltinFunction{Callback: Fmt}),
			"STORAGE":    valPtr(ValueBuiltinVariable{Callback: Storage}),
			"ARGS": valPtr(ValueObject{
				DataType:    "args",
				IsDynamic:   true,
				IsProtected: true,
				ObjFields:   make(map[string]*Value),
			}),
		},
		identifier: nil,
	},
	)
	// Add the optional scope entries
	for key, value := range scopeAdditions {
		if !value.Protected() {
			panic(fmt.Sprintf("Cannot insert scope addition with key `%s`: value is not marked as protected", key))
		}
		// Check if the isertion would be legal
		_, exists := scopes[0].this[key]
		if exists {
			panic(fmt.Sprintf("Cannot insert scope addition with key `%s`: this key is already taken by a builtin value", key))
		}
		// Insert the value into the scope
		var temp Value = value
		scopes[0].this[key] = &temp
	}

	return Analyzer{
		program:     program,
		executor:    executor,
		scopes:      scopes,
		moduleStack: moduleStack,
		filename:    filename,
	}
}

// Can be used to create a diagnostic message at the point where the current function was called
func (self *Analyzer) highlightCaller(kind errors.ErrorKind, errSpan errors.Span) {
	// Only continue if invoked inside a proper function
	if !self.getScope().inFunction || self.getScope().identifier == nil {
		return
	}
	// Backtracking to the point where the scope is no longer inside a function
	scopesCnt := len(self.scopes) - 1
	var prevIdent *string
	for idx := scopesCnt; idx >= 0; idx-- {
		curr := self.scopes[idx]
		// If the current scope is no longer in a function, the correct scope has been found
		// If the current scope is no longer in the same function, the correct scope has been found as well
		if prevIdent != nil && (curr.identifier != nil && *prevIdent != *curr.identifier) || curr.identifier == nil {
			self.info(
				self.scopes[idx+1].span,
				fmt.Sprintf("function with %v at line %d:%d called here", kind, errSpan.Start.Line, errSpan.Start.Column),
			)
			break
		}
		if curr.identifier != nil {
			id := *self.scopes[idx].identifier
			prevIdent = &id
		}
	}
}

func (self *Analyzer) diagnosticError(err errors.Error) {
	self.diagnostics = append(self.diagnostics, Diagnostic{
		Severity: Error,
		Kind:     err.Kind,
		Message:  err.Message,
		Span:     err.Span,
	})
}

func (self *Analyzer) info(span errors.Span, message string) {
	self.diagnostics = append(self.diagnostics, Diagnostic{
		Severity: Info,
		Kind:     errors.Info,
		Message:  message,
		Span:     span,
	})
}

func (self *Analyzer) warn(span errors.Span, message string) {
	self.diagnostics = append(self.diagnostics, Diagnostic{
		Severity: Warning,
		Kind:     errors.Warning,
		Message:  message,
		Span:     span,
	})
}

func (self *Analyzer) issue(span errors.Span, message string, kind errors.ErrorKind) {
	self.diagnostics = append(self.diagnostics, Diagnostic{
		Severity: Error,
		Kind:     kind,
		Message:  message,
		Span:     span,
	})
}

func (self *Analyzer) analyze() ([]Diagnostic, map[string]*Value) {
	_, err := self.visitStatements(self.program)
	if err != nil {
		self.diagnosticError(*err)
		return self.diagnostics, nil
	}
	lastScope := self.scopes[len(self.scopes)-1]
	// Pop the last scope from the scopes in order to analyze top-level functions
	if err := self.popScope(); err != nil {
		self.diagnosticError(*err)
	}
	return self.diagnostics, lastScope.this
}

// Interpreter code
func (self *Analyzer) visitStatements(items []StatementOrExpr) (Result, *errors.Error) {
	lastResult := makeNullResult(errors.Span{})

	unreachable := false
	for _, item := range items {
		// Check if statement is unreachable
		if unreachable {
			self.warn(item.Span(), "unreachable statement")
		}

		if item.IsStatement() {
			res, err := self.visitStatement(item.Statement)
			if err != nil {
				return Result{}, err
			}

			// Handle potential break or return statements
			if res.BreakValue != nil {
				// Check if the use of break is legal here
				if !self.getScope().inLoop {
					self.issue(item.Span(), "Can only use the break statement iside loops", errors.SyntaxError)
				} else {
					unreachable = true
				}
			}

			// Check if the use of continue is legal here
			if res.ShouldContinue {
				if !self.getScope().inLoop {
					self.issue(item.Span(), "Can only use the break statement iside loops", errors.SyntaxError)
				} else {
					unreachable = true
				}
			}

			// Handle potential break or return statements
			if res.ReturnValue != nil {
				// Check if the use of return is legal here
				if !self.getScope().inFunction {
					self.issue(item.Span(), "Can only use the return statement iside function bodies", errors.SyntaxError)
				} else {
					unreachable = true
					lastResult = res
				}
			}
		} else {
			result, err := self.visitExpression(*item.Expression)
			if err != nil {
				return Result{}, err
			}
			if !unreachable {
				lastResult = result
			}
		}
	}
	if lastResult.BreakValue != nil {
		return Result{Value: lastResult.BreakValue}, nil
	} else if lastResult.ReturnValue != nil {
		return Result{Value: lastResult.ReturnValue}, nil
	}
	return lastResult, nil
}

func (self *Analyzer) visitStatement(node Statement) (Result, *errors.Error) {
	// Handle different statement kind
	switch node.Kind() {
	case LetStmtKind:
		return self.visitLetStatement(node.(LetStmt))
	case ImportStmtKind:
		return self.visitImportStatement(node.(ImportStmt))
	case BreakStmtKind:
		return self.visitBreakStatement(node.(BreakStmt))
	case ContinueStmtKind:
		return self.visitContinueStatement(node.(ContinueStmt))
	case ReturnStmtKind:
		return self.visitReturnStatement(node.(ReturnStmt))
	case ExpressionStmtKind:
		// Because visitExpression returns a value instead of a result, it must be transformed here
		value, err := self.visitExpression(node.(ExpressionStmt).Expression)
		if err != nil {
			return Result{}, err
		}
		return value, nil
	default:
		panic("BUG: a new statement kind was introduced without updating this code")
	}
}

func (self *Analyzer) visitLetStatement(node LetStmt) (Result, *errors.Error) {
	// Evaluate the right hand side
	rightResult, err := self.visitExpression(node.Right)
	if err != nil {
		return Result{}, err
	}
	if rightResult.Value == nil || *rightResult.Value == nil {
		// If the right hand side evaluates to nil, still add a placeholder to the scope
		self.addVar(node.Left.Identifier, nil, node.Left.Span)
		return Result{}, nil
	}
	// Insert a span into the value
	right := setValueSpan(*rightResult.Value, node.Left.Span)
	// Add the value to the scope
	self.addVar(node.Left.Identifier, right, node.Left.Span)
	// Finially, return the result
	return rightResult, nil
}

func (self *Analyzer) visitImportStatement(node ImportStmt) (Result, *errors.Error) {
	// Prevent possible circular import
	for _, module := range self.moduleStack {
		if module == node.FromModule {
			// Would import a script which is located (upstream) in the moduleStack
			// Stack is unwided and displayed in order to show the problem to the user
			visual := "=== Import Stack ===\n"
			for idx, visited := range self.moduleStack {
				if idx == 0 {
					visual += fmt.Sprintf("             %2d: %-10s (ORIGIN)\n", 1, self.moduleStack[0])
				} else {
					visual += fmt.Sprintf("  imports -> %2d: %-10s\n", idx+1, visited)
				}
			}
			visual += fmt.Sprintf("  imports -> %2d: %-10s (ILLEGAL)\n", len(self.moduleStack)+1, node.FromModule)
			self.issue(
				node.Range,
				fmt.Sprintf("illegal import: circular import detected:\n%s", visual),
				errors.ImportError,
			)
			return Result{}, nil
		}
	}

	// Check if the function can be imported
	moduleCode, filename, exists, shouldProceed, scopeAdditions, err := self.executor.ResolveModule(node.FromModule)
	if err != nil {
		self.issue(
			node.Range,
			fmt.Sprintf("resolve module: %s", err.Error()),
			errors.ImportError,
		)
		return Result{}, nil
	}
	if !exists {
		self.issue(
			node.Range,
			"resolve module: module not found",
			errors.ImportError,
		)
		return Result{}, nil
	}

	if !shouldProceed {
		for _, imported := range node.Functions {
			function := makeFn(&imported, node.Range)

			// Push a dummy function into the current scope
			self.addVar(imported, function, node.Range)
			// Add the function to the list of imported functions to avoid analysis
			self.getScope().importedFunctions = append(self.getScope().importedFunctions, imported)
		}
		return Result{}, nil
	}

	for _, imported := range node.Functions {
		value := self.getVar(imported)
		function := makeFn(&imported, node.Range)

		// Only report this non-critical error
		if value != nil {
			self.issue(node.Range,
				fmt.Sprintf("the name '%s' is already present in the current scope", imported),
				errors.ImportError,
			)
			return Result{}, nil
		} else {
			// Push a dummy function into the current scope
			self.addVar(imported, function, node.Range)
			// Add the function to the list of imported functions to avoid analysis
			self.getScope().importedFunctions = append(self.getScope().importedFunctions, imported)
		}

		diagnostics, _, rootScope := Analyze(
			self.executor,
			moduleCode,
			scopeAdditions,
			self.moduleStack,
			node.FromModule,
			filename,
		)
		moduleErrors := 0
		firstErrMessage := ""
		for _, diagnostic := range diagnostics {
			if diagnostic.Severity == Error {
				moduleErrors++
				if firstErrMessage == "" {
					firstErrMessage = diagnostic.Message
				}
			}
		}
		if moduleErrors > 0 {
			self.issue(
				node.Range,
				fmt.Sprintf("target module contains %d error(s): %s", moduleErrors, firstErrMessage),
				errors.ImportError,
			)
			return Result{}, nil
		}
		_, found := rootScope[imported]
		if !found {
			self.issue(
				node.Range,
				fmt.Sprintf("no function named `%s` found in module `%s`", imported, node.FromModule),
				errors.ImportError,
			)
			return Result{}, nil
		}
	}

	return Result{}, nil
}

func (self *Analyzer) visitBreakStatement(node BreakStmt) (Result, *errors.Error) {
	// If the break should have a value, make and override it here
	if node.Expression != nil {
		value, err := self.visitExpression(*node.Expression)
		if err != nil {
			return Result{}, err
		}
		return Result{BreakValue: value.Value}, nil
	}
	// The break value defaults to null
	null := makeNull(node.Span())
	return Result{BreakValue: &null}, nil
}

func (self *Analyzer) visitContinueStatement(node ContinueStmt) (Result, *errors.Error) {
	return Result{
		ShouldContinue: true,
		ReturnValue:    nil,
		BreakValue:     nil,
		Value:          nil,
	}, nil
}

func (self *Analyzer) visitReturnStatement(node ReturnStmt) (Result, *errors.Error) {
	// The return value defaults to null
	returnValue := makeNull(node.Span())
	// If the return statement should return a value, make and override it here
	if node.Expression != nil {
		value, err := self.visitExpression(*node.Expression)
		if err != nil {
			return Result{}, err
		}
		if value.Value == nil || *value.Value == nil {
			return Result{
				ShouldContinue: false,
				ReturnValue:    &returnValue,
				BreakValue:     nil,
				Value:          nil,
			}, nil
		}
		returnValue = *value.Value
	}
	return Result{
		ShouldContinue: false,
		ReturnValue:    &returnValue,
		BreakValue:     nil,
		Value:          nil,
	}, nil
}

// Expressions
func (self *Analyzer) visitExpression(node Expression) (Result, *errors.Error) {
	base, err := self.visitAndExpression(node.Base)
	if err != nil {
		return Result{}, err
	}
	// If there are no other expressions, just return the base value
	if len(node.Following) == 0 {
		return base, nil
	}

	// Only continue analysis if the base is not nil
	if base.Value != nil {
		_, err = (*base.Value).IsTrue(self.executor, node.Base.Span)
		if err != nil {
			self.diagnosticError(*err)
			return makeBoolResult(node.Span, false), nil
		}
	}

	// Look at the other expressions
	for _, following := range node.Following {
		followingValue, err := self.visitAndExpression(following)
		if err != nil {
			return Result{}, err
		}

		// Only continue analysis if the current value is not nil
		if followingValue.Value != nil {
			_, err = (*followingValue.Value).IsTrue(self.executor, following.Span)
			if err != nil {
				self.diagnosticError(*err)
				return makeBoolResult(node.Span, false), nil
			}
		}
	}

	return makeBoolResult(node.Span, false), nil
}

func (self *Analyzer) visitAndExpression(node AndExpression) (Result, *errors.Error) {
	base, err := self.visitEqExpression(node.Base)
	if err != nil {
		return Result{}, err
	}
	// If there are no other expressions, just return the base value
	if len(node.Following) == 0 {
		return base, nil
	}

	// Only continue analysis if the base value is not nil
	if base.Value != nil && *base.Value != nil {
		_, err = (*base.Value).IsTrue(self.executor, node.Base.Span)
		if err != nil {
			self.diagnosticError(*err)
			return makeBoolResult(node.Span, false), nil
		}
	}

	// Look at the other expressions
	for _, following := range node.Following {
		followingValue, err := self.visitEqExpression(following)
		if err != nil {
			return Result{}, err
		}

		// Only continue analysis if the current value is not nil
		if followingValue.Value != nil && *followingValue.Value != nil {
			_, err = (*followingValue.Value).IsTrue(self.executor, following.Span)
			if err != nil {
				self.diagnosticError(*err)
				return makeBoolResult(node.Span, false), nil
			}
		}
	}
	return makeBoolResult(node.Span, false), nil
}

func (self *Analyzer) visitEqExpression(node EqExpression) (Result, *errors.Error) {
	base, err := self.visitRelExpression(node.Base)
	if err != nil {
		return Result{}, err
	}

	// If there is nothing to compare to, return the base value
	if node.Other == nil {
		return base, nil
	}

	otherValue, err := self.visitRelExpression(node.Other.Node)
	if err != nil {
		return Result{}, err
	}

	// Prevent further analysis if either the base or the other values are nil
	if base.Value == nil || *base.Value == nil || otherValue.Value == nil || *otherValue.Value == nil {
		self.info(node.Span, "manual validation required")
		return makeBoolResult(node.Span, false), nil
	}

	// Finally, test for equality
	_, err = (*base.Value).IsEqual(self.executor, node.Span, *otherValue.Value)
	if err != nil {
		return makeBoolResult(node.Span, false), err
	}
	return Result{}, nil
}

func (self *Analyzer) visitRelExpression(node RelExpression) (Result, *errors.Error) {
	base, err := self.visitAddExpression(node.Base)
	if err != nil {
		return Result{}, err
	}
	// If there is nothing to compare to, return the base value
	if node.Other == nil {
		return base, nil
	}

	otherValue, err := self.visitAddExpression(node.Other.Node)
	if err != nil {
		return Result{}, err
	}

	// Prevent further analysis if either the base or the other values are nil
	if otherValue.Value == nil || *otherValue.Value == nil {
		self.info(node.Other.Node.Span, "manual type validation required")
		return makeBoolResult(node.Span, false), nil
	}

	if base.Value == nil || *base.Value == nil {
		self.info(node.Base.Span, "manual type validation required")
		return makeBoolResult(node.Span, false), nil
	}

	// Check that the comparison involves a valid left hand side
	var baseVal Value
	switch (*base.Value).Type() {
	case TypeNumber:
		baseVal = (*base.Value).(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = (*base.Value).(ValueBuiltinVariable)
	default:
		self.issue(node.Span, fmt.Sprintf("Cannot compare %v to %v", (*base.Value).Type(), (*otherValue.Value).Type()), errors.TypeError)
		return makeBoolResult(node.Span, false), nil
	}

	// Perform typecast so that comparison operators can be used
	baseComp := baseVal.(ValueRelational)

	var relError *errors.Error

	// Finally, compare the two values
	switch node.Other.RelOperator {
	case RelLessThan:
		_, relError = baseComp.IsLessThan(self.executor, node.Span, *otherValue.Value)
	case RelLessOrEqual:
		_, relError = baseComp.IsLessThanOrEqual(self.executor, node.Span, *otherValue.Value)
	case RelGreaterThan:
		_, relError = baseComp.IsGreaterThan(self.executor, node.Span, *otherValue.Value)
	case RelGreaterOrEqual:
		_, relError = baseComp.IsGreaterThanOrEqual(self.executor, node.Span, *otherValue.Value)
	default:
		panic("BUG: a new rel operator was introduced without updating this code")
	}
	if relError != nil {
		return makeBoolResult(node.Span, false), nil
	}
	return makeBoolResult(node.Span, false), nil
}

func (self *Analyzer) visitAddExpression(node AddExpression) (Result, *errors.Error) {
	base, err := self.visitMulExpression(node.Base)
	if err != nil {
		return Result{}, err
	}

	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil
	}

	// Prevent further analysis if the base value is nil
	if base.Value == nil || *base.Value == nil {
		hadError := false
		// Still lint the other parts
		for _, following := range node.Following {
			followingValue, err := self.visitMulExpression(following.Other)
			if err != nil {
				return Result{}, err
			}
			if followingValue.Value != nil && *followingValue.Value != nil && (*followingValue.Value).Type() != TypeNumber {
				self.issue(following.Span, fmt.Sprintf("cannot apply operation on type %v", (*followingValue.Value).Type()), errors.TypeError)
				hadError = true
			}
		}
		if !hadError {
			self.info(node.Base.Span, "manual type validation required")
		}
		return Result{}, nil
	}

	var baseVal Value
	// Check that the base holds a valid type to perform the requested operations
	switch (*base.Value).Type() {
	case TypeNumber:
		baseVal = (*base.Value).(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = (*base.Value).(ValueBuiltinVariable)
	case TypeString:
		baseVal = (*base.Value).(ValueString)
	case TypeBoolean:
		baseVal = (*base.Value).(ValueBool)
	default:
		self.highlightCaller(errors.TypeError, node.Span)
		self.issue(node.Span, fmt.Sprintf("cannot apply operation on type %v", (*base.Value).Type()), errors.TypeError)

		// Still lint the other parts
		for _, following := range node.Following {
			followingValue, err := self.visitMulExpression(following.Other)
			if err != nil {
				return Result{}, err
			}
			if followingValue.Value != nil && *followingValue.Value != nil && (*followingValue.Value).Type() != TypeNumber {
				self.issue(following.Span, fmt.Sprintf("cannot apply operation on type %v", (*followingValue.Value).Type()), errors.TypeError)
			}
		}
		return Result{}, nil
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(ValueAlg)

	manualInfoIssued := false

	for _, following := range node.Following {
		// Is later filled and evaluated once the correct operator has been applied
		var algError *errors.Error
		var res Value

		followingValue, err := self.visitMulExpression(following.Other)
		if err != nil {
			return Result{}, err
		}

		// Terminate this function's analysis if the followingValue is nil
		if followingValue.Value == nil || *followingValue.Value == nil {
			if !manualInfoIssued {
				self.info(following.Span, "manual type validation required")
				manualInfoIssued = true
			}
			continue
		}

		switch following.AddOperator {
		case AddOpPlus:
			res, algError = baseAlg.Add(self.executor, node.Span, *followingValue.Value)
		case AddOpMinus:
			res, algError = baseAlg.Sub(self.executor, node.Span, *followingValue.Value)
		default:
			panic("BUG: a new add operator has been added without updating this code")
		}

		if algError != nil {
			self.highlightCaller(algError.Kind, algError.Span)
			algError.Span = following.Span
			self.diagnosticError(*algError)
			// Must return a blank result in order to prevent other functions from using a nil value
			return Result{}, nil
		}
		baseAlg = res.(ValueAlg)
	}

	returnValue := baseAlg.(Value)
	return Result{Value: &returnValue}, nil
}

func (self *Analyzer) visitMulExpression(node MulExpression) (Result, *errors.Error) {
	base, err := self.visitCastExpression(node.Base)
	if err != nil {
		return Result{}, err
	}

	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil
	}

	// Check that the base holds a valid type to perform the requested operations
	var baseVal Value

	hideValidationHint := false

	if base.Value != nil && *base.Value != nil && (*base.Value).Type() == TypeNumber {
		baseVal = (*base.Value).(ValueNumber)
	} else {
		if base.Value != nil && *base.Value != nil {
			self.highlightCaller(errors.TypeError, node.Span)
			self.issue(node.Base.Span, fmt.Sprintf("cannot apply operation on type %v", (*base.Value).Type()), errors.TypeError)
			hideValidationHint = true
		}
		// Still lint the other parts
		for _, following := range node.Following {
			followingValue, err := self.visitCastExpression(following.Other)
			if err != nil {
				return Result{}, err
			}
			if followingValue.Value != nil && *followingValue.Value != nil && (*followingValue.Value).Type() != TypeNumber {
				self.issue(following.Span, fmt.Sprintf("cannot apply operation on type %v", (*followingValue.Value).Type()), errors.TypeError)
				hideValidationHint = true
			}
		}
		if !hideValidationHint && (base.Value == nil || *base.Value == nil) {
			self.info(node.Base.Span, "manual type validation required")
		}
		return Result{}, nil
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(ValueAlg)

	for _, following := range node.Following {
		// Is later filled and evaluated once the correct operator has been applied
		var algError *errors.Error

		followingValue, err := self.visitCastExpression(following.Other)
		if err != nil {
			return Result{}, err
		}

		// Terminate this function's analysis if the followingValue is nil
		if baseAlg == nil || followingValue.Value == nil || *followingValue.Value == nil {
			if !hideValidationHint {
				self.info(node.Span, "manual type validation required")
				hideValidationHint = true
			}
			continue
		}

		switch following.MulOperator {
		case MulOpMul:
			_, algError = baseAlg.Mul(self.executor, node.Span, *followingValue.Value)
		case MulOpDiv:
			_, algError = baseAlg.Div(self.executor, node.Span, *followingValue.Value)
		case MulOpIntDiv:
			_, algError = baseAlg.IntDiv(self.executor, node.Span, *followingValue.Value)
		case MulOpReminder:
			_, algError = baseAlg.Rem(self.executor, node.Span, *followingValue.Value)
		default:
			panic("BUG: a new mul operator has been added without updating this code")
		}
		if algError != nil {
			self.highlightCaller(errors.TypeError, (*followingValue.Value).Span())
			self.diagnosticError(*algError)
			hideValidationHint = true
		}
	}
	return Result{Value: base.Value}, nil
}

func (self *Analyzer) visitCastExpression(node CastExpression) (Result, *errors.Error) {
	base, err := self.visitUnaryExpression(node.Base)
	if err != nil {
		return Result{}, err
	}
	// If there is not typecast, only return the base value
	if node.Other == nil {
		return base, nil
	}

	// Stop analysis here if the base value is nil
	// Will still return the expected value type
	if base.Value == nil || *base.Value == nil {
		switch *node.Other {
		case TypeNumber:
			num := makeNum(node.Span, 0)
			return Result{Value: &num}, nil
		case TypeBoolean:
			return makeBoolResult(node.Span, false), nil
		case TypeString:
			str := makeStr(node.Span, "")
			return Result{Value: &str}, nil
		default:
			self.issue(node.Span, fmt.Sprintf("cannot cast to non-primitive type: cast from %v to %v is unsupported", (*base.Value).Type(), *node.Other), errors.TypeError)
			return Result{}, nil
		}
	}

	switch *node.Other {
	case TypeNumber:
		switch (*base.Value).Type() {
		case TypeNumber:
			return base, nil
		case TypeBoolean:
			numericValue := makeNum(node.Span, 0)
			return Result{Value: &numericValue}, nil
		case TypeString:
			_, err := strconv.ParseFloat((*base.Value).(ValueString).Value, 64)
			if err != nil {
				self.issue(node.Base.Span, "cast to number used on non-numeric string", errors.ValueError)
				return Result{}, nil
			}
			num := makeNum(node.Span, 0)
			return Result{Value: &num}, nil
		default:
			self.issue(node.Span, fmt.Sprintf("cannot cast %v to %v", (*base.Value).Type(), *node.Other), errors.TypeError)
			return Result{}, nil
		}
	case TypeString:
		_, err := (*base.Value).Display(self.executor, node.Base.Span)
		if err != nil {
			return Result{}, err
		}
		valueStr := makeStr(node.Span, "")
		return Result{Value: &valueStr}, nil
	case TypeBoolean:
		_, err := (*base.Value).IsTrue(self.executor, node.Base.Span)
		if err != nil {
			return Result{}, err
		}
		truthValue := makeBool(node.Span, false)
		return Result{Value: &truthValue}, nil
	default:
		self.issue(node.Span, fmt.Sprintf("cannot cast to non-primitive type: cast from %v to %v is unsupported", (*base.Value).Type(), *node.Other), errors.TypeError)
		return Result{}, nil
	}
}
func (self *Analyzer) visitUnaryExpression(node UnaryExpression) (Result, *errors.Error) {
	// If there is only a exp expression, return its value (recursion base case)
	if node.ExpExpression != nil {
		return self.visitEpxExpression(*node.ExpExpression)
	}
	unaryBase, err := self.visitUnaryExpression(node.UnaryExpression.UnaryExpression)
	if err != nil {
		return Result{}, err
	}

	// Stop here if the base value is nil
	if unaryBase.Value == nil || *unaryBase.Value == nil {
		return Result{}, nil
	}

	var unaryErr *errors.Error
	switch node.UnaryExpression.UnaryOp {
	case UnaryOpPlus:
		_, unaryErr = ValueNumber{Value: 0.0}.Add(self.executor, node.UnaryExpression.UnaryExpression.Span, *unaryBase.Value)
	case UnaryOpMinus:
		_, unaryErr = ValueNumber{Value: 0.0}.Sub(self.executor, node.UnaryExpression.UnaryExpression.Span, *unaryBase.Value)
	case UnaryOpNot:
		_, err := (*unaryBase.Value).IsTrue(self.executor, node.UnaryExpression.UnaryExpression.Span)
		if err != nil {
			self.diagnosticError(*err)
			return Result{}, nil
		}
		returnValue := makeBool(node.Span, false)
		return Result{Value: &returnValue}, nil
	default:
		panic("BUG: a new unary operator has been added without updating this code")
	}
	if unaryErr != nil {
		self.diagnosticError(*unaryErr)
		return Result{}, nil
	}
	return Result{Value: unaryBase.Value}, nil
}

func (self *Analyzer) visitEpxExpression(node ExpExpression) (Result, *errors.Error) {
	base, err := self.visitAssignExpression(node.Base)
	if err != nil {
		return Result{}, err
	}
	// If there is no exponent, just return the base case
	if node.Other == nil {
		return base, nil
	}
	power, err := self.visitUnaryExpression(*node.Other)
	if err != nil {
		return Result{}, err
	}

	// If the base value is nil, stop this analysis
	if base.Value == nil {
		if power.Value != nil && (*power.Value).Type() == TypeNumber {
			num := makeNum(errors.Span{}, 0)
			return Result{Value: &num}, nil
		}
		return Result{}, nil
	}

	// If the power value is nil, stop the analysis
	if power.Value == nil {
		return Result{}, nil
	}

	// Calculate result based on the base type
	var powErr *errors.Error
	switch (*base.Value).Type() {
	case TypeNumber:
		_, powErr = (*base.Value).(ValueNumber).Pow(self.executor, node.Span, *power.Value)
	case TypeBuiltinVariable:
		_, powErr = (*base.Value).(ValueBuiltinVariable).Pow(self.executor, node.Span, *power.Value)
	default:
		self.issue(node.Span, fmt.Sprintf("cannot perform power operation on type %v", (*base.Value).Type()), errors.TypeError)
		return Result{}, nil
	}
	if powErr != nil {
		self.diagnosticError(*powErr)
		return Result{}, nil
	}
	return Result{Value: base.Value}, nil
}

func (self *Analyzer) visitAssignExpression(node AssignExpression) (Result, *errors.Error) {
	if node.Other != nil {
		self.isAssignLHSCount++
	}
	base, err := self.visitCallExpression(node.Base)
	if err != nil {
		return Result{}, err
	}
	if node.Other != nil {
		self.isAssignLHSCount--
	}
	// If there is no assignment, return the base value here
	if node.Other == nil {
		return base, nil
	}
	rhsValue, err := self.visitExpression(node.Other.Expression)
	if err != nil {
		return Result{}, err
	}

	if base.Value == nil || *base.Value == nil {
		if rhsValue.Value != nil && *rhsValue.Value != nil {
			return Result{Value: rhsValue.Value}, nil
		}
		return Result{}, nil
	}

	if rhsValue.Value == nil || *rhsValue.Value == nil {
		return Result{}, nil
	}

	if node.Other.Operator == OpAssign {
		// Insert an identifier
		right := setValueSpan(*rhsValue.Value, node.Base.Span)
		value, err := assign(base.Value, right, node.Span)
		if err != nil {
			self.diagnosticError(*err)
			return Result{}, nil
		}
		return Result{Value: &value}, nil
	}

	// Check that the base is a type that can be safely assigned to using the complex operators
	if (*base.Value).Type() != TypeString && (*base.Value).Type() != TypeNumber {
		self.issue(node.Base.Span, fmt.Sprintf("cannot use algebraic assignment operators on type %v", (*base.Value).Type()), errors.TypeError)
		return Result{}, nil
	}

	// Perform the more complex assignments
	var newValue Value
	var assignErr *errors.Error
	switch node.Other.Operator {
	case OpAssign:
		panic("BUG: this case should have been handled above")
	case OpPlusAssign:
		newValue, assignErr = (*base.Value).(ValueAlg).Add(self.executor, node.Span, *rhsValue.Value)
	case OpMinusAssign:
		newValue, assignErr = (*base.Value).(ValueAlg).Sub(self.executor, node.Span, *rhsValue.Value)
	case OpMulAssign:
		newValue, assignErr = (*base.Value).(ValueAlg).Mul(self.executor, node.Span, *rhsValue.Value)
	case OpDivAssign:
		newValue, assignErr = (*base.Value).(ValueAlg).Div(self.executor, node.Span, *rhsValue.Value)
	case OpIntDivAssign:
		newValue, assignErr = (*base.Value).(ValueAlg).IntDiv(self.executor, node.Span, *rhsValue.Value)
	case OpReminderAssign:
		newValue, assignErr = (*base.Value).(ValueAlg).Rem(self.executor, node.Span, *rhsValue.Value)
	case OpPowerAssign:
		newValue, assignErr = (*base.Value).(ValueAlg).Pow(self.executor, node.Span, *rhsValue.Value)
	}
	if assignErr != nil {
		return Result{}, assignErr
	}
	right := setValueSpan(newValue, node.Base.Span)
	value, err := assign(base.Value, right, node.Span)
	if err != nil {
		self.diagnosticError(*err)
		return Result{}, nil
	}
	return Result{Value: &value}, nil
}

func (self *Analyzer) visitCallExpression(node CallExpression) (Result, *errors.Error) {
	base, err := self.visitMemberExpression(node.Base)
	if err != nil {
		return Result{}, err
	}
	// Evaluate call / member parts
	for _, part := range node.Parts {
		if base.Value == nil || *base.Value == nil {
			self.info(part.Span, "manual call validation required")
			return Result{}, nil
		}
		// Handle args -> function call
		if part.Args != nil {
			// Call the base using the following args
			result, err := self.callValue(part.Span, *base.Value, *part.Args)
			if err != nil {
				self.diagnosticError(*err)
				return Result{}, nil
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = &result
			//return Result{}, nil
		}
		// Handle member access
		if part.MemberExpressionPart != nil {
			if base.Value == nil || *base.Value == nil {
				self.info(part.Span, "manual validation required")
				return Result{}, nil
			}
			result, err := self.getField(*base.Value, part.Span, *part.MemberExpressionPart, self.isAssignLHSCount != 0)
			if err != nil {
				self.issue(err.Span, err.Message, err.Kind)
				return Result{}, nil
			}
			if result == nil {
				return Result{}, nil
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = result
		}
		// Handle index access
		if part.Index != nil {
			// Make the index expression
			indexValue, err := self.visitExpression(*part.Index)
			if err != nil {
				return Result{}, err
			}
			if base.Value == nil || *base.Value == nil || indexValue.Value == nil || *indexValue.Value == nil {
				self.info(part.Span, "manual index validation required")
				return Result{}, nil
			}
			result, fieldExists, err := (*base.Value).Index(self.executor, *indexValue.Value, part.Span)
			if err != nil {
				self.diagnosticError(*err)
				return Result{}, nil
			}
			// If there was no error but the field does not exist, only create an info
			if !fieldExists {
				self.info(part.Span, "dynamic object: manual validation required")
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = result
		}
	}
	// Return the last base (the result)
	return base, nil
}

func (self *Analyzer) visitMemberExpression(node MemberExpression) (Result, *errors.Error) {
	base, err := self.visitAtom(node.Base)
	if err != nil {
		return Result{}, err
	}

	// Evaluate member expressions
	manualInfo := false
	for _, member := range node.Members {
		// Handle member (field) access
		if member.Identifier != nil {
			if base.Value == nil || *base.Value == nil {
				manualInfo = true
				continue
			}
			value, err := self.getField(*base.Value, member.Span, *member.Identifier, self.isAssignLHSCount != 0)
			if err != nil {
				self.issue(err.Span, err.Message, err.Kind)
				return Result{}, nil
			}
			if value == nil {
				return Result{}, nil
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = value
		}
		if member.Index != nil {
			indexValue, err := self.visitExpression(*member.Index)
			if err != nil {
				return Result{}, err
			}
			if indexValue.Value == nil || *indexValue.Value == nil || base.Value == nil || *base.Value == nil {
				manualInfo = true
				continue
			}
			result, fieldExists, err := (*base.Value).Index(self.executor, *indexValue.Value, member.Span)
			if err != nil {
				self.diagnosticError(*err)
				return Result{}, nil
			}
			// If there was no error but the field does not exist, only create an info
			if !fieldExists {
				self.info(member.Span, "dynamic object: manual validation required")
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = result
		}
	}
	if manualInfo {
		self.info(errors.Span{
			Start:    node.Members[0].Span.Start,
			End:      node.Members[len(node.Members)-1].Span.End,
			Filename: self.filename,
		}, "manual validation required")
	}
	return base, nil
}

func (self *Analyzer) getField(value Value, span errors.Span, key string, isAssignLHS bool) (*Value, *errors.Error) {
	fields, err := value.Fields(self.executor, span)
	if err != nil {
		return nil, err
	}

	val, exists := fields[key]
	if !exists {
		if isAssignLHS {
			ptr := valPtr(ValueNull{})
			fields[key] = ptr
			return ptr, nil
		}
		if value.Type() == TypeObject && value.(ValueObject).IsDynamic {
			self.symbols = append(self.symbols, symbol{
				Span:  span,
				Value: nil,
				Type:  SymbolDynamicMember,
			})
			self.info(span, "dynamic object: manual member validation required")
			return nil, nil
		}
		self.issue(span, fmt.Sprintf("%v has no member named %s", value.Type(), key), errors.TypeError)
		return nil, nil
	}
	self.symbols = append(self.symbols, symbol{
		Span:  span,
		Value: *val,
		Type:  (*val).Type().toSymbolType(),
	})
	return val, nil
}

func (self *Analyzer) visitAtom(node Atom) (Result, *errors.Error) {
	null := makeNull(node.Span())
	result := Result{Value: &null}
	switch node.Kind() {
	case AtomKindNumber:
		num := makeNum(node.Span(), node.(AtomNumber).Num)

		self.symbols = append(self.symbols, symbol{
			Value: num,
			Span:  node.Span(),
			Type:  SymbolTypeNumber,
		})

		result = Result{Value: &num}

	case AtomKindBoolean:
		bool := makeBool(node.Span(), node.(AtomBoolean).Value)

		self.symbols = append(self.symbols, symbol{
			Value: bool,
			Span:  node.Span(),
			Type:  SymbolTypeBoolean,
		})

		result = Result{Value: &bool}
	case AtomKindString:
		str := makeStr(node.Span(), node.(AtomString).Content)

		self.symbols = append(self.symbols, symbol{
			Value: str,
			Span:  node.Span(),
			Type:  SymbolTypeString,
		})

		result = Result{Value: &str}
	case AtomKindListLiteral:
		self.symbols = append(self.symbols, symbol{
			Value: nil,
			Span:  node.Span(),
			Type:  SymbolTypeList,
		})

		return self.makeList(node.(AtomListLiteral))
	case AtomKindObject:
		self.symbols = append(self.symbols, symbol{
			Value: nil,
			Span:  node.Span(),
			Type:  SymbolTypeObject,
		})

		return self.makeObject(node.(AtomObject))
	case AtomKindRange:
		rangeNode := node.(AtomRange)
		current := float64(rangeNode.Start)

		val := valPtr(ValueRange{
			Start:   valPtr(ValueNumber{Value: float64(rangeNode.Start)}),
			End:     valPtr(ValueNumber{Value: float64(rangeNode.End)}),
			Current: &current,
			Range:   node.Span(),
		})

		self.symbols = append(self.symbols, symbol{
			Value: *val,
			Span:  node.Span(),
			Type:  SymbolTypeRange,
		})

		return Result{Value: val}, nil
	case AtomKindPair:
		pairNode := node.(AtomPair)
		// Make the pair's value
		pairValue, err := self.visitExpression(pairNode.ValueExpr)
		if err != nil {
			return Result{}, err
		}
		if pairValue.Value == nil || *pairValue.Value == nil {
			return Result{}, nil
		}
		pair := makePair(node.Span(), ValueString{Value: pairNode.Key}, *pairValue.Value)

		self.symbols = append(self.symbols, symbol{
			Value: pair,
			Span:  node.Span(),
			Type:  SymbolTypePair,
		})

		result = Result{Value: &pair}
	case AtomKindNull:
		null := makeNull(node.Span())

		self.symbols = append(self.symbols, symbol{
			Value: null,
			Span:  node.Span(),
			Type:  SymbolTypeNull,
		})

		result = Result{Value: &null}
	case AtomKindIdentifier:
		// Search the scope for the correct key
		key := node.(AtomIdentifier).Identifier
		scopeValue := self.accessVar(key)

		// If the key is associated with a value, return it
		if scopeValue != nil {
			// This resolves the `true` value of builtin variables directly here
			if (*scopeValue) != nil && (*scopeValue).Type() == TypeBuiltinVariable {
				value, err := (*scopeValue).(ValueBuiltinVariable).Callback(self.executor, node.Span())
				if err != nil {
					return Result{}, err
				}
				self.symbols = append(self.symbols, symbol{
					Value: value,
					Span:  node.Span(),
					Type:  value.Type().toSymbolType(),
				})
				return Result{Value: &value}, nil
			}
			if scopeValue == nil || *scopeValue == nil {
				self.symbols = append(self.symbols, symbol{
					Value: *scopeValue,
					Span:  node.Span(),
					Type:  SymbolTypeUnknown,
				})
			} else {
				self.symbols = append(self.symbols, symbol{
					Value: *scopeValue,
					Span:  node.Span(),
					Type:  (*scopeValue).Type().toSymbolType(),
				})

				if (*scopeValue).Type() == TypeEnum {
					self.issue(node.Span(), fmt.Sprintf("cannot use bare enum `%s` as value", key), errors.ReferenceError)
					return Result{}, nil
				}
			}

			return Result{Value: scopeValue}, nil
		}
		self.issue(node.Span(), fmt.Sprintf("variable or function with name '%s' not found", key), errors.ReferenceError)
		return Result{}, nil
	case AtomKindIfExpr:
		valueTemp, err := self.visitIfExpression(node.(IfExpr))
		if err != nil {
			return Result{}, err
		}
		result = valueTemp
	case AtomKindForExpr:
		valueTemp, err := self.visitForExpression(node.(AtomFor))
		if err != nil {
			return Result{}, err
		}
		result = valueTemp
	case AtomKindWhileExpr:
		valueTemp, err := self.visitWhileExpression(node.(AtomWhile))
		if err != nil {
			return Result{}, err
		}
		result = valueTemp
	case AtomKindLoopExpr:
		valueTemp, err := self.visitLoopExpression(node.(AtomLoop))
		if err != nil {
			return Result{}, err
		}
		result = valueTemp
	case AtomKindFnExpr:
		valueTemp, err := self.visitFunctionDeclaration(node.(AtomFunction))
		if err != nil {
			return Result{}, err
		}
		result.Value = valueTemp
	case AtomKindTryExpr:
		valueTemp, err := self.visitTryExpression(node.(AtomTry))
		if err != nil {
			return Result{}, err
		}
		result = valueTemp
	case AtomKindExpression:
		valueTemp, err := self.visitExpression(node.(AtomExpression).Expression)
		if err != nil {
			return Result{}, err
		}
		result = valueTemp
	case AtomKindEnum:
		if err := self.visitEnum(node.(AtomEnum)); err != nil {
			return Result{}, err
		}
		null := makeNull(node.Span())
		return Result{Value: &null}, nil
	case AtomKindEnumVariant:
		// search for an enum with the referer name
		referrer := node.(AtomEnumVariant).RefersToEnum
		variable := self.getVar(referrer)
		if variable == nil || (*variable) == nil {
			return Result{}, errors.NewError(node.Span(), fmt.Sprintf("use of undefined enum `%s`", referrer), errors.ReferenceError)
		}

		variant := node.(AtomEnumVariant)

		// check if the enum variant is valid
		if !(*variable).(ValueEnum).HasVariant(variant.Name) {
			return Result{}, errors.NewError(variant.Span(), fmt.Sprintf("use of undefined enum variant `%s`", referrer), errors.ReferenceError)
		}

		value := Value(ValueEnumVariant{
			Name:  variant.Name,
			Range: node.(AtomEnumVariant).Range,
		})
		return Result{Value: &value}, nil
	default:
		panic("BUG: A new atom was introduced without updating this code")
	}
	if result.Value != nil && *result.Value != nil {
		self.symbols = append(self.symbols, symbol{
			Span:  (*result.Value).Span(),
			Value: *result.Value,
			Type:  (*result.Value).Type().toSymbolType(),
		})
	}
	return result, nil
}

func (self *Analyzer) makeList(node AtomListLiteral) (Result, *errors.Error) {
	// Validate that all types are the same
	valueType := TypeUnknown
	values := make([]*Value, 0)
	for idx, expression := range node.Values {
		result, err := self.visitExpression(expression)
		if err != nil {
			return Result{}, err
		}
		if result.Value == nil || *result.Value == nil {
			self.info(expression.Span, "manual validation required")
			continue
		}

		value := *result.Value
		if valueType != TypeUnknown && valueType != value.Type() {
			return Result{}, errors.NewError(
				expression.Span,
				fmt.Sprintf("value at index %d is of type %v, but this is a %v<%v>", idx, value.Type(), TypeList, valueType),
				errors.TypeError,
			)
		}
		valueType = value.Type()
		values = append(values, &value)
	}
	return Result{
		Value: valPtr(ValueList{
			Values:      &values,
			ValueType:   &valueType,
			Range:       node.Span(),
			IsProtected: false,
		}),
	}, nil
}

func (self *Analyzer) makeObject(node AtomObject) (Result, *errors.Error) {
	fields := make(map[string]*Value)
	for _, field := range node.Fields {
		_, exists := fields[field.Identifier]
		if exists {
			return Result{}, errors.NewError(
				field.IdentSpan,
				fmt.Sprintf("illegal duplicate key '%s' in object declaration", field.Identifier),
				errors.TypeError,
			)
		}
		defaultFields, err := ValueObject{ObjFields: map[string]*Value{}}.Fields(self.executor, node.Span())
		if err != nil {
			panic("This operation cannot fail since this is not a builtin variable")
		}
		_, isBuiltin := defaultFields[field.Identifier]
		if isBuiltin {
			self.issue(
				field.IdentSpan,
				fmt.Sprintf("key '%s' in object declaration is reserved for a builtin function", field.Identifier),
				errors.TypeError,
			)
		}
		value, err := self.visitExpression(field.Expression)
		if err != nil {
			return Result{}, err
		}

		fields[field.Identifier] = value.Value
	}
	return Result{Value: valPtr(ValueObject{
		IsDynamic:   true,
		ObjFields:   fields,
		Range:       node.Range,
		IsProtected: false,
	})}, nil
}

func (self *Analyzer) visitEnum(node AtomEnum) *errors.Error {
	variants := make([]ValueEnumVariant, 0)

	for _, variant := range node.Variants {
		variants = append(variants, ValueEnumVariant{
			Name:  variant.Value,
			Range: variant.Span,
		})
	}

	zero := 0

	enumValue := Value(ValueEnum{
		Variants:         variants,
		CurrentIterIndex: &zero,
		Range:            node.Span(),
		IsProtected:      false,
	})

	self.addVar(node.Name, enumValue, node.Span())
	return nil
}

func (self *Analyzer) visitTryExpression(node AtomTry) (Result, *errors.Error) {
	// Add a new scope to the try block
	if err := self.pushScope(
		self.getScope().identifier,
		node.Span(),
		make([]string, 0),
		self.getScope().inLoop,
		self.getScope().inFunction,
	); err != nil {
		return Result{}, err
	}

	_, err := self.visitStatements(node.TryBlock.IntoItemsList())
	if err != nil {
		return Result{}, err
	}
	// Remove the scope (cannot simply defer removing it (due to catch block))
	self.popScope()

	// Add a new scope for the catch block
	if err := self.pushScope(
		self.getScope().identifier,
		node.Span(),
		make([]string, 0),
		self.getScope().inLoop,
		self.getScope().inFunction,
	); err != nil {
		return Result{}, err
	}
	defer self.popScope()

	// Add the error variable to the scope (as an error object)
	self.addVar(node.ErrorIdentifier, ValueObject{
		ObjFields: map[string]*Value{
			"kind":    valPtr(makeStr(errors.Span{}, "error")),
			"message": valPtr(makeStr(errors.Span{}, "")),
			"location": valPtr(ValueObject{
				ObjFields: map[string]*Value{
					"start": valPtr(ValueObject{
						ObjFields: map[string]*Value{
							"index":  valPtr(makeNum(errors.Span{}, 0.0)),
							"line":   valPtr(makeNum(errors.Span{}, 0.0)),
							"column": valPtr(makeNum(errors.Span{}, 0.0)),
						},
					}),
					"end": valPtr(ValueObject{
						ObjFields: map[string]*Value{
							"index":  valPtr(makeNum(errors.Span{}, 0.0)),
							"line":   valPtr(makeNum(errors.Span{}, 0.0)),
							"column": valPtr(makeNum(errors.Span{}, 0.0)),
						},
					}),
				},
			}),
		},
		// TODO: improve location here
	}, node.Range)
	// Always visit the catch block
	_, err = self.visitStatements(node.CatchBlock.IntoItemsList())
	if err != nil {
		return Result{}, err
	}
	// Value of the entire expression is unknown so return a nil value
	return Result{}, nil
}

func (self *Analyzer) visitFunctionDeclaration(node AtomFunction) (*Value, *errors.Error) {
	function := ValueFunction{
		Identifier: node.Ident,
		Args:       node.ArgIdentifiers,
		Body:       node.Body,
		Range:      node.Range,
	}

	// If the function declaration contains an identifier, check for conflicts
	if node.Ident != nil {
		// Validate that there is no conflicting value in the scope already
		scopeValue := self.getVar(*node.Ident)
		if scopeValue != nil {
			self.issue(node.Range, fmt.Sprintf("cannot declare function with name '%s': name already taken in scope", *node.Ident), errors.TypeError)
			return nil, nil
		}
		// Add the function to the current scope if there are no conflicts
		// TODO: Improve span here
		self.addVar(*node.Ident, function, node.Range)
	} else {
		// Analyze the function
		args := make([]string, 0)
		for _, param := range function.Args {
			args = append(args, param.Identifier)
		}
		lambda := "<lambda>"
		if err := self.pushScope(&lambda, function.Span(), args, false, true); err != nil {
			return nil, err
		}
		// Add dummy values for the parameters
		for _, param := range function.Args {
			self.addVar(param.Identifier, nil, function.Span())
		}
		if _, err := self.visitStatements(function.Body.IntoItemsList()); err != nil {
			return nil, err
		}
		// Check if there are unused arguments
		for _, arg := range function.Args {
			isUsed := false
			for _, access := range self.getScope().variableAccesses {
				if access == arg.Identifier {
					isUsed = true
					break
				}
			}
			if !isUsed && !strings.HasPrefix(arg.Identifier, "_") {
				self.warn(arg.Span, fmt.Sprintf("function argument '%s' is unused", arg.Identifier))
			}
		}
		if err := self.popScope(); err != nil {
			return nil, err
		}
	}

	// Return the functions value so that assignments like `let a = fn foo() ...` are possible
	return valPtr(function), nil
}

func (self *Analyzer) visitIfExpression(node IfExpr) (Result, *errors.Error) {
	_, err := self.visitExpression(node.Condition)
	if err != nil {
		return Result{}, err
	}

	ifStmts := node.Block.IntoItemsList()

	// If branch
	if err := self.pushScope(
		self.getScope().identifier,
		node.Span(),
		make([]string, 0),
		self.getScope().inLoop,
		self.getScope().inFunction,
	); err != nil {
		return Result{}, err
	}
	resIf, err := self.visitStatements(ifStmts)
	if err != nil {
		return Result{}, err
	}
	self.popScope()

	// Visit potential else if construct
	if node.ElseIfExpr != nil {
		return self.visitIfExpression(*node.ElseIfExpr)
	}

	var ifType ValueType
	if resIf.Value != nil && *resIf.Value != nil {
		ifType = (*resIf.Value).Type()
	}

	// Else branch
	if node.ElseBlock == nil {
		if ifType != TypeNull && ifType != TypeUnknown {
			self.issue(node.Block.Span, fmt.Sprintf("Expected `else` block with `%v` result type", ifType), errors.TypeError)
		}

		return Result{}, nil
	}

	if err := self.pushScope(
		self.getScope().identifier,
		node.Span(),
		make([]string, 0),
		self.getScope().inLoop,
		self.getScope().inFunction,
	); err != nil {
		return Result{}, err
	}

	resElse, err := self.visitStatements((*node.ElseBlock).IntoItemsList())
	if err != nil {
		return Result{}, err
	}
	self.popScope()

	elseStmts := node.ElseBlock.IntoItemsList()

	var elseType ValueType

	if resElse.Value != nil && *resElse.Value != nil {
		elseType = (*resElse.Value).Type()
	}

	if ifType != TypeUnknown && elseType != TypeUnknown && ifType != TypeNull && elseType != TypeNull && ifType != elseType {
		var elseSpan errors.Span

		if len(elseStmts) > 0 {
			elseSpan = elseStmts[len(elseStmts)-1].Span()
		} else {
			elseSpan = node.ElseBlock.Span
		}

		var ifSpan errors.Span

		if len(ifStmts) > 0 {
			ifSpan = ifStmts[len(ifStmts)-1].Span()
		} else {
			ifSpan = node.Block.Span
		}

		self.diagnosticError(errors.Error{
			Span:    elseSpan,
			Message: fmt.Sprintf("Mismatched types: expected `%v`, found `%v`", ifType, elseType),
			Kind:    errors.TypeError,
		})
		self.info(ifSpan, "expected due to this")
		return Result{}, nil
	}

	return Result{Value: resIf.Value}, nil
}

func (self *Analyzer) visitForExpression(node AtomFor) (Result, *errors.Error) {
	// Create the value of the iter expression
	iter, err := self.visitExpression(node.IterExpr)
	if err != nil || iter.Value == nil || (*iter.Value) == nil {
		return Result{}, err
	}

	iterator, err := intoIter(*iter.Value, self.executor, node.IterExpr.Span)
	if err != nil {
		return Result{}, err
	}

	// Performs one iteration

	// Add a new scope for the iteration
	if err := self.pushScope(
		self.getScope().identifier,
		node.Span(),
		make([]string, 0),
		true,
		self.getScope().inFunction,
	); err != nil {
		return Result{}, err
	}

	var next Value
	if iterator != nil {
		_ = iterator(&next, node.HeadIdentifier.Span)

		if next != nil {
			self.symbols = append(self.symbols, symbol{
				Span:       node.HeadIdentifier.Span,
				Type:       next.Type().toSymbolType(),
				Value:      next,
				InFunction: false,
				InLoop:     true,
			})
		}
	}

	// Add the head identifier to the scope (so that loop code can access the iteration variable)
	self.addVar(node.HeadIdentifier.Identifier, next, node.HeadIdentifier.Span)

	_, err = self.visitStatements(node.IterationCode.IntoItemsList())
	if err != nil {
		return Result{}, err
	}

	// Remove the scope again
	self.popScope()
	return Result{}, nil
}

func (self *Analyzer) visitWhileExpression(node AtomWhile) (Result, *errors.Error) {
	// Conditional expression evaluation
	condValue, err := self.visitExpression(node.HeadCondition)
	if err != nil {
		return Result{}, err
	}
	// Only check the condition value's type if it is not null
	if condValue.Value != nil && *condValue.Value != nil {
		_, err := (*condValue.Value).IsTrue(self.executor, node.HeadCondition.Span)
		if err != nil {
			self.diagnosticError(*err)
		}
	}

	// Actual loop iteration code
	// Add a new scope for the loop
	if err := self.pushScope(
		self.getScope().identifier,
		node.Span(),
		make([]string, 0),
		true,
		self.getScope().inFunction,
	); err != nil {
		return Result{}, err
	}

	_, err = self.visitStatements(node.IterationCode.IntoItemsList())
	if err != nil {
		return Result{}, err
	}

	// Remove it as soon as the function is finished
	self.popScope()
	return Result{}, nil
}

func (self *Analyzer) visitLoopExpression(node AtomLoop) (Result, *errors.Error) {
	// Add a new scope for the loop
	if err := self.pushScope(
		self.getScope().identifier,
		node.Span(),
		make([]string, 0),
		true,
		self.getScope().inFunction,
	); err != nil {
		return Result{}, err
	}

	_, err := self.visitStatements(node.IterationCode.IntoItemsList())
	if err != nil {
		return Result{}, err
	}

	// Remove it as soon as the function is finished
	self.popScope()
	return Result{}, nil
}

// Helper functions
func (self *Analyzer) callValue(span errors.Span, value Value, args []Expression) (Value, *errors.Error) {
	switch value.Type() {
	case TypeFunction:
		// Cast the value to a function
		function := value.(ValueFunction)

		// Mark the function as used here
		// Only do it if the function was not already used before
		alreadyMarked := false
		for _, use := range self.getScope().functionCalls {
			if function.Identifier != nil && use == *function.Identifier {
				alreadyMarked = true
				break
			}
		}
		if !alreadyMarked && function.Identifier != nil {
			self.getScope().functionCalls = append(self.getScope().functionCalls, *function.Identifier)
		}

		// Used later for custom errors
		importedFunction := false
		for _, scope := range self.scopes {
			for _, imported := range scope.importedFunctions {
				if function.Identifier != nil && imported == *function.Identifier {
					// Create a note that the function should be verified manually
					self.info(span, "imported function: manual type verification required")
					importedFunction = true
					break
				}
			}
		}

		// Add a new scope for the running function and handle a potential stack overflow
		params := make([]string, 0)
		for _, arg := range function.Args {
			params = append(params, arg.Identifier)
		}
		if err := self.pushScope(function.Identifier, span, params, false, true); err != nil {
			return nil, err
		}
		// Remove the function scope again
		defer self.popScope()

		// Validate that the function has been called using the correct amount of arguments
		if len(args) != len(function.Args) && !importedFunction {
			self.issue(span, fmt.Sprintf("function requires %d argument(s), however %d were supplied", len(function.Args), len(args)), errors.TypeError)
			// Still evaluate the function body, just add dummy elements
			for _, arg := range function.Args {
				self.addVar(arg.Identifier, nil, span)
			}
		} else {
			// Evaluate argument values and add them to the new scope
			if importedFunction {
				for _, arg := range args {
					_, err := self.visitExpression(arg)
					return nil, err
				}
			} else {
				for idx, arg := range function.Args {
					argValue, err := self.visitExpression(args[idx])
					if err != nil {
						return nil, err
					}
					// Add the computed value to the new (current) scope
					if argValue.Value == nil || *argValue.Value == nil {
						self.addVar(arg.Identifier, nil, span)
					} else {
						// This will highlight the param identifier as the location
						//val := setValueSpan(*argValue.Value, arg.Span)

						// This hithlights the entire function
						self.addVar(arg.Identifier, *argValue.Value, span)
					}
				}
			}
		}

		// Prevent recursion here
		for idx := len(self.scopes) - 2; idx >= 0; idx-- {
			if self.scopes[idx].identifier != nil && function.Identifier != nil && *self.scopes[idx].identifier == *function.Identifier {
				return nil, nil
			}
		}

		// Do not visit the body of lamda functions again
		if function.Identifier == nil {
			return nil, nil
		}

		// Visit the function's body
		returnValue, err := self.visitStatements(function.Body.IntoItemsList())
		if err != nil {
			return nil, err
		}

		// Check if there are unused arguments
		for _, arg := range function.Args {
			isUsed := false
			for _, access := range self.getScope().variableAccesses {
				if access == arg.Identifier {
					isUsed = true
					break
				}
			}
			if !isUsed {
				self.warn(arg.Span, fmt.Sprintf("function argument '%s' is unused", arg.Identifier))
			}
		}

		// Required for preventing false errors
		if importedFunction {
			return nil, nil
		}

		if returnValue.ReturnValue != nil && *returnValue.ReturnValue != nil {
			return *returnValue.ReturnValue, nil
		}
		if returnValue.Value != nil && *returnValue.Value != nil {
			return *returnValue.Value, nil
		}
		return nil, nil
	case TypeBuiltinFunction:
		callArgs := make([]Value, 0)
		manualValidationShown := false
		for _, arg := range args {
			value, err := self.visitExpression(arg)
			if err != nil {
				return nil, err
			}
			if value.Value == nil || *value.Value == nil {
				if !manualValidationShown {
					self.info(span, "manual argument type validation required")
					manualValidationShown = true
				}
				continue
			}
			callArgs = append(callArgs, *value.Value)
		}
		// Dont call the function if one or more arguments are nil
		if manualValidationShown {
			return nil, nil
		}
		res, _, err := value.(ValueBuiltinFunction).Callback(self.executor, span, callArgs...)
		if err != nil {
			self.diagnosticError(*err)
			return nil, nil
		}
		return res, nil
	case TypeObject:
		if value.(ValueObject).IsDynamic {
			self.info(span, "dynamic object: manual call validation required")
			return nil, nil
		}
	}
	self.issue(span, fmt.Sprintf("value of type %v is not callable", value.Type()), errors.TypeError)
	return nil, nil
}

// Helper functions for scope management

// Pushes a new scope on top of the scopes stack
// Can return a runtime error if the maximum stack size would be exceeded by this operation
func (self *Analyzer) pushScope(
	ident *string,
	span errors.Span,
	args []string,
	inLoop bool,
	inFunction bool,
) *errors.Error {
	max := 20
	// Check that the stack size will be legal after this operation
	if len(self.scopes) >= max {
		return errors.NewError(span, fmt.Sprintf("Maximum stack size of %d was exceeded", max), errors.StackOverflow)
	}
	// Push a new stack frame onto the stack
	self.scopes = append(self.scopes, scope{
		this:          make(map[string]*Value),
		span:          span,
		identifier:    ident,
		functionCalls: make([]string, 0),
		args:          args,
		inLoop:        inLoop,
		inFunction:    inFunction,
	})
	return nil
}

// Pops a scope from the top of the stack
func (self *Analyzer) popScope() *errors.Error {
	if len(self.scopes) == 0 {
		panic("BUG: no scopes to pop")
	}
	// Check for unused variables before dropping the scope
	scope := self.scopes[len(self.scopes)-1]
	for key, valueTemp := range scope.this {
		if valueTemp == nil {
			continue
		}
		value := *valueTemp

		// If the current value is a function, check if it has been used
		if value != nil && value.Type() == TypeFunction {
			isUsed := false
			for _, use := range scope.functionCalls {
				if use == key {
					isUsed = true
					break
				}
			}
			if !isUsed {
				// Detect if the function is an imported function
				imported := false
				for _, im := range scope.importedFunctions {
					if im == key {
						imported = true
						break
					}
				}
				// Issue a warning if the identifier does not start with an underscore _
				if !strings.HasPrefix(key, "_") {
					if imported {
						self.warn(value.Span(), fmt.Sprintf("import '%s' is unused", key))
					} else {
						self.warn(value.Span(), fmt.Sprintf("function '%s' is unused", key))
					}
				}
				// Analyze the function
				args := make([]string, 0)
				for _, param := range value.(ValueFunction).Args {
					args = append(args, param.Identifier)
				}
				if err := self.pushScope(&key, scope.span, args, false, true); err != nil {
					return err
				}
				// Add dummy values for the parameters
				for _, param := range value.(ValueFunction).Args {
					self.addVar(param.Identifier, nil, value.Span())
				}
				if _, err := self.visitStatements(value.(ValueFunction).Body.IntoItemsList()); err != nil {
					return err
				}
				// Check if there are unused arguments
				for _, arg := range value.(ValueFunction).Args {
					isUsed := false
					for _, access := range self.getScope().variableAccesses {
						if access == arg.Identifier {
							isUsed = true
							break
						}
					}
					if !isUsed && !strings.HasPrefix(arg.Identifier, "_") {
						self.warn(arg.Span, fmt.Sprintf("function argument '%s' is unused", arg.Identifier))
					}
				}
				if err := self.popScope(); err != nil {
					return err
				}
			}
			// If the current value is a variable, analyze variable accesses
			// Only analyze variable access if the variable is not protected
		} else if value != nil && !value.Protected() {
			isUsed := false
			for _, access := range scope.variableAccesses {
				if access == key {
					isUsed = true
					break
				}
			}

			// Check if the current value is also an argument
			// If this is the case, stop here, it's uses will be checked in `callValue`
			isArg := false
			for _, arg := range scope.args {
				if arg == key {
					isArg = true
					break
				}
			}
			// Do not show a warning if the identifier starts with an underscore _
			if !isUsed && !isArg && !strings.HasPrefix(key, "_") {
				self.warn(value.Span(), fmt.Sprintf("variable '%s' is unused", key))
			}
		}
	}
	// Delete the scope
	self.scopes = self.scopes[:len(self.scopes)-1]
	return nil
}

// Adds a variable to the top of the stack
func (self *Analyzer) addVar(key string, value Value, span errors.Span) {
	// Add the entry to the top hashmap
	self.scopes[len(self.scopes)-1].this[key] = &value

	if value != nil {
		// Add the value to the symbols list
		self.symbols = append(self.symbols, symbol{
			Span:  value.Span(),
			Value: value,
			Type:  value.Type().toSymbolType(),
		})
	} else {
		self.symbols = append(self.symbols, symbol{
			Span:  span,
			Value: nil,
			Type:  SymbolTypeUnknown,
		})
	}
}

// Helper function for accessing the scope(s)
// Will also mark the access in the target scope
// Must provide a string key, will return either nil (no such value) or *value (value exists)
func (self *Analyzer) accessVar(key string) *Value {
	// Search the stack scope top to bottom (inner scopes have higher priority)
	scopeLen := len(self.scopes)
	// Must iterate over the slice backwards (0 is root | len-1 is top of the stack)
	for idx := scopeLen - 1; idx >= 0; idx-- {
		// Access the scope in order to get the identifier's value
		scopeValue, exists := self.scopes[idx].this[key]
		// If the correct value has been found, return early
		if exists {
			// Append the access to either function access or variable access
			if scopeValue != nil && *scopeValue != nil && (*scopeValue).Type() == TypeFunction {
				isMarked := false
				for _, funAccess := range self.scopes[idx].functionCalls {
					if funAccess == key {
						isMarked = true
						break
					}
				}
				if !isMarked {
					self.scopes[idx].functionCalls = append(self.scopes[idx].functionCalls, key)
				}
			} else {
				isMarked := false
				for _, access := range self.scopes[idx].variableAccesses {
					if access == key {
						isMarked = true
						break
					}
				}
				if !isMarked {
					self.scopes[idx].variableAccesses = append(self.scopes[idx].variableAccesses, key)
				}
			}
			return scopeValue
		}
	}
	return nil
}

// Helper function for accessing the scope(s)
// Must provide a string key, will return either nil (no such value) or *value (value exists)
// Does not keep a log of accesses
func (self *Analyzer) getVar(key string) *Value {
	// Search the stack scope top to bottom (inner scopes have higher priority)
	scopeLen := len(self.scopes)
	// Must iterate over the slice backwards (0 is root | len-1 is top of the stack)
	for idx := scopeLen - 1; idx >= 0; idx-- {
		// Access the scope in order to get the identifier's value
		scopeValue, exists := self.scopes[idx].this[key]
		// If the correct value has been found, return early
		if exists {
			return scopeValue
		}
	}
	return nil
}

func (self Analyzer) getScope() *scope {
	scopeLen := len(self.scopes)
	return &self.scopes[scopeLen-1]
}
