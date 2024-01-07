package ast

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type EitherStatementOrExpression struct {
	Statement  Statement
	Expression Expression
}

type Statement interface {
	Kind() StatementKind
	Span() errors.Span
	String() string
}

type StatementKind uint8

const (
	ImportStatementKind StatementKind = iota
	TypeDefinitionStatementKind
	LetStatementKind
	FnDefinitionStatementKind
	ReturnStatementKind
	BreakStatementKind
	ContinueStatementKind
	LoopStatementKind
	WhileStatementKind
	ForStatementKind
	ExpressionStatementKind
)

//
// Import statement
//

type ImportStatement struct {
	ToImport   []ImportStatementCandidate
	FromModule SpannedIdent
	Range      errors.Span
}

func (self ImportStatement) Kind() StatementKind { return ImportStatementKind }
func (self ImportStatement) Span() errors.Span   { return self.Range }
func (self ImportStatement) String() string {
	toImport := make([]string, 0)
	for _, imported := range self.ToImport {
		toImport = append(toImport, imported.String())
	}
	return fmt.Sprintf("import { %s } from %s;", strings.Join(toImport, ", "), self.FromModule)
}

type IMPORT_KIND uint8

const (
	// Normal means that the value of this identifier is imported
	IMPORT_KIND_NORMAL IMPORT_KIND = iota
	// A name reference type is brought into the current module's scope
	IMPORT_KIND_TYPE
	// A template is brought into the current module's template-scope
	IMPORT_KIND_TEMPLATE
)

type ImportStatementCandidate struct {
	Ident string
	Kind  IMPORT_KIND
	Span  errors.Span
}

func (self ImportStatementCandidate) String() string {
	modifierStr := ""

	switch self.Kind {
	case IMPORT_KIND_NORMAL:
		break
	case IMPORT_KIND_TYPE:
		modifierStr = "type "
	case IMPORT_KIND_TEMPLATE:
		modifierStr = "templ "
	}

	return fmt.Sprintf("%s%s", modifierStr, self.Ident)
}

//
// Type definition
//

type TypeDefinition struct {
	LhsIdent SpannedIdent
	RhsType  HmsType
	IsPub    bool
	Range    errors.Span
}

func (self TypeDefinition) Kind() StatementKind { return TypeDefinitionStatementKind }
func (self TypeDefinition) Span() errors.Span   { return self.Range }
func (self TypeDefinition) String() string {
	pub := ""
	if self.IsPub {
		pub = "pub "
	}
	return fmt.Sprintf("%stype %s = %s;", pub, self.LhsIdent, self.RhsType)
}

// Let statement
type LetStatement struct {
	Ident      SpannedIdent
	Expression Expression
	OptType    HmsType
	IsPub      bool
	Range      errors.Span
}

func (self LetStatement) Kind() StatementKind { return LetStatementKind }
func (self LetStatement) Span() errors.Span   { return self.Range }
func (self LetStatement) String() string {
	optType := ""
	if self.OptType != nil {
		optType = fmt.Sprintf(": %s", self.OptType)
	}

	return fmt.Sprintf("let %s%s = %s;", self.Ident, optType, self.Expression)
}

//
// Function definition
//

type FunctionModifier uint8

const (
	FN_MODIFIER_NONE FunctionModifier = iota
	FN_MODIFIER_PUB
	FN_MODIFIER_EVENT
)

func (self FunctionModifier) String() string {
	switch self {
	case FN_MODIFIER_NONE:
		return ""
	case FN_MODIFIER_PUB:
		return "pub"
	case FN_MODIFIER_EVENT:
		return "event"
	default:
		panic(fmt.Sprintf("Not implemented: %d", self))
	}
}

type FunctionDefinition struct {
	Ident      SpannedIdent
	Parameters []FnParam
	ParamSpan  errors.Span
	ReturnType HmsType
	Body       Block
	Modifier   FunctionModifier
	Range      errors.Span
}

func (self FunctionDefinition) Kind() StatementKind { return FnDefinitionStatementKind }
func (self FunctionDefinition) Span() errors.Span   { return self.Range }
func (self FunctionDefinition) String() string {
	params := make([]string, 0)
	for _, param := range self.Parameters {
		params = append(params, param.String())
	}

	modifier := ""

	switch self.Modifier {
	case FN_MODIFIER_PUB:
		modifier = "pub "
	case FN_MODIFIER_EVENT:
		modifier = "event"
	case FN_MODIFIER_NONE:
		break
	default:
		panic(fmt.Sprintf("Modifier %d is not implemented.", self.Modifier))
	}

	return fmt.Sprintf("%sfn %s(%s) -> %s %s", modifier, self.Ident, strings.Join(params, ", "), self.ReturnType, self.Body)
}

type FnParam struct {
	Ident SpannedIdent
	Type  HmsType
	Span  errors.Span
}

func (self FnParam) String() string {
	return fmt.Sprintf("%s: %s", self.Ident, self.Type)
}

//
// Return statement
//

type ReturnStatement struct {
	Expression Expression
	Range      errors.Span
}

func (self ReturnStatement) Kind() StatementKind { return ReturnStatementKind }
func (self ReturnStatement) Span() errors.Span   { return self.Range }
func (self ReturnStatement) String() string {
	returnValue := ""
	if self.Expression != nil {
		returnValue = fmt.Sprintf(" %s", self.Expression)
	}

	return fmt.Sprintf("return%s;", returnValue)
}

//
// Break statement
//

type BreakStatement struct {
	Range errors.Span
}

func (self BreakStatement) Kind() StatementKind { return BreakStatementKind }
func (self BreakStatement) Span() errors.Span   { return self.Range }
func (self BreakStatement) String() string      { return "break;" }

//
// Continue statement
//

type ContinueStatement struct {
	Range errors.Span
}

func (self ContinueStatement) Kind() StatementKind { return ContinueStatementKind }
func (self ContinueStatement) Span() errors.Span   { return self.Range }
func (self ContinueStatement) String() string      { return "continue;" }

//
// Loop statement
//

type LoopStatement struct {
	Body  Block
	Range errors.Span
}

func (self LoopStatement) Kind() StatementKind { return LoopStatementKind }
func (self LoopStatement) Span() errors.Span   { return self.Range }
func (self LoopStatement) String() string      { return fmt.Sprintf("loop %s", self.Body) }

//
// While statement
//

type WhileStatement struct {
	Condition Expression
	Body      Block
	Range     errors.Span
}

func (self WhileStatement) Kind() StatementKind { return WhileStatementKind }
func (self WhileStatement) Span() errors.Span   { return self.Range }
func (self WhileStatement) String() string {
	return fmt.Sprintf("while %s %s", self.Condition, self.Body)
}

//
// For statement
//

type ForStatement struct {
	Identifier     SpannedIdent
	IterExpression Expression
	Body           Block
	Range          errors.Span
}

func (self ForStatement) Kind() StatementKind { return ForStatementKind }
func (self ForStatement) Span() errors.Span   { return self.Range }
func (self ForStatement) String() string {
	return fmt.Sprintf("for %s in %s %s", self.Identifier, self.IterExpression, self.Body)
}

//
// Expression statement
//

type ExpressionStatement struct {
	Expression Expression
	Range      errors.Span
}

func (self ExpressionStatement) Kind() StatementKind { return ExpressionStatementKind }
func (self ExpressionStatement) Span() errors.Span   { return self.Range }
func (self ExpressionStatement) String() string      { return fmt.Sprintf("%s;", self.Expression.String()) }
