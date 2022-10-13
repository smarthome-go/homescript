package homescript

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLexer(t *testing.T) {
	program, err := os.ReadFile("../test/lexer_test.hms")
	assert.NoError(t, err)

	start := time.Now()

	lexer := newLexer("testing", string(program))

	tokens := make([]string, 0)
	for {
		current, err := lexer.nextToken()
		if err != nil {
			t.Error(err.Message)
			return
		}
		repr := fmt.Sprintf("(%d:%d--%d:%d) ==> %v(%v)", current.StartLocation.Line, current.StartLocation.Column, current.EndLocation.Line, current.EndLocation.Column, current.Kind, current.Value)
		if current.Kind == EOF {
			break
		}
		if current.Kind == Unknown {
			t.Errorf("Found unknown token %v", current.StartLocation)
		}
		tokens = append(tokens, repr)
	}
	fmt.Printf("Lex: %v\n", time.Since(start))

	// Dump results to file
	err = os.WriteFile("../test/lexer_test.tokens", []byte(strings.Join(tokens, "\n")), 0755)
	assert.NoError(t, err)
}
