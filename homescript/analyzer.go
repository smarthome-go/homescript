package homescript

import (
	"fmt"
	"strconv"

	"github.com/smarthome-go/homescript/homescript/errors"
)

type Analyzer struct {
	program  []Statement
	executor Executor

	// Scope stack: manages scopes (is searched in to (p -> down order)
	// The last element is the top whilst the first element is the bottom of the stack
	scopes []scope

	// Holds the analyzer's diagnostics
	diagnostics []diagnostic

	// If the analyzer is currently handling a loop
	// Will unlock the use of the `break` and `continue` statements inside a statement list
	inLoop bool

	// If the analyzer is currently handling a function
	// Will unlock the use of the `return` statement inside a statement list
	// Used for preventing recursion
	inFunction bool
}

type diagnostic struct {
	severity DiagnosticSeverity
	kind     errors.ErrorKind
	message  string
	span     errors.Span
}

type DiagnosticSeverity uint8

const (
	Warning DiagnosticSeverity = iota
)

type scope struct {
	// Holds the actual scope
	this map[string]Value
	// If this scope belongs to a function
	identifier *string
}

func NewAnalyzer(
	program []Statement,
	executor Executor,
	scopeAdditions map[string]Value, // Allows the user to add more entries to the scope
) Analyzer {
	scopes := make([]scope, 0)
	// Adds the root scope
	scopes = append(scopes, scope{
		identifier: nil,
		this: map[string]Value{
			// Builtin functions implemented by Homescript
			"exit":   ValueBuiltinFunction{Callback: Exit},
			"throw":  ValueBuiltinFunction{Callback: Throw},
			"assert": ValueBuiltinFunction{Callback: Assert},
			// Builtin functions implemented by the executor
			"print":     ValueBuiltinFunction{Callback: Print},
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
	})
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
		program:    program,
		executor:   executor,
		scopes:     scopes,
		inLoop:     false,
		inFunction: false,
	}
}

func (self *Analyzer) diagnostic()

func (self *Analyzer) run() (Value, int, *errors.Error) {
	result, code, err := self.visitStatements(self.program)
	if err != nil {
		return makeNull(), 1, err
	}
	if code != nil {
		return makeNull(), *code, err
	}
	return *result.Value, 0, nil
}

// Analyzer code
func (self *Analyzer) visitStatements(statements []Statement) (Result, *int, *errors.Error) {
	lastResult := makeNullResult()
	var code *int
	var err *errors.Error

	// Is true if a break, return or continue statement is encountered
	unreachable := false

	for _, statement := range statements {
		// Check if the current statement is unreachable
		if unreachable {
			// Issue a warning that the statement is unreachable
		}

		lastResult, code, err = self.visitStatement(statement)
		if code != nil || err != nil {
			return Result{}, code, err
		}

		// Handle potential break or return statements
		if lastResult.BreakValue != nil {
			// Check if the use of break is legal here
			if !self.inLoop {
				return Result{}, nil, errors.NewError(statement.Span(), "Can only use the break statement inside loops", errors.SyntaxError)
			}
			// Continue intentionally
		}

		// Handle potential break or return statements
		if lastResult.ReturnValue != nil {
			// Check if the use of return is legal here
			if !self.inFunction {
				return Result{}, nil, errors.NewError(statement.Span(), "Can only use the return statement inside function bodies", errors.SyntaxError)
			}
			// Continue intentionally
		}
		// If continue is used, return null for this iteration
		if lastResult.ShouldContinue {
			// Check if the use of continue is legal here
			if !self.inLoop {
				return Result{}, nil, errors.NewError(statement.Span(), "Can only use the continue statement inside loops", errors.SyntaxError)
			}
			// Continue intentionally
		}
	}
	return lastResult, code, err
}

func (self *Analyzer) visitStatement(node Statement) (Result, *int, *errors.Error) {
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
			Value:          value.Value,
		}, nil, nil
	default:
		panic("BUG: a new statement kind was introduced without updating this code")
	}
}

