package homescript

import (
	"fmt"
	"strconv"

	"github.com/smarthome-go/homescript/homescript/errors"
)

type Interpreter struct {
	// Source program as AST representation
	program []Statement

	// Holds user-defined functions
	executor Executor

	// Scope stack: manages scopes (is searched in top -> down order)
	// The last element is the top whilst the first element is the bottom of the stack
	scopes []map[string]*Value

	// Holds the modules visited so far (by import statements)
	// in order to prevent a circular import
	moduleStack []string

	// Specifies how many stack frames may lay in the scopes stack
	// Is controlled by self.pushScope
	stackLimit uint

	// Can be used to terminate the script at any point in time
	sigTerm *chan int

	// Will unlock the use of the `break` and `continue` statements inside a statement list (if > 0)
	// If a loop is entered, it is incremented, once the loop exists, it is decremented
	inLoopCount uint

	// Will unlock the use of the `return` statement inside a statement list (if > 0)
	// If a function is entered, it is incremented, once a function exists, it is decremented
	inFunctionCount uint

	// Will enable debug output of the scopes
	debug bool
}

func NewInterpreter(
	program []Statement,
	executor Executor,
	sigTerm *chan int,
	stackLimit uint,
	scopeAdditions map[string]Value, // Allows the user to add more entries to the scope
	args map[string]Value, // Just like scopeAdditions but already named `ARGS.x`
	debug bool,
	moduleStack []string,
	moduleName string,
) Interpreter {
	scopes := make([]map[string]*Value, 0)
	// Adds the root scope
	scopes = append(scopes, map[string]*Value{
		// Builtin functions implemented by Homescript
		"exit":      valPtr(ValueBuiltinFunction{Callback: Exit}),
		"throw":     valPtr(ValueBuiltinFunction{Callback: Throw}),
		"assert":    valPtr(ValueBuiltinFunction{Callback: Assert}),
		"print":     valPtr(ValueBuiltinFunction{Callback: Print}), // Builtin functions implemented by the executor
		"println":   valPtr(ValueBuiltinFunction{Callback: Println}),
		"sleep":     valPtr(ValueBuiltinFunction{Callback: Sleep}),
		"switch_on": valPtr(ValueBuiltinFunction{Callback: SwitchOn}),
		"switch":    valPtr(ValueBuiltinFunction{Callback: Switch}),
		"notify":    valPtr(ValueBuiltinFunction{Callback: Notify}),
		"log":       valPtr(ValueBuiltinFunction{Callback: Log}),
		"exec":      valPtr(ValueBuiltinFunction{Callback: Exec}),
		"get":       valPtr(ValueBuiltinFunction{Callback: Get}),
		"http":      valPtr(ValueBuiltinFunction{Callback: Http}),
		"user":      valPtr(ValueBuiltinVariable{Callback: GetUser}), // Builtin variables implemented by the executor
		"weather":   valPtr(ValueBuiltinVariable{Callback: GetWeather}),
		"time":      valPtr(ValueBuiltinVariable{Callback: GetTime}),
		"ARGS": valPtr(ValueObject{
			DataType:  "args",
			IsDynamic: true,
			ObjFields: args,
		}),
	})
	// Panic if the stackLimit is set to <= 1
	if stackLimit <= 1 {
		panic(fmt.Sprintf("Stack limit is set to %d: such a low stack limit is probably useless", stackLimit))
	}
	// Add the optional scope entries
	for key, value := range scopeAdditions {
		// Check if the isertion would be legal
		_, exists := scopes[0][key]
		if exists {
			panic(fmt.Sprintf("cannot insert scope addition with key %s: this key is already taken by a builtin value", key))
		}
		// Insert the value into the scope
		scopes[0][key] = &value
	}
	// Append the current script to the module stack
	moduleStack = append(moduleStack, moduleName)
	return Interpreter{
		program:         program,
		executor:        executor,
		scopes:          scopes,
		stackLimit:      stackLimit,
		sigTerm:         sigTerm,
		inLoopCount:     0,
		inFunctionCount: 0,
		debug:           debug,
		moduleStack:     moduleStack,
	}
}

func (self *Interpreter) run() (Value, int, *errors.Error) {
	result, code, err := self.visitStatements(self.program)
	if err != nil {
		return makeNull(errors.Span{}), 1, err
	}
	if code != nil {
		return makeNull(errors.Span{}), *code, err
	}
	return *result.Value, 0, nil
}

// Utility function used at the beginning of any AST node's logic
// Is used to allow the abort of a running script at any point in time
// => Checks if a sigTerm has been received
// If this is the case, the code is returned as int, alongside with a bool indicating that a signal has been received
// If no sigTerm has been received, 0 and false are returned
func (self *Interpreter) checkSigTerm() (int, bool) {
	select {
	case code := <-*self.sigTerm:
		return code, true
	default:
		return 0, false
	}
}

// Interpreter code
func (self *Interpreter) visitStatements(statements []Statement) (Result, *int, *errors.Error) {
	lastResult := makeNullResult(errors.Span{})
	var code *int
	var err *errors.Error
	for _, statement := range statements {
		lastResult, code, err = self.visitStatement(statement)
		if code != nil || err != nil {
			return Result{}, code, err
		}

		// Handle potential break or return statements
		if lastResult.BreakValue != nil {
			// Check if the use of break is legal here
			if self.inLoopCount == 0 {
				return Result{}, nil, errors.NewError(statement.Span(), "Can only use the break statement inside loops", errors.SyntaxError)
			}
			return lastResult, nil, nil
		}

		// If continue is used, return null for this iteration
		if lastResult.ShouldContinue {
			// Check if the use of continue is legal here
			if self.inLoopCount == 0 {
				return Result{}, nil, errors.NewError(statement.Span(), "Can only use the continue statement inside loops", errors.SyntaxError)
			}
			return makeNullResult(statement.Span()), nil, nil
		}

		// Handle potential break or return statements
		if lastResult.ReturnValue != nil {
			// Check if the use of return is legal here
			if self.inFunctionCount == 0 {
				return Result{}, nil, errors.NewError(statement.Span(), "Can only use the return statement inside function bodies", errors.SyntaxError)
			}
			return lastResult, nil, nil
		}
	}
	return lastResult, code, err
}

