package interpreter

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

func (self *Interpreter) statement(node ast.AnalyzedStatement) *value.Interrupt {
	// Check for the cancelation signal
	if i := self.checkCancelation(node.Span()); i != nil {
		return i
	}

	switch node.Kind() {
	case ast.TypeDefinitionStatementKind:
		return nil // TODO: it is better to just filter them out during analysis?
	case ast.LetStatementKind:
		node := node.(ast.AnalyzedLetStatement)
		return self.letStatement(node)
	case ast.ReturnStatementKind:
		node := node.(ast.AnalyzedReturnStatement)
		returnValue := value.NewValueNull()
		if node.ReturnValue != nil {
			returnValueTemp, i := self.expression(node.ReturnValue)
			if i != nil {
				return i
			}
			returnValue = returnValueTemp
		}
		return value.NewReturnInterrupt(*returnValue)
	case ast.BreakStatementKind:
		return value.NewBreakInterrupt()
	case ast.ContinueStatementKind:
		return value.NewContinueInterrupt()
	case ast.LoopStatementKind:
		node := node.(ast.AnalyzedLoopStatement)
		return self.loopStatement(node)
	case ast.WhileStatementKind:
		node := node.(ast.AnalyzedWhileStatement)
		return self.whileStatement(node)
	case ast.ForStatementKind:
		node := node.(ast.AnalyzedForStatement)
		return self.forStatement(node)
	case ast.ExpressionStatementKind:
		node := node.(ast.AnalyzedExpressionStatement).Expression
		// ignore the expression value
		_, i := self.expression(node)
		return i
	default:
		panic(fmt.Sprintf("A new statement kind (%v) was added without updating this code", node.Kind()))
	}
}

func (self *Interpreter) letStatement(node ast.AnalyzedLetStatement) *value.Interrupt {
	rhsVal, i := self.expression(node.Expression)
	if i != nil {
		return i
	}

	// runtime validation: equal types, or cast from `any` to other -> check internal compatibilty
	if !node.NeedsRuntimeTypeValidation {
		self.addVar(node.Ident.Ident(), *rhsVal)
		return nil
	}

	if node.Expression.Type().Kind() != ast.AnyTypeKind && node.Expression.Type().Kind() != node.OptType.Kind() {
		return nil
	}

	// TODO: improve performance here (not so much deref)

	newValue, i := value.DeepCast(*rhsVal, node.OptType, node.Range)
	if i != nil {
		return i
	}

	// if i := self.valueIsCompatibleToType(*rhsVal, node.OptType, node.Range); i != nil {
	// 	return i
	// }

	// switch (*rhsVal).Kind() {
	// case value.ObjectValueKind:
	// 	switch node.OptType.Kind() {
	// 	case ast.AnyObjectTypeKind:
	// 		baseObj := (*rhsVal).(value.ValueObject)
	// 		*rhsVal = *baseObj.IntoAnyObject()
	// 	}
	// }

	self.addVar(node.Ident.Ident(), *newValue)

	return nil
}

func (self *Interpreter) loopStatement(node ast.AnalyzedLoopStatement) *value.Interrupt {
loop:
	for {
		_, i := self.block(node.Body, true)
		if i != nil {
			switch (*i).Kind() {
			case value.ContinueInterruptKind:
				continue loop
			case value.BreakInterruptKind:
				break loop
			default:
				return i
			}
		}
	}

	return nil
}

func (self *Interpreter) whileStatement(node ast.AnalyzedWhileStatement) *value.Interrupt {
loop:
	for {
		// analyze expression
		condition, i := self.expression(node.Condition)
		if i != nil {
			return i
		}

		// break if the condition is false
		if !(*condition).(value.ValueBool).Inner {
			break loop
		}

		_, i = self.block(node.Body, true)
		if i != nil {
			switch (*i).Kind() {
			case value.ContinueInterruptKind:
				continue loop
			case value.BreakInterruptKind:
				break loop
			default:
				return i
			}
		}
	}

	return nil
}

func (self *Interpreter) forStatement(node ast.AnalyzedForStatement) *value.Interrupt {
	iterVal, i := self.expression(node.IterExpression)
	if i != nil {
		return i
	}

	iterator := (*iterVal).IntoIter()

	// add a new scope for the loop
	self.pushScope()
	defer func() {
		self.popScope()
	}()

loop:
	for {
		// loop control
		currIterVar, shouldContinue := iterator()
		if !shouldContinue {
			break
		}

		// clear current scope
		self.clearScope()
		self.addVar(node.Identifier.Ident(), currIterVar)

		_, i := self.block(node.Body, false)
		if i != nil {
			switch (*i).Kind() {
			case value.ContinueInterruptKind:
				continue loop
			case value.BreakInterruptKind:
				break loop
			default:
				return i
			}
		}
	}

	return nil
}
