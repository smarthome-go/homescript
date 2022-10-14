package homescript

import (
	"fmt"
	"strconv"

	"github.com/smarthome-go/homescript/homescript/errors"
)

// Rune range helper functions
type runeRange struct {
	min int
	max int
}

func isRuneInRange(char rune, ranges ...runeRange) bool {
	intChar := int(char)
	for _, ran := range ranges {
		if intChar >= ran.min && intChar <= ran.max {
			return true
		}
	}
	return false
}

func isDigit(char rune) bool      { return isRuneInRange(char, runeRange{min: 48, max: 57}) }
func isOctalDigit(char rune) bool { return isRuneInRange(char, runeRange{min: 48, max: 55}) }
func isHexDigit(char rune) bool {
	return isRuneInRange(char, runeRange{min: 48, max: 57}, runeRange{min: 65, max: 70}, runeRange{min: 97, max: 102})
}
func isLetter(char rune) bool {
	return isRuneInRange(char, runeRange /* capital letters */ {min: 65, max: 90}, runeRange /* lowercase letters */ {min: 97, max: 122}, runeRange /* underscore */ {min: 95, max: 95})
}

// end rune range helper functions

type lexer struct {
	currentIndex int
	currentChar  *rune
	nextChar     *rune
	program      []rune
	location     errors.Location
}

func newLexer(filename string, program_source string) lexer {
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
	lexer := lexer{
		currentIndex: 0,
		currentChar:  currentChar,
		nextChar:     nextChar,
		program:      program,
		location: errors.Location{
			Line:   1,
			Column: 1,
			Index:  0,
		},
	}
	return lexer
}

func (self *lexer) advance() {
	// Advance current & next char
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

	// Advance location
	self.location.Index++
	if self.currentChar != nil && *self.currentChar == '\n' {
		self.location.Line++
		self.location.Column = 0
	} else {
		self.location.Column++
	}
}

