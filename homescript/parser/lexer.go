package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/util"
)

//
// Lexer
//

type Lexer struct {
	currentIndex int
	currentChar  *rune
	nextChar     *rune
	program      []rune
	location     errors.Location
	filename     string
}

func NewLexer(program_source string, filename string) Lexer {
	program := []rune(program_source)
	programLen := len(program)
	var currentChar *rune
	var nextChar *rune

	if programLen == 0 {
		currentChar = nil
		nextChar = nil
	} else if programLen == 1 {
		currentChar = &program[0]
		nextChar = nil
	} else {
		currentChar = &program[0]
		nextChar = &program[1]
	}

	lexer := Lexer{
		currentIndex: 0,
		currentChar:  currentChar,
		nextChar:     nextChar,
		program:      program,
		location: errors.Location{
			Index:  0,
			Line:   1,
			Column: 1,
		},
		filename: filename,
	}
	return lexer
}

func (self *Lexer) advance() {
	// ddvance location
	self.location.Advance(self.currentChar != nil && *self.currentChar == '\n')

	// ddvance current & next char
	self.currentIndex++
	programLen := len(self.program)

	if int(self.currentIndex) >= programLen {
		self.currentChar = nil
	} else {
		self.currentChar = &self.program[self.currentIndex]
	}

	if int(self.currentIndex+1) >= programLen {
		self.nextChar = nil
	} else {
		self.nextChar = &self.program[self.currentIndex+1]
	}
}

func (self *Lexer) skipLineComment() {
	self.advance()
	self.advance()

	for self.currentChar != nil && *self.currentChar != '\n' {
		self.advance()
	}

	self.advance()
}

func (self *Lexer) skipBlockComment() {
	self.advance()
	self.advance()

	for {
		if self.currentChar == nil || self.nextChar == nil {
			break
		}
		if *self.currentChar == '*' && *self.nextChar == '/' {
			self.advance()
			self.advance()
			break
		}

		// skip any other character of this comment
		self.advance()
	}
}

func (self *Lexer) NextToken() (Token, *errors.Error) {
outer:
	for self.currentChar != nil {
		switch *self.currentChar {
		case ' ', '\n', '\t' | '\r':
			self.advance()
		case '?':
			return self.makeSingleChar(QuestionMark, '?'), nil
		case '\'', '"':
			return self.makeString()
		case ';':
			return self.makeSingleChar(Semicolon, ';'), nil
		case ',':
			return self.makeSingleChar(Comma, ','), nil
		case ':':
			return self.makeSingleChar(Colon, ':'), nil
		case '.':
			return self.makeDots(), nil
		case '=':
			return self.makeEquals(), nil
		case '(':
			return self.makeSingleChar(LParen, '('), nil
		case ')':
			return self.makeSingleChar(RParen, ')'), nil
		case '{':
			return self.makeSingleChar(LCurly, '{'), nil
		case '}':
			return self.makeSingleChar(RCurly, '}'), nil
		case '[':
			return self.makeSingleChar(LBracket, '['), nil
		case ']':
			return self.makeSingleChar(RBracket, ']'), nil
		case '|':
			return self.makeOr(), nil
		case '&':
			return self.makeAnd(), nil
		case '^':
			return self.makeBitXor(), nil
		case '!':
			return self.makeNot(), nil
		case '<':
			return self.makeLess(), nil
		case '>':
			return self.makeGreater(), nil
		case '+':
			return self.makePlus(), nil
		case '-':
			return self.makeMinus(), nil
		case '*':
			return self.makeStar(), nil
		case '/':
			if self.nextChar != nil {
				switch *self.nextChar {
				case '/':
					self.skipLineComment()
					continue outer
				case '*':
					self.skipBlockComment()
					continue outer
				}
			}
			return self.makeDiv(), nil
		case '%':
			return self.makeReminder(), nil
		default:
			if util.IsDigit(*self.currentChar) {
				return self.makeNumber(), nil
			}
			if util.IsLetter(*self.currentChar) {
				return self.makeName(), nil
			}
			return unknownToken(self.location), errors.NewError(errors.Span{
				Start:    self.location,
				End:      self.location,
				Filename: self.filename,
			}, fmt.Sprintf("illegal character: %c", *self.currentChar), errors.SyntaxError)
		}
	}
	return newToken(
		EOF,
		"EOF",
		errors.Span{
			Start:    self.location,
			End:      self.location,
			Filename: self.filename,
		},
	), nil
}

