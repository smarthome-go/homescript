package homescript

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestParserLexer(t *testing.T) {
	program, err := os.ReadFile("../test/parser_test.hms")
	assert.NoError(t, err)

	start := time.Now()

	lexer := newLexer("testing", string(program))

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
		if current.Kind == Unknown {
			t.Errorf("Found unknown token %v", current.StartLocation)
		}
	}
	fmt.Printf("Lex: %v\n", time.Since(start))
}

func TestParser(t *testing.T) {
	program, err := os.ReadFile("../test/parser_test.hms")
	assert.NoError(t, err)

	start := time.Now()
	parser := newParser("parser_test.hms", string(program))

	ast, errors := parser.parse()

	if len(errors) != 0 {
		for _, err := range errors {
			fmt.Printf("%v: (l:%d c: %d) - (l:%d c: %d): %s", err.Kind, err.Span.Start.Line, err.Span.Start.Column, err.Span.End.Line, err.Span.End.Column, err.Message)
		}
		t.Error("Parsing failed")
		return
	}

	fmt.Printf("Lex + Parse: %v\n", time.Since(start))

	spew.Dump(ast)
	// Dump results to json file
	dump, err := json.MarshalIndent(ast, "", "\t")
	assert.NoError(t, err)
	err = os.WriteFile("../test/parser_test.json", dump, 0755)
	assert.NoError(t, err)
}
