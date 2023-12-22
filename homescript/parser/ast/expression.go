package ast

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/util"
)

type Expression interface {
	Kind() ExpressionKind
	Span() errors.Span
	String() string
}

type ExpressionKind uint8

const (
	// without block
	IntLiteralExpressionKind ExpressionKind = iota
	FloatLiteralExpressionKind
	BoolLiteralExpressionKind
	StringLiteralExpressionKind
	IdentExpressionKind
	NullLiteralExpressionKind
	NoneLiteralExpressionKind
	RangeLiteralExpressionKind
	ListLiteralExpressionKind
	AnyObjectLiteralExpressionKind
	ObjectLiteralExpressionKind
	FunctionLiteralExpressionKind
	GroupedExpressionKind
	PrefixExpressionKind
	InfixExpressionKind
	AssignExpressionKind
	CallExpressionKind
	IndexExpressionKind
	MemberExpressionKind
	CastExpressionKind
	// with block
	BlockExpressionKind
	IfExpressionKind
	MatchExpressionKind
	TryExpressionKind
)

//
// Int literal
//

type IntLiteralExpression struct {
	Value int64
	Range errors.Span
}

func (self IntLiteralExpression) Kind() ExpressionKind { return IntLiteralExpressionKind }
func (self IntLiteralExpression) Span() errors.Span    { return self.Range }
func (self IntLiteralExpression) String() string       { return fmt.Sprint(self.Value) }

//
// Float literal
//

type FloatLiteralExpression struct {
	Value float64
	Range errors.Span
}

func (self FloatLiteralExpression) Kind() ExpressionKind { return FloatLiteralExpressionKind }
func (self FloatLiteralExpression) Span() errors.Span    { return self.Range }
func (self FloatLiteralExpression) String() string {
	// If the float can be replresented as an int without loss, the 'f' extension is forced.
	if float64(int64(self.Value)) == self.Value {
		return fmt.Sprintf("%df", int64(self.Value))
	}

	return fmt.Sprint(self.Value)
}

//
// Bool literal
//

type BoolLiteralExpression struct {
	Value bool
	Range errors.Span
}

func (self BoolLiteralExpression) Kind() ExpressionKind { return BoolLiteralExpressionKind }
func (self BoolLiteralExpression) Span() errors.Span    { return self.Range }
func (self BoolLiteralExpression) String() string       { return fmt.Sprint(self.Value) }

//
// String literal
//

type StringLiteralExpression struct {
	Value string
	Range errors.Span
}

func (self StringLiteralExpression) Kind() ExpressionKind { return StringLiteralExpressionKind }
func (self StringLiteralExpression) Span() errors.Span    { return self.Range }
func (self StringLiteralExpression) String() string       { return fmt.Sprintf("\"%s\"", self.Value) }

//
// Ident expression
//

type IdentExpression struct {
	IsSingleton bool
	Ident       SpannedIdent
}

func (self IdentExpression) Kind() ExpressionKind { return IdentExpressionKind }
func (self IdentExpression) Span() errors.Span    { return self.Ident.span }
func (self IdentExpression) String() string       { return self.Ident.ident }

//
// Null literal
//

type NullLiteralExpression struct{ Range errors.Span }

func (self NullLiteralExpression) Kind() ExpressionKind { return NullLiteralExpressionKind }
func (self NullLiteralExpression) Span() errors.Span    { return self.Range }
func (self NullLiteralExpression) String() string       { return "null" }

//
// None literal
//

type NoneLiteralExpression struct{ Range errors.Span }

func (self NoneLiteralExpression) Kind() ExpressionKind { return NoneLiteralExpressionKind }
func (self NoneLiteralExpression) Span() errors.Span    { return self.Range }
func (self NoneLiteralExpression) String() string       { return "none" }

//
// Range literal
//

type RangeLiteralExpression struct {
	Start Expression
	End   Expression
	Range errors.Span
}

func (self RangeLiteralExpression) Kind() ExpressionKind { return RangeLiteralExpressionKind }
func (self RangeLiteralExpression) Span() errors.Span    { return self.Range }
func (self RangeLiteralExpression) String() string {
	return fmt.Sprintf("%s..%s", self.Start, self.End)
}

//
// List literal
//

type ListLiteralExpression struct {
	Values []Expression
	Range  errors.Span
}