func (self *Analyzer) visitLetStatement(node LetStmt) (Result, *int, *errors.Error) {
	// Check that the left hand side will cause no conflicts
	rightResult, code, err := self.visitExpression(node.Right)
	if code != nil || err != nil {
		return Result{}, code, nil
	}

	// Insert an identifier into the value (if possible)
	value := insertValueIdentifier(*rightResult.Value, node.Left)

	// Add the value to the scope
	self.addVar(node.Left, value)
	// Also update the result value to include the new Identifier
	rightResult.Value = &value
	// Finially, return the result
	return rightResult, nil, nil
}

func (self *Analyzer) visitImportStatement(node ImportStmt) (Result, *int, *errors.Error) {
	return Result{}, nil, errors.NewError(node.Span(), "The import statement is not yet implemented", errors.RuntimeError)
}

func (self *Analyzer) visitBreakStatement(node BreakStmt) (Result, *int, *errors.Error) {
	// If the break should have a value, make and override it here
	if node.Expression != nil {
		value, code, err := self.visitExpression(*node.Expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		return Result{BreakValue: value.Value}, nil, nil
	}
	// The break value defaults to null
	null := makeNull()
	return Result{BreakValue: &null}, nil, nil
}

func (self *Analyzer) visitContinueStatement(node ContinueStmt) (Result, *int, *errors.Error) {
	return Result{
		ShouldContinue: true,
		ReturnValue:    nil,
		BreakValue:     nil,
		Value:          nil,
	}, nil, nil
}
func (self *Analyzer) visitReturnStatement(node ReturnStmt) (Result, *int, *errors.Error) {
	// The return value defaults to null
	returnValue := makeNull()
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

func (self *Analyzer) visitExpression(node Expression) (Result, *int, *errors.Error) {
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
		returnValue := makeBool(true)
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
			returnValue := makeBool(true)
			return Result{Value: &returnValue}, nil, nil
		}
	}
	// If all values before where false, return false
	returnValue := makeBool(false)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Analyzer) visitAndExpression(node AndExpression) (Result, *int, *errors.Error) {
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
		returnValue := makeBool(false)
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
			returnValue := makeBool(false)
			return Result{Value: &returnValue}, nil, nil
		}
	}
	// If all values where true, return true
	returnValue := makeBool(true)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Analyzer) visitEqExpression(node EqExpression) (Result, *int, *errors.Error) {
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
		returnValue := makeBool(!isEqual)
		return Result{Value: &returnValue}, nil, nil
	}
	// If the comparison was not inverted, return the normal result
	returnValue := makeBool(isEqual)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Analyzer) visitRelExression(node RelExpression) (Result, *int, *errors.Error) {
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
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot compare %v to %v", (*base.Value).Type(), (*otherValue.Value).Type()), errors.TypeError)
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
	returnValue := makeBool(relConditionTrue)
	return Result{Value: &returnValue}, nil, nil
}