func (self *Interpreter) visitStatement(node Statement) (Result, *int, *errors.Error) {
	/*
		SIGTERM catching
		Pre-execution validation of potential sigTerm checks if the function has to be aborted
		If a signal is received, the current function aborts using the provided exit-code
	*/
	if code, receivedSignal := self.checkSigTerm(); receivedSignal {
		return Result{}, &code, nil
	}
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
		value, code, err := self.visitExpression(node.(ExpressionStmt).Expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return value, nil, nil
	default:
		panic("BUG: a new statement kind was introduced without updating this code")
	}
}

func (self *Interpreter) visitLetStatement(node LetStmt) (Result, *int, *errors.Error) {
	// Check that the left hand side will cause no conflicts
	fromScope := self.getVar(node.Left.Identifier)
	if fromScope != nil {
		return Result{}, nil, errors.NewError(
			node.Span(),
			fmt.Sprintf("cannot declare variable with name %s: name already taken in scope", node.Left.Identifier),
			errors.SyntaxError,
		)
	}

	// Evaluate the right hand side
	rightResult, code, err := self.visitExpression(node.Right)
	if code != nil || err != nil {
		return Result{}, code, err
	}

	// Insert an identifier into the value (if possible)
	// TODO: contemplate whether to insert a value span
	//value := setValueSpan(*rightResult.Value, node.Left.Span)

	// Add the value to the scope
	self.addVar(node.Left.Identifier, rightResult.Value)
	// Also update the result value to include the new Identifier
	// rightResult.Value =
	// Finially, return the result
	return rightResult, nil, nil
}

func (self *Interpreter) visitImportStatement(node ImportStmt) (Result, *int, *errors.Error) {
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
			visual += fmt.Sprintf("  imports -> %2d: %-10s (HERE)\n", len(self.moduleStack)+1, node.FromModule)
			return Result{}, nil, errors.NewError(
				node.Range,
				fmt.Sprintf("Illegal import: circular import detected:\n%s", visual),
				errors.RuntimeError,
			)
		}
	}
	// Resolve the function to be imported
	function, err := self.ResolveModule(
		node.Span(),
		node.FromModule,
		node.Function,
	)
	if err != nil {
		return Result{}, nil, err
	}
	actualImport := node.Function
	if node.RewriteAs != nil {
		actualImport = *node.RewriteAs
	}
	// Check if the function conflicts with existing values
	value := self.getVar(actualImport)
	if value != nil {
		return Result{}, nil, errors.NewError(
			node.Span(),
			fmt.Sprintf("Import error: the name '%s' is already present in the current scope", actualImport),
			errors.ValueError,
		)
	}
	// Push the function into the current scope
	self.addVar(actualImport, function)
	return Result{Value: function}, nil, nil
}

