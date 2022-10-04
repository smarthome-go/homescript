package homescript

import (
	"fmt"
	"testing"
)

func TestLexer(t *testing.T) {
	program := "* *= ** **="
	lexer := newLexer("testing", program)
	fmt.Printf("::INPUT::\n%s\n", program)

	for {
		current, err := lexer.nextToken()
		if err != nil {
			t.Error(err.Message)
			return
		}
		fmt.Printf("(%d:%d--%d:%d) ==> %v(%v)\n", current.StartLocation.Line, current.StartLocation.Column, current.EndLocation.Line, current.EndLocation.Column, current.Kind, current.Value)
		if current.Kind == EOF {
			break
		}
	}
}
