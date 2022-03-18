package homescript

import "errors"

type Lexer struct {
	CurrentChar  *rune
	CurrentIndex uint32
	Input        []rune
}

var (
	ErrQuotesNotClosed = errors.New("String literal was never closed")
)

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
		case ' ' | '\t' | '\r':
			self.advance()
		case '"' | '\'':
			return self.makeString()
		}
	}
	return Token{}, nil
}

func (self *Lexer) makeString() (Token, error) {
	startQuote := self.CurrentChar
	var value string

	self.advance() // Skip opening quote
	for self.CurrentChar != nil && *self.CurrentChar != *startQuote {
		value += string(*self.CurrentChar)
		self.advance()
	}

	// Check for closing quote
	if self.CurrentChar == nil {
		return Token{}, ErrQuotesNotClosed
	}

	self.advance() // Skip closing quote
	return Token{
		TokenType: String,
		Value:     value,
	}, nil
}

func (self *Lexer) advance() {
	self.CurrentIndex += 1
	if self.CurrentIndex >= uint32(len(self.Input)) {
		self.CurrentChar = nil
		return
	}
	self.CurrentChar = &self.Input[self.CurrentIndex]
}
