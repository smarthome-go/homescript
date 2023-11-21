package parser

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

type Token struct {
	Kind  TokenKind
	Value string
	Span  errors.Span
}

type TokenKind uint8

const (
	Unknown TokenKind = iota
	EOF

	QuestionMark // ?
	Underscore   // _
	Semicolon    // ;
	Comma        // ,
	Colon        // :
	Dot          // .
	DoubleDot    // ..
	Arrow        // ->
	FatArrow     // =>

	LParen   // (
	RParen   // )
	LCurly   // {
	RCurly   // }
	LBracket // [
	RBracket // ]

	Or               // ||
	And              // &&
	Equal            // ==
	NotEqual         // !=
	LessThan         // <
	LessThanEqual    // <=
	GreaterThan      // >
	GreaterThanEqual // >=
	Not              // !

	Plus       // +
	Minus      // -
	Multiply   // *
	Divide     // /
	Modulo     // %
	Power      // **
	ShiftLeft  // <<
	ShiftRight // >>
	BitOr      // |
	BitAnd     // &
	BitXor     // ^

	Assign           // =
	PlusAssign       // +=
	MinusAssign      // -=
	MultiplyAssign   // *=
	DivideAssign     // /=
	PowerAssign      // **=
	ModuloAssign     // %=
	ShiftLeftAssign  // <<=
	ShiftRightAssign // >>=
	BitOrAssign      // |=
	BitAndAssign     // &=
	BitXorAssign     // ^=

	Import   // import
	As       // as
	From     // from
	Try      // try
	Catch    // catch
	In       // in
	Let      // let
	Pub      // pub
	Fn       // fn
	If       // if
	Else     // else
	Match    // match
	For      // for
	While    // while
	Loop     // loop
	Break    // break
	Continue // continue
	Return   // return
	Type     // type
	New      // new
	Spawn    // spawn
	Event    // event

	True  // true
	False // false
	None  // none
	Null  // null

	String     // "foo" (token includes quotes whilst content excludes them)
	Int        // 42
	Float      // 3.1415
	Identifier // foobar
)

func newToken(kind TokenKind, value string, span errors.Span) Token {
	return Token{
		Kind:  kind,
		Value: value,
		Span:  span,
	}
}

func unknownToken(location errors.Location) Token {
	return newToken(Unknown, "", errors.Span{Start: location, End: location})
}

func (self TokenKind) String() string {
	var display string
	switch self {
	case Unknown:
		display = "unknown"
	case EOF:
		display = "EOF"
	case Semicolon:
		display = ";"
	case Colon:
		display = ":"
	case Comma:
		display = ","
	case Dot:
		display = "."
	case DoubleDot:
		display = ".."
	case Arrow:
		display = "->"
	case FatArrow:
		display = "=>"
	case LParen:
		display = "("
	case RParen:
		display = ")"
	case LCurly:
		display = "{"
	case RCurly:
		display = "}"
	case LBracket:
		display = "["
	case RBracket:
		display = "]"
	case Or:
		display = "||"
	case And:
		display = "&&"
	case Equal:
		display = "=="
	case NotEqual:
		display = "!="
	case LessThan:
		display = "<"
	case LessThanEqual:
		display = "<="
	case GreaterThan:
		display = ">"
	case GreaterThanEqual:
		display = ">="
	case Not:
		display = "!"
	case Plus:
		display = "+"
	case Minus:
		display = "-"
	case Multiply:
		display = "*"
	case Divide:
		display = "/"
	case Modulo:
		display = "%"
	case Power:
		display = "**"
	case Assign:
		display = "="
	case PlusAssign:
		display = "+="
	case MinusAssign:
		display = "-="
	case MultiplyAssign:
		display = "*="
	case DivideAssign:
		display = "/="
	case PowerAssign:
		display = "**="
	case ModuloAssign:
		display = "%="
	case Pub:
		display = "pub"
	case Fn:
		display = "fn"
	case If:
		display = "if"
	case Else:
		display = "else"
	case Match:
		display = "match"
	case For:
		display = "for"
	case While:
		display = "while"
	case Loop:
		display = "loop"
	case Break:
		display = "break"
	case Continue:
		display = "continue"
	case Return:
		display = "return"
	case Type:
		display = "type"
	case True:
		display = "true"
	case False:
		display = "false"
	case Null:
		display = "null"
	case None:
		display = "none"
	case String:
		display = "string"
	case Int:
		display = "int"
	case Float:
		display = "float"
	case Identifier:
		display = "identifier"
	case Let:
		display = "let"
	case Import:
		display = "import"
	case As:
		display = "as"
	case From:
		display = "from"
	case In:
		display = "in"
	case Try:
		display = "try"
	case Catch:
		display = "catch"
	case New:
		display = "new"
	case Spawn:
		display = "spawn"
	case Event:
		display = "event"
	case BitOr:
		display = "|"
	case BitXor:
		display = "^"
	case ShiftLeft:
		display = "<<"
	case ShiftRight:
		display = ">>"
	case ShiftLeftAssign:
		display = "<<="
	case ShiftRightAssign:
		display = ">>="
	case BitOrAssign:
		display = "|="
	case BitAndAssign:
		display = "&="
	case BitXorAssign:
		display = "^="
	case QuestionMark:
		display = "?"
	case Underscore:
		display = "_"
	default:
		panic("A new token was introduced without updating this code")
	}
	return display
}

