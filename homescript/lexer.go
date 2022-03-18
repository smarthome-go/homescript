package homescript

import (
	"errors"
	"fmt"
)

type Lexer struct {
	CurrentChar  *rune
	CurrentIndex uint32
	Input        []rune
}

func NewLexer(input string) Lexer {
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
	}
}

func (self *Lexer) Scan() (Token, error) {
	for self.CurrentChar != nil {
		switch *self.CurrentChar {
		case ' ':
			self.advance()
		case '\t':
			self.advance()
		case '\r':
			self.advance()
		case '"':
			return self.makeString()
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
			return Token{}, errors.New(fmt.Sprintf("Illegal character: %c", *self.CurrentChar))
		}
		self.advance()
	}
	return Token{
		TokenType: EOF,
		Value:     "EOF",
	}, nil
}

func (self *Lexer) makeName() Token {
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
	}

}

func (self *Lexer) makeNumber() Token {
	value := string(*self.CurrentChar)
	self.advance()
	for self.CurrentChar != nil && isDigit(*self.CurrentChar) {
		value += string(*self.CurrentChar)
		self.advance()
	}
	return Token{
		TokenType: Number,
		Value:     value,
	}
}

func (self *Lexer) makeOptionalEquals(standardTokenType TokenType, withEqualTokenType TokenType) Token {
	char := *self.CurrentChar
	self.advance()
	if self.CurrentChar != nil && *self.CurrentChar == '=' {
		self.advance()
		return Token{
			TokenType: withEqualTokenType,
			Value:     string(char) + "=",
		}
	}
	return Token{
		TokenType: standardTokenType,
		Value:     string(char),
	}
}

func (self *Lexer) makeDoubleChar(char rune, tokenType TokenType) (Token, error) {
	self.advance()
	if self.CurrentChar == nil || *self.CurrentChar != char {
		return Token{}, errors.New(fmt.Sprintf("Expected character: %c, found: %c", char, *self.CurrentChar))
	}
	self.advance()
	return Token{
		TokenType: tokenType,
		Value:     string(char) + string(char),
	}, nil
}

func (self *Lexer) makeSingleChar(tokenType TokenType) Token {
	char := *self.CurrentChar
	self.advance()
	return Token{
		TokenType: tokenType,
		Value:     string(char),
	}
}

func (self *Lexer) makeString() (Token, error) {
	startQuote := *self.CurrentChar
	fmt.Printf("start quote: %c", startQuote)
	var value string

	fmt.Printf("%c\n", *self.CurrentChar)
	self.advance() // Skip opening quote
	for self.CurrentChar != nil {
		if *self.CurrentChar == startQuote {
			fmt.Println("start quote reached")
			break
		}
		value += string(*self.CurrentChar)
		fmt.Printf("%c\n", *self.CurrentChar)
		self.advance()
	}

	// Check for closing quote
	if self.CurrentChar == nil {
		return Token{}, errors.New("String literal never closed")
	}

	self.advance() // Skip closing quote
	return Token{
		TokenType: String,
		Value:     value,
	}, nil
}

func (self *Lexer) skipComment() {
	self.advance()
	for self.CurrentChar != nil && *self.CurrentChar != '\n' {
		self.advance()
	}
}

func (self *Lexer) advance() {
	self.CurrentIndex += 1
	if self.CurrentIndex >= uint32(len(self.Input)) {
		self.CurrentChar = nil
		return
	}
	self.CurrentChar = &self.Input[self.CurrentIndex]
}