func (self ListLiteralExpression) Kind() ExpressionKind { return ListLiteralExpressionKind }
func (self ListLiteralExpression) Span() errors.Span    { return self.Range }
func (self ListLiteralExpression) String() string {
	inner := make([]string, 0)
	for _, value := range self.Values {
		inner = append(inner, value.String())
	}

	return fmt.Sprintf("[%s]", strings.Join(inner, ", "))
}

//
// Any object literal
//

type AnyObjectLiteralExpression struct {
	Range errors.Span
}

func (self AnyObjectLiteralExpression) Kind() ExpressionKind { return AnyObjectLiteralExpressionKind }
func (self AnyObjectLiteralExpression) Span() errors.Span    { return self.Range }
func (self AnyObjectLiteralExpression) String() string       { return "{ ? }" }

//
// Object literal
//

type ObjectLiteralExpression struct {
	Fields []ObjectLiteralField
	Range  errors.Span
}

func (self ObjectLiteralExpression) Kind() ExpressionKind { return ObjectLiteralExpressionKind }
func (self ObjectLiteralExpression) Span() errors.Span    { return self.Range }
func (self ObjectLiteralExpression) String() string {
	fields := make([]string, 0)
	for _, field := range self.Fields {
		fields = append(fields, strings.ReplaceAll(field.String(), "\n", "\n    "))
	}
	return fmt.Sprintf("new {\n    %s\n}", strings.Join(fields, ",\n    "))
}

type ObjectLiteralField struct {
	Key        SpannedIdent
	Expression Expression
	Range      errors.Span
}

func (self ObjectLiteralField) String() string {
	var key string
	if !util.IsIdent(self.Key.ident) {
		key = fmt.Sprintf("\"%s\"", self.Key.ident)
	} else {
		key = self.Key.ident
	}
	return fmt.Sprintf("%s: %s", key, self.Expression)
}

//
// Function literal
//

type FunctionLiteralExpression struct {
	Parameters []FnParam
	ParamSpan  errors.Span
	ReturnType HmsType
	Body       Block
	Range      errors.Span
}

func (self FunctionLiteralExpression) Kind() ExpressionKind { return FunctionLiteralExpressionKind }
func (self FunctionLiteralExpression) Span() errors.Span    { return self.Range }
func (self FunctionLiteralExpression) String() string {
	params := make([]string, 0)
	for _, param := range self.Parameters {
		params = append(params, param.String())
	}
	return fmt.Sprintf("fn(%s) -> %s %s", strings.Join(params, ", "), self.ReturnType, self.Body)
}

//
// Grouped expression
//

type GroupedExpression struct {
	Inner Expression
	Range errors.Span
}

func (self GroupedExpression) Kind() ExpressionKind { return GroupedExpressionKind }
func (self GroupedExpression) Span() errors.Span    { return self.Range }
func (self GroupedExpression) String() string       { return fmt.Sprintf("(%s)", self.Inner) }

//
// Prefix expression
//

type PrefixExpression struct {
	Operator PrefixOperator
	Base     Expression
	Range    errors.Span
}

func (self PrefixExpression) Kind() ExpressionKind { return PrefixExpressionKind }
func (self PrefixExpression) Span() errors.Span    { return self.Range }
func (self PrefixExpression) String() string       { return fmt.Sprintf("%s%s", self.Operator, self.Base) }

type PrefixOperator uint8

const (
	MinusPrefixOperator PrefixOperator = iota
	NegatePrefixOperator
	IntoSomePrefixOperator
)

func (self PrefixOperator) String() string {
	switch self {
	case MinusPrefixOperator:
		return "-"
	case NegatePrefixOperator:
		return "!"
	case IntoSomePrefixOperator:
		return "?"
	default:
		panic("A new prefix-operator was added without updating this code")
	}
}

//
// Infix expression (NOTE: refer to `infix_expr.go`)
//

//
// Assign expression (NOTE: refer to `assign_expr.go`)
//

//
// Call expression
//

type CallExpression struct {
	Base      Expression
	Arguments []Expression
	Range     errors.Span
	IsSpawn   bool
}

