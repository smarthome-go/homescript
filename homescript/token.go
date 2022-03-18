package homescript

type TokenType uint8

const (
	Number TokenType = iota
	Identifier
	If
	Else
)

type Position struct {
	Line   uint32
	Column int32
}

type Token struct {
	TokenType TokenType
	Value     string
	Position  Position
}
