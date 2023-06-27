package ast

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

//
// Infix expression
//

type InfixExpression struct {
	Lhs      Expression
	Rhs      Expression
	Operator InfixOperator
	Range    errors.Span
}

func (self InfixExpression) Kind() ExpressionKind { return InfixExpressionKind }
func (self InfixExpression) Span() errors.Span    { return self.Range }
func (self InfixExpression) String() string {
	return fmt.Sprintf("%s %s %s", self.Lhs, self.Operator, self.Rhs)
}

//
// Infix operators
//

type InfixOperator uint8

const (
	PlusInfixOperator InfixOperator = iota
	MinusInfixOperator
	MultiplyInfixOperator
	DivideInfixOperator
	ModuloInfixOperator
	PowerInfixOperator
	ShiftLeftInfixOperator
	ShiftRightInfixOperator
	BitOrInfixOperator
	BitAndInfixOperator
	BitXorInfixOperator
	LogicalOrInfixOperator
	LogicalAndInfixOperator
	EqualInfixOperator
	NotEqualInfixOperator
	LessThanInfixOperator
	LessThanEqualInfixOperator
	GreaterThanInfixOperator
	GreaterThanEqualInfixOperator
)

func (self InfixOperator) String() string {
	switch self {
	case PlusInfixOperator:
		return "+"
	case MinusInfixOperator:
		return "-"
	case MultiplyInfixOperator:
		return "*"
	case DivideInfixOperator:
		return "/"
	case ModuloInfixOperator:
		return "%"
	case PowerInfixOperator:
		return "**"
	case ShiftLeftInfixOperator:
		return "<<"
	case ShiftRightInfixOperator:
		return ">>"
	case BitOrInfixOperator:
		return "|"
	case BitAndInfixOperator:
		return "&"
	case BitXorInfixOperator:
		return "^"
	case LogicalOrInfixOperator:
		return "||"
	case LogicalAndInfixOperator:
		return "&&"
	case EqualInfixOperator:
		return "=="
	case NotEqualInfixOperator:
		return "!="
	case LessThanInfixOperator:
		return "<"
	case LessThanEqualInfixOperator:
		return "<="
	case GreaterThanInfixOperator:
		return ">"
	case GreaterThanEqualInfixOperator:
		return ">="
	default:
		panic("A new infix-operator was added without updating this code")
	}
}