func (self *Interpreter) visitBreakStatement(node BreakStmt) (Result, *int, *errors.Error) {
	// If the break should have a value, make and override it here
	if node.Expression != nil {
		value, code, err := self.visitExpression(*node.Expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return Result{BreakValue: value.Value}, nil, nil
	}
	// The break value defaults to null
	null := makeNull(node.Range)
	return Result{BreakValue: &null}, nil, nil
}

func (self *Interpreter) visitContinueStatement(node ContinueStmt) (Result, *int, *errors.Error) {
	return Result{
		ShouldContinue: true,
		ReturnValue:    nil,
		BreakValue:     nil,
		Value:          nil,
	}, nil, nil
}
func (self *Interpreter) visitReturnStatement(node ReturnStmt) (Result, *int, *errors.Error) {
	// The return value defaults to null
	returnValue := makeNull(node.Range)
	// If the return statment should return a value, make and override it here
	if node.Expression != nil {
		value, code, err := self.visitExpression(*node.Expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		returnValue = *value.Value
	}
	return Result{
		ShouldContinue: false,
		ReturnValue:    &returnValue,
		BreakValue:     nil,
		Value:          nil,
	}, nil, nil
}

// Expressions
// Expressions return

func (self *Interpreter) visitExpression(node Expression) (Result, *int, *errors.Error) {
	base, code, err := self.visitAndExpression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// If there are no other expressions, just return the base value
	if len(node.Following) == 0 {
		return base, nil, nil
	}
	// If the base is already true, return true without looking at the other expressions
	baseIsTrue, err := (*base.Value).IsTrue(self.executor, node.Base.Span)
	if err != nil {
		return Result{}, nil, err
	}
	if baseIsTrue {
		returnValue := makeBool(node.Span, true)
		return Result{Value: &returnValue}, nil, nil
	}
	// Look at the other expressions
	for _, following := range node.Following {
		followingValue, code, err := self.visitAndExpression(following)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		followingIsTrue, err := (*followingValue.Value).IsTrue(self.executor, following.Span)
		if err != nil {
			return Result{}, nil, err
		}
		// If the current value is true, return true without looking at the other expressions
		if followingIsTrue {
			returnValue := makeBool(node.Span, true)
			return Result{Value: &returnValue}, nil, nil
		}
	}
	// If all values before where false, return false
	returnValue := makeBool(node.Span, false)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Interpreter) visitAndExpression(node AndExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitEqExpression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// If there are no other expressions, just return the base value
	if len(node.Following) == 0 {
		return base, nil, nil
	}
	// If the base is false, stop here and return false
	baseIsTrue, err := (*base.Value).IsTrue(self.executor, node.Base.Span)
	if err != nil {
		return Result{}, nil, err
	}
	if !baseIsTrue {
		returnValue := makeBool(node.Span, false)
		return Result{Value: &returnValue}, nil, nil
	}
	// Look at the other expressions
	for _, following := range node.Following {
		followingValue, code, err := self.visitEqExpression(following)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		followingIsTrue, err := (*followingValue.Value).IsTrue(self.executor, following.Span)
		if err != nil {
			return Result{}, nil, err
		}
		// Stop here if the value is false
		if !followingIsTrue {
			returnValue := makeBool(node.Span, false)
			return Result{Value: &returnValue}, nil, nil
		}
	}
	// If all values where true, return true
	returnValue := makeBool(node.Span, true)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Interpreter) visitEqExpression(node EqExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitRelExression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// If there is nothing to compare to, return the base value
	if node.Other == nil {
		return base, nil, nil
	}
	otherValue, code, err := self.visitRelExression(node.Other.Node)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// Finally, test for equality
	isEqual, err := (*base.Value).IsEqual(self.executor, node.Span, *otherValue.Value)
	if err != nil {
		return Result{}, nil, err
	}
	// Check if the comparison should be inverted (using the != operator over the == operator)
	if node.Other.Inverted {
		returnValue := makeBool(node.Span, !isEqual)
		return Result{Value: &returnValue}, nil, nil
	}
	// If the comparison was not inverted, return the normal result
	returnValue := makeBool(node.Span, isEqual)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Interpreter) visitRelExression(node RelExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitAddExression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// If there is nothing to compare to, return the base value
	if node.Other == nil {
		return base, nil, nil
	}
	otherValue, code, err := self.visitAddExression(node.Other.Node)
	if code != nil || err != nil {
		return Result{}, code, err
	}

	// Check that the comparison involves a valid left hand side
	var baseVal Value
	switch (*base.Value).Type() {
	case TypeNumber:
		baseVal = (*base.Value).(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = (*base.Value).(ValueBuiltinVariable)
	default:
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("cannot compare %v to %v", (*base.Value).Type(), (*otherValue.Value).Type()), errors.TypeError)
	}

	// Perform typecast so that comparison operators can be used
	baseComp := baseVal.(ValueRelational)

	// Is later filled and evaluated once the correct check has been performed
	var relConditionTrue bool
	var relError *errors.Error

	// Finally, compare the two values
	switch node.Other.RelOperator {
	case RelLessThan:
		relConditionTrue, relError = baseComp.IsLessThan(self.executor, node.Span, *otherValue.Value)
	case RelLessOrEqual:
		relConditionTrue, relError = baseComp.IsLessThanOrEqual(self.executor, node.Span, *otherValue.Value)
	case RelGreaterThan:
		relConditionTrue, relError = baseComp.IsGreaterThan(self.executor, node.Span, *otherValue.Value)
	case RelGreaterOrEqual:
		relConditionTrue, relError = baseComp.IsGreaterThanOrEqual(self.executor, node.Span, *otherValue.Value)
	default:
		panic("BUG: a new rel operator was introduced without updating this code")
	}
	if relError != nil {
		return Result{}, nil, err
	}
	returnValue := makeBool(node.Span, relConditionTrue)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Interpreter) visitAddExression(node AddExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitMulExression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil, nil
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
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("cannot apply operation on type %v", (*base.Value).Type()), errors.TypeError)
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(ValueAlg)

	for _, following := range node.Following {
		// Is later filled and evaluated once the correct operator has been applied
		var algResult Value
		var algError *errors.Error

		followingValue, code, err := self.visitMulExression(following.Other)
		if code != nil || err != nil {
			return Result{}, code, err
		}

		switch following.AddOperator {
		case AddOpPlus:
			algResult, algError = baseAlg.Add(self.executor, node.Span, *followingValue.Value)
		case AddOpMinus:
			algResult, algError = baseAlg.Sub(self.executor, node.Span, *followingValue.Value)
		default:
			panic("BUG: a new add operator has been added without updating this code")
		}
		if algError != nil {
			return Result{}, nil, algError
		}
		// This is okay because the result of an algebraic operation should ALWAYS result in an algebraic type
		baseAlg = algResult.(ValueAlg)
	}
	returnValue := baseAlg.(Value)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Interpreter) visitMulExression(node MulExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitCastExpression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil, nil
	}

	// Check that the base holds a valid type to perform the requested operations
	var baseVal Value
	switch (*base.Value).Type() {
	case TypeNumber:
		baseVal = (*base.Value).(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = (*base.Value).(ValueBuiltinVariable)
	default:
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("cannot apply operation on type %v", (*base.Value).Type()), errors.TypeError)
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(ValueAlg)

	for _, following := range node.Following {
		// Is later filled and evaluated once the correct operator has been applied
		var algResult Value
		var algError *errors.Error

		followingValue, code, err := self.visitCastExpression(following.Other)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		switch following.MulOperator {
		case MulOpMul:
			algResult, algError = baseAlg.Mul(self.executor, node.Span, *followingValue.Value)
		case MulOpDiv:
			algResult, algError = baseAlg.Div(self.executor, node.Span, *followingValue.Value)
		case MulOpIntDiv:
			algResult, algError = baseAlg.IntDiv(self.executor, node.Span, *followingValue.Value)
		case MulOpReminder:
			algResult, algError = baseAlg.Rem(self.executor, node.Span, *followingValue.Value)
		default:
			panic("BUG: a new mul operator has been added without updating this code")
		}
		if algError != nil {
			return Result{}, nil, algError
		}
		// This is okay because the result of an algebraic operation should ALWAYS result in an algebraic type
		baseAlg = algResult.(ValueAlg)
	}
	// Holds the return value
	returnValue := baseAlg.(Value)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Interpreter) visitCastExpression(node CastExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitUnaryExpression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// If there is not typecast, only return the base value
	if node.Other == nil {
		return base, nil, nil
	}
	switch *node.Other {
	case TypeNumber:
		switch (*base.Value).Type() {
		case TypeNumber:
			return base, nil, nil
		case TypeBoolean:
			numeric := 0.0
			if (*base.Value).(ValueBool).Value {
				numeric = 1.0
			}
			// Holds the final result
			numericValue := makeNum(node.Span, numeric)
			return Result{Value: &numericValue}, nil, nil
		case TypeString:
			numeric, err := strconv.ParseFloat((*base.Value).(ValueString).Value, 64)
			if err != nil {
				return Result{}, nil, errors.NewError(node.Base.Span, fmt.Sprintf("cannot cast non-numeric string to number: %s", err.Error()), errors.ValueError)
			}

			// Holds the return value
			numericValue := makeNum(node.Span, numeric)
			return Result{Value: &numericValue}, nil, nil
		default:
			return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("cannot cast %v to %v", (*base.Value).Type(), *node.Other), errors.TypeError)
		}
	case TypeString:
		display, err := (*base.Value).Display(self.executor, node.Base.Span)
		if err != nil {
			return Result{}, nil, err
		}
		// Holds the return value
		valueStr := makeStr(node.Span, display)
		return Result{Value: &valueStr}, nil, nil
	case TypeBoolean:
		isTrue, err := (*base.Value).IsTrue(self.executor, node.Base.Span)
		if err != nil {
			return Result{}, nil, err
		}
		// Holds the return value
		truthValue := makeBool(node.Span, isTrue)
		return Result{Value: &truthValue}, nil, nil
	default:
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot cast to non-primitive type: cast from %v to %v is unsupported", (*base.Value).Type(), *node.Other), errors.TypeError)
	}
}
func (self *Interpreter) visitUnaryExpression(node UnaryExpression) (Result, *int, *errors.Error) {
	// If there is only a exp exression, return its value (recursion base case)
	if node.ExpExpression != nil {
		return self.visitEpxExpression(*node.ExpExpression)
	}
	unaryBase, code, err := self.visitUnaryExpression(node.UnaryExpression.UnaryExpression)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	var unaryResult Value
	var unaryErr *errors.Error
	switch node.UnaryExpression.UnaryOp {
	case UnaryOpPlus:
		unaryResult, unaryErr = ValueNumber{Value: 0.0}.Add(self.executor, node.UnaryExpression.UnaryExpression.Span, *unaryBase.Value)
	case UnaryOpMinus:
		unaryResult, unaryErr = ValueNumber{Value: 0.0}.Sub(self.executor, node.UnaryExpression.UnaryExpression.Span, *unaryBase.Value)
	case UnaryOpNot:
		unaryBaseIsTrueTemp, err := (*unaryBase.Value).IsTrue(self.executor, node.UnaryExpression.UnaryExpression.Span)
		if err != nil {
			return Result{}, nil, err
		}
		// Truth value is inverted due to the unary not (!)
		returnValue := makeBool(node.Span, !unaryBaseIsTrueTemp)
		return Result{Value: &returnValue}, nil, nil
	default:
		panic("BUG: a new unary operator has been added without updating this code")
	}
	if unaryErr != nil {
		return Result{}, nil, unaryErr
	}
	return Result{Value: &unaryResult}, nil, nil
}

