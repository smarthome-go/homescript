package homescript

import (
	"fmt"

	"github.com/MikMuellerDev/homescript/homescript/error"
)

type Lexer struct {
	CurrentChar  *rune
	CurrentIndex uint
	Input        []rune
	Location     error.Location
}

func NewLexer(filename string, input string) Lexer {
	var currentChar *rune
	if input == "" {
		currentChar = nil
	} else {
		currentChar = &[]rune(input)[0]
	}
	return Lexer{
		CurrentChar:  currentChar,
		CurrentIndex: 0,
		Input:        []rune(input),
		Location:     error.NewLocation(filename),
	}
}

func (self *Lexer) Scan() (Token, *error.Error) {
	for self.CurrentChar != nil {
		switch *self.CurrentChar {
		case ' ':
			fallthrough
		case '\t':
			fallthrough
		case '\r':
			self.advance()
		case '"':
			fallthrough
		case '\'':
			return self.makeString()
		case '#':
			self.skipComment()
		case '|':
			return self.makeDoubleChar('|', Or)
		case '&':
			return self.makeDoubleChar('&', And)
		case '=':
			return self.makeDoubleChar('=', Equal)
		case '!':
			return self.makeOptionalEquals(Not, NotEqual), nil
		case '<':
			return self.makeOptionalEquals(LessThan, LessThanOrEqual), nil
		case '>':
			return self.makeOptionalEquals(GreaterThan, GreaterThanOrEqual), nil
		case '(':
			return self.makeSingleChar(LeftParenthesis), nil
		case ')':
			return self.makeSingleChar(RightParenthesis), nil
		case '{':
			return self.makeSingleChar(LeftCurlyBrace), nil
		case '}':
			return self.makeSingleChar(RightCurlyBrace), nil
		case ',':
			return self.makeSingleChar(Comma), nil
		case '\n':
			return self.makeSingleChar(EOL), nil
		default:
			if isDigit(*self.CurrentChar) {
				return self.makeNumber(), nil
			}
			if isLetter(*self.CurrentChar) {
				return self.makeName(), nil
			}
			return UnknownToken(self.Location), error.NewError(
				error.SyntaxError,
				self.Location,
				fmt.Sprintf("Illegal character: %c", *self.CurrentChar),
			)
		}
	}
	return Token{
		TokenType: EOF,
		Value:     "EOF",
		Location:  self.Location,
	}, nil
}

func (self *Lexer) makeName() Token {
	location := self.Location
	value := string(*self.CurrentChar)
	self.advance()
	for self.CurrentChar != nil && isLetter(*self.CurrentChar) {
		value += string(*self.CurrentChar)
		self.advance()
	}
	var tokenType TokenType
	switch value {
	case "true":
		tokenType = True
	case "on":
		tokenType = True
	case "false":
		tokenType = False
	case "off":
		tokenType = False
	case "if":
		tokenType = If
	case "else":
		tokenType = Else
	default:
		tokenType = Identifier
	}
	return Token{
		TokenType: tokenType,
		Value:     value,
		Location:  location,
	}

}

func (self *Lexer) makeNumber() Token {
	location := self.Location
	value := string(*self.CurrentChar)
	self.advance()
	for self.CurrentChar != nil && isDigit(*self.CurrentChar) {
		value += string(*self.CurrentChar)
		self.advance()
	}
	return Token{
		TokenType: Number,
		Value:     value,
		Location:  location,
	}
}

func (self *Lexer) makeOptionalEquals(standardTokenType TokenType, withEqualTokenType TokenType) Token {
	location := self.Location
	char := *self.CurrentChar
	self.advance()
	if self.CurrentChar != nil && *self.CurrentChar == '=' {
		self.advance()
		return Token{
			TokenType: withEqualTokenType,
			Value:     string(char) + "=",
			Location:  location,
		}
	}
	return Token{
		TokenType: standardTokenType,
		Value:     string(char),
		Location:  location,
	}
}

func (self *Lexer) makeDoubleChar(char rune, tokenType TokenType) (Token, *error.Error) {
	location := self.Location
	self.advance()
	if self.CurrentChar == nil || *self.CurrentChar != char {
		return UnknownToken(location), error.NewError(
			error.SyntaxError,
			location,
			fmt.Sprintf("Expected character: %c, found: %c", char, *self.CurrentChar),
		)
	}
	self.advance()
	return Token{
		TokenType: tokenType,
		Value:     string(char) + string(char),
		Location:  location,
	}, nil
}

func (self *Lexer) makeSingleChar(tokenType TokenType) Token {
	location := self.Location
	char := *self.CurrentChar
	self.advance()
	return Token{
		TokenType: tokenType,
		Value:     string(char),
		Location:  location,
	}
}

func (self *Lexer) makeString() (Token, *error.Error) {
	location := self.Location
	startQuote := *self.CurrentChar
	var value string

	self.advance() // Skip opening quote
	for self.CurrentChar != nil {
		if *self.CurrentChar == startQuote {
			break
		}
		value += string(*self.CurrentChar)
		self.advance()
	}

	// Check for closing quote
	if self.CurrentChar == nil {
		return UnknownToken(location), error.NewError(error.SyntaxError, location, "String literal never closed")
	}

	self.advance() // Skip closing quote
	return Token{
		TokenType: String,
		Value:     value,
		Location:  location,
	}, nil
}

func (self *Lexer) skipComment() {
	self.advance()
	for self.CurrentChar != nil && *self.CurrentChar != '\n' {
		self.advance()
	}
}

func (self *Lexer) advance() {
	self.Location.Advance(self.CurrentChar != nil && *self.CurrentChar == '\n')
	self.CurrentIndex += 1
	if self.CurrentIndex >= uint(len(self.Input)) {
		self.CurrentChar = nil
		return
	}
	self.CurrentChar = &self.Input[self.CurrentIndex]
}
