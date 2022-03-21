package homescript

import (
	"fmt"
	"strconv"

	"github.com/MikMuellerDev/homescript/homescript/error"
)

type Parser struct {
	Lexer        Lexer
	CurrentToken Token
	Errors       []error.Error
}

func NewParser(lexer Lexer) Parser {
	return Parser{
		Lexer:        lexer,
		CurrentToken: Token{},
		Errors:       make([]error.Error, 0),
	}
}

func (self *Parser) Parse() (Expressions, []error.Error) {
	self.advance()
	expressions, err := self.expressions()
	if err != nil {
		self.Errors = append(self.Errors, *err)
		return nil, self.Errors
	}
	if self.CurrentToken.TokenType != EOF {
		self.Errors = append(self.Errors, *error.NewError(
			error.SyntaxError,
			self.CurrentToken.Location,
			fmt.Sprintf("Expected EOF, got %s", self.CurrentToken.Value),
		))
	}
	if len(self.Errors) > 0 {
		return nil, self.Errors
	}
	return expressions, nil
}

func (self *Parser) expect(tokenType TokenType, name string) {
	if self.CurrentToken.TokenType != tokenType {
		self.Errors = append(self.Errors, *error.NewError(
			error.SyntaxError,
			self.CurrentToken.Location,
			fmt.Sprintf("Expected %s, found '%s'", name, self.CurrentToken.Value),
		))
	}
}

func (self *Parser) isType(tokenType TokenType) bool {
	return self.CurrentToken.TokenType == tokenType
}

func (self *Parser) isOfTypes(tokenTypes ...TokenType) bool {
	for _, currentType := range tokenTypes {
		if currentType == self.CurrentToken.TokenType {
			return true
		}
	}
	return false
}

func (self *Parser) expressions() (Expressions, *error.Error) {
	for self.isType(EOL) {
		self.advance()
	}
	if !self.isOfTypes(EOF, RightCurlyBrace) {
		expressions := make(Expressions, 0)
		expression, err := self.expression()
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expression)
		for {
			if self.isOfTypes(EOF, RightCurlyBrace) {
				break
			}
			self.expect(EOL, "line break")
			for self.isType(EOL) {
				self.advance()
			}
			if self.isOfTypes(EOF, RightCurlyBrace) {
				break
			}
			expression, err := self.expression()
			if err != nil {
				return nil, err
			}
			expressions = append(expressions, expression)
		}
		return expressions, nil
	}
	return make(Expressions, 0), nil
}

func (self *Parser) expression() (Expression, *error.Error) {
	base, err := self.andExpression()
	if err != nil {
		return Expression{}, err
	}
	following := make([]AndExpr, 0)
	for self.isType(Or) {
		self.advance()
		followingTemp, err := self.andExpression()
		if err != nil {
			return Expression{}, err
		}
		following = append(following, followingTemp)
	}
	return Expression{
		Base:      base,
		Following: following,
	}, nil
}

func (self *Parser) andExpression() (AndExpr, *error.Error) {
	base, err := self.eqExpr()
	if err != nil {
		return AndExpr{}, err
	}
	following := make([]EqExpr, 0)
	for self.isType(And) {
		self.advance()
		followingTemp, err := self.eqExpr()
		if err != nil {
			return AndExpr{}, err
		}
		following = append(following, followingTemp)
	}
	return AndExpr{
		Base:      base,
		Following: following,
	}, nil
}

func (self *Parser) eqExpr() (EqExpr, *error.Error) {
	base, err := self.relExpr()
	if err != nil {
		return EqExpr{}, err
	}
	if self.isOfTypes(Equal, NotEqual) {
		operator := self.CurrentToken.TokenType
		self.advance()
		other, err := self.relExpr()
		if err != nil {
			return EqExpr{}, err
		}
		return EqExpr{
			Base: base,
			Other: &struct {
				TokenType
				RelExpr
			}{
				TokenType: operator,
				RelExpr:   other,
			},
		}, nil
	}
	return EqExpr{
		Base:  base,
		Other: nil,
	}, nil
}

