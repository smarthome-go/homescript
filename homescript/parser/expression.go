package parser

import (
	"fmt"
	"strconv"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
//	Expression
//

func (self *Parser) expression(prec uint8) (ast.Expression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	var lhs ast.Expression
	switch self.CurrentToken.Kind {
	case Int, Float:
		num, err := self.intFloatLiteral()
		if err != nil {
			return nil, err
		}
		lhs = num
	case True, False:
		if err := self.next(); err != nil {
			return nil, err
		}
		lhs = ast.BoolLiteralExpression{
			Value: self.PreviousToken.Kind == True,
			Range: self.PreviousToken.Span,
		}
	case String:
		if err := self.next(); err != nil {
			return nil, err
		}
		lhs = ast.StringLiteralExpression{
			Value: self.PreviousToken.Value,
			Range: self.PreviousToken.Span,
		}
	case Identifier:
		ident, err := self.identExpression()
		if err != nil {
			return nil, err
		}
		lhs = ident
	case Null:
		null, err := self.nullLiteral()
		if err != nil {
			return nil, err
		}
		lhs = null
	case None:
		null, err := self.noneLiteral()
		if err != nil {
			return nil, err
		}
		lhs = null
	case LBracket:
		listLiteral, err := self.listLiteral()
		if err != nil {
			return nil, err
		}
		lhs = listLiteral
	case New:
		objectLiteral, err := self.objectLiteral()
		if err != nil {
			return nil, err
		}
		lhs = objectLiteral
	case Fn:
		fnLiteral, err := self.functionLiteral()
		if err != nil {
			return nil, err
		}
		lhs = fnLiteral
	case LParen:
		grouped, err := self.groupedExpression()
		if err != nil {
			return nil, err
		}
		lhs = grouped
	case Not, Minus, QuestionMark:
		prefixExpr, err := self.prefixExpression()
		if err != nil {
			return nil, err
		}
		lhs = prefixExpr
	case LCurly:
		blockExpr, err := self.blockExpression()
		if err != nil {
			return nil, err
		}
		lhs = blockExpr
	case If:
		ifExpr, err := self.ifExpression()
		if err != nil {
			return nil, err
		}
		lhs = ifExpr
	case Try:
		tryExpr, err := self.tryExpression()
		if err != nil {
			return nil, err
		}
		lhs = tryExpr
	default:
		return nil, errors.NewSyntaxError(
			self.CurrentToken.Span,
			fmt.Sprintf("Expected an expression, found '%s'", self.CurrentToken.Kind),
		)
	}

	for left, _ := self.CurrentToken.Kind.prec(); left > prec; left, _ = self.CurrentToken.Kind.prec() {
		switch self.CurrentToken.Kind {
		case DoubleDot:
			newLhs, err := self.rangeLiteral(startLoc, lhs)
			if err != nil {
				return nil, err
			}
			lhs = newLhs
		case Plus, Minus, Multiply, Divide, Modulo,
			Power, ShiftLeft, ShiftRight, BitOr, BitAnd,
			BitXor, Or, And, Equal, NotEqual, LessThan,
			LessThanEqual, GreaterThan, GreaterThanEqual:
			// infix expression

			newLhs, err := self.infixExpression(startLoc, lhs)
			if err != nil {
				return nil, err
			}
			lhs = newLhs
		case Assign, PlusAssign, MinusAssign, MultiplyAssign,
			DivideAssign, ModuloAssign, PowerAssign,
			ShiftLeftAssign, ShiftRightAssign,
			BitOrAssign, BitAndAssign, BitXorAssign:
			// assign expression

			newLhs, err := self.assignExpression(startLoc, lhs)
			if err != nil {
				return nil, err
			}
			lhs = newLhs
		case LParen:
			newLhs, err := self.callExpression(startLoc, lhs)
			if err != nil {
				return nil, err
			}
			lhs = newLhs
		case LBracket:
			newLhs, err := self.indexExpression(startLoc, lhs)
			if err != nil {
				return nil, err
			}
			lhs = newLhs
		case Dot:
			newLhs, err := self.memberExpression(startLoc, lhs)
			if err != nil {
				return nil, err
			}
			lhs = newLhs
		case As:
			newLhs, err := self.castExpression(startLoc, lhs)
			if err != nil {
				return nil, err
			}
			lhs = newLhs
		default:
			return lhs, nil
		}
	}

	return lhs, nil
}

//
//	Integer + float literal
//

func (self *Parser) intFloatLiteral() (ast.Expression, *errors.Error) {
	if err := self.next(); err != nil {
		return nil, err
	}

	switch self.PreviousToken.Kind {
	case Int:
		intRes, err := strconv.ParseInt(self.PreviousToken.Value, 10, 64)
		if err != nil {
			return nil, errors.NewSyntaxError(
				self.PreviousToken.Span,
				fmt.Sprintf("Cannot use '%s' as integer: %s", self.PreviousToken.Value, err),
			)
		}

		return ast.Expression(ast.IntLiteralExpression{
			Range: self.PreviousToken.Span,
			Value: intRes,
		}), nil
	case Float:
		floatRes, err := strconv.ParseFloat(self.PreviousToken.Value, 64)
		if err != nil {
			return nil, errors.NewSyntaxError(
				self.PreviousToken.Span,
				fmt.Sprintf("Cannot use '%s' as float: %s", self.PreviousToken.Value, err),
			)
		}

		return ast.Expression(ast.FloatLiteralExpression{
			Range: self.PreviousToken.Span,
			Value: floatRes,
		}), nil
	}

	panic("Unreachable: this method is only called on integers and floats")
}

//
//	Ident expression
//

func (self *Parser) identExpression() (ast.IdentExpression, *errors.Error) {
	ident := ast.NewSpannedIdent(self.CurrentToken.Value, self.CurrentToken.Span)
	if err := self.next(); err != nil {
		return ast.IdentExpression{}, err
	}

	return ast.IdentExpression{
		Ident: ident,
	}, nil
}

//
// Null literal
//

func (self *Parser) nullLiteral() (ast.NullLiteralExpression, *errors.Error) {
	if err := self.next(); err != nil {
		return ast.NullLiteralExpression{}, err
	}
	return ast.NullLiteralExpression{Range: self.PreviousToken.Span}, nil
}

//
// None literal
//

func (self *Parser) noneLiteral() (ast.NoneLiteralExpression, *errors.Error) {
	if err := self.next(); err != nil {
		return ast.NoneLiteralExpression{}, err
	}
	return ast.NoneLiteralExpression{Range: self.PreviousToken.Span}, nil
}

//
// Range literal
//

func (self *Parser) rangeLiteral(start errors.Location, rangeStart ast.Expression) (ast.RangeLiteralExpression, *errors.Error) {
	// skip the `..`
	if err := self.next(); err != nil {
		return ast.RangeLiteralExpression{}, err
	}

	rangeEnd, err := self.expression(0)
	if err != nil {
		return ast.RangeLiteralExpression{}, err
	}

	return ast.RangeLiteralExpression{
		Start: rangeStart,
		End:   rangeEnd,
		Range: start.Until(self.PreviousToken.Span.End, self.Filename),
	}, err
}

//
// List literal
//

func (self *Parser) listLiteral() (ast.ListLiteralExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `[`
	if err := self.next(); err != nil {
		return ast.ListLiteralExpression{}, err
	}

	values := make([]ast.Expression, 0)
	if self.CurrentToken.Kind != RBracket && self.CurrentToken.Kind != EOF {
		// make initial value
		expr, err := self.expression(0)
		if err != nil {
			return ast.ListLiteralExpression{}, err
		}
		values = append(values, expr)

		// make remaining values
		for self.CurrentToken.Kind == Comma {
			if err := self.next(); err != nil {
				return ast.ListLiteralExpression{}, err
			}

			if self.CurrentToken.Kind == RBracket || self.CurrentToken.Kind == EOF {
				break
			}

			expr, err := self.expression(0)
			if err != nil {
				return ast.ListLiteralExpression{}, err
			}
			values = append(values, expr)
		}
	}

	if err := self.expectRecoverable(RBracket); err != nil {
		return ast.ListLiteralExpression{}, err
	}

	return ast.ListLiteralExpression{
		Values: values,
		Range:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Object literal
//

func (self *Parser) objectLiteral() (ast.Expression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `new`
	if err := self.next(); err != nil {
		return nil, err
	}

	if err := self.expect(LCurly); err != nil {
		return nil, err
	}

	fields := make([]ast.ObjectLiteralField, 0)

	if self.CurrentToken.Kind == QuestionMark {
		if err := self.next(); err != nil {
			return nil, err
		}
		if err := self.expectRecoverable(RCurly); err != nil {
			return nil, err
		}
		return ast.AnyObjectLiteralExpression{
			Range: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
		}, nil
	}

	if self.CurrentToken.Kind != RCurly && self.CurrentToken.Kind != EOF {
		// make initial field
		field, err := self.objectLiteralField()
		if err != nil {
			return nil, err
		}

		fields = append(fields, field)

		// make remaining fields
		for self.CurrentToken.Kind == Comma {
			// skip comma
			if err := self.next(); err != nil {
				return nil, err
			}

			if self.CurrentToken.Kind == RCurly || self.CurrentToken.Kind == EOF {
				break
			}

			field, err := self.objectLiteralField()
			if err != nil {
				return nil, err
			}

			fields = append(fields, field)
		}
	}

	if err := self.expectRecoverable(RCurly); err != nil {
		return nil, err
	}

	return ast.ObjectLiteralExpression{
		Fields: fields,
		Range:  startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) functionLiteral() (ast.FunctionLiteralExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `fn`
	if err := self.next(); err != nil {
		return ast.FunctionLiteralExpression{}, err
	}

	paramStartLoc := self.CurrentToken.Span.Start
	params, err := self.parameterList()
	if err != nil {
		return ast.FunctionLiteralExpression{}, err
	}
	paramEndLoc := self.PreviousToken.Span.End

	returnType := ast.HmsType(ast.NameReferenceType{
		Ident: ast.NewSpannedIdent("null", self.PreviousToken.Span.End.Until(self.CurrentToken.Span.End, self.Filename)),
	})

	if self.CurrentToken.Kind == Arrow {
		if err := self.next(); err != nil {
			return ast.FunctionLiteralExpression{}, err
		}

		resType, err := self.hmsType()
		if err != nil {
			return ast.FunctionLiteralExpression{}, err
		}

		returnType = resType
	}

	body, err := self.block()
	if err != nil {
		return ast.FunctionLiteralExpression{}, err
	}

	return ast.FunctionLiteralExpression{
		Parameters: params,
		ParamSpan:  paramStartLoc.Until(paramEndLoc, self.Filename),
		ReturnType: returnType,
		Body:       body,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) objectLiteralField() (ast.ObjectLiteralField, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	if err := self.expectMultiple([]TokenKind{Identifier, String}); err != nil {
		return ast.ObjectLiteralField{}, err
	}
	key := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	if err := self.expect(Colon); err != nil {
		return ast.ObjectLiteralField{}, err
	}

	value, err := self.expression(0)
	if err != nil {
		return ast.ObjectLiteralField{}, err
	}

	return ast.ObjectLiteralField{
		Key:        key,
		Expression: value,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Grouped expression
//

func (self *Parser) groupedExpression() (ast.GroupedExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip opening `(`
	if err := self.next(); err != nil {
		return ast.GroupedExpression{}, err
	}

	inner, err := self.expression(0)
	if err != nil {
		return ast.GroupedExpression{}, err
	}

	if err := self.expectRecoverable(RParen); err != nil {
		return ast.GroupedExpression{}, err
	}

	return ast.GroupedExpression{
		Inner: inner,
		Range: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, err
}

//
// Prefix expression
//

func (self *Parser) prefixExpression() (ast.PrefixExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	operator := self.CurrentToken.Kind.asPrefixOperator()
	if err := self.next(); err != nil {
		return ast.PrefixExpression{}, err
	}

	// precedence is higher than all infix-precedences except call / member
	base, err := self.expression(29)
	if err != nil {
		return ast.PrefixExpression{}, err
	}

	return ast.PrefixExpression{
		Operator: operator,
		Base:     base,
		Range:    startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
//	Infix expression
//

func (self *Parser) infixExpression(start errors.Location, lhs ast.Expression) (ast.InfixExpression, *errors.Error) {
	// determine infix operator kind
	op := self.CurrentToken.Kind.asInfixOperator()
	_, rhsPrec := self.CurrentToken.Kind.prec()

	if err := self.next(); err != nil {
		return ast.InfixExpression{}, err
	}

	rhs, err := self.expression(rhsPrec)
	if err != nil {
		return ast.InfixExpression{}, err
	}

	return ast.InfixExpression{
		Lhs:      lhs,
		Rhs:      rhs,
		Operator: op,
		Range:    start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Assign expression
//

func (self *Parser) assignExpression(start errors.Location, lhs ast.Expression) (ast.AssignExpression, *errors.Error) {
	operator := self.CurrentToken.Kind.asAssignOperator()
	_, rhsPrec := self.CurrentToken.Kind.prec()

	if err := self.next(); err != nil {
		return ast.AssignExpression{}, err
	}

	rhs, err := self.expression(rhsPrec)
	if err != nil {
		return ast.AssignExpression{}, err
	}

	switch lhs.Kind() {
	case ast.IdentExpressionKind, ast.IndexExpressionKind, ast.MemberExpressionKind, ast.CastExpressionKind:
		// do nothing, this is legal
	default:
		return ast.AssignExpression{}, errors.NewSyntaxError(lhs.Span(), "Invalid left-hand side of assignment")
	}

	return ast.AssignExpression{
		Lhs:            lhs,
		AssignOperator: operator,
		Rhs:            rhs,
		Range:          start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
//	Call expression
//

func (self *Parser) callExpression(start errors.Location, base ast.Expression) (ast.CallExpression, *errors.Error) {
	// skip opening parenthesis
	if err := self.next(); err != nil {
		return ast.CallExpression{}, err
	}

	args := make([]ast.Expression, 0)
	if self.CurrentToken.Kind != RParen && self.CurrentToken.Kind != EOF {
		// make first argument
		expr, err := self.expression(0)
		if err != nil {
			return ast.CallExpression{}, err
		}
		args = append(args, expr)

		// make remaining arguments
		for self.CurrentToken.Kind == Comma {
			if err := self.next(); err != nil {
				return ast.CallExpression{}, err
			}

			if self.CurrentToken.Kind == RParen || self.CurrentToken.Kind == EOF {
				break
			}

			expr, err := self.expression(0)
			if err != nil {
				return ast.CallExpression{}, err
			}

			args = append(args, expr)
		}
	}

	if err := self.expectRecoverable(RParen); err != nil {
		return ast.CallExpression{}, err
	}

	return ast.CallExpression{
		Base:      base,
		Arguments: args,
		Range:     start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
//	Index expression
//

func (self *Parser) indexExpression(start errors.Location, base ast.Expression) (ast.IndexExpression, *errors.Error) {
	// skip opening bracket
	if err := self.next(); err != nil {
		return ast.IndexExpression{}, err
	}

	index, err := self.expression(0)
	if err != nil {
		return ast.IndexExpression{}, err
	}

	if err := self.expectRecoverable(RBracket); err != nil {
		return ast.IndexExpression{}, err
	}

	return ast.IndexExpression{
		Base:  base,
		Index: index,
		Range: start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
//	Member expression
//

func (self *Parser) memberExpression(start errors.Location, base ast.Expression) (ast.MemberExpression, *errors.Error) {
	// skip the `.`
	if err := self.next(); err != nil {
		return ast.MemberExpression{}, err
	}

	if err := self.expect(Identifier); err != nil {
		return ast.MemberExpression{}, err
	}
	member := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	return ast.MemberExpression{
		Base:   base,
		Member: member,
		Range:  start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Cast expression
//

func (self *Parser) castExpression(start errors.Location, base ast.Expression) (ast.CastExpression, *errors.Error) {
	// skip the `as`
	if err := self.next(); err != nil {
		return ast.CastExpression{}, err
	}

	asType, err := self.hmsType()
	if err != nil {
		return ast.CastExpression{}, err
	}

	return ast.CastExpression{
		Base:   base,
		AsType: asType,
		Range:  start.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Block expression
//

func (self *Parser) blockExpression() (ast.BlockExpression, *errors.Error) {
	block, err := self.block()
	if err != nil {
		return ast.BlockExpression{}, err
	}

	return ast.BlockExpression{
		Block: block,
	}, nil
}

//
// If expression
//

func (self *Parser) ifExpression() (ast.IfExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `if`
	if err := self.next(); err != nil {
		return ast.IfExpression{}, err
	}

	condition, err := self.expression(0)
	if err != nil {
		return ast.IfExpression{}, err
	}

	thenBlock, err := self.block()
	if err != nil {
		return ast.IfExpression{}, err
	}

	// make optional `else if` block
	var elseBlock *ast.Block

	if self.CurrentToken.Kind == Else {
		if err := self.next(); err != nil {
			return ast.IfExpression{}, err
		}

		if self.CurrentToken.Kind == If {
			elseIf, err := self.ifExpression()
			if err != nil {
				return ast.IfExpression{}, err
			}
			elseBlock = &ast.Block{
				Statements: make([]ast.Statement, 0),
				Expression: elseIf,
				Range:      elseIf.Range.Start.Until(elseIf.ThenBlock.Range.End, self.Filename),
			}
		} else {
			block, err := self.block()
			if err != nil {
				return ast.IfExpression{}, err
			}

			elseBlock = &block
		}

	}

	return ast.IfExpression{
		Condition: condition,
		ThenBlock: thenBlock,
		ElseBlock: elseBlock,
		Range:     startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Try expression
//

func (self *Parser) tryExpression() (ast.TryExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `try`
	if err := self.next(); err != nil {
		return ast.TryExpression{}, err
	}

	tryBlock, err := self.block()
	if err != nil {
		return ast.TryExpression{}, err
	}

	if err := self.expect(Catch); err != nil {
		return ast.TryExpression{}, err
	}

	if err := self.expect(Identifier); err != nil {
		return ast.TryExpression{}, err
	}
	catchIdentifier := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	catchBlock, err := self.block()
	if err != nil {
		return ast.TryExpression{}, err
	}

	return ast.TryExpression{
		TryBlock:   tryBlock,
		CatchIdent: catchIdentifier,
		CatchBlock: catchBlock,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Block
//

func (self *Parser) block() (ast.Block, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	if err := self.expect(LCurly); err != nil {
		return ast.Block{}, err
	}

	statements := make([]ast.Statement, 0)
	var trailingExpr ast.Expression

	for self.CurrentToken.Kind != RCurly && self.CurrentToken.Kind != EOF {
		item, err := self.statemtent()
		if err != nil {
			return ast.Block{}, err
		}

		if item.Statement != nil {
			statements = append(statements, item.Statement)
		} else if item.Expression != nil {
			trailingExpr = item.Expression
			break
		} else {
			panic("Unreachable: either statement or expression should be non-nil")
		}
	}

	if err := self.expectRecoverable(RCurly); err != nil {
		return ast.Block{}, err
	}

	return ast.Block{
		Statements: statements,
		Expression: trailingExpr,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}
