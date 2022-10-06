package homescript

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

func (self *parser) parse() (Block, []Error) {
	return nil, nil
}
