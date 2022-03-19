package homescript

import "fmt"

func Test() {
	program := `print(42, 12, "homescript gut")`
	parser := NewParser(NewLexer(program))
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
