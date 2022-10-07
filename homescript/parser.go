package homescript

import "fmt"

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

func (self *parser) letStmt() (LetStmt, *Error) {
	startLocation := self.currToken.StartLocation
	// Skip the `let`
	self.advance()

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
	self.advance()
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
	self.advance()
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
	self.advance()
	return letStmt, nil
}

func (self *parser) importStmt() (ImportStmt, *Error) {
	startLocation := self.currToken.StartLocation
	// Skip the `import`
	self.advance()

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
	self.advance()

	var rewriteName *string
	if self.currToken.Kind == As {
		self.advance()
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
		self.advance()
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
	self.advance()
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
	self.advance()
	return importStmt, nil
}

func (self *parser) breakStmt() (BreakStmt, *Error) {
	startLocation := self.currToken.StartLocation
	// Skip the break
	self.advance()
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
		additionalExpr, err := self.expression()
		if err != nil {
			return BreakStmt{}, err
		}
	}
	breakStmt := BreakStmt{
		Expression: additionalExpr,
		Range: Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	self.advance()
	return breakStmt, nil
}

func (self *parser) continueStmt() (ContinueStmt, *Error) {
	continueStmt := ContinueStmt{
		Range: Span{
			Start: self.currToken.StartLocation,
			End:   self.currToken.EndLocation,
		},
	}
	self.advance()
	return continueStmt, nil
}

func (self *parser) returnStmt() (ReturnStmt, *Error) {
	startLocation := self.currToken.StartLocation
	self.advance()
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
		additionalExpr, err := self.expression()
		if err != nil {
			return ReturnStmt{}, err
		}
	}
	returnStmt := ReturnStmt{
		Expression: additionalExpr,
		Range: Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	self.advance()
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
		self.advance()
	}
	return statements, nil
}

func (self *parser) parse() (Block, []Error) {
	block, err := self.block()
	if err != nil {
		self.errors = append(self.errors, *err)
		return nil, self.errors
	}
	return block, nil
}