func (self *Lexer) makeString() (Token, *errors.Error) {
	startLocation := self.location
	startQuote := *self.currentChar
	var value_buf []rune

	// skip opening quote
	self.advance()

	for self.currentChar != nil {
		if *self.currentChar == startQuote {
			break
		}
		if *self.currentChar == '\\' {
			char, err := self.makeEscapeSequence()
			if err != nil {
				return unknownToken(startLocation), err
			}
			value_buf = append(value_buf, char)
		} else {
			value_buf = append(value_buf, *self.currentChar)
			self.advance()
		}
	}

	// check for closing quote
	if self.currentChar == nil {
		return unknownToken(startLocation), errors.NewError(errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		}, "String literal never closed", errors.SyntaxError)
	}

	token := newToken(
		String,
		string(value_buf),
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)

	// skip closing quote
	self.advance()
	return token, nil
}

func (self *Lexer) makeEscapeSequence() (rune, *errors.Error) {
	startLocation := self.location
	self.advance()
	if self.currentChar == nil {
		return ' ', errors.NewError(errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		}, "Unfinished escape sequence", errors.SyntaxError)
	}

	var char rune
	var err *errors.Error
	switch *self.currentChar {
	case '\\':
		char = '\\'
		self.advance()
	case '\'':
		char = '\''
		self.advance()
	case '"':
		char = '"'
		self.advance()
	case 'b':
		char = '\b'
		self.advance()
	case 'n':
		char = '\n'
		self.advance()
	case 'r':
		char = '\r'
		self.advance()
	case 't':
		char = '\t'
		self.advance()
	case 'x':
		char, err = self.escapePart("", startLocation, 16, 2)
	case 'u':
		char, err = self.escapePart("", startLocation, 16, 4)
	case 'U':
		char, err = self.escapePart("", startLocation, 16, 8)
	default:
		if util.IsOctalDigit(*self.currentChar) {
			char, err = self.escapePart(string(*self.currentChar), startLocation, 8, 2)
		} else {
			err = errors.NewError(errors.Span{
				Start:    startLocation,
				End:      self.location,
				Filename: self.filename,
			}, "Invalid escape sequence", errors.SyntaxError)
		}
	}
	return char, err
}

func (self *Lexer) escapePart(esc string, startLocation errors.Location, radix int, digits uint8) (rune, *errors.Error) {
	self.advance()
	var digitFun func(rune) bool
	if radix == 16 {
		digitFun = util.IsHexDigit
	} else {
		digitFun = util.IsOctalDigit
	}
	for i := 0; i < int(digits); i++ {
		if self.currentChar == nil || !digitFun(*self.currentChar) {
			return ' ', errors.NewError(errors.Span{
				Start:    startLocation,
				End:      self.location,
				Filename: self.filename,
			}, "Invalid escape sequence", errors.SyntaxError)
		}
		esc += string(*self.currentChar)
		self.advance()
	}
	code, _ := strconv.ParseInt(esc, radix, 32)
	return rune(code), nil
}

func (self *Lexer) makeNumber() Token {
	startLocation := self.location
	value := string(*self.currentChar)
	kind := Int

	self.advance()

	for self.currentChar != nil && *self.currentChar == '_' {
		self.advance()
	}

	lastEnd := startLocation
	for self.currentChar != nil && util.IsDigit(*self.currentChar) {
		value += string(*self.currentChar)
		lastEnd = self.location
		self.advance()
	}

	if self.currentChar != nil && *self.currentChar == '.' && self.nextChar != nil && util.IsDigit(*self.nextChar) {
		kind = Float

		value += string(*self.currentChar)
		self.advance()
		for self.currentChar != nil && util.IsDigit(*self.currentChar) {
			value += string(*self.currentChar)
			lastEnd = self.location
			self.advance()
		}
	} else if self.currentChar != nil && *self.currentChar == 'f' {
		self.advance()
		kind = Float // this number is now a float
	}

	return newToken(
		kind,
		strings.ReplaceAll(value, "_", ""),
		errors.Span{
			Start:    startLocation,
			End:      lastEnd,
			Filename: self.filename,
		},
	)
}

