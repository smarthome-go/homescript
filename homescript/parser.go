package homescript

import (
	"fmt"
)

// A list of possible tokens which can start an expression
var firstExpr = []TokenKind{
	LParen,
	Not,
	Plus,
	Minus,
	If,
	Fn,
	Try,
	For,
	While,
	Loop,
	True,
	False,
	String,
	Number,
	Identifier,
}

type parser struct {
	lexer     lexer
	prevToken Token
	currToken Token
	errors    []Error
}

func newParser(filename string, program string) parser {
	return parser{
		lexer:     newLexer(filename, program),
		prevToken: unknownToken(Location{}),
		currToken: unknownToken(Location{}),
		errors:    make([]Error, 0),
	}
}

func (self *parser) advance() *Error {
	next, err := self.lexer.nextToken()
	if err != nil {
		return err
	}
	self.prevToken = self.currToken
	self.currToken = next
	return nil
}

func (self *parser) expression() (Expression, *Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.andExpr()
	if err != nil {
		return Expression{}, nil
	}

	following := make([]AndExpression, 0)
	for self.currToken.Kind == Or {
		followingExpr, err := self.andExpr()
		if err != nil {
			return Expression{}, err
		}
		following = append(following, followingExpr)
	}

	expr := Expression{
		Base:      base,
		Following: following,
		Span: Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	if err := self.advance(); err != nil {
		return Expression{}, err
	}
	return expr, nil
}

func (self *parser) andExpr() (AndExpression, *Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.eqExpr()
	if err != nil {
		return AndExpression{}, err
	}

	following := make([]EqExpression, 0)
	for self.currToken.Kind == And {
		followingExpr, err := self.eqExpr()
		if err != nil {
			return AndExpression{}, err
		}
		following = append(following, followingExpr)
	}

	andExpr := AndExpression{
		Base:      base,
		Following: following,
		Span: Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	if err := self.advance(); err != nil {
		return AndExpression{}, err
	}
	return andExpr, nil
}

func (self *parser) eqExpr() (EqExpression, *Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.relExpr()
	if err != nil {
		return EqExpression{}, err
	}
	if self.currToken.Kind == Equal || self.currToken.Kind == NotEqual {
		isInverted := self.currToken.Kind == NotEqual
		if err := self.advance(); err != nil {
			return EqExpression{}, err
		}
		other, err := self.relExpr()
		if err != nil {
			return EqExpression{}, err
		}
		eqExpr := EqExpression{
			Base: RelExpression{},
			Other: &struct {
				Inverted bool
				Other    RelExpression
			}{
				Inverted: isInverted,
				Other:    other,
			},
			Span: Span{
				Start: startLocation,
				End:   self.currToken.EndLocation,
			},
		}
		if err := self.advance(); err != nil {
			return EqExpression{}, err
		}
		return eqExpr, nil
	}
	eqExpr := EqExpression{
		Base:  RelExpression{},
		Other: nil,
		Span: Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	if err := self.advance(); err != nil {
		return EqExpression{}, err
	}
	return eqExpr, nil
}

func (self *parser) relExpr() (RelExpression, *Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.addExpr()
	if err != nil {
		return RelExpression{}, err
	}

	var otherOp RelOperator

	switch self.currToken.Kind {
	case LessThan:
		otherOp = RelLessThan
	case LessThanEqual:
		otherOp = RelLessOrEqual
	case GreaterThan:
		otherOp = RelGreaterThan
	case GreaterThanEqual:
		otherOp = RelGreaterOrEqual
	default:
		return RelExpression{
			Base:  base,
			Other: nil,
			Span: Span{
				Start: startLocation,
				End:   self.currToken.EndLocation,
			},
		}, nil
	}
	other, err := self.addExpr()
	if err != nil {
		return RelExpression{}, err
	}
	return RelExpression{
		Base: AddExpression{},
		Other: &struct {
			RelOperator RelOperator
			Other       AddExpression
		}{
			RelOperator: otherOp,
			Other:       other,
		},
		Span: Span{},
	}, nil
}

func (self *parser) addExpr() (AddExpression, *Error) {
	return AddExpression{}, nil
}

func (self *parser) letStmt() (LetStmt, *Error) {
	startLocation := self.currToken.StartLocation
	// Skip the `let`
	if err := self.advance(); err != nil {
		return LetStmt{}, err
	}

	if self.currToken.Kind != Identifier {
		return LetStmt{}, &Error{
			Kind:    SyntaxError,
			Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
			Span: Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	// Copy the assignment identifier name
	assignIdentifier := self.currToken.Value
	if err := self.advance(); err != nil {
		return LetStmt{}, err
	}
	if self.currToken.Kind != Assign {
		return LetStmt{}, &Error{
			Kind:    SyntaxError,
			Message: fmt.Sprintf("Expected assignment operator '=', found %v", self.currToken.Kind),
			Span: Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	if err := self.advance(); err != nil {
		return LetStmt{}, err
	}
	assignExpression, err := self.expression()
	if err != nil {
		return LetStmt{}, err
	}
	letStmt := LetStmt{
		Left:  assignIdentifier,
		Right: assignExpression,
		Range: Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	// TODO: does this belong here?
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}
	return letStmt, nil
}

func (self *parser) importStmt() (ImportStmt, *Error) {
	startLocation := self.currToken.StartLocation
	// Skip the `import`
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}

	if self.currToken.Kind != Identifier {
		return ImportStmt{}, &Error{
			Kind:    SyntaxError,
			Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
			Span: Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	functionName := self.currToken.Value
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}

	var rewriteName *string
	if self.currToken.Kind == As {
		if err := self.advance(); err != nil {
			return ImportStmt{}, err
		}
		if self.currToken.Kind != Identifier {
			return ImportStmt{}, &Error{
				Kind:    SyntaxError,
				Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
				Span: Span{
					Start: self.currToken.StartLocation,
					End:   self.currToken.EndLocation,
				},
			}
		}
		rewriteName = &self.currToken.Value
		if err := self.advance(); err != nil {
			return ImportStmt{}, err
		}
	}

	if self.currToken.Kind != From {
		return ImportStmt{}, &Error{
			Kind:    SyntaxError,
			Message: fmt.Sprintf("Expected 'from', found %v", self.currToken.Kind),
			Span: Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}
	if self.currToken.Kind != Identifier {
		return ImportStmt{}, &Error{
			Kind:    SyntaxError,
			Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
			Span: Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	importStmt := ImportStmt{
		Function:   functionName,
		RewriteAs:  rewriteName,
		FromModule: self.currToken.Value,
		Range: Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}
	return importStmt, nil
}

func (self *parser) breakStmt() (BreakStmt, *Error) {
	startLocation := self.currToken.StartLocation
	// Skip the break
	if err := self.advance(); err != nil {
		return BreakStmt{}, err
	}
	// Check for an additional expression
	shouldMakeAdditionExpr := false
	for _, possible := range firstExpr {
		if possible == self.currToken.Kind {
			shouldMakeAdditionExpr = true
			break
		}
	}
	var additionalExpr *Expression
	if shouldMakeAdditionExpr {
		additionalExprTemp, err := self.expression()
		if err != nil {
			return BreakStmt{}, err
		}
		additionalExpr = &additionalExprTemp
	}
	breakStmt := BreakStmt{
		Expression: additionalExpr,
		Range: Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	if err := self.advance(); err != nil {
		return BreakStmt{}, err
	}
	return breakStmt, nil
}

func (self *parser) continueStmt() (ContinueStmt, *Error) {
	continueStmt := ContinueStmt{
		Range: Span{
			Start: self.currToken.StartLocation,
			End:   self.currToken.EndLocation,
		},
	}
	if err := self.advance(); err != nil {
		return ContinueStmt{}, err
	}
	return continueStmt, nil
}

func (self *parser) returnStmt() (ReturnStmt, *Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return ReturnStmt{}, err
	}
	// Check for an additional expression
	shouldMakeAdditionExpr := false
	for _, possible := range firstExpr {
		if possible == self.currToken.Kind {
			shouldMakeAdditionExpr = true
			break
		}
	}
	var additionalExpr *Expression
	if shouldMakeAdditionExpr {
		additionalExprTemp, err := self.expression()
		if err != nil {
			return ReturnStmt{}, err
		}
		additionalExpr = &additionalExprTemp
	}
	returnStmt := ReturnStmt{
		Expression: additionalExpr,
		Range: Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	if err := self.advance(); err != nil {
		return ReturnStmt{}, err
	}
	return returnStmt, nil
}

func (self *parser) statement() (Statement, *Error) {
	switch self.currToken.Kind {
	case Let:
		return self.letStmt()
	case Import:
		return self.importStmt()
	case Break:
		return self.breakStmt()
	case Continue:
		return self.continueStmt()
	case Return:
		return self.returnStmt()
	default:
		canBeExpr := false
		for _, possible := range firstExpr {
			if self.currToken.Kind == possible {
				canBeExpr = true
				break
			}
		}
		if canBeExpr {
			expr, err := self.expression()
			if err != nil {
				return nil, err
			}
			return ExpressionStmt{
				Expression: expr,
				Range:      expr.Span,
			}, nil
		}
		return nil, &Error{
			Kind:    SyntaxError,
			Message: fmt.Sprintf("Expected one of the tokens 'let', 'import', 'break', 'continue', 'return', found %v", self.currToken.Kind),
			Span: Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
}

func (self *parser) block() (Block, *Error) {
	statements := make([]Statement, 0)

	for self.currToken.Kind != EOF {
		statement, err := self.statement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, statement)

		// Check for semicolon after a statement
		if self.currToken.Kind != Semicolon {
			return nil, &Error{
				Kind:    SyntaxError,
				Message: fmt.Sprintf("Expected semicolon, found %v", self.currToken.Kind),
				Span: Span{
					Start: self.currToken.StartLocation,
					End:   self.currToken.EndLocation,
				},
			}
		}
		// Advance to the next statement
		if err := self.advance(); err != nil {
			return nil, err
		}
	}
	return statements, nil
}

func (self *parser) parse() (Block, []Error) {
	if err := self.advance(); err != nil {
		self.errors = append(self.errors, *err)
		return nil, self.errors
	}
	block, err := self.block()
	if err != nil {
		self.errors = append(self.errors, *err)
		return nil, self.errors
	}
	return block, nil
}
