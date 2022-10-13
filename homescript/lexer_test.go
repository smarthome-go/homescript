package homescript

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLexer(t *testing.T) {
	program, err := os.ReadFile("../test/lexer_test.hms")
	assert.NoError(t, err)

	start := time.Now()

	lexer := newLexer("testing", string(program))
	fmt.Printf("::INPUT::\n%s\n", program)

	tokens := make([]Token, 0)
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
		tokens = append(tokens, current)
	}
	fmt.Printf("Lex: %v\n", time.Since(start))
	// Dump results to json file
	dump, err := json.MarshalIndent(tokens, "", "\t")
	assert.NoError(t, err)
	err = os.WriteFile("../test/lexer_test.json", dump, 0755)
	assert.NoError(t, err)
}