func (self *Analyzer) visitAddExression(node AddExpression) (Result, *int, *errors.Error) {
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
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot apply operation on type %v", (*base.Value).Type()), errors.TypeError)
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

		// This is okay because the result of an algebraic operation should ALWAYS result in the same type
		baseAlg = algResult.(ValueAlg)
	}
	returnValue := baseAlg.(Value)
	return Result{Value: &returnValue}, nil, nil
}
func (self *Analyzer) visitMulExression(node MulExpression) (Result, *int, *errors.Error) {
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
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot apply operation on type %v", (*base.Value).Type()), errors.TypeError)
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
		case MullOpReminder:
			algResult, algError = baseAlg.Rem(self.executor, node.Span, *followingValue.Value)
		default:
			panic("BUG: a new mul operator has been added without updating this code")
		}
		if algError != nil {
			return Result{}, nil, algError
		}
		// This is okay because the result of an algebraic operation should ALWAYS result in the same type
		baseAlg = algResult.(ValueAlg)
	}
	// Holds the return value
	returnValue := baseAlg.(Value)
	return Result{Value: &returnValue}, nil, nil
}
func (self *Analyzer) visitCastExpression(node CastExpression) (Result, *int, *errors.Error) {
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
			numericValue := makeNum(numeric)
			return Result{Value: &numericValue}, nil, nil
		case TypeString:
			numeric, err := strconv.ParseFloat((*base.Value).(ValueString).Value, 64)
			if err != nil {
				return Result{}, nil, errors.NewError(node.Base.Span, fmt.Sprintf("Cannot cast non-numeric string to number: %s", err.Error()), errors.ValueError)
			}

			// Holds the return value
			numericValue := makeNum(numeric)
			return Result{Value: &numericValue}, nil, nil
		default:
			return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot cast %v to %v", (*base.Value).Type(), *node.Other), errors.TypeError)
		}
	case TypeString:
		display, err := (*base.Value).Display(self.executor, node.Base.Span)
		if err != nil {
			return Result{}, nil, err
		}
		// Holds the return value
		valueStr := makeStr(display)
		return Result{Value: &valueStr}, nil, nil
	case TypeBoolean:
		isTrue, err := (*base.Value).IsTrue(self.executor, node.Base.Span)
		if err != nil {
			return Result{}, nil, err
		}
		// Holds the return value
		truthValue := makeBool(isTrue)
		return Result{Value: &truthValue}, nil, nil
	default:
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot cast to non-primitive type: cast from %v to %v is unsupported", (*base.Value).Type(), *node.Other), errors.TypeError)
	}
}
func (self *Analyzer) visitUnaryExpression(node UnaryExpression) (Result, *int, *errors.Error) {
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
		unaryResult, unaryErr = ValueNumber{Value: 0.0}.Sub(self.executor, node.UnaryExpression.UnaryExpression.Span, *unaryBase.Value)
	case UnaryOpMinus:
		unaryResult, unaryErr = ValueNumber{Value: 0.0}.Add(self.executor, node.UnaryExpression.UnaryExpression.Span, *unaryBase.Value)
	case UnaryOpNot:
		unaryBaseIsTrueTemp, err := (*unaryBase.Value).IsTrue(self.executor, node.UnaryExpression.UnaryExpression.Span)
		if err != nil {
			return Result{}, nil, err
		}
		// Truth value is inverted due to the unary not (!)
		returnValue := makeBool(!unaryBaseIsTrueTemp)
		return Result{Value: &returnValue}, nil, nil
	default:
		panic("BUG: a new unary operator has been added without updating this code")
	}
	if unaryErr != nil {
		return Result{}, nil, unaryErr
	}
	return Result{Value: &unaryResult}, nil, nil
}
func (self *Analyzer) visitEpxExpression(node ExpExpression) (Result, *int, *errors.Error) {
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
		return Result{}, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot perform power operation on type %v", (*base.Value).Type()), errors.TypeError)
	}
	if powErr != nil {
		return Result{}, nil, powErr
	}
	return Result{Value: &powRes}, nil, nil
}
func (self *Analyzer) visitAssignExression(node AssignExpression) (Result, *int, *errors.Error) {
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
		if ident := (*base.Value).Ident(); ident != nil {
			// Insert the original identifier back into the new value (if possible)
			*rhsValue.Value = insertValueIdentifier(*rhsValue.Value, *ident)

			// Need to manually search through the scopes to find the right stack frame
			for _, scope := range self.scopes {
				_, exist := scope[*ident]
				if exist {
					// Validate type equality
					if (*base.Value).Type() != (*rhsValue.Value).Type() {
						return Result{}, nil, errors.NewError(
							node.Span,
							fmt.Sprintf("Cannot assign %v to %v: type inequality", (*rhsValue.Value).Type(), (*base.Value).Type()),
							errors.TypeError,
						)
					}

					// Perform actual assignment
					scope[*ident] = *rhsValue.Value
					// Return the rhs as the return value of the entire assignment
					return rhsValue, nil, nil
				}
			}
			panic("BUG: value holds an identifer but is not present in scope")
		}
		// Return an error, which states that this type is not assignable to
		return Result{}, nil, errors.NewError(node.Base.Span, fmt.Sprintf("Cannot assign to %v", (*base.Value).Type()), errors.TypeError)

	}
	// Check that the base is a type that can be safely assigned to using the complex operators
	if (*base.Value).Type() != TypeString && (*base.Value).Type() != TypeNumber {
		return Result{}, nil, errors.NewError(node.Base.Span, fmt.Sprintf("Cannot use algebraic assignment operators on the %v type", (*base.Value).Type()), errors.TypeError)
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
		return Result{}, nil, assignErr
	}
	// Perform actual (complex) assignment
	if ident := (*base.Value).Ident(); ident != nil {
		// Insert the original identifier back into the new value (if possible)
		newValue = insertValueIdentifier(newValue, *ident)

		// Need to manually search through the scopes to find the right stack frame
		for _, scope := range self.scopes {
			_, exist := scope[*ident]
			if exist {
				// Type equality validation is omitted due to check above (in add / .. / div)
				// Perform actual assignment
				scope[*ident] = newValue
				// Return the rhs as the return value of the entire assignment
				return Result{Value: &newValue}, nil, nil
			}
		}
		panic("BUG: value holds an identifer but is not present in scope")
	}
	// Return an error, which states that this type is not assignable to
	return Result{}, nil, errors.NewError(node.Base.Span, fmt.Sprintf("Cannot assign to %v", (*base.Value).Type()), errors.TypeError)
}

