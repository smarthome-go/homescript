package parser

import (
	"fmt"
	"strconv"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
//	Expression
//

func (self *Parser) expression(prec uint8) (expr ast.Expression, isWithBlock bool, err *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	var lhs ast.Expression

	switch self.CurrentToken.Kind {
	case lexer.SINGLETON_TOKEN, lexer.Identifier, lexer.Underscore:
		ident, err := self.identExpression()
		if err != nil {
			return nil, false, err
		}
		lhs = ident
	case lexer.LParen:
		grouped, err := self.groupedExpression()
		if err != nil {
			return nil, false, err
		}
		lhs = grouped
	case lexer.Not, lexer.Minus, lexer.QuestionMark:
		prefixExpr, err := self.prefixExpression(false)
		if err != nil {
			return nil, false, err
		}
		lhs = prefixExpr
	case lexer.LCurly:
		blockExpr, err := self.blockExpression()
		if err != nil {
			return nil, false, err
		}
		lhs = blockExpr
		isWithBlock = true
	case lexer.If:
		ifExpr, err := self.ifExpression()
		if err != nil {
			return nil, false, err
		}
		lhs = ifExpr
		isWithBlock = true
	case lexer.Match:
		matchExpr, err := self.matchExpression()
		if err != nil {
			return nil, false, err
		}
		lhs = matchExpr
		isWithBlock = true
	case lexer.Try:
		tryExpr, err := self.tryExpression()
		if err != nil {
			return nil, false, err
		}
		lhs = tryExpr
		isWithBlock = true
	case lexer.Spawn:
		spawnExpr, err := self.spawnExpr()
		if err != nil {
			return nil, false, err
		}
		lhs = spawnExpr
	default:
		expr, err := self.literal(false)
		if err != nil {
			return nil, false, err
		}
		lhs = expr
	}

	for left, _ := self.CurrentToken.Kind.Prec(); left > prec; left, _ = self.CurrentToken.Kind.Prec() {
		isWithBlock = false

		switch self.CurrentToken.Kind {
		case lexer.DoubleDot:
			newLhs, err := self.rangeLiteral(startLoc, lhs)
			if err != nil {
				return nil, false, err
			}
			lhs = newLhs
		case lexer.Plus, lexer.Minus, lexer.Multiply, lexer.Divide, lexer.Modulo,
			lexer.Power, lexer.ShiftLeft, lexer.ShiftRight, lexer.BitOr, lexer.BitAnd,
			lexer.BitXor, lexer.Or, lexer.And, lexer.Equal, lexer.NotEqual, lexer.LessThan,
			lexer.LessThanEqual, lexer.GreaterThan, lexer.GreaterThanEqual:
			// infix expression

			newLhs, err := self.infixExpression(startLoc, lhs)
			if err != nil {
				return nil, false, err
			}
			lhs = newLhs
		case lexer.Assign, lexer.PlusAssign, lexer.MinusAssign, lexer.MultiplyAssign,
			lexer.DivideAssign, lexer.ModuloAssign, lexer.PowerAssign,
			lexer.ShiftLeftAssign, lexer.ShiftRightAssign,
			lexer.BitOrAssign, lexer.BitAndAssign, lexer.BitXorAssign:
			// assign expression

			newLhs, err := self.assignExpression(startLoc, lhs)
			if err != nil {
				return nil, false, err
			}
			lhs = newLhs
		case lexer.LParen:
			newLhs, err := self.callExpression(startLoc, lhs)
			if err != nil {
				return nil, false, err
			}
			lhs = newLhs
		case lexer.LBracket:
			newLhs, err := self.indexExpression(startLoc, lhs)
			if err != nil {
				return nil, false, err
			}
			lhs = newLhs
		case lexer.Dot, lexer.Arrow, lexer.TildeArrow:
			newLhs, err := self.memberExpression(startLoc, lhs, ast.NewMemberOperator(self.CurrentToken.Kind))
			if err != nil {
				return nil, false, err
			}
			lhs = newLhs
		case lexer.As:
			newLhs, err := self.castExpression(startLoc, lhs)
			if err != nil {
				return nil, false, err
			}
			lhs = newLhs
		default:
			literal, err := self.literal(false)
			return literal, false, err
		}
	}

	return lhs, isWithBlock, nil
}

func (self *Parser) literal(useLiteralInErr bool) (ast.Expression, *errors.Error) {
	switch self.CurrentToken.Kind {
	case lexer.Int, lexer.Float:
		return self.intFloatLiteral()
	case lexer.True, lexer.False:
		if err := self.next(); err != nil {
			return nil, err
		}
		return ast.BoolLiteralExpression{
			Value: self.PreviousToken.Kind == lexer.True,
			Range: self.PreviousToken.Span,
		}, nil
	case lexer.String:
		if err := self.next(); err != nil {
			return nil, err
		}
		return ast.StringLiteralExpression{
			Value: self.PreviousToken.Value,
			Range: self.PreviousToken.Span,
		}, nil
	case lexer.Null:
		return self.nullLiteral()
	case lexer.None:
		return self.noneLiteral()
	case lexer.LBracket:
		return self.listLiteral()
	case lexer.New:
		return self.objectLiteral()
	case lexer.Fn:
		return self.functionLiteral()
	default:
		message := fmt.Sprintf("Expected an expression, found '%s'", self.CurrentToken.Kind)
		if useLiteralInErr {
			message = fmt.Sprintf("Expected a literal expression, found '%s'", self.CurrentToken.Kind)
		}

		return nil, errors.NewSyntaxError(
			self.CurrentToken.Span,
			message,
		)
	}
}

//
//	Integer + float literal
//

func (self *Parser) intFloatLiteral() (ast.Expression, *errors.Error) {
	if err := self.next(); err != nil {
		return nil, err
	}

	switch self.PreviousToken.Kind {
	case lexer.Int:
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
	case lexer.Float:
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
// (Singleton) Ident expression
//

func (self *Parser) identExpression() (ast.IdentExpression, *errors.Error) {
	ident, isSingleton, err := self.singletonIdentOrNormal()
	if err != nil {
		return ast.IdentExpression{}, err
	}

	return ast.IdentExpression{
		IsSingleton: isSingleton,
		Ident:       ident,
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

	// if there is an `=`, include the upper bound
	endIsInclusive := false
	if self.CurrentToken.Kind == lexer.Assign {
		endIsInclusive = true
		if err := self.next(); err != nil {
			return ast.RangeLiteralExpression{}, err
		}
	}

	rangeEnd, _, err := self.expression(0)
	if err != nil {
		return ast.RangeLiteralExpression{}, err
	}

	return ast.RangeLiteralExpression{
		Start:          rangeStart,
		End:            rangeEnd,
		EndIsInclusive: endIsInclusive,
		Range:          start.Until(self.PreviousToken.Span.End, self.Filename),
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
	if self.CurrentToken.Kind != lexer.RBracket && self.CurrentToken.Kind != lexer.EOF {
		// make initial value
		expr, _, err := self.expression(0)
		if err != nil {
			return ast.ListLiteralExpression{}, err
		}
		values = append(values, expr)

		// make remaining values
		for self.CurrentToken.Kind == lexer.Comma {
			if err := self.next(); err != nil {
				return ast.ListLiteralExpression{}, err
			}

			if self.CurrentToken.Kind == lexer.RBracket || self.CurrentToken.Kind == lexer.EOF {
				break
			}

			expr, _, err := self.expression(0)
			if err != nil {
				return ast.ListLiteralExpression{}, err
			}
			values = append(values, expr)
		}
	}

	if err := self.expectRecoverable(lexer.RBracket); err != nil {
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

	if err := self.expect(lexer.LCurly); err != nil {
		return nil, err
	}

	fields := make([]ast.ObjectLiteralField, 0)

	if self.CurrentToken.Kind == lexer.QuestionMark {
		if err := self.next(); err != nil {
			return nil, err
		}
		if err := self.expectRecoverable(lexer.RCurly); err != nil {
			return nil, err
		}
		return ast.AnyObjectLiteralExpression{
			Range: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
		}, nil
	}

	if self.CurrentToken.Kind != lexer.RCurly && self.CurrentToken.Kind != lexer.EOF {
		// make initial field
		field, err := self.objectLiteralField()
		if err != nil {
			return nil, err
		}

		fields = append(fields, field)

		// make remaining fields
		for self.CurrentToken.Kind == lexer.Comma {
			// skip comma
			if err := self.next(); err != nil {
				return nil, err
			}

			if self.CurrentToken.Kind == lexer.RCurly || self.CurrentToken.Kind == lexer.EOF {
				break
			}

			field, err := self.objectLiteralField()
			if err != nil {
				return nil, err
			}

			fields = append(fields, field)
		}
	}

	if err := self.expectRecoverable(lexer.RCurly); err != nil {
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

	if self.CurrentToken.Kind == lexer.Arrow {
		if err := self.next(); err != nil {
			return ast.FunctionLiteralExpression{}, err
		}

		resType, err := self.hmsType(false)
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

	if err := self.expectMultiple(lexer.Identifier, lexer.Underscore, lexer.String); err != nil {
		return ast.ObjectLiteralField{}, err
	}
	key := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	if err := self.expect(lexer.Colon); err != nil {
		return ast.ObjectLiteralField{}, err
	}

	value, _, err := self.expression(0)
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

	inner, _, err := self.expression(0)
	if err != nil {
		return ast.GroupedExpression{}, err
	}

	if err := self.expectRecoverable(lexer.RParen); err != nil {
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

func (self *Parser) prefixExpression(restrictBaseToLiterals bool) (ast.PrefixExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	operator := ast.TokenAsPrefixOperator(self.CurrentToken.Kind)
	if err := self.next(); err != nil {
		return ast.PrefixExpression{}, err
	}

	var base ast.Expression

	if restrictBaseToLiterals {
		baseTemp, err := self.literal(true)
		if err != nil {
			return ast.PrefixExpression{}, err
		}
		base = baseTemp
	} else {
		// precedence is higher than all infix-precedences except call / member
		baseTemp, _, err := self.expression(29)
		if err != nil {
			return ast.PrefixExpression{}, err
		}
		base = baseTemp
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
	op := ast.TokenAsInfixOperator(self.CurrentToken.Kind)
	_, rhsPrec := self.CurrentToken.Kind.Prec()

	if err := self.next(); err != nil {
		return ast.InfixExpression{}, err
	}

	rhs, _, err := self.expression(rhsPrec)
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
	operator := ast.TokenAsAssignOperator(self.CurrentToken.Kind)
	_, rhsPrec := self.CurrentToken.Kind.Prec()

	if err := self.next(); err != nil {
		return ast.AssignExpression{}, err
	}

	rhs, _, err := self.expression(rhsPrec)
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

func (self *Parser) callArgs() (argsRet ast.CallArgs, err *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip opening parenthesis
	if err := self.next(); err != nil {
		return ast.CallArgs{}, err
	}

	args := make([]ast.Expression, 0)
	if self.CurrentToken.Kind != lexer.RParen && self.CurrentToken.Kind != lexer.EOF {
		// make first argument
		expr, _, err := self.expression(0)
		if err != nil {
			return ast.CallArgs{}, err
		}
		args = append(args, expr)

		// make remaining arguments
		for self.CurrentToken.Kind == lexer.Comma {
			if err := self.next(); err != nil {
				return ast.CallArgs{}, err
			}

			if self.CurrentToken.Kind == lexer.RParen || self.CurrentToken.Kind == lexer.EOF {
				break
			}

			expr, _, err := self.expression(0)
			if err != nil {
				return ast.CallArgs{}, err
			}

			args = append(args, expr)
		}
	}

	if err := self.expectRecoverable(lexer.RParen); err != nil {
		return ast.CallArgs{}, err
	}

	return ast.CallArgs{
		Span: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
		List: args,
	}, nil
}

func (self *Parser) callExpression(start errors.Location, base ast.Expression) (ast.CallExpression, *errors.Error) {
	args, err := self.callArgs()
	if err != nil {
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

	index, _, err := self.expression(0)
	if err != nil {
		return ast.IndexExpression{}, err
	}

	if err := self.expectRecoverable(lexer.RBracket); err != nil {
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

func (self *Parser) memberExpression(
	start errors.Location,
	base ast.Expression,
	operator ast.MemberOperator,
) (ast.MemberExpression, *errors.Error) {
	// Skip the operator.
	if err := self.next(); err != nil {
		return ast.MemberExpression{}, err
	}

	if err := self.expectMultiple(lexer.Identifier, lexer.Underscore); err != nil {
		return ast.MemberExpression{}, err
	}
	member := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	return ast.MemberExpression{
		Base:     base,
		Member:   member,
		Range:    start.Until(self.PreviousToken.Span.End, self.Filename),
		Operator: operator,
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

	asType, err := self.hmsType(false)
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

	condition, _, err := self.expression(0)
	if err != nil {
		return ast.IfExpression{}, err
	}

	thenBlock, err := self.block()
	if err != nil {
		return ast.IfExpression{}, err
	}

	// make optional `else if` block
	var elseBlock *ast.Block

	if self.CurrentToken.Kind == lexer.Else {
		if err := self.next(); err != nil {
			return ast.IfExpression{}, err
		}

		if self.CurrentToken.Kind == lexer.If {
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
// Match expression
//

func (self *Parser) matchExpression() (ast.MatchExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `match`
	if err := self.next(); err != nil {
		return ast.MatchExpression{}, err
	}

	controlExpr, _, err := self.expression(0)
	if err != nil {
		return ast.MatchExpression{}, err
	}

	if err := self.expect(lexer.LCurly); err != nil {
		return ast.MatchExpression{}, err
	}

	arms := make([]ast.MatchArm, 0)

	if self.CurrentToken.Kind != lexer.EOF && self.CurrentToken.Kind != lexer.RCurly {
		// make initial value
		arm, withBlockTemp, err := self.matchArm()
		if err != nil {
			return ast.MatchExpression{}, err
		}
		arms = append(arms, arm)

		withBlock := withBlockTemp

		// make remaining values
		for self.CurrentToken.Kind == lexer.Comma || withBlock {
			if self.CurrentToken.Kind == lexer.Comma {
				if err := self.next(); err != nil {
					return ast.MatchExpression{}, err
				}
			}

			if self.CurrentToken.Kind == lexer.RCurly || self.CurrentToken.Kind == lexer.EOF {
				break
			}

			arm, withBlockTemp, err := self.matchArm()
			if err != nil {
				return ast.MatchExpression{}, err
			}
			withBlock = withBlockTemp
			arms = append(arms, arm)
		}
	}

	if err := self.expectRecoverable(lexer.RCurly); err != nil {
		return ast.MatchExpression{}, err
	}

	return ast.MatchExpression{
		ControlExpression: controlExpr,
		Arms:              arms,
		Range:             startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

func (self *Parser) matchArm() (arm ast.MatchArm, withBlock bool, err *errors.Error) {
	startLoc := self.CurrentToken.Span.Start
	literals := make([]ast.DefaultOrLiteral, 0)

	switch self.CurrentToken.Kind {
	case lexer.Underscore:
		literals = append(literals, ast.NewDefaultOrLiteralDefault())
		if err := self.next(); err != nil {
			return ast.MatchArm{}, false, err
		}
	case lexer.Not, lexer.Minus, lexer.QuestionMark:
		expr, err := self.prefixExpression(true)
		if err != nil {
			return ast.MatchArm{}, false, err
		}
		literals = append(literals, ast.NewDefaultOrLiteralLiteral(expr))
	default:
		for {
			expr, err := self.literal(true)
			if err != nil {
				return ast.MatchArm{}, false, err
			}
			literals = append(literals, ast.NewDefaultOrLiteralLiteral(expr))

			if self.CurrentToken.Kind != lexer.BitOr {
				break
			}

			if err := self.next(); err != nil {
				return ast.MatchArm{}, false, err
			}
		}
	}

	if err := self.expect(lexer.FatArrow); err != nil {
		return ast.MatchArm{}, false, err
	}

	action, withBlock, err := self.expression(0)
	if err != nil {
		return ast.MatchArm{}, false, err
	}

	return ast.MatchArm{
		Literals: literals,
		Action:   action,
		Range:    startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, withBlock, nil
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

	if err := self.expect(lexer.Catch); err != nil {
		return ast.TryExpression{}, err
	}

	if err := self.expectMultiple(lexer.Identifier, lexer.Underscore); err != nil {
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
// Spawn expression
//

func (self *Parser) spawnExpr() (ast.CallExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start
	if err := self.next(); err != nil {
		return ast.CallExpression{}, err
	}

	ident := ast.NewSpannedIdent(self.CurrentToken.Value, self.CurrentToken.Span)
	if err := self.expect(lexer.Identifier); err != nil {
		return ast.CallExpression{}, err
	}

	call, err := self.callExpression(startLoc, ast.IdentExpression{
		Ident: ident,
	})

	if err != nil {
		return ast.CallExpression{}, err
	}

	call.IsSpawn = true
	return call, nil
}

//
// Block
//

func (self *Parser) block() (ast.Block, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	if err := self.expect(lexer.LCurly); err != nil {
		return ast.Block{}, err
	}

	statements := make([]ast.Statement, 0)
	var trailingExpr ast.Expression

	for self.CurrentToken.Kind != lexer.RCurly && self.CurrentToken.Kind != lexer.EOF {
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

	if err := self.expectRecoverable(lexer.RCurly); err != nil {
		return ast.Block{}, err
	}

	return ast.Block{
		Statements: statements,
		Expression: trailingExpr,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}
