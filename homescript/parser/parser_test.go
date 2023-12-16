package parser

import (
	"fmt"
	"os"
	"testing"
)

const EXAMPLE_DIR = "../../examples/"

func FuzzParser(f *testing.F) {
	files, err := os.ReadDir(EXAMPLE_DIR)
	if err != nil {
		panic(err.Error())
	}

	for _, file := range files {
		content, err := os.ReadFile(fmt.Sprintf("%s/%s", EXAMPLE_DIR, file.Name()))
		if err != nil {
			panic(err.Error())
		}
		f.Add(string(content))
	}

	f.Fuzz(func(t *testing.T, input string) {
		l := NewLexer(input, t.Name())
		p := NewParser(l, t.Name())

		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("Parser panicked for input: %s\n", input)
			}
		}()

		_, _, _ = p.Parse()
	})
}
