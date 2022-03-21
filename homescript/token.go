package homescript

import "github.com/MikMuellerDev/homescript/homescript/error"

type TokenType uint8

const (
	Unknown TokenType = iota
	EOF
	EOL        // \n
	Number     // int
	String     // " "
	Identifier // temperature, sleep

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
	True               // true | on
	False              // false | off
)

type Token struct {
	TokenType TokenType
	Value     string
	Location  error.Location
}

func UnknownToken(location error.Location) Token {
	return Token{
		TokenType: Unknown,
		Value:     "Unknown",
		Location:  location,
	}
}