func (self *Parser) relExpr() (RelExpr, *error.Error) {
	location := self.CurrentToken.Location
	base, err := self.notExpr()
	if err != nil {
		return RelExpr{}, err
	}
	if self.isOfTypes(
		LessThan,
		LessThanOrEqual,
		GreaterThan,
		GreaterThanOrEqual,
	) {
		operator := self.CurrentToken.TokenType
		self.advance()
		other, err := self.notExpr()
		if err != nil {
			return RelExpr{}, err
		}
		return RelExpr{
			Base: base,
			Other: &struct {
				TokenType
				NotExpr
			}{
				TokenType: operator,
				NotExpr:   other,
			},
			Location: location,
		}, nil
	}
	return RelExpr{
		Base:     base,
		Other:    nil,
		Location: location,
	}, nil
}

func (self *Parser) notExpr() (NotExpr, *error.Error) {
	negated := false
	if self.isType(Not) {
		negated = true
		self.advance()
	}
	atom, err := self.atom()
	if err != nil {
		return NotExpr{}, err
	}
	return NotExpr{
		Negated: negated,
		Base:    atom,
	}, nil
}

func (self *Parser) atom() (Atom, *error.Error) {
	if self.isType(Number) {
		// TODO: handle possible errors
		value, _ := strconv.Atoi(self.CurrentToken.Value)
		self.advance()
		return AtomNumber{
			Num: value,
		}, nil
	}
	if self.isType(String) {
		value := self.CurrentToken.Value
		self.advance()
		return AtomString{
			Content: value,
		}, nil
	}
	if self.isOfTypes(True, False) {
		value := self.isType(True)
		self.advance()
		return AtomBoolean{
			Value: value,
		}, nil
	}
	if self.isType(If) {
		ifExpr, err := self.ifExpr()
		if err != nil {
			return nil, err
		}
		return AtomIf{
			IfExpr: ifExpr,
		}, nil
	}
	if self.isType(LeftParenthesis) {
		self.advance()
		expr, err := self.expression()
		if err != nil {
			return nil, err
		}
		self.expect(RightParenthesis, "')'")
		self.advance()
		return AtomExpr{
			Expr: expr,
		}, nil
	}
	if self.isType(Identifier) {
		location := self.CurrentToken.Location
		name := self.CurrentToken.Value
		self.advance()
		if self.isType(LeftParenthesis) {
			callExpr, err := self.callExpr(name, location)
			if err != nil {
				return nil, err
			}
			return AtomCall{
				CallExpr: callExpr,
			}, nil
		}
		return AtomIdentifier{
			Name:     name,
			Location: location,
		}, nil
	}
	return nil, error.NewError(
		error.SyntaxError,
		self.CurrentToken.Location,
		fmt.Sprintf("Expected expression, found '%s'", self.CurrentToken.Value),
	)
}

func (self *Parser) callExpr(name string, location error.Location) (CallExpr, *error.Error) {
	self.advance()
	args := make(Expressions, 0)
	if !self.isType(RightParenthesis) {
		argument, err := self.expression()
		if err != nil {
			return CallExpr{}, err
		}
		args = append(args, argument)

		for self.isType(Comma) {
			self.advance()
			argument, err := self.expression()
			if err != nil {
				return CallExpr{}, err
			}
			args = append(args, argument)
		}
	}
	self.expect(RightParenthesis, "')'")
	self.advance()
	return CallExpr{
		Name:      name,
		Arguments: args,
		Location:  location,
	}, nil
}

func (self *Parser) ifExpr() (IfExpr, *error.Error) {
	self.advance()
	condition, err := self.expression()
	if err != nil {
		return IfExpr{}, nil
	}
	self.expect(LeftCurlyBrace, "'{'")
	self.advance()
	ifBody, err := self.expressions()
	if err != nil {
		return IfExpr{}, err
	}
	self.expect(RightCurlyBrace, "'}'")
	self.advance()
	if self.isType(Else) {
		self.advance()
		self.expect(LeftCurlyBrace, "'{'")
		self.advance()
		elseBody, err := self.expressions()
		if err != nil {
			return IfExpr{}, err
		}
		self.expect(RightCurlyBrace, "'}'")
		self.advance()
		return IfExpr{
			Condition: condition,
			Body:      ifBody,
			ElseBody:  elseBody,
		}, nil
	}
	return IfExpr{
		Condition: condition,
		Body:      ifBody,
		ElseBody:  nil,
	}, nil
}

func (self *Parser) advance() {
	token, err := self.Lexer.Scan()
	if err != nil {
		self.Errors = append(self.Errors, *err)
	}
	self.CurrentToken = token
}
