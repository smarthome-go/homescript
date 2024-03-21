package fuzzer

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

// This method is used for determining whether a tree node contains the `break` or `continue` keyword.
// If this is the case, loop obfuscations are not applied on this node.
func (self *Transformer) stmtCanControlLoop(node ast.AnalyzedStatement) bool {
	switch node.Kind() {
	case ast.TypeDefinitionStatementKind:
		return false
	case ast.LetStatementKind:
		node := node.(ast.AnalyzedLetStatement)
		return self.exprCanControlLoop(node.Expression)
	case ast.ReturnStatementKind:
		node := node.(ast.AnalyzedReturnStatement)
		return self.exprCanControlLoop(node.ReturnValue)
	case ast.BreakStatementKind, ast.ContinueStatementKind:
		// These statements are what we are looking for, so return `true`
		return true
	case ast.LoopStatementKind:
		// The body is irrelevant here, if it contains a loop control keyword,
		// it will be addressing this loop node
		return false
	case ast.WhileStatementKind:
		node := node.(ast.AnalyzedWhileStatement)
		// The body is irrelevant here, if it contains a loop control keyword,
		// it will be addressing this loop node
		// If there is a `break` in the loop control code however,
		// its interrupt will propagate until another loop is reached.
		// Therefore, this is the only factor that influences whether this node counts as a break
		return self.exprCanControlLoop(node.Condition)
	case ast.ForStatementKind:
		// The body is irrelevant here, if it contains a loop control keyword,
		// it will be addressing this loop node
		return false
	case ast.ExpressionStatementKind:
		node := node.(ast.AnalyzedExpressionStatement)
		return self.exprCanControlLoop(node.Expression)
	default:
		panic("A new statement kind was introduced without updating this code")
	}
}

// This method is used for determining whether a tree node contains the `break` or `continue` keyword.
// If this is the case, loop obfuscations are not applied on this node.
func (self *Transformer) exprCanControlLoop(node ast.AnalyzedExpression) bool {
	switch node.Kind() {
	case ast.UnknownExpressionKind:
		return false
	case ast.IntLiteralExpressionKind:
		return false
	case ast.FloatLiteralExpressionKind:
		return false
	case ast.BoolLiteralExpressionKind:
		return false
	case ast.StringLiteralExpressionKind:
		return false
	case ast.IdentExpressionKind:
		return false
	case ast.NullLiteralExpressionKind:
		return false
	case ast.NoneLiteralExpressionKind:
		return false
	case ast.RangeLiteralExpressionKind:
		return false
	case ast.ListLiteralExpressionKind:
		node := node.(ast.AnalyzedListLiteralExpression)
		for _, expr := range node.Values {
			if self.exprCanControlLoop(expr) {
				return true
			}
		}
		return false
	case ast.AnyObjectLiteralExpressionKind:
		return false
	case ast.ObjectLiteralExpressionKind:
		node := node.(ast.AnalyzedObjectLiteralExpression)
		for _, field := range node.Fields {
			if self.exprCanControlLoop(field.Expression) {
				return true
			}
		}
		return false
	case ast.FunctionLiteralExpressionKind:
		return false
	case ast.GroupedExpressionKind:
		node := node.(ast.AnalyzedGroupedExpression)
		return self.exprCanControlLoop(node.Inner)
	case ast.PrefixExpressionKind:
		node := node.(ast.AnalyzedPrefixExpression)
		return self.exprCanControlLoop(node.Base)
	case ast.InfixExpressionKind:
		node := node.(ast.AnalyzedInfixExpression)
		return self.exprCanControlLoop(node.Lhs) || self.exprCanControlLoop(node.Rhs)
	case ast.AssignExpressionKind:
		node := node.(ast.AnalyzedAssignExpression)
		return self.exprCanControlLoop(node.Lhs) || self.exprCanControlLoop(node.Rhs)
	case ast.CallExpressionKind:
		node := node.(ast.AnalyzedCallExpression)
		if self.exprCanControlLoop(node.Base) {
			return true
		}

		for _, arg := range node.Arguments.List {
			if self.exprCanControlLoop(arg.Expression) {
				return true
			}
		}

		return false
	case ast.IndexExpressionKind:
		node := node.(ast.AnalyzedIndexExpression)
		return self.exprCanControlLoop(node.Base) || self.exprCanControlLoop(node.Index)
	case ast.MemberExpressionKind:
		node := node.(ast.AnalyzedMemberExpression)
		return self.exprCanControlLoop(node.Base)
	case ast.CastExpressionKind:
		node := node.(ast.AnalyzedCastExpression)
		return self.exprCanControlLoop(node.Base)
	case ast.BlockExpressionKind:
		node := node.(ast.AnalyzedBlockExpression)
		return self.blockCanControlLoop(node.Block)
	case ast.IfExpressionKind:
		node := node.(ast.AnalyzedIfExpression)

		if self.exprCanControlLoop(node.Condition) {
			return true
		}

		if self.blockCanControlLoop(node.ThenBlock) {
			return true
		}

		if node.ElseBlock != nil {
			return self.blockCanControlLoop(*node.ElseBlock)
		}

		return false
	case ast.MatchExpressionKind:
		node := node.(ast.AnalyzedMatchExpression)
		if self.exprCanControlLoop(node.ControlExpression) {
			return true
		}

		for _, arm := range node.Arms {
			if self.exprCanControlLoop(arm.Action) {
				return true
			}
		}

		return false
	case ast.TryExpressionKind:
		node := node.(ast.AnalyzedTryExpression)

		if self.blockCanControlLoop(node.TryBlock) {
			return true
		}

		return self.blockCanControlLoop(node.CatchBlock)
	default:
		panic(fmt.Sprintf("A new expression kind was added without updating this code: %v", node))
	}
}

func (self *Transformer) blockCanControlLoop(node ast.AnalyzedBlock) bool {
	for _, stmt := range node.Statements {
		if self.stmtCanControlLoop(stmt) {
			return true
		}
	}

	if node.Expression != nil {
		return self.exprCanControlLoop(node.Expression)
	}

	return false
}
