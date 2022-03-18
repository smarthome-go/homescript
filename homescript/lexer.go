package homescript

type Lexer struct {
	CurrentChar  rune
	CurrentIndex uint32
	Input        []rune
}

func NewLexer(input string) Lexer {
	return Lexer{
		CurrentChar:  []rune(input)[0],
		CurrentIndex: 0,
		Input:        []rune(input),
	}
}

func (self *Lexer) Scan() ([]Token, error) {
	tokens := make([]Token, 0)

	for self.CurrentIndex < uint32(len(self.Input)) {

	}

	return tokens, nil
}

func (self *Lexer) next() {
	self.CurrentIndex += 1
	self.CurrentChar = self.Input[self.CurrentIndex]
}
