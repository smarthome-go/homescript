package homescript

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	program, err := os.ReadFile("lexer_test.hms")
	assert.NoError(t, err)

	start := time.Now()
	parser := newParser("parser_test.hms", string(program))

	ast, errors := parser.parse()

	if len(errors) != 0 {
		for _, err := range errors {
			fmt.Printf("%v: (l:%d c: %d) - (l:%d c: %d): %s", err.Kind, err.Span.Start.Line, err.Span.Start.Column, err.Span.End.Line, err.Span.End.Column, err.Message)
		}
		return
	}

	fmt.Printf("Lex + Parse: %v\n", time.Since(start))

	fmt.Printf("%v\n", ast)

}
