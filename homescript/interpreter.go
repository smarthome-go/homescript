package homescript

import (
	"fmt"
	"strconv"

	"github.com/smarthome-go/homescript/homescript/errors"
)

type Interpreter struct {
	program  []Statement
	executor Executor

	// Scope stack: manages scopes (is searched in top -> down order)
	// The last element is the top whilst the first element is the bottom of the stack
	scopes []map[string]Value

	// Specifies how many stack frames may lay in the scopes stack
	// Is controlled by self.pushScope
	stackLimit uint

	// Can be used to terminate the script at any point in time
	sigTerm *chan int
}

func NewInterpreter(
	program []Statement,
	executor Executor,
	sigTerm *chan int,
	stackLimit uint,
) Interpreter {
	scopes := make([]map[string]Value, 0)
	scopes = append(scopes, map[string]Value{
		// Builtin functions implemented by Homescript
		"exit":  ValueBuiltinFunction{}, // Special function implemented below
		"throw": ValueBuiltinFunction{Callback: Throw},
		// Builtin functions implemented by the executor
		"sleep":     ValueBuiltinFunction{Callback: Sleep},
		"switch_on": ValueBuiltinFunction{Callback: SwitchOn},
		"switch":    ValueBuiltinFunction{Callback: Switch},
		"notify":    ValueBuiltinFunction{Callback: Notify},
		"log":       ValueBuiltinFunction{Callback: Log},
		"exec":      ValueBuiltinFunction{Callback: Exec},
		"get":       ValueBuiltinFunction{Callback: Get},
		"http":      ValueBuiltinFunction{Callback: Http},
		// Builtin variables
		"user":    ValueBuiltinVariable{Callback: GetUser},
		"weather": ValueBuiltinVariable{Callback: GetWeather},
		"time":    ValueBuiltinVariable{Callback: GetTime},
	})
	return Interpreter{
		program:    program,
		executor:   executor,
		scopes:     scopes,
		stackLimit: stackLimit,
		sigTerm:    sigTerm,
	}
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
	null := makeNull()
	var lastResult Result = Result{
		ShouldContinue: false,
		ReturnValue:    nil,
		BreakValue:     nil,
		Value:          &null,
	}
	var code *int
	var err *errors.Error
	for _, statement := range statements {
		lastResult, code, err = self.visitStatement(statement)
		if code != nil || err != nil {
			return Result{}, code, err
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
		// Because visitExpression returns a value instead of a result, it must be transformed here
		value, code, err := self.visitExpression(node.(ExpressionStmt).Expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return Result{
			ShouldContinue: false,
			ReturnValue:    nil,
			BreakValue:     nil,
			Value:          &value,
		}, nil, nil
	default:
		panic("BUG: a new statement kind was introduced without updating this code")
	}
}

func (self *Interpreter) visitLetStatement(node LetStmt) (Result, *int, *errors.Error) {
	// TODO: implement this
	return Result{}, nil, nil
}
func (self *Interpreter) visitImportStatement(node ImportStmt) (Result, *int, *errors.Error) {
	// TODO: implement this
	return Result{}, nil, nil
}
func (self *Interpreter) visitBreakStatement(node BreakStmt) (Result, *int, *errors.Error) {
	// The break value defaults to null
	breakValue := makeNull()
	// If the break should have a value, make and override it here
	if node.Expression != nil {
		value, code, err := self.visitExpression(*node.Expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		breakValue = value
	}
	return Result{
		ShouldContinue: false,
		ReturnValue:    nil,
		BreakValue:     &breakValue,
		Value:          nil,
	}, nil, nil
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
	returnValue := makeNull()
	// If the return statment should return a value, make and override it here
	if node.Expression != nil {
		value, code, err := self.visitExpression(*node.Expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		returnValue = value
	}
	return Result{
		ShouldContinue: false,
		ReturnValue:    &returnValue,
		BreakValue:     nil,
		Value:          nil,
	}, nil, nil
}

// Expressions
func (self *Interpreter) visitExpression(node Expression) (Value, *int, *errors.Error) {
	base, code, err := self.visitAndExpression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If there are no other expressions, just return the base value
	if len(node.Following) == 0 {
		return base, nil, nil
	}
	// If the base is already true, return true without looking at the other expressions
	baseIsTrue, err := base.IsTrue(self.executor, node.Base.Span)
	if err != nil {
		return nil, nil, err
	}
	if baseIsTrue {
		return ValueBool{Value: true}, nil, nil
	}
	// Look at the other expressions
	for _, following := range node.Following {
		followingValue, code, err := self.visitAndExpression(following)
		if code != nil || err != nil {
			return nil, code, err
		}
		followingIsTrue, err := followingValue.IsTrue(self.executor, following.Span)
		if err != nil {
			return nil, nil, err
		}
		// If the current value is true, return true without looking at the other expressions
		if followingIsTrue {
			return ValueBool{Value: true}, nil, nil
		}
	}
	// If all values before where false, return false
	return ValueBool{Value: false}, nil, nil
}

func (self *Interpreter) visitAndExpression(node AndExpression) (Value, *int, *errors.Error) {
	base, code, err := self.visitEqExpression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If there are no other expressions, just return the base value
	if len(node.Following) == 0 {
		return base, nil, nil
	}
	// If the base is false, stop here and return false
	baseIsTrue, err := base.IsTrue(self.executor, node.Base.Span)
	if err != nil {
		return nil, nil, err
	}
	if !baseIsTrue {
		return ValueBool{Value: false}, nil, nil
	}
	// Look at the other expressions
	for _, following := range node.Following {
		followingValue, code, err := self.visitEqExpression(following)
		if code != nil || err != nil {
			return nil, code, err
		}
		followingIsTrue, err := followingValue.IsTrue(self.executor, following.Span)
		if err != nil {
			return nil, nil, err
		}
		// Stop here if the value is false
		if !followingIsTrue {
			return ValueBool{Value: false}, nil, nil
		}
	}
	// If all values where true, return true
	return ValueBool{Value: true}, nil, nil
}

func (self *Interpreter) visitEqExpression(node EqExpression) (Value, *int, *errors.Error) {
	base, code, err := self.visitRelExression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If there is nothing to compare to, return the base value
	if node.Other == nil {
		return base, nil, nil
	}
	otherValue, code, err := self.visitRelExression(node.Other.Node)
	if code != nil || err != nil {
		return nil, code, err
	}
	// Finally, test for equality
	isEqual, err := base.IsEqual(self.executor, node.Span, otherValue)
	if err != nil {
		return nil, nil, err
	}
	// Check if the comparison should be inverted (using the != operator over the == operator)
	if node.Other.Inverted {
		return ValueBool{Value: !isEqual}, nil, nil
	}
	// If the comparison was not inverted, return the normal result
	return ValueBool{Value: isEqual}, nil, nil
}

func (self *Interpreter) visitRelExression(node RelExpression) (Value, *int, *errors.Error) {
	base, code, err := self.visitAddExression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If there is nothing to compare to, return the base value
	if node.Other == nil {
		return base, nil, nil
	}
	otherValue, code, err := self.visitAddExression(node.Other.Node)
	if code != nil || err != nil {
		return nil, code, err
	}

	// Check that the comparison involves a valid left hand side
	var baseVal Value
	switch base.Type() {
	case TypeNumber:
		baseVal = base.(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = base.(ValueBuiltinVariable)
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot compare %v to %v", base.Type(), otherValue.Type()), errors.TypeError)
	}

	// Perform typecast so that comparison operators can be used
	baseComp := baseVal.(ValueRelational)

	// Is later filled and evaluated once the correct check has been performed
	var relConditionTrue bool
	var relError *errors.Error

	// Finally, compare the two values
	switch node.Other.RelOperator {
	case RelLessThan:
		relConditionTrue, relError = baseComp.IsLessThan(self.executor, node.Span, otherValue)
	case RelLessOrEqual:
		relConditionTrue, relError = baseComp.IsLessThanOrEqual(self.executor, node.Span, otherValue)
	case RelGreaterThan:
		relConditionTrue, relError = baseComp.IsGreaterThan(self.executor, node.Span, otherValue)
	case RelGreaterOrEqual:
		relConditionTrue, relError = baseComp.IsGreaterThanOrEqual(self.executor, node.Span, otherValue)
	default:
		panic("BUG: a new rel operator was introduced without updating this code")
	}
	if relError != nil {
		return nil, nil, err
	}
	return ValueBool{Value: relConditionTrue}, nil, nil
}

func (self *Interpreter) visitAddExression(node AddExpression) (Value, *int, *errors.Error) {
	base, code, err := self.visitMulExression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil, nil
	}

	// Check that the base holds a valid type to perform the requested operations
	var baseVal Value
	switch base.Type() {
	case TypeNumber:
		baseVal = base.(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = base.(ValueBuiltinVariable)
	case TypeString:
		baseVal = base.(ValueString)
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot apply operation on type %v", base.Type()), errors.TypeError)
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(ValueAlg)

	for _, following := range node.Following {
		// Is later filled and evaluated once the correct operator has been applied
		var algResult Value
		var algError *errors.Error

		followingValue, code, err := self.visitMulExression(following.Other)
		if code != nil || err != nil {
			return nil, code, err
		}
		switch following.AddOperator {
		case AddOpPlus:
			algResult, algError = baseAlg.Add(self.executor, node.Span, followingValue)
		case AddOpMinus:
			algResult, algError = baseAlg.Sub(self.executor, node.Span, followingValue)
		default:
			panic("BUG: a new add operator has been added without updating this code")
		}
		if algError != nil {
			return nil, nil, err
		}

		// This is okay because the result of an algebraic operation should ALWAYS result in the same type
		baseAlg = algResult.(ValueAlg)
	}
	return baseAlg.(Value), nil, nil
}
func (self *Interpreter) visitMulExression(node MulExpression) (Value, *int, *errors.Error) {
	base, code, err := self.visitCastExpression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil, nil
	}

	// Check that the base holds a valid type to perform the requested operations
	var baseVal Value
	switch base.Type() {
	case TypeNumber:
		baseVal = base.(ValueNumber)
	case TypeBuiltinVariable:
		baseVal = base.(ValueBuiltinVariable)
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot apply operation on type %v", base.Type()), errors.TypeError)
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(ValueAlg)

	for _, following := range node.Following {
		// Is later filled and evaluated once the correct operator has been applied
		var algResult Value
		var algError *errors.Error

		followingValue, code, err := self.visitCastExpression(following.Other)
		if code != nil || err != nil {
			return nil, code, err
		}
		switch following.MulOperator {
		case MulOpMul:
			algResult, algError = baseAlg.Mul(self.executor, node.Span, followingValue)
		case MulOpDiv:
			algResult, algError = baseAlg.Div(self.executor, node.Span, followingValue)
		case MullOpReminder:
			algResult, algError = baseAlg.Rem(self.executor, node.Span, followingValue)
		default:
			panic("BUG: a new mul operator has been added without updating this code")
		}
		if algError != nil {
			return nil, nil, err
		}

		// This is okay because the result of an algebraic operation should ALWAYS result in the same type
		baseAlg = algResult.(ValueAlg)
	}
	return baseAlg.(Value), nil, nil
}
func (self *Interpreter) visitCastExpression(node CastExpression) (Value, *int, *errors.Error) {
	base, code, err := self.visitUnaryExpression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If there is not typecast, only return the base value
	if node.Other == nil {
		return base, nil, nil
	}
	switch *node.Other {
	case TypeNumber:
		switch base.Type() {
		case TypeNumber:
			return base, nil, nil
		case TypeBoolean:
			numeric := 0.0
			if base.(ValueBool).Value {
				numeric = 1.0
			}
			return ValueNumber{Value: numeric}, nil, nil
		case TypeString:
			numeric, err := strconv.ParseFloat(base.(ValueString).Value, 64)
			if err != nil {
				return nil, nil, errors.NewError(node.Base.Span, fmt.Sprintf("Cannot cast non-numeric string to number: %s", err.Error()), errors.ValueError)
			}
			return ValueNumber{Value: numeric}, nil, nil
		default:
			return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot cast %v to %v", base.Type(), *node.Other), errors.TypeError)
		}
	case TypeString:
		display, err := base.Display(self.executor, node.Base.Span)
		if err != nil {
			return nil, nil, err
		}
		return ValueString{Value: display}, nil, nil
	case TypeBoolean:
		isTrue, err := base.IsTrue(self.executor, node.Base.Span)
		if err != nil {
			return nil, nil, err
		}
		return ValueBool{Value: isTrue}, nil, nil
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot cast to non-primitive type: cast from %v to %v is unsupported", base.Type(), *node.Other), errors.TypeError)
	}
}
func (self *Interpreter) visitUnaryExpression(node UnaryExpression) (Value, *int, *errors.Error) {
	// If there is only a exp exression, return its value (recursion base case)
	if node.ExpExpression != nil {
		return self.visitEpxExpression(*node.ExpExpression)
	}
	unaryBase, code, err := self.visitUnaryExpression(node.UnaryExpression.UnaryExpression)
	if code != nil || err != nil {
		return nil, code, err
	}
	var unaryResult Value
	var unaryErr *errors.Error
	switch node.UnaryExpression.UnaryOp {
	case UnaryOpPlus:
		unaryResult, unaryErr = ValueNumber{Value: 0.0}.Sub(self.executor, node.UnaryExpression.UnaryExpression.Span, unaryBase)
	case UnaryOpMinus:
		unaryResult, unaryErr = ValueNumber{Value: 0.0}.Add(self.executor, node.UnaryExpression.UnaryExpression.Span, unaryBase)
	case UnaryOpNot:
		unaryBaseIsTrueTemp, err := unaryBase.IsTrue(self.executor, node.UnaryExpression.UnaryExpression.Span)
		if err != nil {
			return nil, nil, err
		}
		return ValueBool{Value: !unaryBaseIsTrueTemp}, nil, nil
	default:
		panic("BUG: a new unary operator has been added without updating this code")
	}
	if unaryErr != nil {
		return nil, nil, unaryErr
	}
	return unaryResult, nil, nil
}
func (self *Interpreter) visitEpxExpression(node ExpExpression) (Value, *int, *errors.Error) {
	base, code, err := self.visitAssignExression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If there is no exponent, just return the base case
	if node.Other == nil {
		return base, nil, nil
	}
	power, code, err := self.visitUnaryExpression(*node.Other)
	if code != nil || err != nil {
		return nil, code, err
	}
	// Calculate result based on the base type
	var powRes Value
	var powErr *errors.Error
	switch base.Type() {
	case TypeNumber:
		powRes, powErr = base.(ValueNumber).Pow(self.executor, node.Span, power)
	case TypeBuiltinVariable:
		powRes, powErr = base.(ValueBuiltinVariable).Pow(self.executor, node.Span, power)
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot perform power operation on type %v", base.Type()), errors.TypeError)
	}
	if powErr != nil {
		return nil, nil, powErr
	}
	return powRes, nil, nil
}
func (self *Interpreter) visitAssignExression(node AssignExpression) (Value, *int, *errors.Error) {
	baseTmp, code, err := self.visitCallExpression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	base := *baseTmp
	// If there is no assignment, return the base value here
	if node.Other == nil {
		return base, nil, nil
	}
	rhsValue, code, err := self.visitExpression(node.Other.Expression)
	if code != nil || err != nil {
		return nil, code, err
	}
	// Check if this type legal to assign to
	if base.Type() == TypeObject || base.Type() == TypeBuiltinFunction || base.Type() == TypeBuiltinVariable {
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot reassign to type %v", base.Type()), errors.TypeError)
	}
	// Perform a simpple assignment
	if node.Other.Operator == OpAssign {
		// TODO: answer questions below
		// - Is this memory-safe?
		// - Could it lead to unexpected behaviour?
		baseTmp = &rhsValue
		return rhsValue, nil, nil
	}
	// Check that the base is a type that can be safely assigned to using the complex operators
	if base.Type() != TypeString && base.Type() != TypeNumber {
		return nil, nil, errors.NewError(node.Base.Span, fmt.Sprintf("Cannot use algebraic assignment operators on the %v type", base.Type()), errors.TypeError)
	}
	// Perform the more complex assignments
	var newValue Value
	var assignErr *errors.Error
	switch node.Other.Operator {
	case OpAssign:
		panic("BUG: this case should have been handled above")
	case OpPlusAssign:
		newValue, assignErr = base.(ValueAlg).Add(self.executor, node.Span, rhsValue)
	case OpMinusAssign:
		newValue, assignErr = base.(ValueAlg).Sub(self.executor, node.Span, rhsValue)
	case OpMulAssign:
		newValue, assignErr = base.(ValueAlg).Mul(self.executor, node.Span, rhsValue)
	case OpDivAssign:
		newValue, assignErr = base.(ValueAlg).Div(self.executor, node.Span, rhsValue)
	case OpReminderAssign:
		newValue, assignErr = base.(ValueAlg).Rem(self.executor, node.Span, rhsValue)
	case OpPowerAssign:
		newValue, assignErr = base.(ValueAlg).Pow(self.executor, node.Span, rhsValue)
	}
	if assignErr != nil {
		return nil, nil, assignErr
	}
	// Perform actual assignment
	baseTmp = &newValue
	// Return rhs value as a result of the entire expression
	return rhsValue, nil, nil
}

// Functions from here on downstream return a pointer to a value so that it can be modified in a assign expression
func (self *Interpreter) visitCallExpression(node CallExpression) (*Value, *int, *errors.Error) {
	base, code, err := self.visitMemberExpression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// Evaluate call / member parts
	for _, part := range node.Parts {
		// Handle args -> function call
		if part.Args != nil {
			// Call the base using the following ars
			result, code, err := self.callValue(node.Span, *base, *part.Args)
			if code != nil || err != nil {
				return nil, code, err
			}
			// Swap the result and the base so that the next iteration uses this result
			base = &result
		}

		// Handle member access
		if part.MemberExpressionPart != nil {
			result, err := getField(node.Span, *base, *part.MemberExpressionPart)
			if err != nil {
				return nil, nil, err
			}
			// Swap the result and the base so that the next iteration uses this result
			base = &result
		}
	}
	// Return the last base (the result)
	return base, nil, nil
}

func (self *Interpreter) visitMemberExpression(node MemberExpression) (*Value, *int, *errors.Error) {
	base, code, err := self.visitAtom(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// Evaluate member expressions
	for _, member := range node.Members {
		result, err := getField(node.Span, *base, member)
		if err != nil {
			return nil, nil, err
		}
		// Swap the result and the base so that the next iteration uses this result
		base = &result
	}
	return base, nil, nil
}

func (self *Interpreter) visitAtom(node Atom) (*Value, *int, *errors.Error) {
	value := makeNull()
	switch node.Kind() {
	case AtomKindNumber:
		value = ValueNumber{Value: node.(AtomNumber).Num}
	case AtomKindBoolean:
		value = ValueBool{Value: node.(AtomBoolean).Value}
	case AtomKindString:
		value = ValueString{Value: node.(AtomString).Content}
	case AtomKindPair:
		pairNode := node.(AtomPair)
		// Make the pair's value
		pairValue, code, err := self.visitExpression(pairNode.ValueExpr)
		if code != nil || err != nil {
			return nil, code, err
		}
		value = ValuePair{Key: pairNode.Key, Value: pairValue}
	case AtomKindNull:
		value = ValueNull{}
	case AtomKindIdentifier:
		key := node.(AtomIdentifier).Identifier
		// Seach the stack scope top to bottom (inner scopes have higher priority)
		for _, scope := range self.scopes {
			// Access the scope in order to get the identifier's value
			scopeValue, exists := scope[key]
			// If the correct value has been found, return early
			if exists {
				return &scopeValue, nil, nil
			}
		}
		// If the value has not been found in any scope, return an error
		return nil, nil, errors.NewError(
			node.Span(),
			fmt.Sprintf("Variable or function with name %s not found", key),
			errors.ReferenceError,
		)
	case AtomKindIfExpr:
		valueTemp, code, err := self.visitIfExpression(node.(IfExpr))
		if code != nil || err != nil {
			return nil, code, err
		}
		value = valueTemp
		// TODO: impl other atom kinds
	}
	return &value, nil, nil
}

func (self *Interpreter) visitIfExpression(node IfExpr) (Value, *int, *errors.Error) {
	conditionValue, code, err := self.visitExpression(node.Condition)
	if code != nil || err != nil {
		return nil, code, err
	}
	conditionIsTrue, err := conditionValue.IsTrue(self.executor, node.Span())
	if err != nil {
		return nil, nil, err
	}
	// If the condition is true, visit the true branch
	if conditionIsTrue {
		value, code, err := self.visitStatements(node.Block)
		if code != nil || err != nil {
			return nil, code, err
		}
		// Return the value of the block
		return *value.Value, nil, nil
	} else {
		// Otherwise, visit the else branch
		value, code, err := self.visitStatements(*node.ElseBlock)
		if code != nil || err != nil {
			return nil, code, err
		}
		// Return the value of the else block
		return *value.Value, nil, nil
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

		// Add a new scope for the running function and handle a potential stack overflow
		if err := self.pushScope(); err != nil {
			return nil, nil, err
		}

		// Evaluate argument values and add them to the new scope
		for index, argKey := range function.Args {
			argValue, code, err := self.visitExpression(args[index])
			if code != nil || err != nil {
				return nil, code, err
			}

			// Add the newly computed value to the scoe so the function can access it
			self.addVar(argKey, argValue)
		}

		// Visit the function's body
		returnValue, code, err := self.visitStatements(function.Body)
		if code != nil || err != nil {
			return nil, code, err
		}

		// Remove the function scope again
		self.popScope()

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
			callArgs = append(callArgs, argValue)
		}

		// Call the builtin function
		returnValue, err := value.(ValueBuiltinFunction).Callback(self.executor, span, callArgs...)
		if err != nil {
			return nil, nil, err
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
func (self *Interpreter) pushScope() *errors.Error {
	// Check that the stack size will be legal after this operation
	if len(self.scopes) >= int(self.stackLimit) {
		return errors.NewError(errors.Span{}, fmt.Sprintf("Maximum call stack size of %d was exceeded", self.stackLimit), errors.StackOverflow)
	}
	// Push a new stack frame onto the stack
	self.scopes = append(self.scopes, make(map[string]Value))
	return nil
}

// Pops a scope from the top of the stack
func (self *Interpreter) popScope() {
	// Check that the root scope is not popped
	if len(self.scopes) == 1 {
		panic("BUG: Cannot pop root scope")
	}
	// Remove the last (top) element from the slice / stack
	self.scopes = self.scopes[:len(self.scopes)]
}

// Adds a varable to the top of the stack
func (self *Interpreter) addVar(key string, value Value) {
	// Add the entry to the top hashmap
	self.scopes[len(self.scopes)-1][key] = value
}