func (self *Lexer) makeSingleChar(kind TokenKind, value rune) Token {
	token := newToken(
		kind,
		string(value),
		errors.Span{
			Start:    self.location,
			End:      self.location,
			Filename: self.filename,
		},
	)
	self.advance()
	return token
}

func (self *Lexer) makeDots() Token {
	startLocation := self.location

	var tokenKind TokenKind
	var tokenKindValue string

	if self.nextChar != nil && *self.nextChar == '.' {
		tokenKind = DoubleDot
		tokenKindValue = ".."
		self.advance()
	} else {
		tokenKind = Dot
		tokenKindValue = "."
	}

	token := newToken(
		tokenKind,
		tokenKindValue,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)

	self.advance()
	return token
}

func (self *Lexer) makeEquals() Token {
	startLocation := self.location

	if self.nextChar != nil {
		switch *self.nextChar {
		case '>':
			self.advance()

			token := newToken(
				FatArrow,
				"=>",
				errors.Span{
					Start:    startLocation,
					End:      self.location,
					Filename: self.filename,
				},
			)

			self.advance()
			return token
		case '=':
			self.advance()

			token := newToken(
				Equal,
				"==",
				errors.Span{
					Start:    startLocation,
					End:      self.location,
					Filename: self.filename,
				},
			)

			self.advance()
			return token
		}
	}

	self.advance()

	return newToken(
		Assign,
		"=",
		errors.Span{
			Start:    startLocation,
			End:      startLocation,
			Filename: self.filename,
		},
	)
}

func (self *Lexer) makeOr() Token {
	startLocation := self.location
	self.advance()

	tokenKind := BitOr
	value := "|"

	if self.currentChar != nil {
		switch *self.currentChar {
		case '|':
			tokenKind = Or
			value = "||"
			self.advance()
		case '=':
			tokenKind = BitOrAssign
			value = "|="
			self.advance()
		}
	}

	token := newToken(
		tokenKind,
		value,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)
	self.advance()
	return token

}

func (self *Lexer) makeAnd() Token {
	startLocation := self.location
	self.advance()

	tokenKind := BitAnd
	value := "&"

	if self.currentChar != nil {
		switch *self.currentChar {
		case '&':
			tokenKind = And
			value = "&&"
			self.advance()
		case '=':
			tokenKind = BitAndAssign
			value = "&="
			self.advance()
		}
	}

	token := newToken(
		tokenKind,
		value,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)
	self.advance()
	return token
}

func (self *Lexer) makeBitXor() Token {
	startLocation := self.location
	self.advance()

	tokenKind := BitXor
	value := "^"

	if self.currentChar != nil && *self.currentChar == '=' {
		tokenKind = BitXorAssign
		value = "^="
		self.advance()
	}

	return newToken(
		tokenKind,
		value,
		errors.Span{
			Start: startLocation,
			End:   self.location,
		},
	)
}

func (self *Lexer) makeNot() Token {
	startLocation := self.location

	tokenKind := Not
	tokenValue := "!"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = NotEqual
		tokenValue = "!="
		self.advance()
	}

	token := newToken(
		tokenKind,
		tokenValue,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)
	self.advance()
	return token
}

func (self *Lexer) makeLess() Token {
	startLocation := self.location

	tokenKind := LessThan
	tokenValue := "<"

	if self.nextChar != nil {
		switch *self.nextChar {
		case '<':
			tokenKind = ShiftLeft
			tokenValue = "<<"
			self.advance()

			if self.nextChar != nil && *self.nextChar == '=' {
				tokenKind = ShiftLeftAssign
				tokenValue = "<<="
				self.advance()
			}
		case '=':
			tokenKind = LessThanEqual
			tokenValue = "<="
			self.advance()
		}
	}

	token := newToken(
		tokenKind,
		tokenValue,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)
	self.advance()
	return token
}

func (self *Lexer) makeGreater() Token {
	startLocation := self.location

	tokenKind := GreaterThan
	tokenValue := ">"

	if self.nextChar != nil {
		switch *self.nextChar {
		case '>':
			tokenKind = ShiftRight
			tokenValue = ">>"
			self.advance()

			if self.nextChar != nil && *self.nextChar == '=' {
				tokenKind = ShiftRightAssign
				tokenValue = ">>="
				self.advance()
			}
		case '=':
			tokenKind = GreaterThanEqual
			tokenValue = ">="
			self.advance()
		}
	}

	token := newToken(
		tokenKind,
		tokenValue,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)
	self.advance()
	return token
}

