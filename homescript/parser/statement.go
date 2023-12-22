package parser

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
//	Statements
//

func (self *Parser) statemtent() (ast.EitherStatementOrExpression, *errors.Error) {
	res := ast.EitherStatementOrExpression{}

	switch self.CurrentToken.Kind {
	case Type:
		typeDef, err := self.typeDefinition(false)
		if err != nil {
			return ast.EitherStatementOrExpression{}, err
		}
		res.Statement = typeDef
	case Let:
		letStmt, err := self.letStatement(false) // no longer top-level
		if err != nil {
			return ast.EitherStatementOrExpression{}, err
		}
		res.Statement = letStmt
	case Return:
		returnStmt, err := self.returnStatement()
		if err != nil {
			return ast.EitherStatementOrExpression{}, err
		}
		res.Statement = returnStmt
	case Break:
		breakStmt, err := self.breakStatement()
		if err != nil {
			return ast.EitherStatementOrExpression{}, err
		}
		res.Statement = breakStmt
	case Continue:
		continueStmt, err := self.continueStatement()
		if err != nil {
			return ast.EitherStatementOrExpression{}, err
		}
		res.Statement = continueStmt
	case Loop:
		loopStmt, err := self.loopStatement()
		if err != nil {
			return ast.EitherStatementOrExpression{}, err
		}
		res.Statement = loopStmt
	case While:
		whileStmt, err := self.whileStatement()
		if err != nil {
			return ast.EitherStatementOrExpression{}, err
		}
		res.Statement = whileStmt
	case For:
		forStmt, err := self.forStatement()
		if err != nil {
			return ast.EitherStatementOrExpression{}, err
		}
		res.Statement = forStmt
	default:
		return self.expressionStatement()
	}
	return res, nil
}

///
/// Type Definition
///

