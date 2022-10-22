package homescript

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/smarthome-go/homescript/homescript/errors"
)

type Analyzer struct {
	program  []Statement
	executor Executor

	// Scope stack: manages scopes (is searched in to (p -> down order)
	// The last element is the top whilst the first element is the bottom of the stack
	scopes []scope

	// Holds the analyzer's diagnostics
	diagnostics []Diagnostic
}

func (self Diagnostic) Display(program string, filename string) string {
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
		"\x1b[1;36m%v\x1b[39m at %s:%d:%d\x1b[0m\n%s\n%s\n%s%s\n\n\x1b[1;3%dm%s\x1b[0m\n",
		self.Kind,
		filename,
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
	this map[string]Value
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
	// Used for issueing warnings about unused varirables
	variableAccesses []string
	// Saves which functions were imported
	// Because import is currently not implemented, the function should not be analyzed
	importedFunctions []string
	// Specifies whether the current scope is inside a function body / is a function body
	// Is used for backtracking the original caller of a function
	// Used for showing where a in-function type error was caused
	// Unlocks the use of the `return` statement
	inFunction bool
	// Specifies whether the current scope is inside a loop
	// Unlocks the use of the `break` and `continue` statements
	inLoop bool
}

func NewAnalyzer(
	program []Statement,
	executor Executor,
	scopeAdditions map[string]Value, // Allows the user to add more entries to the scope
) Analyzer {
	scopes := make([]scope, 0)
	// Adds the root scope
	scopes = append(scopes, scope{
		this: map[string]Value{
			// Builtin functions implemented by Homescript
			"exit":   ValueBuiltinFunction{Callback: Exit},
			"throw":  ValueBuiltinFunction{Callback: Throw},
			"assert": ValueBuiltinFunction{Callback: Assert},
			// Builtin functions implemented by the executor
			"print":     ValueBuiltinFunction{Callback: Print},
			"println":   ValueBuiltinFunction{Callback: Print},
			"sleep":     ValueBuiltinFunction{Callback: Sleep},
			"switch_on": ValueBuiltinFunction{Callback: SwitchOn},
			"switch":    ValueBuiltinFunction{Callback: Switch},
			"notify":    ValueBuiltinFunction{Callback: Notify},
			"log":       ValueBuiltinFunction{Callback: Log},
			"exec":      ValueBuiltinFunction{Callback: Exec},
			"get":       ValueBuiltinFunction{Callback: Get},
			"http":      ValueBuiltinFunction{Callback: Http},
			// Builtin variables implemented by the executor
			"user":    ValueBuiltinVariable{Callback: GetUser},
			"weather": ValueBuiltinVariable{Callback: GetWeather},
			"time":    ValueBuiltinVariable{Callback: GetTime},
		},
		identifier: nil,
	},
	)
	// Add the optional scope entries
	for key, value := range scopeAdditions {
		// Check if the isertion would be legal
		_, exists := scopes[0].this[key]
		if exists {
			panic(fmt.Sprintf("Cannot insert scope addition with key %s: this key is already taken by a builtin value", key))
		}
		// Insert the value into the scope
		scopes[0].this[key] = value
	}
	return Analyzer{
		program:  program,
		executor: executor,
		scopes:   scopes,
	}
}

func (self *Analyzer) diagnostic(span errors.Span, message string, severity DiagnosticSeverity, kind errors.ErrorKind) {
	self.diagnostics = append(self.diagnostics, Diagnostic{
		Severity: severity,
		Kind:     kind,
		Message:  message,
		Span:     span,
	})
}

