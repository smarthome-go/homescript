package homescript

import (
	"fmt"
	"strconv"

	"github.com/smarthome-go/homescript/homescript/errors"
	"github.com/smarthome-go/homescript/homescript/interpreter"
)

type Interpreter struct {
	program  []Statement
	executor interpreter.Executor
	scopes   []map[string]interpreter.Value

	// Can be used to terminate the script at any point in time
	sigTerm *chan int
}

func NewInterpreter(
	program []Statement,
	executor interpreter.Executor,
	sigTerm *chan int,
) Interpreter {
	scopes := make([]map[string]interpreter.Value, 0)
	scopes = append(scopes, map[string]interpreter.Value{
		// Builtin functions implemented by Homescript
		"exit":  interpreter.ValueBuiltinFunction{}, // Special function implemented below
		"throw": interpreter.ValueBuiltinFunction{Callback: interpreter.Throw},
		// Builtin functions implemented by the executor
		"sleep":     interpreter.ValueBuiltinFunction{Callback: interpreter.Sleep},
		"switch_on": interpreter.ValueBuiltinFunction{Callback: interpreter.SwitchOn},
		"switch":    interpreter.ValueBuiltinFunction{Callback: interpreter.Switch},
		"notify":    interpreter.ValueBuiltinFunction{Callback: interpreter.Notify},
		"log":       interpreter.ValueBuiltinFunction{Callback: interpreter.Log},
		"exec":      interpreter.ValueBuiltinFunction{Callback: interpreter.Exec},
		"get":       interpreter.ValueBuiltinFunction{Callback: interpreter.Get},
		"http":      interpreter.ValueBuiltinFunction{Callback: interpreter.Http},
		// Builtin variables
		"user":    interpreter.ValueBuiltinVariable{Callback: interpreter.GetUser},
		"weather": interpreter.ValueBuiltinVariable{Callback: interpreter.GetWeather},
		"time":    interpreter.ValueBuiltinVariable{Callback: interpreter.GetTime},
	})
	return Interpreter{
		program:  program,
		executor: executor,
		scopes:   scopes,
		sigTerm:  sigTerm,
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
func (self *Interpreter) visitStatements() (interpreter.Value, *int, *errors.Error) {
	var value interpreter.Value
	var code *int
	var err *errors.Error
	for _, statement := range self.program {
		value, code, err = self.visitStatement(statement)
		if code != nil || err != nil {
			return nil, code, err
		}
	}
	return value, code, err
}

func (self *Interpreter) visitStatement(node Statement) (interpreter.Value, *int, *errors.Error) {
	/*
		SIGTERM catching
		Pre-execution validation of potential sigTerm checks if the function has to be aborted
		If a signal is received, the current function aborts using the provided exit-code
	*/
	if code, receivedSignal := self.checkSigTerm(); receivedSignal {
		return interpreter.ValueNull{}, &code, nil
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
		return self.visitExpression(node.(ExpressionStmt).Expression)
	default:
		panic("BUG: a new statement kind was introduced without updating this code")
	}
}

func (self *Interpreter) visitLetStatement(node LetStmt) (interpreter.Value, *int, *errors.Error) {
	return interpreter.ValueNull{}, nil, nil
}
func (self *Interpreter) visitImportStatement(node ImportStmt) (interpreter.Value, *int, *errors.Error) {
	return interpreter.ValueNull{}, nil, nil
}
func (self *Interpreter) visitBreakStatement(node BreakStmt) (interpreter.Value, *int, *errors.Error) {
	return interpreter.ValueNull{}, nil, nil
}
func (self *Interpreter) visitContinueStatement(node ContinueStmt) (interpreter.Value, *int, *errors.Error) {
	return interpreter.ValueNull{}, nil, nil
}
func (self *Interpreter) visitReturnStatement(node ReturnStmt) (interpreter.Value, *int, *errors.Error) {
	return interpreter.ValueNull{}, nil, nil
}

// Expressions
func (self *Interpreter) visitExpression(node Expression) (interpreter.Value, *int, *errors.Error) {
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
		return interpreter.ValueBool{Value: true}, nil, nil
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
			return interpreter.ValueBool{Value: true}, nil, nil
		}
	}
	// If all values before where false, return false
	return interpreter.ValueBool{Value: false}, nil, nil
}

func (self *Interpreter) visitAndExpression(node AndExpression) (interpreter.Value, *int, *errors.Error) {
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
		return interpreter.ValueBool{Value: false}, nil, nil
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
			return interpreter.ValueBool{Value: false}, nil, nil
		}
	}
	// If all values where true, return true
	return interpreter.ValueBool{Value: true}, nil, nil
}

