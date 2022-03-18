package homescript

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
	inputLen := uint32(len(self.Input))
	for self.CurrentIndex < inputLen {

	}
	return Token{}, nil
}

func (self *Lexer) next() {
	self.CurrentIndex += 1
	self.CurrentChar = &self.Input[self.CurrentIndex]
}