func (self *Lexer) makePlus() Token {
	startLocation := self.location

	tokenKind := Plus
	tokenValue := "+"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = PlusAssign
		tokenValue = "+="
		self.advance()
	}

	token := newToken(
		tokenKind,
		tokenValue,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)
	self.advance()
	return token
}

func (self *Lexer) makeMinus() Token {
	startLocation := self.location

	tokenKind := Minus
	tokenValue := "-"

	if self.nextChar != nil {
		switch *self.nextChar {
		case '=':
			tokenKind = MinusAssign
			tokenValue = "-="
			self.advance()
		case '>':
			self.advance()
			tokenKind = Arrow
			tokenValue = "->"
		}
	}

	token := newToken(
		tokenKind,
		tokenValue,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)

	self.advance()
	return token
}

func (self *Lexer) makeStar() Token {
	startLocation := self.location
	self.advance()

	if self.currentChar != nil && *self.currentChar == '=' {
		token := newToken(
			MultiplyAssign,
			"*=",
			errors.Span{
				Start:    startLocation,
				End:      self.location,
				Filename: self.filename,
			},
		)
		self.advance()
		return token
	}
	if self.currentChar != nil && *self.currentChar == '*' {
		if self.nextChar != nil && *self.nextChar == '=' {
			self.advance()
			token := newToken(
				PowerAssign,
				"**=",
				errors.Span{
					Start:    startLocation,
					End:      self.location,
					Filename: self.filename,
				},
			)
			self.advance()
			return token
		}

		token := newToken(
			Power,
			"**",
			errors.Span{
				Start:    startLocation,
				End:      self.location,
				Filename: self.filename,
			},
		)

		self.advance()
		return token
	}

	return newToken(
		Multiply,
		"*",
		errors.Span{
			Start:    startLocation,
			End:      startLocation,
			Filename: self.filename,
		},
	)
}

func (self *Lexer) makeDiv() Token {
	startLocation := self.location

	tokenKind := Divide
	tokenValue := "/"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = DivideAssign
		tokenValue = "/="
		self.advance()
	}

	token := newToken(
		tokenKind,
		tokenValue,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)

	self.advance()
	return token
}

func (self *Lexer) makeReminder() Token {
	startLocation := self.location

	tokenKind := Modulo
	tokenValue := "%"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = ModuloAssign
		tokenValue = "%="
		self.advance()
	}

	token := newToken(
		tokenKind,
		tokenValue,
		errors.Span{
			Start:    startLocation,
			End:      self.location,
			Filename: self.filename,
		},
	)

	self.advance()
	return token
}

func (self *Lexer) makeName() Token {
	startLocation := self.location
	value := string(*self.currentChar)
	self.advance()

	lastEnd := startLocation
	for self.currentChar != nil && (util.IsDigit(*self.currentChar) || util.IsLetter(*self.currentChar)) {
		value += string(*self.currentChar)
		lastEnd = self.location
		self.advance()
	}

	var tokenKind TokenKind
	switch value {
	case "true", "on":
		tokenKind = True
	case "false", "off":
		tokenKind = False
	case "null":
		tokenKind = Null
	case "none":
		tokenKind = None
	case "pub":
		tokenKind = Pub
	case "fn":
		tokenKind = Fn
	case "if":
		tokenKind = If
	case "else":
		tokenKind = Else
	case "match":
		tokenKind = Match
	case "for":
		tokenKind = For
	case "while":
		tokenKind = While
	case "loop":
		tokenKind = Loop
	case "break":
		tokenKind = Break
	case "continue":
		tokenKind = Continue
	case "return":
		tokenKind = Return
	case "import":
		tokenKind = Import
	case "as":
		tokenKind = As
	case "from":
		tokenKind = From
	case "let":
		tokenKind = Let
	case "in":
		tokenKind = In
	case "type":
		tokenKind = Type
	case "try":
		tokenKind = Try
	case "catch":
		tokenKind = Catch
	case "new":
		tokenKind = New
	case "event":
		tokenKind = Event
	case "_":
		tokenKind = Underscore
	default:
		tokenKind = Identifier
	}

	return newToken(
		tokenKind,
		value,
		errors.Span{
			Start:    startLocation,
			End:      lastEnd,
			Filename: self.filename,
		},
	)
}
