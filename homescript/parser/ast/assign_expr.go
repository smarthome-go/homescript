package ast

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

//
// Assign expression
//

type AssignExpression struct {
	Lhs            Expression
	AssignOperator AssignOperator
	Rhs            Expression
	Range          errors.Span
}

func (self AssignExpression) Kind() ExpressionKind { return AssignExpressionKind }
func (self AssignExpression) Span() errors.Span    { return self.Range }
func (self AssignExpression) String() string {
	return fmt.Sprintf("%s %s %s", self.Lhs, self.AssignOperator, self.Rhs)
}

//
// Assign operators
//

type AssignOperator uint8

const (
	StdAssignOperatorKind AssignOperator = iota
	PlusAssignOperatorKind
	MinusAssignOperatorKind
	MultiplyAssignOperatorKind
	DivideAssignOperatorKind
	ModuloAssignOperatorKind
	PowerAssignOperatorKind
	ShiftLeftAssignOperatorKind
	ShiftRightAssignOperatorKind
	BitOrAssignOperatorKind
	BitAndAssignOperatorKind
	BitXorAssignOperatorKind
)

func (self AssignOperator) String() string {
	switch self {
	case StdAssignOperatorKind:
		return "="
	case PlusAssignOperatorKind:
		return "+="
	case MinusAssignOperatorKind:
		return "-="
	case MultiplyAssignOperatorKind:
		return "*="
	case DivideAssignOperatorKind:
		return "/="
	case ModuloAssignOperatorKind:
		return "%="
	case PowerAssignOperatorKind:
		return "**="
	case ShiftLeftAssignOperatorKind:
		return "<<="
	case ShiftRightAssignOperatorKind:
		return ">>="
	case BitOrAssignOperatorKind:
		return "|="
	case BitAndAssignOperatorKind:
		return "&="
	case BitXorAssignOperatorKind:
		return "^="
	default:
		panic("A new assign-operator was introduced without updating this code")
	}
}

func (self AssignOperator) IntoInfixOperator() InfixOperator {
	switch self {
	case PlusAssignOperatorKind:
		return PlusInfixOperator
	case MinusAssignOperatorKind:
		return MinusInfixOperator
	case MultiplyAssignOperatorKind:
		return MultiplyInfixOperator
	case DivideAssignOperatorKind:
		return DivideInfixOperator
	case ModuloAssignOperatorKind:
		return ModuloInfixOperator
	case PowerAssignOperatorKind:
		return PowerInfixOperator
	case ShiftLeftAssignOperatorKind:
		return ShiftLeftInfixOperator
	case ShiftRightAssignOperatorKind:
		return ShiftRightInfixOperator
	case BitOrAssignOperatorKind:
		return BitOrInfixOperator
	case BitAndAssignOperatorKind:
		return BitAndInfixOperator
	case BitXorAssignOperatorKind:
		return BitXorInfixOperator
	default:
		panic("Not supported")
	}
}
