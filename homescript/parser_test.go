package homescript

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

	tokens := make([]string, 0)
	for {
		current, err := lexer.nextToken()
		if err != nil {
			t.Error(err.Message)
			return
		}
		repr := fmt.Sprintf("(%d:%d--%d:%d) ==> %v(%v)", current.StartLocation.Line, current.StartLocation.Column, current.EndLocation.Line, current.EndLocation.Column, current.Kind, current.Value)
		fmt.Println(repr)
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
	err = os.WriteFile("../test/parser_test.tokens", []byte(strings.Join(tokens, "\n")), 0755)
	assert.NoError(t, err)
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
	err = os.WriteFile("../test/parser_test_ast.json", dump, 0755)
	assert.NoError(t, err)
}
