package homescript

import (
	"fmt"
	"io/ioutil"
)

func Test() {
	content, err1 := ioutil.ReadFile("demo.hms")
	if err1 != nil {
		panic(err1.Error())
	}

	parser := NewParser(NewLexer(string(content)))
	res, err := parser.Parse()
	if len(err) > 0 {
		for i := 0; i < len(err); i += 1 {
			fmt.Println(err[i].Error())
		}
		return
	}
	fmt.Println(res)

	// lexer := NewLexer(program)
	// for {
	// 	res, err := lexer.Scan()
	// 	if err != nil {
	// 		panic(err.Error())
	// 	}
	// 	fmt.Println(res.Value)
	// 	if res.TokenType == EOF {
	// 		return
	// 	}
	// }
}
