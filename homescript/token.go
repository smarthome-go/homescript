package homescript

type TokenType uint8

const (
	EOF        TokenType = iota
	EOL                  // \n
	Number               // int
	String               // " "
	Identifier           // true | false | on | off

	// Terminal symbols
	Or                 // ||
	And                // &&
	Equal              // ==
	NotEqual           // !=
	LessThan           // <
	LessThanOrEqual    // <=
	GreaterThan        // >
	GreaterThanOrEqual // >=
	Not                // !
	LeftParenthesis    // (
	RightParenthesis   // )
	LeftCurlyBrace     // {
	RightCurlyBrace    // }
	Comma              // ,
	If                 // if
	Else               // else
)

// TODO: Add positions to tokens and errors
// type Position struct {
// 	Filename string
// 	Index    uint32
// 	Line     uint32
// 	Column   int32
// }

type Token struct {
	TokenType TokenType
	Value     string
	// Position  Position
}
