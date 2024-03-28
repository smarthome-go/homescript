package lexer

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLexer(t *testing.T) {
	start := time.Now()

	program := "[]{}+-*/%**"
	lexer := NewLexer(string(program), "test")

	tokens := make([]string, 0)
	for {
		current, err := lexer.NextToken()
		if err != nil {
			t.Error(err.Message)
			return
		}
		repr := fmt.Sprintf("(%d:%d--%d:%d) ==> %v | %v", current.Span.Start.Line, current.Span.Start.Column, current.Span.End.Line, current.Span.End.Column, current.Kind, current.Value)
		if current.Kind == EOF {
			break
		}
		if current.Kind == Unknown {
			t.Errorf("Found unknown token %v", current.Span.Start)
		}
		tokens = append(tokens, repr)
	}
	fmt.Printf("Lex: %v\n", time.Since(start))

	// Dump results to file
	err := os.WriteFile("../../test/lexer_test.tokens", []byte(strings.Join(tokens, "\n")), 0755)
	assert.NoError(t, err)
}
