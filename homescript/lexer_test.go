package homescript

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexer(t *testing.T) {
	program, err := os.ReadFile("lexer_test.hms")
	assert.NoError(t, err)
	lexer := newLexer("testing", string(program))
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
