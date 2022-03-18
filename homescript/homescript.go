package homescript

import "fmt"

func Test() {
	fmt.Println("hallo")
	lexer := NewLexer("")
	lexer.Scan()
}