// Functions from here on downstream return a pointer to a value so that it can be modified in a assign expression
func (self *Analyzer) visitCallExpression(node CallExpression) (Result, *int, *errors.Error) {
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
			result, err := getField(part.Span, *base.Value, *part.MemberExpressionPart)
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

func (self *Analyzer) visitMemberExpression(node MemberExpression) (Result, *int, *errors.Error) {
	base, code, err := self.visitAtom(node.Base)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// Evaluate member expressions
	for _, member := range node.Members {
		result, err := getField(node.Span, *base.Value, member)
		if err != nil {
			return Result{}, nil, err
		}
		// Swap the result and the base so that the next iteration uses this result
		base.Value = &result
	}
	return base, nil, nil
}

func (self *Analyzer) visitAtom(node Atom) (Result, *int, *errors.Error) {
	null := makeNull()
	result := Result{Value: &null}
	switch node.Kind() {
	case AtomKindNumber:
		num := makeNum(node.(AtomNumber).Num)
		result = Result{Value: &num}
	case AtomKindBoolean:
		bool := makeBool(node.(AtomBoolean).Value)
		result = Result{Value: &bool}
	case AtomKindString:
		str := makeStr(node.(AtomString).Content)
		result = Result{Value: &str}
	case AtomKindPair:
		pairNode := node.(AtomPair)
		// Make the pair's value
		pairValue, code, err := self.visitExpression(pairNode.ValueExpr)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		pair := makePair(pairNode.Key, *pairValue.Value)
		result = Result{Value: &pair}
	case AtomKindNull:
		null := makeNull()
		result = Result{Value: &null}
	case AtomKindIdentifier:
		// Search the scope for the correct key
		key := node.(AtomIdentifier).Identifier
		scopeValue := self.getVar(key)
		// If the key is associated with a value, return it
		if scopeValue != nil {
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
		result = valueTemp
	case AtomKindForExpr:
		valueTemp, code, err := self.visitForExpression(node.(AtomFor))
		if code != nil || err != nil {
			return Result{}, code, err
		}
		result = valueTemp
	case AtomKindWhileExpr:
		valueTemp, code, err := self.visitWhileExpression(node.(AtomWhile))
		if code != nil || err != nil {
			return Result{}, code, err
		}
		result = valueTemp
	case AtomKindLoopExpr:
		valueTemp, code, err := self.visitLoopExpression(node.(AtomLoop))
		if code != nil || err != nil {
			return Result{}, code, err
		}
		result = valueTemp
	case AtomKindFnExpr:
		valueTemp, err := self.visitFunctionDeclaration(node.(AtomFunction))
		if err != nil {
			return Result{}, nil, err
		}
		result.Value = &valueTemp
	case AtomKindTryExpr:
		valueTemp, code, err := self.visitTryExpression(node.(AtomTry))
		if code != nil || err != nil {
			return Result{}, code, err
		}
		result = valueTemp
	case AtomKindExpression:
		valueTemp, code, err := self.visitExpression(node.(AtomExpression).Expression)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		result = valueTemp
	}
	return result, nil, nil
}

func (self *Analyzer) visitTryExpression(node AtomTry) (Result, *int, *errors.Error) {
	// Add a new scope to the try block
	if err := self.pushScope(node.Span()); err != nil {
		return Result{}, nil, err
	}
	tryBlockResult, code, err := self.visitStatements(node.TryBlock)
	if code != nil {
		return Result{}, code, nil
	}
	// Remove the scope (cannot simly defer removing it (due to catch block))
	self.popScope()

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
		self.addVar(node.ErrorIdentifier, ValueObject{
			Fields: map[string]Value{
				"kind":    makeStr(err.Kind.String()),
				"message": makeStr(err.Message),
				"location": ValueObject{
					Fields: map[string]Value{
						"start": ValueObject{
							Fields: map[string]Value{
								"index":  makeNum(float64(err.Span.Start.Index)),
								"line":   makeNum(float64(err.Span.Start.Line)),
								"column": makeNum(float64(err.Span.Start.Column)),
							},
						},
						"end": ValueObject{
							Fields: map[string]Value{
								"index":  makeNum(float64(err.Span.End.Index)),
								"line":   makeNum(float64(err.Span.End.Line)),
								"column": makeNum(float64(err.Span.End.Column)),
							},
						},
					},
				},
			},
		})

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

func (self *Analyzer) visitFunctionDeclaration(node AtomFunction) (Value, *errors.Error) {
	function := ValueFunction{
		Identifier: node.Name,
		Args:       node.ArgIdentifiers,
		Body:       node.Body,
	}
	// Validate that there is no conflicting value in the scope already
	scopeValue := self.getVar(node.Name)
	if scopeValue != nil {
		return nil, errors.NewError(node.Span(), fmt.Sprintf("Cannot declare function with name %s: name already taken in scope", node.Name), errors.SyntaxError)
	}

	// Add the function to the current scope if there are no conflicts
	self.addVar(node.Name, function)
	// Return the functions value so that assignments like `let a = fn foo() ...` are possible
	return function, nil
}

func (self *Analyzer) visitIfExpression(node IfExpr) (Result, *int, *errors.Error) {
	conditionValue, code, err := self.visitExpression(node.Condition)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	conditionIsTrue, err := (*conditionValue.Value).IsTrue(self.executor, node.Span())
	if err != nil {
		return Result{}, nil, err
	}

	// Before visiting any branch, push a new scope
	if err := self.pushScope(node.Span()); err != nil {
		return Result{}, nil, err
	}
	// When this function is done, pop the scope again
	defer self.popScope()

	// If the condition is true, visit the true branch
	if conditionIsTrue {
		value, code, err := self.visitStatements(node.Block)
		if code != nil || err != nil {
			return Result{}, code, err
		}
		// Forward any return statement
		if value.ReturnValue != nil {
			return value, nil, nil
		}
		// Otherwise, just return the result of the block
		return Result{Value: value.Value}, nil, nil
	}
	// Otherwise, visit the else branch (if it exists)
	if node.ElseBlock == nil {
		return makeNullResult(), nil, nil
	}
	value, code, err := self.visitStatements(*node.ElseBlock)
	if code != nil || err != nil {
		return Result{}, code, err
	}
	// Forward any return statement
	if value.ReturnValue != nil {
		return value, nil, nil
	}
	// Otherwise, just return the result of the block
	return Result{Value: value.Value}, nil, nil
}

func (self *Analyzer) visitForExpression(node AtomFor) (Result, *int, *errors.Error) {
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
				fmt.Sprintf("Cannot use value of type %v in a range", callBackResult.Type()),
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
				fmt.Sprintf("Cannot use value of type %v in a range", callBackResult.Type()),
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
	null := makeNull()
	var lastValue *Value = &null

	// Enable the `inLoop` flag on the interpreter
	self.inLoop = true
	// Release the `inLoop` flag as soon as possible
	defer func() { self.inLoop = false }()

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
		self.addVar(node.HeadIdentifier, ValueNumber{Value: float64(loopIter)})

		value, code, err := self.visitStatements(node.IterationCode)
		if code != nil || err != nil {
			return Result{}, code, err
		}

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

func (self *Analyzer) visitWhileExpression(node AtomWhile) (Result, *int, *errors.Error) {
	lastResult := makeNullResult()
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
		self.inLoop = true
		// Release the `inLoop` flag as soon as this function is finished
		defer func() { self.inFunction = false }()

		result, code, err := self.visitStatements(node.IterationCode)
		if code != nil || err != nil {
			return Result{}, code, err
		}

		// Check if there is a break statement
		if result.BreakValue != nil {
			return Result{Value: result.BreakValue}, nil, nil
		}

		// Otherwise, update the lastResult
		lastResult = result

		// Remove it as soon as the function is finished
		self.popScope()
	}
	return lastResult, nil, nil
}

func (self *Analyzer) visitLoopExpression(node AtomLoop) (Result, *int, *errors.Error) {
	for {
		// Add a new scope for the loop
		if err := self.pushScope(node.Span()); err != nil {
			return Result{}, nil, err
		}

		// Enable the `inLoop` flag
		self.inLoop = true
		// Release the `inLoop` flag as soon as this function is finished
		defer func() { self.inFunction = false }()

		result, code, err := self.visitStatements(node.IterationCode)
		if code != nil || err != nil {
			return Result{}, code, err
		}

		// Check if there is a break statement
		if result.BreakValue != nil {
			return Result{Value: result.BreakValue}, nil, nil
		}

		// Remove it as soon as the function is finished
		self.popScope()
	}
}

// Helper functions
func (self *Analyzer) callValue(span errors.Span, value Value, args []Expression) (Value, *int, *errors.Error) {
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
		self.inFunction = true
		// Release the `inFunction` flag as soon as possible
		defer func() { self.inFunction = false }()

		// Add a new scope for the running function and handle a potential stack overflow
		self.pushScope(span)

		// Remove the function scope again
		defer self.popScope()

		// Evaluate argument values and add them to the new scope
		for idx, arg := range function.Args {
			argValue, code, err := self.visitExpression(args[idx])
			if code != nil || err != nil {
				return nil, code, err
			}
			// Add the computed value to the new (current) scope
			self.addVar(arg, *argValue.Value)
		}

		// If th
		a

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
func (self *Analyzer) pushScope(ident *string, span errors.Span) {
	// Push a new stack frame onto the stack
	self.scopes = append(self.scopes, make(map[string]Value))
}

// Pops a scope from the top of the stack
func (self *Analyzer) popScope() {
	// Check that the root scope is not popped
	if len(self.scopes) == 1 {
		panic("BUG: Cannot pop root scope")
	}
	// Remove the last (top) element from the slice / stack
	self.scopes = self.scopes[:len(self.scopes)-1]
}

// Adds a varable to the top of the stack
func (self *Analyzer) addVar(key string, value Value) {
	// Add the entry to the top hashmap
	self.scopes[len(self.scopes)-1][key] = value
}

// Debug function for printing the scope(s)
func (self *Analyzer) debugScopes() {
	fmt.Printf("\n")
	for idx, scope := range self.scopes {
		for k, v := range scope {
			dis, err := v.Display(self.executor, errors.Span{})
			if err != nil {
				panic(err.Message)
			}
			fmt.Printf("%10s => %s\n", k, dis)
		}
		fmt.Printf("---------- [NUM: %d] \n", idx)
	}
	fmt.Printf("\n")
}

// Helper function for accessing the scope(s)
// Must provide a string key, will return either nil (no such value) or *value (value exists)
func (self *Analyzer) getVar(key string) *Value {
	// Search the stack scope top to bottom (inner scopes have higher priority)
	scopeLen := len(self.scopes)
	// Must iterate over the slice backwards (0 is root | len-1 is top of the stack)
	for idx := scopeLen - 1; idx >= 0; idx-- {
		// Access the scope in order to get the identifier's value
		scopeValue, exists := self.scopes[idx][key]
		// If the correct value has been found, return early
		if exists {
			return &scopeValue
		}
	}
	return nil
}
