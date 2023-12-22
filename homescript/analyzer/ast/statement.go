package ast

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

type AnalyzedStatement interface {
	Kind() AnalyzedStatementKind
	Span() errors.Span
	String() string
	Type() Type
}

type AnalyzedStatementKind uint8

const (
	TypeDefinitionStatementKind AnalyzedStatementKind = iota
	SingletonTypeDefinitionStatementKind
	LetStatementKind
	ReturnStatementKind
	BreakStatementKind
	ContinueStatementKind
	LoopStatementKind
	WhileStatementKind
	ForStatementKind
	ExpressionStatementKind
)

//
// Singleton (Type definition)
//

type AnalyzedSingletonTypeDefinition struct {
	Ident   ast.SpannedIdent
	TypeDef AnalyzedTypeDefinition
	Range   errors.Span
}

func (self AnalyzedSingletonTypeDefinition) Kind() AnalyzedStatementKind {
	return SingletonTypeDefinitionStatementKind
}
func (self AnalyzedSingletonTypeDefinition) Span() errors.Span { return self.Range }
func (self AnalyzedSingletonTypeDefinition) String() string {
	return fmt.Sprintf("%s\n%s", self.Ident.Ident(), self.TypeDef)
}
func (self AnalyzedSingletonTypeDefinition) Type() Type { return NewNullType(self.Range) }

//
// Type definition
//

type AnalyzedTypeDefinition struct {
	LhsIdent string
	RhsType  Type
	Range    errors.Span
}

func (self AnalyzedTypeDefinition) Kind() AnalyzedStatementKind { return TypeDefinitionStatementKind }
func (self AnalyzedTypeDefinition) Span() errors.Span           { return self.Range }
func (self AnalyzedTypeDefinition) String() string {
	return fmt.Sprintf("type %s = %s;", self.LhsIdent, self.RhsType)
}
func (self AnalyzedTypeDefinition) Type() Type { return NewNullType(self.Range) }

// Let statement
type AnalyzedLetStatement struct {
	Ident                      ast.SpannedIdent
	Expression                 AnalyzedExpression
	VarType                    Type
	NeedsRuntimeTypeValidation bool // is set to `true` if the rhs is of type `any`
	OptType                    Type
	Range                      errors.Span
}

func (self AnalyzedLetStatement) Kind() AnalyzedStatementKind { return LetStatementKind }
func (self AnalyzedLetStatement) Span() errors.Span           { return self.Range }
func (self AnalyzedLetStatement) String() string {
	return fmt.Sprintf("let %s: %s = %s;", self.Ident, self.VarType, self.Expression)
}
func (self AnalyzedLetStatement) Type() Type { return NewNullType(self.Range) }

//
// Return statement
//

type AnalyzedReturnStatement struct {
	ReturnValue AnalyzedExpression
	Range       errors.Span
}

func (self AnalyzedReturnStatement) Kind() AnalyzedStatementKind { return ReturnStatementKind }
func (self AnalyzedReturnStatement) Span() errors.Span           { return self.Range }
func (self AnalyzedReturnStatement) String() string {
	returnValue := ""
	if self.ReturnValue != nil {
		returnValue = fmt.Sprintf(" %s", self.ReturnValue)
	}

	return fmt.Sprintf("return%s;", returnValue)
}
func (self AnalyzedReturnStatement) Type() Type { return NewNeverType() }

//
// Break statement
//

type AnalyzedBreakStatement struct {
	Range errors.Span
}

func (self AnalyzedBreakStatement) Kind() AnalyzedStatementKind { return BreakStatementKind }
func (self AnalyzedBreakStatement) Span() errors.Span           { return self.Range }
func (self AnalyzedBreakStatement) String() string              { return "break;" }
func (self AnalyzedBreakStatement) Type() Type                  { return NewNeverType() }

//
// Continue statement
//

type AnalyzedContinueStatement struct {
	Range errors.Span
}

func (self AnalyzedContinueStatement) Kind() AnalyzedStatementKind { return ContinueStatementKind }
func (self AnalyzedContinueStatement) Span() errors.Span           { return self.Range }
func (self AnalyzedContinueStatement) String() string              { return "continue;" }
func (self AnalyzedContinueStatement) Type() Type                  { return NewNeverType() }

//
// Loop statement
//

type AnalyzedLoopStatement struct {
	Body            AnalyzedBlock
	NeverTerminates bool
	Range           errors.Span
}

func (self AnalyzedLoopStatement) Kind() AnalyzedStatementKind { return LoopStatementKind }
func (self AnalyzedLoopStatement) Span() errors.Span           { return self.Range }
func (self AnalyzedLoopStatement) String() string              { return fmt.Sprintf("loop %s", self.Body) }
func (self AnalyzedLoopStatement) Type() Type {
	if self.NeverTerminates {
		return NewNeverType()
	}
	return NewNullType(self.Range)
}

//
// While statement
//

type AnalyzedWhileStatement struct {
	Condition       AnalyzedExpression
	Body            AnalyzedBlock
	NeverTerminates bool
	Range           errors.Span
}

func (self AnalyzedWhileStatement) Kind() AnalyzedStatementKind { return WhileStatementKind }
func (self AnalyzedWhileStatement) Span() errors.Span           { return self.Range }
func (self AnalyzedWhileStatement) String() string {
	return fmt.Sprintf("while %s %s", self.Condition, self.Body)
}
func (self AnalyzedWhileStatement) Type() Type {
	if self.NeverTerminates {
		return NewNeverType()
	}
	return NewNullType(self.Range)
}

//
// For statement
//

type AnalyzedForStatement struct {
	Identifier      ast.SpannedIdent
	IterExpression  AnalyzedExpression
	IterVarType     Type
	Body            AnalyzedBlock
	NeverTerminates bool
	Range           errors.Span
}

func (self AnalyzedForStatement) Kind() AnalyzedStatementKind { return ForStatementKind }
func (self AnalyzedForStatement) Span() errors.Span           { return self.Range }
func (self AnalyzedForStatement) String() string {
	return fmt.Sprintf("for %s in %s %s", self.Identifier, self.IterExpression, self.Body)
}
func (self AnalyzedForStatement) Type() Type {
	if self.NeverTerminates {
		return NewNeverType()
	}
	return NewNullType(self.Range)
}

//
// Expression statement
//

type AnalyzedExpressionStatement struct {
	Expression AnalyzedExpression
	Range      errors.Span
}

func (self AnalyzedExpressionStatement) Kind() AnalyzedStatementKind { return ExpressionStatementKind }
func (self AnalyzedExpressionStatement) Span() errors.Span           { return self.Range }
func (self AnalyzedExpressionStatement) String() string {
	return fmt.Sprintf("%s;", self.Expression.String())
}
func (self AnalyzedExpressionStatement) Type() Type { return self.Expression.Type() }