func (self TokenKind) prec() (left uint8, right uint8) {
	switch self {
	case Assign, PlusAssign, MinusAssign, MultiplyAssign,
		DivideAssign, ModuloAssign, PowerAssign,
		ShiftLeftAssign, ShiftRightAssign,
		BitOrAssign, BitAndAssign, BitXorAssign:
		return 1, 2
	case Or:
		return 3, 4
	case And:
		return 5, 6
	case BitOr:
		return 7, 8
	case BitXor:
		return 9, 10
	case BitAnd:
		return 11, 12
	case Equal, NotEqual:
		return 13, 14
	case LessThan, GreaterThan, LessThanEqual, GreaterThanEqual:
		return 15, 16
	case ShiftLeft, ShiftRight:
		return 17, 18
	case Plus, Minus:
		return 19, 20
	case Multiply, Divide, Modulo:
		return 21, 22
	case As:
		return 23, 24
	case Power:
		// inverse order for right-associativity
		return 26, 25
	case DoubleDot:
		return 27, 28
	case LParen, LBracket:
		return 30, 31
	case Dot:
		// inverse order for right-associativity
		return 33, 32
	default:
		return 0, 0
	}
}

func (self TokenKind) asInfixOperator() ast.InfixOperator {
	switch self {
	case Plus:
		return ast.PlusInfixOperator
	case Minus:
		return ast.MinusInfixOperator
	case Multiply:
		return ast.MultiplyInfixOperator
	case Divide:
		return ast.DivideInfixOperator
	case Modulo:
		return ast.ModuloInfixOperator
	case Power:
		return ast.PowerInfixOperator
	case ShiftLeft:
		return ast.ShiftLeftInfixOperator
	case ShiftRight:
		return ast.ShiftRightInfixOperator
	case BitOr:
		return ast.BitOrInfixOperator
	case BitAnd:
		return ast.BitAndInfixOperator
	case BitXor:
		return ast.BitXorInfixOperator
	case Or:
		return ast.LogicalOrInfixOperator
	case And:
		return ast.LogicalAndInfixOperator
	case Equal:
		return ast.EqualInfixOperator
	case NotEqual:
		return ast.NotEqualInfixOperator
	case LessThan:
		return ast.LessThanInfixOperator
	case LessThanEqual:
		return ast.LessThanEqualInfixOperator
	case GreaterThan:
		return ast.GreaterThanInfixOperator
	case GreaterThanEqual:
		return ast.GreaterThanEqualInfixOperator
	default:
		panic(fmt.Sprintf("Unreachable: this method was called on an unsupported token `%s`", self))
	}
}

func (self TokenKind) asPrefixOperator() ast.PrefixOperator {
	switch self {
	case Minus:
		return ast.MinusPrefixOperator
	case Not:
		return ast.NegatePrefixOperator
	case QuestionMark:
		return ast.IntoSomePrefixOperator
	default:
		panic(fmt.Sprintf("Unreachable: this method was called on an unsupported token `%s`", self))
	}
}

func (self TokenKind) asAssignOperator() ast.AssignOperator {
	switch self {
	case Assign:
		return ast.StdAssignOperatorKind
	case PlusAssign:
		return ast.PlusAssignOperatorKind
	case MinusAssign:
		return ast.MinusAssignOperatorKind
	case MultiplyAssign:
		return ast.MultiplyAssignOperatorKind
	case DivideAssign:
		return ast.DivideAssignOperatorKind
	case ModuloAssign:
		return ast.ModuloAssignOperatorKind
	case PowerAssign:
		return ast.PowerAssignOperatorKind
	case ShiftLeftAssign:
		return ast.ShiftLeftAssignOperatorKind
	case ShiftRightAssign:
		return ast.ShiftRightAssignOperatorKind
	case BitOrAssign:
		return ast.BitOrAssignOperatorKind
	case BitAndAssign:
		return ast.BitAndAssignOperatorKind
	case BitXorAssign:
		return ast.BitXorAssignOperatorKind
	default:
		panic(fmt.Sprintf("Unreachable: this method was called on an unsupported token `%s`", self))
	}
}
