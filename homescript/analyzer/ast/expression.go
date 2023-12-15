package ast

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
	"github.com/smarthome-go/homescript/v3/homescript/parser/util"
)

//
// Expression
//

type AnalyzedExpression interface {
	Kind() ExpressionKind
	Span() errors.Span
	String() string
	Type() Type
	Constant() bool
}

type ExpressionKind uint8

const (
	// without block
	UnknownExpressionKind ExpressionKind = iota
	IntLiteralExpressionKind
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
// Unknown expression
//

type UnknownExpression struct{}

func (self UnknownExpression) Kind() ExpressionKind { return UnknownExpressionKind }
func (self UnknownExpression) Span() errors.Span    { return errors.Span{} }
func (self UnknownExpression) String() string       { return "" }
func (self UnknownExpression) Type() Type           { return NewUnknownType() }
func (self UnknownExpression) Constant() bool       { return true }

//
// Int literal
//

type AnalyzedIntLiteralExpression struct {
	Value int64
	Range errors.Span
}

func (self AnalyzedIntLiteralExpression) Kind() ExpressionKind {
	return IntLiteralExpressionKind
}
func (self AnalyzedIntLiteralExpression) Span() errors.Span { return self.Range }
func (self AnalyzedIntLiteralExpression) String() string    { return fmt.Sprint(self.Value) }
func (self AnalyzedIntLiteralExpression) Type() Type        { return NewIntType(self.Range) }
func (self AnalyzedIntLiteralExpression) Constant() bool    { return true }

//
// Float literal
//

type AnalyzedFloatLiteralExpression struct {
	Value float64
	Range errors.Span
}

func (self AnalyzedFloatLiteralExpression) Kind() ExpressionKind {
	return FloatLiteralExpressionKind
}
func (self AnalyzedFloatLiteralExpression) Span() errors.Span { return self.Range }
func (self AnalyzedFloatLiteralExpression) String() string {
	// If the float can be replresented as an int without loss, the 'f' extension is forced.
	if float64(int64(self.Value)) == self.Value {
		return fmt.Sprintf("%df", int64(self.Value))
	}

	return fmt.Sprint(self.Value)
}
func (self AnalyzedFloatLiteralExpression) Type() Type     { return NewFloatType(self.Range) }
func (self AnalyzedFloatLiteralExpression) Constant() bool { return true }

//
// Bool literal
//

type AnalyzedBoolLiteralExpression struct {
	Value bool
	Range errors.Span
}

func (self AnalyzedBoolLiteralExpression) Kind() ExpressionKind {
	return BoolLiteralExpressionKind
}
func (self AnalyzedBoolLiteralExpression) Span() errors.Span { return self.Range }
func (self AnalyzedBoolLiteralExpression) String() string    { return fmt.Sprint(self.Value) }
func (self AnalyzedBoolLiteralExpression) Type() Type        { return NewBoolType(self.Range) }
func (self AnalyzedBoolLiteralExpression) Constant() bool    { return true }

//
// String literal
//

// TODO: add more escapes
func escapeHmsString(input string) string {
	output := input

	escapes := map[string]string{
		"\n": "\\n",
		"\"": "\\\"",
		"\t": "\\n",
	}

	for from, to := range escapes {
		output = strings.ReplaceAll(output, from, to)
	}

	return output

}

type AnalyzedStringLiteralExpression struct {
	Value string
	Range errors.Span
}

func (self AnalyzedStringLiteralExpression) Kind() ExpressionKind {
	return StringLiteralExpressionKind
}
func (self AnalyzedStringLiteralExpression) Span() errors.Span { return self.Range }
func (self AnalyzedStringLiteralExpression) String() string {
	return fmt.Sprintf("\"%s\"", escapeHmsString(self.Value))
}
func (self AnalyzedStringLiteralExpression) Type() Type     { return NewStringType(self.Range) }
func (self AnalyzedStringLiteralExpression) Constant() bool { return true }

//
// Ident expression
//

type AnalyzedIdentExpression struct {
	Ident      ast.SpannedIdent
	ResultType Type
	IsGlobal   bool
	IsFunction bool
}

func (self AnalyzedIdentExpression) Kind() ExpressionKind { return IdentExpressionKind }
func (self AnalyzedIdentExpression) Span() errors.Span    { return self.Ident.Span() }
func (self AnalyzedIdentExpression) String() string       { return self.Ident.Ident() }
func (self AnalyzedIdentExpression) Type() Type           { return self.ResultType }
func (self AnalyzedIdentExpression) Constant() bool       { return false }

//
// Null literal
//

type AnalyzedNullLiteralExpression struct{ Range errors.Span }

func (self AnalyzedNullLiteralExpression) Kind() ExpressionKind {
	return NullLiteralExpressionKind
}
func (self AnalyzedNullLiteralExpression) Span() errors.Span { return self.Range }
func (self AnalyzedNullLiteralExpression) String() string    { return "null" }
func (self AnalyzedNullLiteralExpression) Type() Type        { return NewNullType(self.Range) }
func (self AnalyzedNullLiteralExpression) Constant() bool    { return true }

//
// None literal
//

type AnalyzedNoneLiteralExpression struct{ Range errors.Span }

func (self AnalyzedNoneLiteralExpression) Kind() ExpressionKind {
	return NoneLiteralExpressionKind
}
func (self AnalyzedNoneLiteralExpression) Span() errors.Span { return self.Range }
func (self AnalyzedNoneLiteralExpression) String() string    { return "one" }
func (self AnalyzedNoneLiteralExpression) Type() Type {
	return NewOptionType(NewAnyType(self.Span()), self.Span())
}
func (self AnalyzedNoneLiteralExpression) Constant() bool { return true }

//
// Range literal
//

type AnalyzedRangeLiteralExpression struct {
	Start AnalyzedExpression
	End   AnalyzedExpression
	Range errors.Span
}

func (self AnalyzedRangeLiteralExpression) Kind() ExpressionKind { return RangeLiteralExpressionKind }
func (self AnalyzedRangeLiteralExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedRangeLiteralExpression) String() string {
	return fmt.Sprintf("%s..%s", self.Start, self.End)
}
func (self AnalyzedRangeLiteralExpression) Type() Type     { return NewRangeType(self.Range) }
func (self AnalyzedRangeLiteralExpression) Constant() bool { return true }

//
// List literal
//

type AnalyzedListLiteralExpression struct {
	Values   []AnalyzedExpression
	Range    errors.Span
	ListType Type
}

func (self AnalyzedListLiteralExpression) Kind() ExpressionKind { return ListLiteralExpressionKind }
func (self AnalyzedListLiteralExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedListLiteralExpression) String() string {
	inner := make([]string, 0)
	for _, value := range self.Values {
		inner = append(inner, value.String())
	}

	return fmt.Sprintf("[%s]", strings.Join(inner, ", "))
}
func (self AnalyzedListLiteralExpression) Type() Type {
	return NewListType(self.ListType, self.Range)
}
func (self AnalyzedListLiteralExpression) Constant() bool {
	for _, value := range self.Values {
		if !value.Constant() {
			return false
		}
	}
	return true
}

//
// Any object expression
//

type AnalyzedAnyObjectExpression struct {
	Range errors.Span
}

func (self AnalyzedAnyObjectExpression) Kind() ExpressionKind { return AnyObjectLiteralExpressionKind }

func (self AnalyzedAnyObjectExpression) Span() errors.Span { return self.Range }
func (self AnalyzedAnyObjectExpression) String() string    { return "{ ? }" }
func (self AnalyzedAnyObjectExpression) Type() Type        { return NewAnyObjectType(self.Range) }
func (self AnalyzedAnyObjectExpression) Constant() bool    { return true }

//
// Object literal
//

type AnalyzedObjectLiteralExpression struct {
	Fields []AnalyzedObjectLiteralField
	Range  errors.Span
}

func (self AnalyzedObjectLiteralExpression) Kind() ExpressionKind { return ObjectLiteralExpressionKind }
func (self AnalyzedObjectLiteralExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedObjectLiteralExpression) String() string {
	fields := make([]string, 0)
	for _, field := range self.Fields {
		fields = append(fields, strings.ReplaceAll(field.String(), "\n", "\n    "))
	}
	return fmt.Sprintf("new {\n    %s\n}", strings.Join(fields, ",\n    "))
}
func (self AnalyzedObjectLiteralExpression) Type() Type {
	fields := make([]ObjectTypeField, 0)

	for _, field := range self.Fields {
		fields = append(fields, ObjectTypeField{
			FieldName: field.Key,
			Type:      field.Expression.Type(),
			Span:      field.Range,
		})
	}

	return NewObjectType(fields, self.Span())
}
func (self AnalyzedObjectLiteralExpression) Constant() bool {
	for _, field := range self.Fields {
		if !field.Expression.Constant() {
			return false
		}
	}
	return true
}

type AnalyzedObjectLiteralField struct {
	Key        ast.SpannedIdent
	Expression AnalyzedExpression
	Range      errors.Span
}

func (self AnalyzedObjectLiteralField) String() string {
	var key string
	if !util.IsIdent(self.Key.Ident()) {
		key = fmt.Sprintf("\"%s\"", self.Key.Ident())
	} else {
		key = self.Key.Ident()
	}
	return fmt.Sprintf("%s: %s", key, self.Expression)
}

//
// Function literal
//

type AnalyzedFunctionLiteralExpression struct {
	Parameters []AnalyzedFnParam
	ParamSpan  errors.Span
	ReturnType Type
	Body       AnalyzedBlock
	Range      errors.Span
}

func (self AnalyzedFunctionLiteralExpression) Kind() ExpressionKind {
	return FunctionLiteralExpressionKind
}
func (self AnalyzedFunctionLiteralExpression) Span() errors.Span { return self.Range }
func (self AnalyzedFunctionLiteralExpression) String() string {
	params := make([]string, 0)
	for _, param := range self.Parameters {
		params = append(params, param.String())
	}
	return fmt.Sprintf("fn(%s) -> %s %s", strings.Join(params, ", "), self.ReturnType, self.Body)
}
func (self AnalyzedFunctionLiteralExpression) Type() Type {
	params := make([]FunctionTypeParam, 0)
	for _, param := range self.Parameters {
		params = append(params, NewFunctionTypeParam(param.Ident, param.Type))
	}
	return NewFunctionType(NewNormalFunctionTypeParamKind(params), self.ParamSpan, self.ReturnType, self.Range)
}

// If this was constant, it would open up a whole new category of bugs.
// Therefore, using a function literal as a global ist just forbidden.
func (self AnalyzedFunctionLiteralExpression) Constant() bool { return false }

//
// Grouped expression
//

type AnalyzedGroupedExpression struct {
	Inner AnalyzedExpression
	Range errors.Span
}

func (self AnalyzedGroupedExpression) Kind() ExpressionKind { return GroupedExpressionKind }
func (self AnalyzedGroupedExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedGroupedExpression) String() string       { return fmt.Sprintf("(%s)", self.Inner) }
func (self AnalyzedGroupedExpression) Type() Type           { return self.Inner.Type() }
func (self AnalyzedGroupedExpression) Constant() bool       { return self.Inner.Constant() }

//
// Prefix expression
//

type AnalyzedPrefixExpression struct {
	Operator   PrefixOperator
	Base       AnalyzedExpression
	ResultType Type
	Range      errors.Span
}

func (self AnalyzedPrefixExpression) Kind() ExpressionKind { return PrefixExpressionKind }
func (self AnalyzedPrefixExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedPrefixExpression) String() string {
	return fmt.Sprintf("%s%s", self.Operator, self.Base)
}
func (self AnalyzedPrefixExpression) Type() Type     { return self.ResultType }
func (self AnalyzedPrefixExpression) Constant() bool { return self.Base.Constant() }

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
// Infix expression
//

type AnalyzedInfixExpression struct {
	Lhs        AnalyzedExpression
	Rhs        AnalyzedExpression
	Operator   ast.InfixOperator
	ResultType Type
	Range      errors.Span
}

func (self AnalyzedInfixExpression) Kind() ExpressionKind { return InfixExpressionKind }
func (self AnalyzedInfixExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedInfixExpression) String() string {
	return fmt.Sprintf("%s %s %s", self.Lhs, self.Operator, self.Rhs)
}
func (self AnalyzedInfixExpression) Type() Type { return self.ResultType }
func (self AnalyzedInfixExpression) Constant() bool {
	return self.Lhs.Constant() && self.Rhs.Constant()
}

//
// Assign expression
//

type AnalyzedAssignExpression struct {
	Lhs        AnalyzedExpression
	Rhs        AnalyzedExpression
	Operator   ast.AssignOperator
	ResultType Type
	Range      errors.Span
}

func (self AnalyzedAssignExpression) Kind() ExpressionKind { return AssignExpressionKind }
func (self AnalyzedAssignExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedAssignExpression) String() string {
	return fmt.Sprintf("%s %s %s", self.Lhs, self.Operator, self.Rhs)
}
func (self AnalyzedAssignExpression) Type() Type     { return self.ResultType }
func (self AnalyzedAssignExpression) Constant() bool { return false }

//
// Call expression
//

type AnalyzedCallExpression struct {
	Base       AnalyzedExpression
	Arguments  []AnalyzedCallArgument
	ResultType Type
	Range      errors.Span
	IsSpawn    bool
	// Specifies whether the call is referring to a `real` function or a closure or other stuff
	IsNormalFunction bool
}

func (self AnalyzedCallExpression) Kind() ExpressionKind { return CallExpressionKind }
func (self AnalyzedCallExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedCallExpression) String() string {
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
func (self AnalyzedCallExpression) Type() Type     { return self.ResultType }
func (self AnalyzedCallExpression) Constant() bool { return false }

type AnalyzedCallArgument struct {
	Name       string
	Expression AnalyzedExpression
}

func (self AnalyzedCallArgument) String() string { return self.Expression.String() }

//
// Index expression
//

type AnalyzedIndexExpression struct {
	Base       AnalyzedExpression
	Index      AnalyzedExpression
	ResultType Type
	Range      errors.Span
}

func (self AnalyzedIndexExpression) Kind() ExpressionKind { return IndexExpressionKind }
func (self AnalyzedIndexExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedIndexExpression) String() string {
	return fmt.Sprintf("%s[%s]", self.Base, self.Index)
}
func (self AnalyzedIndexExpression) Type() Type     { return self.ResultType }
func (self AnalyzedIndexExpression) Constant() bool { return self.Base.Constant() }

//
// Member expression
//

type AnalyzedMemberExpression struct {
	Base       AnalyzedExpression
	Member     ast.SpannedIdent
	ResultType Type
	Range      errors.Span
}

func (self AnalyzedMemberExpression) Kind() ExpressionKind { return MemberExpressionKind }
func (self AnalyzedMemberExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedMemberExpression) String() string {
	return fmt.Sprintf("%s.%s", self.Base, self.Member.Ident())
}
func (self AnalyzedMemberExpression) Type() Type     { return self.ResultType }
func (self AnalyzedMemberExpression) Constant() bool { return self.Base.Constant() }

//
// Cast expression
//

type AnalyzedCastExpression struct {
	Base   AnalyzedExpression
	AsType Type
	Range  errors.Span
}

func (self AnalyzedCastExpression) Kind() ExpressionKind { return CastExpressionKind }
func (self AnalyzedCastExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedCastExpression) String() string {
	return fmt.Sprintf("%s as %s", self.Base, self.AsType)
}
func (self AnalyzedCastExpression) Type() Type     { return self.AsType }
func (self AnalyzedCastExpression) Constant() bool { return self.Base.Constant() }

//
// Block expression
//

type AnalyzedBlockExpression struct {
	Block AnalyzedBlock
}

func (self AnalyzedBlockExpression) Kind() ExpressionKind { return BlockExpressionKind }
func (self AnalyzedBlockExpression) Span() errors.Span    { return self.Block.Range }
func (self AnalyzedBlockExpression) String() string       { return self.Block.String() }
func (self AnalyzedBlockExpression) Type() Type           { return self.Block.Type() }
func (self AnalyzedBlockExpression) Constant() bool {
	if len(self.Block.Statements) > 0 {
		return false
	} else if self.Block.Expression != nil {
		return self.Block.Expression.Constant()
	}
	return false
}

//
// If expression
//

type AnalyzedIfExpression struct {
	Condition  AnalyzedExpression
	ThenBlock  AnalyzedBlock
	ElseBlock  *AnalyzedBlock
	ResultType Type
	Range      errors.Span
}

func (self AnalyzedIfExpression) Kind() ExpressionKind { return IfExpressionKind }
func (self AnalyzedIfExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedIfExpression) String() string {
	elseString := ""

	if self.ElseBlock != nil {
		elseString = fmt.Sprintf(" else %s", self.ElseBlock.String())
	}

	return fmt.Sprintf("if %s %s%s", self.Condition, self.ThenBlock, elseString)
}
func (self AnalyzedIfExpression) Type() Type     { return self.ResultType }
func (self AnalyzedIfExpression) Constant() bool { return false }

//
// Match expression
//

type AnalyzedMatchExpression struct {
	ControlExpression AnalyzedExpression
	Arms              []AnalyzedMatchArm
	DefaultArmAction  *AnalyzedExpression
	Range             errors.Span
	ResultType        Type
}

func (self AnalyzedMatchExpression) Kind() ExpressionKind { return MatchExpressionKind }
func (self AnalyzedMatchExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedMatchExpression) String() string {
	arms := make([]string, 0)
	for _, arm := range self.Arms {
		arms = append(arms, strings.ReplaceAll(arm.String(), "\n", "\n    "))
	}
	return fmt.Sprintf("match %s {\n    %s\n}", self.ControlExpression, strings.Join(arms, ",\n    "))
}
func (self AnalyzedMatchExpression) Type() Type     { return self.ResultType }
func (self AnalyzedMatchExpression) Constant() bool { return false }

type AnalyzedMatchArm struct {
	Literal AnalyzedExpression
	Action  AnalyzedExpression
}

func (self AnalyzedMatchArm) String() string {
	return fmt.Sprintf("%s => %s", self.Literal, self.Action)
}

//
// Try expression
//

type AnalyzedTryExpression struct {
	TryBlock   AnalyzedBlock
	CatchIdent ast.SpannedIdent
	CatchBlock AnalyzedBlock
	ResultType Type
	Range      errors.Span
}

func (self AnalyzedTryExpression) Kind() ExpressionKind { return TryExpressionKind }
func (self AnalyzedTryExpression) Span() errors.Span    { return self.Range }
func (self AnalyzedTryExpression) String() string {
	return fmt.Sprintf("try %s catch %s %s", self.TryBlock, self.CatchIdent, self.CatchBlock)
}
func (self AnalyzedTryExpression) Type() Type     { return self.ResultType }
func (self AnalyzedTryExpression) Constant() bool { return false }