// Can be used to create a diagnostic message at the point where the current function was called
func (self *Analyzer) badArgError(kind errors.ErrorKind, errSpan errors.Span) {
	// Only continue if invoked inside a function
	if !self.getScope().inFunction {
		return
	}
	// Backtracking to the point where the scope is no longer inside a function
	scopesCnt := len(self.scopes) - 1
	var prevIdent *string
	for idx := scopesCnt; idx >= 0; idx-- {
		curr := self.scopes[idx]
		// If the current scope is no longer in a function, the correct scope has been found
		// If the current scope is no longer in the same function, the correct scope has been found as well
		if !curr.inFunction ||
			(prevIdent != nil && curr.identifier != nil && *prevIdent != *curr.identifier) {
			self.info(
				self.scopes[idx+1].span,
				fmt.Sprintf("%v at line %d:%d caused here", kind, errSpan.Start.Line, errSpan.Start.Column),
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

func (self *Analyzer) analyze() []Diagnostic {
	_, err := self.visitStatements(self.program)
	if err != nil {
		self.diagnosticError(*err)
		return self.diagnostics
	}
	// Pop the last scope from the scopes in order to analyze top-level functions
	if err := self.popScope(); err != nil {
		self.diagnosticError(*err)
	}
	return self.diagnostics
}

// Interpreter code
func (self *Analyzer) visitStatements(statements []Statement) (Result, *errors.Error) {
	lastResult := makeNullResult(errors.Span{})

	unreachable := false
	for _, statement := range statements {
		// Check if statement is unreachable
		if unreachable {
			self.warn(statement.Span(), "unreachable statement")
		}

		result, err := self.visitStatement(statement)
		if err != nil {
			return Result{}, err
		}
		lastResult = result

		// Handle potential break or return statements
		if lastResult.BreakValue != nil {
			// Check if the use of break is legal here
			if !self.getScope().inLoop {
				self.issue(statement.Span(), "Can only use the break statement iside loops", errors.SyntaxError)
			} else {
				unreachable = true
			}
		}

		// If continue is used, return null for this iteration
		if lastResult.ShouldContinue {
			// Check if the use of continue is legal here
			if !self.getScope().inLoop {
				self.issue(statement.Span(), "Can only use the break statement iside loops", errors.SyntaxError)
			} else {
				unreachable = true
			}
		}

		// Handle potential break or return statements
		if lastResult.ReturnValue != nil {
			// Check if the use of return is legal here
			if !self.getScope().inFunction {
				self.issue(statement.Span(), "Can only use the return statement iside function bodies", errors.SyntaxError)
			} else {
				unreachable = true
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
	// Check that the left hand side will cause no conflicts
	fromScope := self.getVar(node.Left)
	if fromScope != nil {
		self.issue(node.Range, fmt.Sprintf("cannot declare variable with name %s: name already taken in scope", node.Left), errors.SyntaxError)
		return Result{}, nil
	}

	// Evaluate the right hand side
	rightResult, err := self.visitExpression(node.Right)
	if err != nil {
		return Result{}, err
	}

	if rightResult.Value == nil || *rightResult.Value == nil {
		// If the right hand side evaluates to nil, still add a placeholder to the scope
		self.addVar(node.Left, nil)
		return Result{}, nil
	}

	// Insert an identifier into the value (if possible)
	value := insertValueMetadata(*rightResult.Value, node.Left, node.Range)

	// Add the value to the scope
	self.addVar(node.Left, value)
	// Also update the result value to include the new Identifier
	rightResult.Value = &value
	// Finially, return the result
	return rightResult, nil
}

func (self *Analyzer) visitImportStatement(node ImportStmt) (Result, *errors.Error) {
	actualImport := node.Function
	if node.RewriteAs != nil {
		actualImport = *node.RewriteAs
	}

	// Check if the function conflicts with existing values
	value := self.getVar(actualImport)
	function := makeFn(&actualImport, node.Range)

	// Only report this non-critical error
	if value != nil {
		self.issue(node.Range,
			fmt.Sprintf("import error: the name '%s' is already present in the current scope", actualImport),
			errors.TypeError,
		)
	} else {
		// Push a dummy function into the current scope
		self.addVar(actualImport, function)
		// Add the funtion to the list of imported functions to avoid analysis
		self.getScope().importedFunctions = append(self.getScope().importedFunctions, actualImport)
	}

	return Result{Value: &function}, nil
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
	// If the return statment should return a value, make and override it here
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
	if base.Value != nil {
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

func (self *Analyzer) visitEqExpression(node EqExpression) (Result, *errors.Error) {
	base, err := self.visitRelExression(node.Base)
	if err != nil {
		return Result{}, err
	}

	// If there is nothing to compare to, return the base value
	if node.Other == nil {
		return base, nil
	}

	otherValue, err := self.visitRelExression(node.Other.Node)
	if err != nil {
		return Result{}, err
	}

	// Prevent further analysis if either the base or the other values are nil
	if base.Value == nil || otherValue.Value == nil {
		return makeBoolResult(node.Span, false), nil
	}

	// Finally, test for equality
	_, err = (*base.Value).IsEqual(self.executor, node.Span, *otherValue.Value)
	if err != nil {
		return makeBoolResult(node.Span, false), err
	}
	return Result{}, nil
}

func (self *Analyzer) visitRelExression(node RelExpression) (Result, *errors.Error) {
	base, err := self.visitAddExression(node.Base)
	if err != nil {
		return Result{}, err
	}
	// If there is nothing to compare to, return the base value
	if node.Other == nil {
		return base, nil
	}

	otherValue, err := self.visitAddExression(node.Other.Node)
	if err != nil {
		return Result{}, err
	}

	// Prevent further analysis if either the base or the other values are nil
	if base.Value == nil || otherValue.Value == nil {
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

func (self *Analyzer) visitAddExression(node AddExpression) (Result, *errors.Error) {
	base, err := self.visitMulExression(node.Base)
	if err != nil {
		return Result{}, err
	}

	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil
	}

	// Prevent further analysis if the base value is nil
	if base.Value == nil {
		self.info(node.Span, "manual type validation required")
		return Result{}, nil
	}

	// Check that the base holds a valid type to perform the requested operations
	var baseVal Value
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
		if base.Value != nil && (*base.Value).Ident() != nil {
			for _, arg := range self.getScope().args {
				if arg == *(*base.Value).Ident() {
					// Only show the function caller if the type error was caused by an argument
					self.badArgError(errors.TypeError, node.Span)
					break
				}
			}
		}
		self.issue(node.Span, fmt.Sprintf("cannot apply operation on type %v", (*base.Value).Type()), errors.TypeError)
		return Result{}, nil
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(ValueAlg)

	for _, following := range node.Following {
		// Is later filled and evaluated once the correct operator has been applied
		var algError *errors.Error
		var res Value

		followingValue, err := self.visitMulExression(following.Other)
		if err != nil {
			return Result{}, err
		}

		// Terminate this function's analysis if the followingValue is nil
		if followingValue.Value == nil || *followingValue.Value == nil {
			self.info(node.Span, "manual type validation required")
			return Result{}, nil
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
			if followingValue.Value != nil && (*followingValue.Value).Ident() != nil {
				for _, arg := range self.getScope().args {
					if arg == *(*followingValue.Value).Ident() {
						// Only show the function caller if the type error was caused by an argument
						self.badArgError(algError.Kind, algError.Span)
						break
					}
				}
			} else if base.Value != nil && (*base.Value).Ident() != nil {
				for _, arg := range self.getScope().args {
					fmt.Println(arg, *(*base.Value).Ident())
					if arg == *(*base.Value).Ident() {
						// Only show the function caller if the type error was caused by an argument
						self.badArgError(algError.Kind, algError.Span)
						break
					}
				}
			}
			self.diagnosticError(*algError)
			// Must return a blank result in order to prevent other functions from using a nil value
			return Result{}, nil
		}
		baseAlg = res.(ValueAlg)
	}

	returnValue := baseAlg.(Value)
	return Result{Value: &returnValue}, nil
}

func (self *Analyzer) visitMulExression(node MulExpression) (Result, *errors.Error) {
	base, err := self.visitCastExpression(node.Base)
	if err != nil {
		return Result{}, err
	}

	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil
	}

	// Prevent further analysis if the base value is nil
	if base.Value == nil {
		return Result{}, nil
	}

	// Check that the base holds a valid type to perform the requested operations
	var baseVal Value
	switch (*base.Value).Type() {
	case TypeNumber:
		baseVal = (*base.Value).(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = (*base.Value).(ValueBuiltinVariable)
	default:
		self.issue(node.Span, fmt.Sprintf("cannot apply operation on type %v", (*base.Value).Type()), errors.TypeError)
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
		if followingValue.Value == nil {
			return Result{}, nil
		}

		switch following.MulOperator {
		case MulOpMul:
			_, algError = baseAlg.Mul(self.executor, node.Span, *followingValue.Value)
		case MulOpDiv:
			_, algError = baseAlg.Div(self.executor, node.Span, *followingValue.Value)
		case MullOpReminder:
			_, algError = baseAlg.Rem(self.executor, node.Span, *followingValue.Value)
		default:
			panic("BUG: a new mul operator has been added without updating this code")
		}
		if algError != nil {
			return Result{}, algError
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
	if base.Value == nil {
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
	// If there is only a exp exression, return its value (recursion base case)
	if node.ExpExpression != nil {
		return self.visitEpxExpression(*node.ExpExpression)
	}
	unaryBase, err := self.visitUnaryExpression(node.UnaryExpression.UnaryExpression)
	if err != nil {
		return Result{}, err
	}

	// Stop here if the base value is nil
	if unaryBase.Value == nil {
		return Result{}, nil
	}

	var unaryErr *errors.Error
	switch node.UnaryExpression.UnaryOp {
	case UnaryOpPlus:
		_, unaryErr = ValueNumber{Value: 0.0}.Sub(self.executor, node.UnaryExpression.UnaryExpression.Span, *unaryBase.Value)
	case UnaryOpMinus:
		_, unaryErr = ValueNumber{Value: 0.0}.Add(self.executor, node.UnaryExpression.UnaryExpression.Span, *unaryBase.Value)
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
	base, err := self.visitAssignExression(node.Base)
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

func (self *Analyzer) visitAssignExression(node AssignExpression) (Result, *errors.Error) {
	base, err := self.visitCallExpression(node.Base)
	if err != nil {
		return Result{}, err
	}
	// If there is no assignment, return the base value here
	if node.Other == nil {
		return base, nil
	}
	rhsValue, err := self.visitExpression(node.Other.Expression)
	if err != nil {
		return Result{}, err
	}

	if base.Value == nil {
		if rhsValue.Value != nil {
			return Result{Value: rhsValue.Value}, nil
		}
		return Result{}, nil
	}

	if rhsValue.Value == nil {
		return Result{}, nil
	}

	// Perform a simple assignment
	if node.Other.Operator == OpAssign {
		if ident := (*base.Value).Ident(); ident != nil {
			// Need to manually search through the scopes to find the right stack frame
			for _, scope := range self.scopes {
				_, exist := scope.this[*ident]
				if exist {
					// Validate type equality (if the left type is null, anything can be assigned to it)
					if (*base.Value).Type() != (*rhsValue.Value).Type() && (*base.Value).Type() != TypeNull {
						self.issue(node.Span, fmt.Sprintf("cannot assign %v to %v: type inequality", (*rhsValue.Value).Type(), (*base.Value).Type()), errors.TypeError)
						return Result{}, nil
					}
					// Perform actual assignment (required for checking against null)
					value := insertValueMetadata(*rhsValue.Value, *ident, (*rhsValue.Value).Span())
					scope.this[*ident] = value
					// Return the rhs as the return value of the entire assignment
					return Result{Value: &value}, nil
				}
			}
			panic("BUG: value holds an identifer but is not present in scope")
		}
		self.issue(node.Base.Span, fmt.Sprintf("cannot assign to value of type %v", (*base.Value).Type()), errors.TypeError)
		return Result{}, nil
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
	case OpReminderAssign:
		newValue, assignErr = (*base.Value).(ValueAlg).Rem(self.executor, node.Span, *rhsValue.Value)
	case OpPowerAssign:
		newValue, assignErr = (*base.Value).(ValueAlg).Pow(self.executor, node.Span, *rhsValue.Value)
	}
	if assignErr != nil {
		return Result{}, assignErr
	}

	// Perform actual (complex) assignment
	if ident := (*base.Value).Ident(); ident != nil {
		// Insert the original identifier back into the new value (if possible)
		newValue = insertValueMetadata(newValue, *ident, newValue.Span())

		// Need to manually search through the scopes to find the right stack frame
		for _, scope := range self.scopes {
			_, exist := scope.this[*ident]
			if exist {
				// Perform actual assignment
				scope.this[*ident] = newValue
				// Return the rhs as the return value of the entire assignment
				return Result{Value: &newValue}, nil
			}
		}
		panic("BUG: value holds an identifer but is not present in scope")
	}
	// Return an error, which states that this type is not assignable to
	self.issue(node.Base.Span, fmt.Sprintf("cannot assign to value of type %v", (*base.Value).Type()), errors.TypeError)
	return rhsValue, nil
}

func (self *Analyzer) visitCallExpression(node CallExpression) (Result, *errors.Error) {
	base, err := self.visitMemberExpression(node.Base)
	if err != nil {
		return Result{}, err
	}
	// Evaluate call / member parts
	for _, part := range node.Parts {
		// Handle args -> function call
		if part.Args != nil {
			// Call the base using the following args
			// TODO: maybe include this
			if base.Value == nil || *base.Value == nil {
				return Result{}, nil
			}
			result, err := self.callValue(node.Span, *base.Value, *part.Args)
			if err != nil {
				self.diagnosticError(*err)
				return Result{}, nil
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = &result
			return Result{}, nil
		}
		// Handle member access
		if part.MemberExpressionPart != nil {
			result, err := getField(part.Span, *base.Value, *part.MemberExpressionPart)
			if err != nil {
				self.diagnosticError(*err)
				return Result{}, nil
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = &result
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

	if base.Value == nil || *base.Value == nil {
		return Result{}, nil
	}

	// Evaluate member expressions
	for _, member := range node.Members {
		result, err := getField(node.Span, *base.Value, member)
		if err != nil {
			self.diagnosticError(*err)
			return Result{}, nil
		}
		// Swap the result and the base so that the next iteration uses this result
		base.Value = &result
	}
	return base, nil
}

func (self *Analyzer) visitAtom(node Atom) (Result, *errors.Error) {
	null := makeNull(node.Span())
	result := Result{Value: &null}
	switch node.Kind() {
	case AtomKindNumber:
		num := makeNum(node.Span(), node.(AtomNumber).Num)
		result = Result{Value: &num}
	case AtomKindBoolean:
		bool := makeBool(node.Span(), node.(AtomBoolean).Value)
		result = Result{Value: &bool}
	case AtomKindString:
		str := makeStr(node.Span(), node.(AtomString).Content)
		result = Result{Value: &str}
	case AtomKindPair:
		pairNode := node.(AtomPair)
		// Make the pair's value
		pairValue, err := self.visitExpression(pairNode.ValueExpr)
		if err != nil {
			return Result{}, err
		}
		pair := makePair(node.Span(), pairNode.Key, *pairValue.Value)
		result = Result{Value: &pair}
	case AtomKindNull:
		null := makeNull(node.Span())
		result = Result{Value: &null}
	case AtomKindIdentifier:
		// Search the scope for the correct key
		key := node.(AtomIdentifier).Identifier
		scopeValue := self.accessVar(key)

		// Add this variable to the variable accesses
		// Only add the access if it is not already in the list
		isMarked := false
		for _, access := range self.getScope().variableAccesses {
			if access == key {
				isMarked = true
				break
			}
		}
		if !isMarked {
			self.getScope().variableAccesses = append(self.getScope().variableAccesses, key)
		}

		// If the key is associated with a value, return it
		if scopeValue != nil {
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
		result.Value = &valueTemp
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
	}
	return result, nil
}

func (self *Analyzer) visitTryExpression(node AtomTry) (Result, *errors.Error) {
	// Add a new scope to the try block
	if err := self.pushScope(self.getScope().identifier, node.Span(), self.getScope().inFunction, self.getScope().inLoop, make([]string, 0)); err != nil {
		return Result{}, err
	}
	_, err := self.visitStatements(node.TryBlock)
	if err != nil {
		return Result{}, err
	}
	// Remove the scope (cannot simly defer removing it (due to catch block))
	self.popScope()

	// Add a new scope for the catch block
	if err := self.pushScope(self.getScope().identifier, node.Span(), self.getScope().inFunction, self.getScope().inLoop, make([]string, 0)); err != nil {
		return Result{}, err
	}
	defer self.popScope()

	// Add the error variable to the scope (as an error object)
	self.addVar(node.ErrorIdentifier, ValueObject{
		Fields: map[string]Value{
			"kind":    makeStr(errors.Span{}, ""),
			"message": makeStr(errors.Span{}, ""),
			"location": ValueObject{
				Fields: map[string]Value{
					"start": ValueObject{
						Fields: map[string]Value{
							"index":  makeNum(errors.Span{}, 0.0),
							"line":   makeNum(errors.Span{}, 0.0),
							"column": makeNum(errors.Span{}, 0.0),
						},
					},
					"end": ValueObject{
						Fields: map[string]Value{
							"index":  makeNum(errors.Span{}, 0.0),
							"line":   makeNum(errors.Span{}, 0.0),
							"column": makeNum(errors.Span{}, 0.0),
						},
					},
				},
			},
		},
	})
	// Always visit the catch block
	_, err = self.visitStatements(node.CatchBlock)
	if err != nil {
		return Result{}, err
	}
	// Value of the entire expression is unknown so return a nil value
	return Result{}, nil
}

func (self *Analyzer) visitFunctionDeclaration(node AtomFunction) (Value, *errors.Error) {
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
		self.addVar(*node.Ident, function)
	}

	// TODO: move this
	/*

		if err := self.pushScope(node.Ident, node.Range); err != nil {
			return nil, err
		}

		self.inFunction = true

		// Add the function's parameters into the new scope
		for _, param := range node.ArgIdentifiers {
			self.addVar(param, nil)
		}

		// Before returning the function, analyze its body
		if _, err := self.visitStatements(node.Body); err != nil {
			return nil, err
		}

		self.inFunction = false
		self.popScope()

	*/

	// Return the functions value so that assignments like `let a = fn foo() ...` are possible

	return function, nil
}

func (self *Analyzer) visitIfExpression(node IfExpr) (Result, *errors.Error) {
	_, err := self.visitExpression(node.Condition)
	if err != nil {
		return Result{}, err
	}

	// If branch
	if err := self.pushScope(self.getScope().identifier, node.Span(), self.getScope().inFunction, self.getScope().inLoop, make([]string, 0)); err != nil {
		return Result{}, err
	}
	_, err = self.visitStatements(node.Block)
	if err != nil {
		return Result{}, err
	}
	self.popScope()

	// Else branch
	if node.ElseBlock == nil {
		return Result{}, nil
	}

	if err := self.pushScope(self.getScope().identifier, node.Span(), self.getScope().inFunction, self.getScope().inLoop, make([]string, 0)); err != nil {
		return Result{}, err
	}

	_, err = self.visitStatements(*node.ElseBlock)
	if err != nil {
		return Result{}, err
	}
	return Result{}, nil
}

func (self *Analyzer) visitForExpression(node AtomFor) (Result, *errors.Error) {
	// Make the value of the lower range
	rangeLowerValue, err := self.visitExpression(node.RangeLowerExpr)
	if err != nil {
		return Result{}, err
	}

	// Only check the lower value's type if it is not nil
	rangeLowerNumeric := 0.0
	if rangeLowerValue.Value != nil {
		switch (*rangeLowerValue.Value).Type() {
		case TypeNumber:
			rangeLowerNumeric = (*rangeLowerValue.Value).(ValueNumber).Value
		case TypeBuiltinVariable:
			callBackResult, err := (*rangeLowerValue.Value).(ValueBuiltinVariable).Callback(self.executor, node.RangeLowerExpr.Span)
			if err != nil {
				self.diagnosticError(*err)
				break
			}
			if callBackResult.Type() != TypeNumber {
				self.issue(node.RangeLowerExpr.Span, fmt.Sprintf("cannot use value of type %v in a range", callBackResult.Type()), errors.TypeError)
				break
			}
			rangeLowerNumeric = callBackResult.(ValueNumber).Value
		}
	}

	rangeUpperValue, err := self.visitExpression(node.RangeUpperExpr)
	if err != nil {
		return Result{}, err
	}

	// Only check the upper value's type if it is not nil
	rangeUpperNumeric := 0.0
	if rangeUpperValue.Value != nil {
		switch (*rangeUpperValue.Value).Type() {
		case TypeNumber:
			rangeUpperNumeric = (*rangeUpperValue.Value).(ValueNumber).Value
		case TypeBuiltinVariable:
			callBackResult, err := (*rangeUpperValue.Value).(ValueBuiltinVariable).Callback(self.executor, node.RangeUpperExpr.Span)
			if err != nil {
				self.diagnosticError(*err)
				break
			}
			if callBackResult.Type() != TypeNumber {
				self.issue(node.RangeUpperExpr.Span, fmt.Sprintf("cannot use value of type %v in a range", callBackResult.Type()), errors.TypeError)
				break
			}
			rangeUpperNumeric = callBackResult.(ValueNumber).Value
		}
	}

	// Check that both ranges are whole numbers
	if rangeLowerNumeric != float64(int(rangeLowerNumeric)) || rangeUpperNumeric != float64(int(rangeUpperNumeric)) {
		self.issue(
			errors.Span{
				Start: node.RangeLowerExpr.Span.Start,
				End:   node.RangeUpperExpr.Span.End,
			},
			"range bounds have to be integers",
			errors.TypeError,
		)
	}

	// Performs one iteration

	// Add a new scope for the iteration
	if err := self.pushScope(self.getScope().identifier, node.Span(), self.getScope().inFunction, true, make([]string, 0)); err != nil {
		return Result{}, err
	}

	// Add the head identifier to the scope (so that loop code can access the iteration variable)
	self.addVar(node.HeadIdentifier, ValueNumber{Value: 0.0})

	_, err = self.visitStatements(node.IterationCode)
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
	if condValue.Value != nil {
		_, err := (*condValue.Value).IsTrue(self.executor, node.HeadCondition.Span)
		if err != nil {
			self.diagnosticError(*err)
		}
	}

	// Actual loop iteration code
	// Add a new scope for the loop
	if err := self.pushScope(self.getScope().identifier, node.Span(), self.getScope().inFunction, true, make([]string, 0)); err != nil {
		return Result{}, err
	}

	_, err = self.visitStatements(node.IterationCode)
	if err != nil {
		return Result{}, err
	}

	// Remove it as soon as the function is finished
	self.popScope()
	return Result{}, nil
}

func (self *Analyzer) visitLoopExpression(node AtomLoop) (Result, *errors.Error) {
	// Add a new scope for the loop
	if err := self.pushScope(self.getScope().identifier, node.Span(), self.getScope().inFunction, true, make([]string, 0)); err != nil {
		return Result{}, err
	}

	_, err := self.visitStatements(node.IterationCode)
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

		// Do not call the function if it was imported
		for _, scope := range self.scopes {
			for _, imported := range scope.importedFunctions {
				if function.Identifier != nil && imported == *function.Identifier {
					// Create a note that the function should be verified manually
					self.info(span, "imported function: manual type verification required")
					return nil, nil
				}
			}
		}

		// Prevent recursion here
		for _, scope := range self.scopes {
			if scope.identifier != nil && function.Identifier != nil && *scope.identifier == *function.Identifier {
				return nil, nil
			}
		}

		// Add a new scope for the running function and handle a potential stack overflow
		params := make([]string, 0)
		for _, arg := range function.Args {
			params = append(params, arg.Identifier)
		}
		if err := self.pushScope(function.Identifier, span, true, false, params); err != nil {
			return nil, err
		}

		// Validate that the function has been called using the correct amount of arguments
		if len(args) != len(function.Args) {
			self.issue(span, fmt.Sprintf("function requires %d argument(s), however %d were supplied", len(function.Args), len(args)), errors.TypeError)
			// Still evaluate the function body, just add dummy elements
			for _, arg := range function.Args {
				self.addVar(arg.Identifier, nil)
			}
		} else {
			// Evaluate argument values and add them to the new scope
			for idx, arg := range function.Args {
				argValue, err := self.visitExpression(args[idx])
				if err != nil {
					return nil, err
				}
				// Add the computed value to the new (current) scope
				if argValue.Value == nil || *argValue.Value == nil {
					self.addVar(arg.Identifier, nil)
				} else {
					// This will highlight the param identifier as the location
					val := insertValueMetadata(*argValue.Value, arg.Identifier, arg.Span)

					// This hithlights the entire function
					// TODO: remove this
					//val := insertValueMetadata(*argValue.Value, arg, value.Span())
					self.addVar(arg.Identifier, val)
				}
			}
		}

		// Visit the function's body
		_, err := self.visitStatements(function.Body)
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

		// Remove the function scope again
		self.popScope()
		return nil, nil
	case TypeBuiltinFunction:
		for _, arg := range args {
			_, err := self.visitExpression(arg)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	default:
		self.issue(span, fmt.Sprintf("value of type %v is not callable", value.Type()), errors.TypeError)
		return nil, nil
	}
}

// Helper functions for scope management

// Pushes a new scope on top of the scopes stack
// Can return a runtime error if the maximum stack size would be exceeded by this operation
func (self *Analyzer) pushScope(ident *string, span errors.Span, inFunction bool, inLoop bool, args []string) *errors.Error {
	max := 20
	// Check that the stack size will be legal after this operation
	if len(self.scopes) >= max {
		return errors.NewError(span, fmt.Sprintf("Maximum stack size of %d was exceeded", max), errors.StackOverflow)
	}
	// Push a new stack frame onto the stack
	self.scopes = append(self.scopes, scope{
		this:          make(map[string]Value),
		span:          span,
		identifier:    ident,
		functionCalls: make([]string, 0),
		args:          args,
		inFunction:    inFunction,
		inLoop:        inLoop,
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
	for key, value := range scope.this {
		// If the current value is a variable, analyze variable accesses
		if value != nil && value.Ident() != nil {
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
			if !isUsed && !isArg {
				self.warn(value.Span(), fmt.Sprintf("variable '%s' is unused", key))
			}
		}

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
				// Issue a warning
				self.warn(value.Span(), fmt.Sprintf("function '%s' is unused", key))
				// Analyze the function
				args := make([]string, 0)
				for _, param := range value.(ValueFunction).Args {
					args = append(args, param.Identifier)
				}
				if err := self.pushScope(&key, scope.span, true, false, args); err != nil {
					return err
				}
				// Add dummy values for the parameters
				for _, param := range value.(ValueFunction).Args {
					self.addVar(param.Identifier, nil)
				}
				if _, err := self.visitStatements(value.(ValueFunction).Body); err != nil {
					return err
				}
				if err := self.popScope(); err != nil {
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
					if !isUsed {
						self.warn(arg.Span, fmt.Sprintf("function argument '%s' is unused", arg.Identifier))
					}
				}
			}
		}
	}
	// Delete the scope
	self.scopes = self.scopes[:len(self.scopes)-1]
	return nil
}

// Adds a varable to the top of the stack
func (self *Analyzer) addVar(key string, value Value) {
	// Add the entry to the top hashmap
	self.scopes[len(self.scopes)-1].this[key] = value
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
			if scopeValue != nil && scopeValue.Type() == TypeFunction {
				self.scopes[idx].functionCalls = append(self.scopes[idx].functionCalls, key)
			} else {
				self.scopes[idx].variableAccesses = append(self.scopes[idx].variableAccesses, key)
			}
			return &scopeValue
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
			return &scopeValue
		}
	}
	return nil
}

func (self Analyzer) getScope() *scope {
	scopeLen := len(self.scopes)
	return &self.scopes[scopeLen-1]
}