func (self *lexer) makeString() (Token, *errors.Error) {
	startLocation := self.location
	startQuote := *self.currentChar
	var value_buf []rune
	self.advance() // Skip opening quote
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
	// Check for closing quote
	if self.currentChar == nil {
		return unknownToken(startLocation), errors.NewError(errors.Span{
			Start: startLocation,
			End:   self.location,
		}, "String literal never closed", errors.SyntaxError)
	}
	token := Token{
		Kind:          String,
		Value:         string(value_buf),
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	self.advance() // Skip closing quote
	return token, nil
}

func (self *lexer) makeEscapeSequence() (rune, *errors.Error) {
	startLocation := self.location
	self.advance()
	if self.currentChar == nil {
		return ' ', errors.NewError(errors.Span{
			Start: startLocation,
			End:   self.location,
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
		if isOctalDigit(*self.currentChar) {
			char, err = self.escapePart(string(*self.currentChar), startLocation, 8, 2)
		} else {
			err = errors.NewError(errors.Span{
				Start: startLocation,
				End:   self.location,
			}, "Invalid escape sequence", errors.SyntaxError)
		}
	}
	return char, err
}

func (self *lexer) escapePart(esc string, startLocation errors.Location, radix int, digits uint8) (rune, *errors.Error) {
	self.advance()
	var digitFun func(rune) bool
	if radix == 16 {
		digitFun = isHexDigit
	} else {
		digitFun = isOctalDigit
	}
	for i := 0; i < int(digits); i++ {
		if self.currentChar == nil || !digitFun(*self.currentChar) {
			return ' ', errors.NewError(errors.Span{
				Start: startLocation,
				End:   self.location,
			}, "Invalid escape sequence", errors.SyntaxError)
		}
		esc += string(*self.currentChar)
		self.advance()
	}
	code, _ := strconv.ParseInt(esc, radix, 32)
	return rune(code), nil
}

func (self *lexer) makeNumber() Token {
	startLocation := self.location
	value := string(*self.currentChar)

	self.advance()

	for self.currentChar != nil && isDigit(*self.currentChar) {
		value += string(*self.currentChar)
		self.advance()
	}

	if self.currentChar != nil && *self.currentChar == '.' && self.nextChar != nil && isDigit(*self.nextChar) {
		value += string(*self.currentChar)
		self.advance()
		for self.currentChar != nil && isDigit(*self.currentChar) {
			value += string(*self.currentChar)
			self.advance()
		}
	}
	return Token{
		Kind:          Number,
		Value:         value,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
}

func (self *lexer) makeSingleChar(kind TokenKind, value rune) Token {
	token := Token{
		Kind:          kind,
		Value:         string(value),
		StartLocation: self.location,
		EndLocation:   self.location,
	}
	self.advance()
	return token
}

func (self *lexer) makeDots() Token {
	startLocation := self.location

	var tokenKind TokenKind
	var tokenKindValue string

	if self.nextChar != nil && *self.nextChar == '.' {
		tokenKind = Range
		tokenKindValue = ".."
		self.advance()
	} else {
		tokenKind = Dot
		tokenKindValue = "."
	}
	token := Token{
		Kind:          tokenKind,
		Value:         tokenKindValue,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	self.advance()
	return token
}

func (self *lexer) makeEquals() Token {
	startLocation := self.location

	if self.nextChar != nil {
		tokenKind := Assign
		tokenValue := "="

		switch *self.nextChar {
		case '=':
			self.advance()
			tokenKind = Equal
			tokenValue = "=="
		case '>':
			self.advance()
			tokenKind = Arrow
			tokenValue = "=>"
		}
		token := Token{
			Kind:          tokenKind,
			Value:         tokenValue,
			StartLocation: startLocation,
			EndLocation:   self.location,
		}
		self.advance()
		return token
	}

	return Token{
		Kind:          Assign,
		Value:         "=",
		StartLocation: startLocation,
		EndLocation:   startLocation,
	}
}

func (self *lexer) makeOr() (Token, *errors.Error) {
	startLocation := self.location
	self.advance()

	if self.currentChar != nil && *self.currentChar == '|' {
		token := Token{
			Kind:          Or,
			Value:         "||",
			StartLocation: startLocation,
			EndLocation:   self.location,
		}
		self.advance()
		return token, nil
	}

	foundChar := "EOF"
	if self.currentChar != nil {
		foundChar = string(*self.currentChar)
	}
	return unknownToken(self.location), &errors.Error{
		Span: errors.Span{
			Start: self.location,
			End:   self.location,
		},
		Kind:    errors.SyntaxError,
		Message: fmt.Sprintf("Expected '|', found %s", foundChar),
	}
}

func (self *lexer) makeAnd() (Token, *errors.Error) {
	startLocation := self.location
	self.advance()

	if self.currentChar != nil && *self.currentChar == '&' {
		token := Token{
			Kind:          And,
			Value:         "&&",
			StartLocation: startLocation,
			EndLocation:   self.location,
		}
		self.advance()
		return token, nil
	}

	foundChar := "EOF"
	if self.currentChar != nil {
		foundChar = string(*self.currentChar)
	}
	return unknownToken(self.location), &errors.Error{
		Span: errors.Span{
			Start: self.location,
			End:   self.location,
		},
		Kind:    errors.SyntaxError,
		Message: fmt.Sprintf("Expected '&', found %s", foundChar),
	}
}

func (self *lexer) makeNot() Token {
	startLocation := self.location

	tokenKind := Not
	tokenValue := "!"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = NotEqual
		tokenValue = "!="
		self.advance()
	}

	token := Token{
		Kind:          tokenKind,
		Value:         tokenValue,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	self.advance()
	return token
}

func (self *lexer) makeLess() Token {
	startLocation := self.location

	tokenKind := LessThan
	tokenValue := "<"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = LessThanEqual
		tokenValue = "<="
		self.advance()
	}

	token := Token{
		Kind:          tokenKind,
		Value:         tokenValue,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	self.advance()
	return token
}

func (self *lexer) makeGreater() Token {
	startLocation := self.location

	tokenKind := GreaterThan
	tokenValue := ">"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = GreaterThanEqual
		tokenValue = ">="
		self.advance()
	}

	token := Token{
		Kind:          tokenKind,
		Value:         tokenValue,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	self.advance()
	return token
}

func (self *lexer) makePlus() Token {
	startLocation := self.location

	tokenKind := Plus
	tokenValue := "+"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = PlusAssign
		tokenValue = "+="
		self.advance()
	}

	token := Token{
		Kind:          tokenKind,
		Value:         tokenValue,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	self.advance()
	return token
}

func (self *lexer) makeMinus() Token {
	startLocation := self.location

	tokenKind := Minus
	tokenValue := "-"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = MinusAssign
		tokenValue = "-="
		self.advance()
	}

	token := Token{
		Kind:          tokenKind,
		Value:         tokenValue,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	self.advance()
	return token
}

func (self *lexer) makeStar() Token {
	startLocation := self.location
	self.advance()

	if self.currentChar != nil {
		if *self.currentChar == '=' {
			token := Token{
				Kind:          MultiplyAssign,
				Value:         "*=",
				StartLocation: startLocation,
				EndLocation:   self.location,
			}
			self.advance()
			return token
		}
		if *self.currentChar == '*' {
			if self.nextChar != nil && *self.nextChar == '=' {
				self.advance()
				token := Token{
					Kind:          PowerAssign,
					Value:         "**=",
					StartLocation: startLocation,
					EndLocation:   self.location,
				}
				self.advance()
				return token
			}
			token := Token{
				Kind:          Power,
				Value:         "**",
				StartLocation: startLocation,
				EndLocation:   self.location,
			}
			self.advance()
			return token
		}
	}
	token := Token{
		Kind:          Multiply,
		Value:         "*",
		StartLocation: startLocation,
		EndLocation:   startLocation,
	}
	self.advance()
	return token
}

func (self *lexer) makeDiv() Token {
	startLocation := self.location

	tokenKind := Divide
	tokenValue := "/"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = DivideAssign
		tokenValue = "/="
		self.advance()
	}

	token := Token{
		Kind:          tokenKind,
		Value:         tokenValue,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	self.advance()
	return token
}

func (self *lexer) makeReminder() Token {
	startLocation := self.location

	tokenKind := Reminder
	tokenValue := "%"

	if self.nextChar != nil && *self.nextChar == '=' {
		tokenKind = ReminderAssign
		tokenValue = "%="
		self.advance()
	}

	token := Token{
		Kind:          tokenKind,
		Value:         tokenValue,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	self.advance()
	return token
}

func (self *lexer) makeName() Token {
	startLocation := self.location

	value := string(*self.currentChar)
	self.advance()

	for self.currentChar != nil && (isLetter(*self.currentChar) || isDigit(*self.currentChar)) {
		value += string(*self.currentChar)
		self.advance()
	}

	var tokenKind TokenKind
	switch value {
	case "true":
		tokenKind = True
	case "on":
		tokenKind = True
	case "false":
		tokenKind = False
	case "off":
		tokenKind = False
	case "fn":
		tokenKind = Fn
	case "if":
		tokenKind = If
	case "else":
		tokenKind = Else
	case "try":
		tokenKind = Try
	case "catch":
		tokenKind = Catch
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
	case "str":
		tokenKind = StringType
	case "num":
		tokenKind = NumberType
	case "bool":
		tokenKind = BooleanType
	case "null":
		tokenKind = NullType
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
	default:
		tokenKind = Identifier
	}
	token := Token{
		Kind:          tokenKind,
		Value:         value,
		StartLocation: startLocation,
		EndLocation:   self.location,
	}
	return token
}

func (self *lexer) skipComment() {
	self.advance()
	for self.currentChar != nil && *self.currentChar != '\n' {
		self.advance()
	}
}

func (self *lexer) nextToken() (Token, *errors.Error) {
	for self.currentChar != nil {
		switch *self.currentChar {
		case ' ', '\n', '\t':
			self.advance()
		case '\'', '"':
			return self.makeString()
		case ';':
			return self.makeSingleChar(Semicolon, ';'), nil
		case ',':
			return self.makeSingleChar(Comma, ','), nil
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
		case '|':
			return self.makeOr()
		case '&':
			return self.makeAnd()
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
			return self.makeDiv(), nil
		case '%':
			return self.makeReminder(), nil
		case '#':
			self.skipComment()
		default:
			if isDigit(*self.currentChar) {
				return self.makeNumber(), nil
			}
			if isLetter(*self.currentChar) {
				return self.makeName(), nil
			}
			return unknownToken(self.location), errors.NewError(errors.Span{
				Start: self.location,
				End:   self.location,
			}, fmt.Sprintf("Illegal characer: %c", *self.currentChar), errors.SyntaxError)
		}
	}
	return Token{
		Kind:          EOF,
		Value:         "EOF",
		StartLocation: self.location,
		EndLocation:   self.location,
	}, nil
}