func (self *Interpreter) visitEpxExpression(node ExpExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitAssignExression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// If there is no exponent, just return the base case
	if node.Other == nil {
		return base, nil, nil
	}
	power, code, err := self.visitUnaryExpression(*node.Other)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// Calculate result based on the base type
	var powRes Value
	var powErr *errors.Error
	switch (*base.Value).Type() {
	case TypeNumber:
		powRes, powErr = (*base.Value).(ValueNumber).Pow(self.executor, node.Span, *power.Value)
	case TypeBuiltinVariable:
		powRes, powErr = (*base.Value).(ValueBuiltinVariable).Pow(self.executor, node.Span, *power.Value)
	default:
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("cannot perform power operation on type %v", (*base.Value).Type()), errors.TypeError)
	}
	if powErr != nil {
		return Result{}, nil, powErr
	}
	return Result{Value: &powRes}, nil, nil
}

func assign(left *Value, right Value, span errors.Span) (Value, *errors.Error) {
	// Validate that the left hand side can be assigned to
	if (*left).Protected() {
		return nil, errors.NewError(
			span,
			fmt.Sprintf("cannot assign to protected value of type %v", (*left).Type()),
			errors.TypeError,
		)
	}
	// Check type equality
	if (*left).Type() != right.Type() {
		return nil, errors.NewError(
			span,
			fmt.Sprintf("cannot assign %v to %v: type inequality", right.Type(), (*left).Type()),
			errors.TypeError,
		)
	}
	// Insert new span into the right value
	// value := setValueSpan(
	//right,
	// *(*left).Span(),
	//right.Span(),
	//)
	// Perform actual assignment
	*left = right
	// Return the value of the right hand side
	return right, nil
}

func (self *Interpreter) visitAssignExression(node AssignExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitCallExpression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// If there is no assignment, return the base value here
	if node.Other == nil {
		return base, nil, nil
	}

	rhsValue, code, err := self.visitExpression(node.Other.Expression)
	if code != nil || err != nil {
		return Result{}, code, err
	}

	// Perform a simple assignment
	if node.Other.Operator == OpAssign {
		value, err := assign(base.Value, *rhsValue.Value, node.Span)
		return Result{Value: &value}, nil, err
	}
	// Check that the base is a type that can be safely assigned to using the complex operators
	if (*base.Value).Type() != TypeString && (*base.Value).Type() != TypeNumber {
		return Result{}, nil, errors.NewError(node.Base.Span, fmt.Sprintf("cannot use algebraic assignment operators on the %v type", (*base.Value).Type()), errors.TypeError)
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
		return Result{}, nil, assignErr
	}
	value, err := assign(base.Value, newValue, node.Span)
	return Result{Value: &value}, nil, err
}