func (self CallExpression) Kind() ExpressionKind { return CallExpressionKind }
func (self CallExpression) Span() errors.Span    { return self.Range }
func (self CallExpression) String() string {
	args := make([]string, 0)
	for _, arg := range self.Arguments {
		args = append(args, arg.String())
	}

	spawnPrefix := ""
	if self.IsSpawn {
		spawnPrefix = "spawn "
	}

	return fmt.Sprintf("%s%s(%s)", spawnPrefix, self.Base, strings.Join(args, ", "))
}

//
// Index expression
//

type IndexExpression struct {
	Base  Expression
	Index Expression
	Range errors.Span
}

func (self IndexExpression) Kind() ExpressionKind { return IndexExpressionKind }
func (self IndexExpression) Span() errors.Span    { return self.Range }
func (self IndexExpression) String() string {
	return fmt.Sprintf("%s[%s]", self.Base, self.Index)
}

//
// Member expression
//

type MemberExpression struct {
	Base   Expression
	Member SpannedIdent
	Range  errors.Span
}

func (self MemberExpression) Kind() ExpressionKind { return MemberExpressionKind }
func (self MemberExpression) Span() errors.Span    { return self.Range }
func (self MemberExpression) String() string {
	return fmt.Sprintf("%s.%s", self.Base, self.Member.ident)
}

//
// Cast expression
//

type CastExpression struct {
	Base   Expression
	AsType HmsType
	Range  errors.Span
}

func (self CastExpression) Kind() ExpressionKind { return CastExpressionKind }
func (self CastExpression) Span() errors.Span    { return self.Range }
func (self CastExpression) String() string {
	return fmt.Sprintf("%s as %s", self.Base, self.AsType)
}

//
// Block expression
//

type BlockExpression struct {
	Block Block
}

func (self BlockExpression) Kind() ExpressionKind { return BlockExpressionKind }
func (self BlockExpression) Span() errors.Span    { return self.Block.Range }
func (self BlockExpression) String() string       { return self.Block.String() }

//
// If expression
//

type IfExpression struct {
	Condition Expression
	ThenBlock Block
	ElseBlock *Block
	Range     errors.Span
}

func (self IfExpression) Kind() ExpressionKind { return IfExpressionKind }
func (self IfExpression) Span() errors.Span    { return self.Range }
func (self IfExpression) String() string {
	elseString := ""

	if self.ElseBlock != nil {
		elseString = fmt.Sprintf(" else %s", self.ElseBlock.String())
	}

	return fmt.Sprintf("if %s %s%s", self.Condition, self.ThenBlock, elseString)
}

//
// Match expression
//

type MatchExpression struct {
	ControlExpression Expression
	Arms              []MatchArm
	Range             errors.Span
}

func (self MatchExpression) Kind() ExpressionKind { return MatchExpressionKind }
func (self MatchExpression) Span() errors.Span    { return self.Range }
func (self MatchExpression) String() string {
	arms := make([]string, 0)
	for _, arm := range self.Arms {
		arms = append(arms, strings.ReplaceAll(arm.String(), "\n", "\n    "))
	}
	return fmt.Sprintf("match %s {\n    %s\n}", self.ControlExpression, strings.Join(arms, ",\n    "))
}

type MatchArm struct {
	Literal DefaultOrLiteral
	Action  Expression
	Range   errors.Span
}

func (self MatchArm) String() string {
	return fmt.Sprintf("%s => %s", self.Literal, self.Action)
}

type DefaultOrLiteral struct {
	Literal Expression
}

func (self DefaultOrLiteral) String() string {
	if self.IsLiteral() {
		return self.Literal.String()
	}
	return "_"
}

func (self DefaultOrLiteral) IsLiteral() bool {
	return self.Literal != nil
}

func NewDefaultOrLiteralDefault() DefaultOrLiteral { return DefaultOrLiteral{Literal: nil} }
func NewDefaultOrLiteralLiteral(literal Expression) DefaultOrLiteral {
	return DefaultOrLiteral{Literal: literal}
}

//
// Try expression
//

type TryExpression struct {
	TryBlock   Block
	CatchIdent SpannedIdent
	CatchBlock Block
	Range      errors.Span
}

func (self TryExpression) Kind() ExpressionKind { return TryExpressionKind }
func (self TryExpression) Span() errors.Span    { return self.Range }
func (self TryExpression) String() string {
	return fmt.Sprintf("try %s catch %s %s", self.TryBlock, self.CatchIdent, self.CatchBlock)
}
