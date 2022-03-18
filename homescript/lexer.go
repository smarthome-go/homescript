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

func (l *Lexer) Scan() (Token, error) {
	for l.CurrentChar != nil {
		switch *l.CurrentChar {
		case ' ':
			l.advance()
		case '\t':
			l.advance()
		case '\r':
			l.advance()
		case '"':
			return l.makeString()
		case '\'':
			return l.makeString()
		case '#':
			l.skipComment()
		case '|':
			return l.makeDoubleChar('|', Or)
		case '&':
			return l.makeDoubleChar('&', And)
		case '=':
			return l.makeDoubleChar('=', Equal)
		case '!':
			return l.makeOptionalEquals(Not, NotEqual), nil
		case '<':
			return l.makeOptionalEquals(LessThan, LessThanOrEqual), nil
		case '>':
			return l.makeOptionalEquals(GreaterThan, GreaterThanOrEqual), nil
		case '(':
			return l.makeSingleChar(LeftParenthesis), nil
		case ')':
			return l.makeSingleChar(RightParenthesis), nil
		case '{':
			return l.makeSingleChar(LeftCurlyBrace), nil
		case '}':
			return l.makeSingleChar(RightCurlyBrace), nil
		case ',':
			return l.makeSingleChar(Comma), nil
		case '\n':
			return l.makeSingleChar(EOL), nil
		default:
			if isDigit(*l.CurrentChar) {
				return l.makeNumber(), nil
			}
			if isLetter(*l.CurrentChar) {
				return l.makeName(), nil
			}
			return Token{}, errors.New(fmt.Sprintf("Illegal character: %c", *l.CurrentChar))
		}
		l.advance()
	}
	return Token{
		TokenType: EOF,
		Value:     "EOF",
	}, nil
}

func (l *Lexer) makeName() Token {
	value := string(*l.CurrentChar)
	l.advance()
	for l.CurrentChar != nil && isLetter(*l.CurrentChar) {
		value += string(*l.CurrentChar)
		l.advance()
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

func (l *Lexer) makeNumber() Token {
	value := string(*l.CurrentChar)
	l.advance()
	for l.CurrentChar != nil && isDigit(*l.CurrentChar) {
		value += string(*l.CurrentChar)
		l.advance()
	}
	return Token{
		TokenType: Number,
		Value:     value,
	}
}

func (l *Lexer) makeOptionalEquals(standardTokenType TokenType, withEqualTokenType TokenType) Token {
	char := *l.CurrentChar
	l.advance()
	if l.CurrentChar != nil && *l.CurrentChar == '=' {
		l.advance()
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

func (l *Lexer) makeDoubleChar(char rune, tokenType TokenType) (Token, error) {
	l.advance()
	if l.CurrentChar == nil || *l.CurrentChar != char {
		return Token{}, errors.New(fmt.Sprintf("Expected character: %c, found: %c", char, *l.CurrentChar))
	}
	l.advance()
	return Token{
		TokenType: tokenType,
		Value:     string(char) + string(char),
	}, nil
}

func (l *Lexer) makeSingleChar(tokenType TokenType) Token {
	char := *l.CurrentChar
	l.advance()
	return Token{
		TokenType: tokenType,
		Value:     string(char),
	}
}

func (l *Lexer) makeString() (Token, error) {
	startQuote := *l.CurrentChar
	fmt.Printf("start quote: %c", startQuote)
	var value string

	fmt.Printf("%c\n", *l.CurrentChar)
	l.advance() // Skip opening quote
	for l.CurrentChar != nil {
		if *l.CurrentChar == startQuote {
			fmt.Println("start quote reached")
			break
		}
		value += string(*l.CurrentChar)
		fmt.Printf("%c\n", *l.CurrentChar)
		l.advance()
	}

	// Check for closing quote
	if l.CurrentChar == nil {
		return Token{}, errors.New("String literal never closed")
	}

	l.advance() // Skip closing quote
	return Token{
		TokenType: String,
		Value:     value,
	}, nil
}

func (l *Lexer) skipComment() {
	l.advance()
	for l.CurrentChar != nil && *l.CurrentChar != '\n' {
		l.advance()
	}
}

func (l *Lexer) advance() {
	l.CurrentIndex += 1
	if l.CurrentIndex >= uint32(len(l.Input)) {
		l.CurrentChar = nil
		return
	}
	l.CurrentChar = &l.Input[l.CurrentIndex]
}