func (self *Parser) typeDefinition(isPub bool) (ast.TypeDefinition, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	if err := self.next(); err != nil {
		return ast.TypeDefinition{}, err
	}

	if err := self.expectMultiple(Identifier, Underscore); err != nil {
		return ast.TypeDefinition{}, err
	}
	newTypeIdent := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	// Check that the lhs is not a builtin type
	isBuiltin := false
	for _, typ := range ast.HMS_BUILTIN_TYPES {
		if typ == newTypeIdent.Ident() {
			isBuiltin = true
			break
		}
	}

	// Prevent redeclaration of builtin types
	if isBuiltin {
		return ast.TypeDefinition{}, errors.NewSyntaxError(
			self.PreviousToken.Span,
			fmt.Sprintf("Cannot redeclare builtin type '%s'", self.PreviousToken.Value),
		)
	}

	if err := self.expect(Assign); err != nil {
		return ast.TypeDefinition{}, err
	}

	rhsType, err := self.hmsType()
	if err != nil {
		return ast.TypeDefinition{}, err
	}

	if err := self.expectRecoverable(Semicolon); err != nil {
		return ast.TypeDefinition{}, err
	}

	return ast.TypeDefinition{
		LhsIdent: newTypeIdent,
		RhsType:  rhsType,
		IsPub:    isPub,
		Range:    startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Let statement
//

func (self *Parser) letStatement(isPub bool) (ast.LetStatement, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start
	// skip the `let`
	if err := self.next(); err != nil {
		return ast.LetStatement{}, err
	}

	if err := self.expectMultiple(Identifier, Underscore); err != nil {
		return ast.LetStatement{}, err
	}
	ident := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	if self.CurrentToken.Kind != Assign && self.CurrentToken.Kind != Colon {
		return ast.LetStatement{}, self.expectedOneOfErr([]TokenKind{Assign, Colon})
	}

	// make optional type
	var optType ast.HmsType
	if self.CurrentToken.Kind == Colon {
		// skip the colon
		if err := self.next(); err != nil {
			return ast.LetStatement{}, err
		}

		typ, err := self.hmsType()
		if err != nil {
			return ast.LetStatement{}, err
		}

		optType = typ

		if err := self.expect(Assign); err != nil {
			return ast.LetStatement{}, err
		}
	} else {
		if err := self.next(); err != nil {
			return ast.LetStatement{}, err
		}
	}

	expr, _, err := self.expression(0)
	if err != nil {
		return ast.LetStatement{}, err
	}

	if err := self.expectRecoverable(Semicolon); err != nil {
		return ast.LetStatement{}, err
	}

	return ast.LetStatement{
		Ident:      ident,
		Expression: expr,
		OptType:    optType,
		IsPub:      isPub,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Return statement
//

func (self *Parser) returnStatement() (ast.ReturnStatement, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip `return`
	if err := self.next(); err != nil {
		return ast.ReturnStatement{}, err
	}

	var expr ast.Expression
	if self.CurrentToken.Kind != Semicolon &&
		self.CurrentToken.Kind != EOF &&
		self.CurrentToken.Kind != RCurly { // this may lead to a hard error being a soft error
		returnExpr, _, err := self.expression(0)
		if err != nil {
			return ast.ReturnStatement{}, err
		}
		expr = returnExpr
	}

	if err := self.expectRecoverable(Semicolon); err != nil {
		return ast.ReturnStatement{}, err
	}

	return ast.ReturnStatement{
		Expression: expr,
		Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Break statement
//

func (self *Parser) breakStatement() (ast.BreakStatement, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `break`
	if err := self.next(); err != nil {
		return ast.BreakStatement{}, err
	}

	if err := self.expectRecoverable(Semicolon); err != nil {
		return ast.BreakStatement{}, err
	}

	return ast.BreakStatement{
		Range: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Continue statement
//

func (self *Parser) continueStatement() (ast.ContinueStatement, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `break`
	if err := self.next(); err != nil {
		return ast.ContinueStatement{}, err
	}

	if err := self.expectRecoverable(Semicolon); err != nil {
		return ast.ContinueStatement{}, err
	}

	return ast.ContinueStatement{
		Range: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Loop statement
//

func (self *Parser) loopStatement() (ast.LoopStatement, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `loop`
	if err := self.next(); err != nil {
		return ast.LoopStatement{}, err
	}

	body, err := self.block()
	if err != nil {
		return ast.LoopStatement{}, err
	}

	return ast.LoopStatement{
		Body:  body,
		Range: startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// While statement
//

func (self *Parser) whileStatement() (ast.WhileStatement, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `while`
	if err := self.next(); err != nil {
		return ast.WhileStatement{}, err
	}

	condition, _, err := self.expression(0)
	if err != nil {
		return ast.WhileStatement{}, err
	}

	body, err := self.block()
	if err != nil {
		return ast.WhileStatement{}, err
	}

	return ast.WhileStatement{
		Condition: condition,
		Body:      body,
		Range:     startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// For statement
//

func (self *Parser) forStatement() (ast.ForStatement, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// skip the `for`
	if err := self.next(); err != nil {
		return ast.ForStatement{}, err
	}

	if err := self.expectMultiple(Identifier, Underscore); err != nil {
		return ast.ForStatement{}, err
	}
	ident := ast.NewSpannedIdent(self.PreviousToken.Value, self.PreviousToken.Span)

	if err := self.expect(In); err != nil {
		return ast.ForStatement{}, err
	}

	iterExpression, _, err := self.expression(0)
	if err != nil {
		return ast.ForStatement{}, err
	}

	body, err := self.block()
	if err != nil {
		return ast.ForStatement{}, err
	}

	return ast.ForStatement{
		Identifier:     ident,
		IterExpression: iterExpression,
		Body:           body,
		Range:          startLoc.Until(self.PreviousToken.Span.End, self.Filename),
	}, nil
}

//
// Expression statement
//

func (self *Parser) expressionStatement() (ast.EitherStatementOrExpression, *errors.Error) {
	startLoc := self.CurrentToken.Span.Start

	// var expression ast.Expression
	// withBlock := false

	// switch self.CurrentToken.Kind {
	// case If:
	// 	ifExpr, err := self.ifExpression()
	// 	if err != nil {
	// 		return ast.EitherStatementOrExpression{}, err
	// 	}
	// 	expression = ifExpr
	// 	withBlock = true
	// case Match:
	// 	matchExpr, err := self.matchExpression()
	// 	if err != nil {
	// 		return ast.EitherStatementOrExpression{}, err
	// 	}
	// 	expression = matchExpr
	// 	withBlock = true
	// case Try:
	// 	tryExpr, err := self.tryExpression()
	// 	if err != nil {
	// 		return ast.EitherStatementOrExpression{}, err
	// 	}
	// 	expression = tryExpr
	// 	withBlock = true
	// case LCurly:
	// 	block, err := self.block()
	// 	if err != nil {
	// 		return ast.EitherStatementOrExpression{}, err
	// 	}
	// 	expression = ast.BlockExpression{Block: block}
	// 	withBlock = true
	// default:
	// 	expr, _, err := self.expression(0)
	// 	if err != nil {
	// 		return ast.EitherStatementOrExpression{}, err
	// 	}
	// 	expression = expr
	// }

	expression, withBlock, err := self.expression(0)
	if err != nil {
		return ast.EitherStatementOrExpression{}, err
	}

	if self.CurrentToken.Kind == RCurly {
		return ast.EitherStatementOrExpression{
			Expression: expression,
		}, nil
	}

	if self.CurrentToken.Kind == Semicolon {
		if err := self.next(); err != nil {
			return ast.EitherStatementOrExpression{}, err
		}
	} else if !withBlock {
		self.Errors = append(self.Errors, *errors.NewSyntaxError(
			expression.Span(),
			"Missing semicolon after statemtent",
		))
	}

	return ast.EitherStatementOrExpression{
		Statement: ast.ExpressionStatement{
			Expression: expression,
			Range:      startLoc.Until(self.PreviousToken.Span.End, self.Filename),
		},
	}, nil
}