// Functions from here on downstream return a pointer to a value so that it can be modified in a assign expression
func (self *Interpreter) visitCallExpression(node CallExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitMemberExpression(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// Evaluate call / member parts
	for _, part := range node.Parts {
		// Handle args -> function call
		if part.Args != nil {
			// Call the base using the following ars
			result, code, err := self.callValue(node.Span, *base.Value, *part.Args)
			if code != nil || err != nil {
				return Result{}, code, err
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = &result
		}
		// Handle member access
		if part.MemberExpressionPart != nil {
			result, err := getField(self.executor, part.Span, *base.Value, *part.MemberExpressionPart)
			if err != nil {
				return Result{}, nil, err
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = &result
		}
	}
	// Return the last base (the result)
	return base, nil, nil
}

func (self *Interpreter) visitMemberExpression(node MemberExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitAtom(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// Evaluate member expressions
	for _, member := range node.Members {
		// If the current member is a regular member access, perform it
		if member.Identifier != nil {
			result, err := getField(self.executor, member.Span, *base.Value, *member.Identifier)
			if err != nil {
				return Result{}, nil, err
			}
			// Swap the result and the base so that the next iteration uses this result
			base.Value = &result
		} else if member.Index != nil {
			indexValue, code, err := self.visitExpression(*member.Index)
			if code != nil || err != nil {
				return Result{}, code, err
			}
			// Check that the index is a number which is also an integer
			if (*indexValue.Value).Type() != TypeNumber {
				return Result{}, nil, errors.NewError(
					member.Span,
					fmt.Sprintf("type '%v' cannot be indexed by type '%v'", (*base.Value).Type(), (*indexValue.Value).Type()),
					errors.TypeError,
				)
			}
			index := (*indexValue.Value).(ValueNumber).Value
			// Check that the number is whole
			if index != float64(int(index)) {
				return Result{}, nil, errors.NewError(
					member.Span,
					"indices must be integer numbers",
					errors.ValueError,
				)
			}
			result, err := (*base.Value).Index(self.executor, int(index), member.Span)
			// Swap the result and the base so that the next iteration uses this result
			base.Value = &result
		} else {
			panic("BUG: a new member kind was introduced without updating this code")
		}
		// TODO: maybe include the base + member in the span (foo.bar.baz)
		// Like this	     									  ~~~~~~~
	}
	return base, nil, nil
}

func (self *Interpreter) visitAtom(node Atom) (Result, *int, *errors.Error) {
	switch node.Kind() {
	case AtomKindNumber:
		num := makeNum(node.Span(), node.(AtomNumber).Num)
		return Result{Value: &num}, nil, nil
	case AtomKindBoolean:
		bool := makeBool(node.Span(), node.(AtomBoolean).Value)
		return Result{Value: &bool}, nil, nil
	case AtomKindString:
		str := makeStr(node.Span(), node.(AtomString).Content)
		return Result{Value: &str}, nil, nil
	case AtomKindListLiteral:
		return self.makeList(node.(AtomListLiteral))
	case AtomKindPair:
		pairNode := node.(AtomPair)
		// Make the pair's value
		pairValue, code, err := self.visitExpression(pairNode.ValueExpr)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		pair := makePair(node.Span(), pairNode.Key, *pairValue.Value)
		return Result{Value: &pair}, nil, nil
	case AtomKindNull:
		null := makeNull(node.Span())
		return Result{Value: &null}, nil, nil
	case AtomKindIdentifier:
		// Search the scope for the correct key
		key := node.(AtomIdentifier).Identifier
		scopeValue := self.getVar(key)
		// If the key is associated with a value, return it
		if scopeValue != nil {
			// This resolves the `true` value of builtin variables directly here
			if (*scopeValue).Type() == TypeBuiltinVariable {
				value, err := (*scopeValue).(ValueBuiltinVariable).Callback(self.executor, node.Span())
				if err != nil {
					return Result{}, nil, err
				}
				return Result{Value: &value}, nil, nil
			}
			return Result{Value: scopeValue}, nil, nil
		}
		// If the value has not been found in any scope, return an error
		return Result{}, nil, errors.NewError(
			node.Span(),
			fmt.Sprintf("Variable or function with name %s not found", key),
			errors.ReferenceError,
		)
	case AtomKindIfExpr:
		valueTemp, code, err := self.visitIfExpression(node.(IfExpr))
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return valueTemp, nil, nil
	case AtomKindForExpr:
		valueTemp, code, err := self.visitForExpression(node.(AtomFor))
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return valueTemp, nil, nil
	case AtomKindWhileExpr:
		valueTemp, code, err := self.visitWhileExpression(node.(AtomWhile))
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return valueTemp, nil, nil
	case AtomKindLoopExpr:
		valueTemp, code, err := self.visitLoopExpression(node.(AtomLoop))
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return valueTemp, nil, nil
	case AtomKindFnExpr:
		valueTemp, err := self.visitFunctionDeclaration(node.(AtomFunction))
		if err != nil {
			return Result{}, nil, err
		}
		return Result{Value: &valueTemp}, nil, nil
	case AtomKindTryExpr:
		valueTemp, code, err := self.visitTryExpression(node.(AtomTry))
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return valueTemp, nil, nil
	case AtomKindExpression:
		valueTemp, code, err := self.visitExpression(node.(AtomExpression).Expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return valueTemp, nil, nil
	}
	panic("BUG: A new atom was introduced without updating this code")
}

func (self *Interpreter) makeList(node AtomListLiteral) (Result, *int, *errors.Error) {
	// Validate that all types are the same
	var valueType *ValueType
	typ := TypeUnknown
	valueType = &typ
	values := make([]Value, 0)
	for idx, expression := range node.Values {
		result, code, err := self.visitExpression(expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		value := *result.Value
		if valueType != nil && *valueType != value.Type() {
			return Result{}, nil, errors.NewError(
				expression.Span,
				fmt.Sprintf("value at index %d is of type %v, but this is a %v<%v>", idx, value.Type(), TypeList, *valueType),
				errors.TypeError,
			)
		}
		typ := value.Type()
		valueType = &typ
		values = append(values, value)
	}
	return Result{
		Value: valPtr(ValueList{
			Values:      &values,
			ValueType:   valueType,
			Range:       node.Span(),
			IsProtected: false,
		}),
	}, nil, nil
}

func (self *Interpreter) visitTryExpression(node AtomTry) (Result, *int, *errors.Error) {
	// Add a new scope to the try block
	if err := self.pushScope(node.Span()); err != nil {
		return Result{}, nil, err
	}
	tryBlockResult, code, err := self.visitStatements(node.TryBlock)

	// Remove the scope (cannot simly defer removing it (due to catch block))
	self.popScope()

	if code != nil {
		// Error is not caught intentionally
		return Result{}, code, nil
	}

	// if there is an error, handle it (try to catch it)
	if err != nil {
		// If the error is not a runtime / thrown error, do not catch it and propagate it
		if err.Kind != errors.RuntimeError && err.Kind != errors.ThrowError {
			// Modify the error's message so that the try attempt is apparent
			err.Message = fmt.Sprintf("Uncatchable error: %s", err.Message)
			return Result{}, nil, err
		}
		// Add a new scope for the catch block
		if err := self.pushScope(node.Span()); err != nil {
			return Result{}, nil, err
		}
		defer self.popScope()

		// Add the error variable to the scope (as an error object)
		errVal := Value(ValueObject{
			ObjFields: map[string]Value{
				"kind":    makeStr(errors.Span{}, err.Kind.String()),
				"message": makeStr(errors.Span{}, err.Message),
				"location": ValueObject{
					ObjFields: map[string]Value{
						"start": ValueObject{
							ObjFields: map[string]Value{
								"index":  makeNum(errors.Span{}, float64(err.Span.Start.Index)),
								"line":   makeNum(errors.Span{}, float64(err.Span.Start.Line)),
								"column": makeNum(errors.Span{}, float64(err.Span.Start.Column)),
							},
						},
						"end": ValueObject{
							ObjFields: map[string]Value{
								"index":  makeNum(errors.Span{}, float64(err.Span.End.Index)),
								"line":   makeNum(errors.Span{}, float64(err.Span.End.Line)),
								"column": makeNum(errors.Span{}, float64(err.Span.End.Column)),
							},
						},
					},
				},
			},
		})
		self.addVar(node.ErrorIdentifier, &errVal)

		// Visit the catch block
		catchResult, code, err := self.visitStatements(node.CatchBlock)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		// Return the result of the catch block
		return catchResult, nil, nil
	}
	return tryBlockResult, nil, nil
}

func (self *Interpreter) visitFunctionDeclaration(node AtomFunction) (Value, *errors.Error) {
	function := Value(ValueFunction{
		Identifier: node.Ident,
		Args:       node.ArgIdentifiers,
		Body:       node.Body,
		Range:      node.Span(),
	})

	// If the function declaration contains no identifier, just return the function's value
	if node.Ident != nil {
		// Validate that there is no conflicting value in the scope already
		scopeValue := self.getVar(*node.Ident)
		if scopeValue != nil {
			return nil, errors.NewError(node.Span(), fmt.Sprintf("cannot declare function with name %s: name already taken in scope", *node.Ident), errors.SyntaxError)
		}
		// Add the function to the current scope if there are no conflicts

		self.addVar(*node.Ident, &function)
	}

	// Return the functions value so that assignments like `let a = fn foo() ...` are possible
	return function, nil
}

func (self *Interpreter) visitIfExpression(node IfExpr) (Result, *int, *errors.Error) {
	conditionValue, code, err := self.visitExpression(node.Condition)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	conditionIsTrue, err := (*conditionValue.Value).IsTrue(self.executor, node.Span())
	if err != nil {
		return Result{}, nil, err
	}

	// If the condition is true, visit the true branch
	if conditionIsTrue {
		if err := self.pushScope(node.Span()); err != nil {
			return Result{}, nil, err
		}
		value, code, err := self.visitStatements(node.Block)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		self.popScope()
		// Forward any return, break or continue statement
		if value.ReturnValue != nil || value.BreakValue != nil || value.ShouldContinue {
			return value, nil, nil
		}
		// Otherwise, just return the result of the block
		return Result{Value: value.Value}, nil, nil
	}

	// If there is a else if construct, handle it here
	if node.ElseIfExpr != nil {
		return self.visitIfExpression(*node.ElseIfExpr)
	}

	// Otherwise, visit the else branch (if it exists)
	if node.ElseBlock == nil {
		return makeNullResult(node.Range), nil, nil
	}
	if err := self.pushScope(node.Span()); err != nil {
		return Result{}, nil, err
	}
	value, code, err := self.visitStatements(*node.ElseBlock)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	self.popScope()
	// Forward any return, break or continue statement
	if value.ReturnValue != nil || value.BreakValue != nil || value.ShouldContinue {
		return value, nil, nil
	}
	// Otherwise, just return the result of the block
	return Result{Value: value.Value}, nil, nil
}

func (self *Interpreter) visitForExpression(node AtomFor) (Result, *int, *errors.Error) {
	// Make the value of the lower range
	rangeLowerValue, code, err := self.visitExpression(node.RangeLowerExpr)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	rangeLowerNumeric := 0.0 // Placeholder is later filled
	// Assert that the value is of type number or builtin variable
	switch (*rangeLowerValue.Value).Type() {
	case TypeNumber:
		rangeLowerNumeric = (*rangeLowerValue.Value).(ValueNumber).Value
	case TypeBuiltinVariable:
		callBackResult, err := (*rangeLowerValue.Value).(ValueBuiltinVariable).Callback(self.executor, node.RangeLowerExpr.Span)
		if err != nil {
			return Result{}, nil, err
		}
		if callBackResult.Type() != TypeNumber {
			return Result{}, nil, errors.NewError(
				node.RangeLowerExpr.Span,
				fmt.Sprintf("cannot use value of type %v in a range", callBackResult.Type()),
				errors.TypeError,
			)
		}
		rangeLowerNumeric = callBackResult.(ValueNumber).Value
	}

	// Make the value of the upper range
	rangeUpperValue, code, err := self.visitExpression(node.RangeUpperExpr)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// Placeholder is later filled
	rangeUpperNumeric := 0.0
	// Assert that the value is of type number or builtin variable
	switch (*rangeUpperValue.Value).Type() {
	case TypeNumber:
		rangeUpperNumeric = (*rangeUpperValue.Value).(ValueNumber).Value
	case TypeBuiltinVariable:
		callBackResult, err := (*rangeUpperValue.Value).(ValueBuiltinVariable).Callback(self.executor, node.RangeUpperExpr.Span)
		if err != nil {
			return Result{}, nil, err
		}
		if callBackResult.Type() != TypeNumber {
			return Result{}, nil, errors.NewError(
				node.RangeUpperExpr.Span,
				fmt.Sprintf("cannot use value of type %v in a range", callBackResult.Type()),
				errors.TypeError,
			)
		}
		rangeUpperNumeric = callBackResult.(ValueNumber).Value
	}

	// Check that both ranges are whole numbers
	if rangeLowerNumeric != float64(int(rangeLowerNumeric)) || rangeUpperNumeric != float64(int(rangeUpperNumeric)) {
		return Result{}, nil, errors.NewError(
			errors.Span{
				Start: node.RangeLowerExpr.Span.Start,
				End:   node.RangeUpperExpr.Span.End,
			},
			"Range bounds have to be integers",
			errors.ValueError,
		)
	}

	// Saves the last result of the loop
	null := makeNull(node.Range)
	var lastValue *Value = &null

	// If the lower range bound is greater than the upper bound, reverse the order
	isReversed := rangeLowerNumeric > rangeUpperNumeric

	loopIter := int(rangeLowerNumeric)

	// Performs the iteration code
	for {
		// Loop control code
		if loopIter >= int(rangeUpperNumeric) && !isReversed || // Is normal (0..10)
			loopIter <= int(rangeUpperNumeric) && isReversed { // Is inverted (10..0)
			break
		}

		// Add a new scope for the iteration
		if err := self.pushScope(node.Span()); err != nil {
			return Result{}, nil, err
		}

		// Add the head identifier to the scope (so that loop code can access the iteration variable)
		num := Value(ValueNumber{Value: float64(loopIter)})
		self.addVar(node.HeadIdentifier, &num)

		// Enable the `inLoop` flag on the interpreter
		self.inLoopCount++

		value, code, err := self.visitStatements(node.IterationCode)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		self.inLoopCount--

		if value.BreakValue != nil {
			return Result{
				Value: value.BreakValue,
			}, nil, nil
		}

		// Assign to the last value
		lastValue = value.Value

		// Remove the scope again
		self.popScope()

		// Loop iter variabel control
		if isReversed {
			loopIter--
		} else {
			loopIter++
		}
	}

	// Returns the last of the loop's statements
	return Result{
		Value: lastValue,
	}, nil, nil
}

func (self *Interpreter) visitWhileExpression(node AtomWhile) (Result, *int, *errors.Error) {
	lastResult := makeNullResult(node.Range)
	for {
		// Conditional expression evaluation
		condValue, code, err := self.visitExpression(node.HeadCondition)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		condIsTrue, err := (*condValue.Value).IsTrue(self.executor, node.HeadCondition.Span)
		if err != nil {
			return Result{}, nil, err
		}
		// Break out of the loop when the condition is false
		if !condIsTrue {
			break
		}

		// Actual loop iteration code
		// Add a new scope for the loop
		if err := self.pushScope(node.Span()); err != nil {
			return Result{}, nil, err
		}
		// Enable the `inLoop` flag
		self.inLoopCount++

		result, code, err := self.visitStatements(node.IterationCode)
		if code != nil || err != nil {
			return Result{}, code, err
		}

		self.popScope()
		self.inLoopCount--

		// Check if there is a break statement
		if result.BreakValue != nil {
			return Result{Value: result.BreakValue}, nil, nil
		}

		// Otherwise, update the lastResult
		lastResult = result
	}
	return lastResult, nil, nil
}

func (self *Interpreter) visitLoopExpression(node AtomLoop) (Result, *int, *errors.Error) {
	for {
		// Add a new scope for the loop iteration
		if err := self.pushScope(node.Span()); err != nil {
			return Result{}, nil, err
		}
		self.inLoopCount++

		result, code, err := self.visitStatements(node.IterationCode)
		if code != nil || err != nil {
			return Result{}, code, err
		}

		self.popScope()
		self.inLoopCount--

		// Check if there is a break statement
		if result.BreakValue != nil {
			return Result{Value: result.BreakValue}, nil, nil
		}
	}
}

// Helper functions
func (self *Interpreter) callValue(span errors.Span, value Value, args []Expression) (Value, *int, *errors.Error) {
	switch value.Type() {
	case TypeFunction:
		// Cast the value to a function
		function := value.(ValueFunction)

		cntArgsRequired := len(function.Args)
		cntArgsGiven := len(args)

		// Validate that the function has been called using the correct amount of arguments
		if cntArgsGiven != cntArgsRequired {
			return nil, nil, errors.NewError(
				span,
				fmt.Sprintf("Function requires %d arguments, however %d were supplied", cntArgsRequired, cntArgsGiven),
				errors.TypeError,
			)
		}

		// Enable the `inFunction` flag on the interpreter
		self.inFunctionCount++
		// Release the `inFunction` flag as soon as possible
		defer func() { self.inFunctionCount-- }()

		// Add a new scope for the running function and handle a potential stack overflow
		if err := self.pushScope(span); err != nil {
			return nil, nil, err
		}
		// Remove the function scope again
		defer self.popScope()

		// Evaluate argument values and add them to the new scope
		for idx, arg := range function.Args {
			argValue, code, err := self.visitExpression(args[idx])
			if code != nil || err != nil {
				return nil, code, err
			}
			// Add the computed value to the new (current) scope
			self.addVar(arg.Identifier, argValue.Value)
		}

		// Visit the function's body
		returnValue, code, err := self.visitStatements(function.Body)
		if code != nil || err != nil {
			return nil, code, err
		}

		// Check if the return value is nil (no explicit return by the function)
		if returnValue.ReturnValue == nil {
			// In this case, return the default value of the last statement
			return *returnValue.Value, nil, nil
		}

		// Return the functions return value (concrete value implemented in `visitStatements`)
		return *returnValue.ReturnValue, nil, nil
	case TypeBuiltinFunction:
		// Prepare the call arguments for the function
		callArgs := make([]Value, 0)
		for _, arg := range args {
			argValue, code, err := self.visitExpression(arg)
			if code != nil || err != nil {
				return nil, code, err
			}
			callArgs = append(callArgs, *argValue.Value)
		}

		// Call the builtin function
		returnValue, code, err := value.(ValueBuiltinFunction).Callback(self.executor, span, callArgs...)
		if code != nil || err != nil {
			return nil, code, err
		}

		// Return the functions return value
		return returnValue, nil, nil
	default:
		return nil, nil, errors.NewError(span, fmt.Sprintf("Type %v is not callable", value.Type()), errors.TypeError)
	}
}

// Helper functions for scope management

// Pushes a new scope on top of the scopes stack
// Can return a runtime error if the maximum stack size would be exceeded by this operation
func (self *Interpreter) pushScope(span errors.Span) *errors.Error {
	// Check that the stack size will be legal after this operation
	if len(self.scopes) >= int(self.stackLimit) {
		return errors.NewError(span, fmt.Sprintf("Maximum stack size of %d was exceeded", self.stackLimit), errors.StackOverflow)
	}
	// Push a new stack frame onto the stack
	self.scopes = append(self.scopes, make(map[string]*Value))
	if self.debug {
		self.debugScope()
	}
	return nil
}

// Pops a scope from the top of the stack
func (self *Interpreter) popScope() {
	// Check that the root scope is not popped
	if len(self.scopes) == 1 {
		panic("BUG: cannot pop root scope")
	}
	self.scopes = self.scopes[:len(self.scopes)-1]
	// Remove the last (top) element from the slice / stack
	if self.debug {
		self.debugScope()
	}
}

// Adds a variable to the top of the stack
func (self *Interpreter) addVar(key string, value *Value) {
	// Add the entry to the top hashmap
	self.scopes[len(self.scopes)-1][key] = value
	if self.debug {
		self.debugScope()
	}
}

// Debug function for printing the scope(s)
func (self *Interpreter) debugScope() {
	for k, v := range self.scopes[len(self.scopes)-1] {
		dis, err := (*v).Display(self.executor, errors.Span{})
		if err != nil {
			panic(err.Message)
		}
		fmt.Printf("%10s => %s\n", k, dis)
	}
	fmt.Println()
	fmt.Println()
}

// Helper function for accessing the scope(s)
// Must provide a string key, will return either nil (no such value) or *value (value exists)
func (self *Interpreter) getVar(key string) *Value {
	// Search the stack scope top to bottom (inner scopes have higher priority)
	scopeLen := len(self.scopes)
	// Must iterate over the slice backwards (0 is root | len-1 is top of the stack)
	for idx := scopeLen - 1; idx >= 0; idx-- {
		// Access the scope in order to get the identifier's value
		scopeValue, exists := self.scopes[idx][key]
		// If the correct value has been found, return early
		if exists {
			return scopeValue
		}
	}
	return nil
}

// Resolves a function imported by an 'import' statement
// The builtin just has the task of providing the target module code
// This function then runs the target module code and returns the value of the target function (analyzes the root scope)
// If the target module contains top level code, it is also executed
func (self Interpreter) ResolveModule(span errors.Span, module string, function string) (*Value, *errors.Error) {
	moduleCode, found, err := self.executor.ResolveModule(module)
	if err != nil {
		return nil, errors.NewError(
			span,
			fmt.Sprintf("Import error: resolve module: %s", err.Error()),
			errors.RuntimeError,
		)
	}
	if !found {
		return nil, errors.NewError(
			span,
			fmt.Sprintf("Import error: cannot resolve module '%s' no such module", module),
			errors.RuntimeError,
		)
	}
	_, exitCode, rootScope, runErr := Run(
		self.executor,
		self.sigTerm,
		moduleCode,
		make(map[string]Value),
		make(map[string]Value),
		false,
		10,
		self.moduleStack,
		function,
	)
	if len(runErr) > 0 {
		if runErr != nil {
			runErr[0].Message = "Import error: target returned error: " + runErr[0].Message
			runErr[0].Span = span
			return nil, &runErr[0]
		}
	}
	if exitCode != 0 {
		return nil, errors.NewError(
			span,
			fmt.Sprintf("Import error: resolve module: script '%s' terminated with exit-code %d", module, exitCode),
			errors.RuntimeError,
		)
	}
	functionValue, found := rootScope[function]
	if !found {
		return nil, errors.NewError(
			span,
			fmt.Sprintf("Import error: no function named '%s' found in module '%s'", function, module),
			errors.RuntimeError,
		)
	}
	return functionValue, nil
}