func (self *Interpreter) visitEqExpression(node EqExpression) (interpreter.Value, *int, *errors.Error) {
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
		return interpreter.ValueBool{Value: !isEqual}, nil, nil
	}
	// If the comparison was not inverted, return the normal result
	return interpreter.ValueBool{Value: isEqual}, nil, nil
}

func (self *Interpreter) visitRelExression(node RelExpression) (interpreter.Value, *int, *errors.Error) {
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
	var baseVal interpreter.Value
	switch base.Type() {
	case interpreter.Number:
		baseVal = base.(interpreter.ValueNumber)
	case interpreter.BuiltinVariable:
		baseVal = base.(interpreter.ValueBuiltinVariable)
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot compare %v to %v", base.Type(), otherValue.Type()), errors.TypeError)
	}

	// Perform typecast so that comparison operators can be used
	baseComp := baseVal.(interpreter.ValueRelational)

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
	return interpreter.ValueBool{Value: relConditionTrue}, nil, nil
}

func (self *Interpreter) visitAddExression(node AddExpression) (interpreter.Value, *int, *errors.Error) {
	base, code, err := self.visitMulExression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil, nil
	}

	// Check that the base holds a valid type to perform the requested operations
	var baseVal interpreter.Value
	switch base.Type() {
	case interpreter.Number:
		baseVal = base.(interpreter.ValueNumber)
	case interpreter.BuiltinVariable:
		baseVal = base.(interpreter.ValueBuiltinVariable)
	case interpreter.String:
		baseVal = base.(interpreter.ValueString)
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot apply operation on type %v", base.Type()), errors.TypeError)
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(interpreter.ValueAlg)

	for _, following := range node.Following {
		// Is later filled and evaluated once the correct operator has been applied
		var algResult interpreter.Value
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
		baseAlg = algResult.(interpreter.ValueAlg)
	}
	return baseAlg.(interpreter.Value), nil, nil
}
func (self *Interpreter) visitMulExression(node MulExpression) (interpreter.Value, *int, *errors.Error) {
	base, code, err := self.visitCastExpression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If only the base is present, return its value
	if len(node.Following) == 0 {
		return base, nil, nil
	}

	// Check that the base holds a valid type to perform the requested operations
	var baseVal interpreter.Value
	switch base.Type() {
	case interpreter.Number:
		baseVal = base.(interpreter.ValueNumber)
	case interpreter.BuiltinVariable:
		baseVal = base.(interpreter.ValueBuiltinVariable)
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot apply operation on type %v", base.Type()), errors.TypeError)
	}

	// Performs typecase so that the algebraic functions are available on the base type
	baseAlg := baseVal.(interpreter.ValueAlg)

	for _, following := range node.Following {
		// Is later filled and evaluated once the correct operator has been applied
		var algResult interpreter.Value
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
		baseAlg = algResult.(interpreter.ValueAlg)
	}
	return baseAlg.(interpreter.Value), nil, nil
}
func (self *Interpreter) visitCastExpression(node CastExpression) (interpreter.Value, *int, *errors.Error) {
	base, code, err := self.visitUnaryExpression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If there is not typecast, only return the base value
	if node.Other == nil {
		return base, nil, nil
	}
	switch *node.Other {
	case interpreter.Number:
		switch base.Type() {
		case interpreter.Number:
			return base, nil, nil
		case interpreter.Boolean:
			numeric := 0.0
			if base.(interpreter.ValueBool).Value {
				numeric = 1.0
			}
			return interpreter.ValueNumber{Value: numeric}, nil, nil
		case interpreter.String:
			numeric, err := strconv.ParseFloat(base.(interpreter.ValueString).Value, 64)
			if err != nil {
				return nil, nil, errors.NewError(node.Base.Span, fmt.Sprintf("Cannot cast non-numeric string to number: %s", err.Error()), errors.ValueError)
			}
			return interpreter.ValueNumber{Value: numeric}, nil, nil
		default:
			return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot cast %v to %v", base.Type(), *node.Other), errors.TypeError)
		}
	case interpreter.String:
		display, err := base.Display(self.executor, node.Base.Span)
		if err != nil {
			return nil, nil, err
		}
		return interpreter.ValueString{Value: display}, nil, nil
	case interpreter.Boolean:
		isTrue, err := base.IsTrue(self.executor, node.Base.Span)
		if err != nil {
			return nil, nil, err
		}
		return interpreter.ValueBool{Value: isTrue}, nil, nil
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot cast to non-primitive type: cast from %v to %v is unsupported", base.Type(), *node.Other), errors.TypeError)
	}
}
func (self *Interpreter) visitUnaryExpression(node UnaryExpression) (interpreter.Value, *int, *errors.Error) {
	// If there is only a exp exression, return its value (recursion base case)
	if node.ExpExpression != nil {
		return self.visitEpxExpression(*node.ExpExpression)
	}
	unaryBase, code, err := self.visitUnaryExpression(node.UnaryExpression.UnaryExpression)
	if code != nil || err != nil {
		return nil, code, err
	}
	var unaryResult interpreter.Value
	var unaryErr *errors.Error
	switch node.UnaryExpression.UnaryOp {
	case UnaryOpPlus:
		unaryResult, unaryErr = interpreter.ValueNumber{Value: 0.0}.Sub(self.executor, node.UnaryExpression.UnaryExpression.Span, unaryBase)
	case UnaryOpMinus:
		unaryResult, unaryErr = interpreter.ValueNumber{Value: 0.0}.Add(self.executor, node.UnaryExpression.UnaryExpression.Span, unaryBase)
	case UnaryOpNot:
		unaryBaseIsTrueTemp, err := unaryBase.IsTrue(self.executor, node.UnaryExpression.UnaryExpression.Span)
		if err != nil {
			return nil, nil, err
		}
		return interpreter.ValueBool{Value: !unaryBaseIsTrueTemp}, nil, nil
	default:
		panic("BUG: a new unary operator has been added without updating this code")
	}
	if unaryErr != nil {
		return nil, nil, unaryErr
	}
	return unaryResult, nil, nil
}
func (self *Interpreter) visitEpxExpression(node ExpExpression) (interpreter.Value, *int, *errors.Error) {
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
	var powRes interpreter.Value
	var powErr *errors.Error
	switch base.Type() {
	case interpreter.Number:
		powRes, powErr = base.(interpreter.ValueNumber).Pow(self.executor, node.Span, power)
	case interpreter.BuiltinVariable:
		powRes, powErr = base.(interpreter.ValueBuiltinVariable).Pow(self.executor, node.Span, power)
	default:
		return nil, nil, errors.NewError(node.Span, fmt.Sprintf("Cannot perform power operation on type %v", base.Type()), errors.TypeError)
	}
	if powErr != nil {
		return nil, nil, powErr
	}
	return powRes, nil, nil
}
func (self *Interpreter) visitAssignExression(node AssignExpression) (interpreter.Value, *int, *errors.Error) {
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
	if base.Type() == interpreter.Object || base.Type() == interpreter.BuiltinFunction || base.Type() == interpreter.BuiltinVariable {
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
	if base.Type() != interpreter.String && base.Type() != interpreter.Number {
		return nil, nil, errors.NewError(node.Base.Span, fmt.Sprintf("Cannot use algebraic assignment operators on the %v type", base.Type()), errors.TypeError)
	}
	// Perform the more complex assignments
	var newValue interpreter.Value
	var assignErr *errors.Error
	switch node.Other.Operator {
	case OpAssign:
		panic("BUG: this case should have been handled above")
	case OpPlusAssign:
		newValue, assignErr = base.(interpreter.ValueAlg).Add(self.executor, node.Span, rhsValue)
	case OpMinusAssign:
		newValue, assignErr = base.(interpreter.ValueAlg).Sub(self.executor, node.Span, rhsValue)
	case OpMulAssign:
		newValue, assignErr = base.(interpreter.ValueAlg).Mul(self.executor, node.Span, rhsValue)
	case OpDivAssign:
		newValue, assignErr = base.(interpreter.ValueAlg).Div(self.executor, node.Span, rhsValue)
	case OpReminderAssign:
		newValue, assignErr = base.(interpreter.ValueAlg).Rem(self.executor, node.Span, rhsValue)
	case OpPowerAssign:
		newValue, assignErr = base.(interpreter.ValueAlg).Pow(self.executor, node.Span, rhsValue)
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
func (self *Interpreter) visitCallExpression(node CallExpression) (*interpreter.Value, *int, *errors.Error) {
	base, code, err := self.visitMemberExpression(node.Base)
	if code != nil || err != nil {
		return nil, code, err
	}
	// If there are no args and no parts, return the base here
	if len(node.Parts) == 0 {
		return base, nil, nil
	}

	// TODO: continue evaluating parts here

	return nil, nil, nil
}

func (self *Interpreter) visitMemberExpression(node MemberExpression) (*interpreter.Value, *int, *errors.Error) {
	panic("Not imlemented")
	return nil, nil, nil
}

func (self *Interpreter) visitAtom(node Atom) (*interpreter.Value, *int, *errors.Error) {
	panic("Not imlemented")
	return nil, nil, nil
}
