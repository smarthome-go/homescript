package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/smarthome-go/homescript/v2/homescript"
)

type analysis struct {
	Name string
	File string
	Skip bool
}

var analysisTasks = []analysis{
	{
		Name: "Main",
		File: "./test/programs/main.hms",
		Skip: false,
	},
	{
		Name: "Fibonacci",
		File: "./test/programs/fibonacci.hms",
		Skip: false,
	},
	{
		Name: "StackOverFlow",
		File: "./test/programs/stack_overflow.hms",
		Skip: false,
	},
	{
		Name: "ImportExport",
		File: "./test/programs/import_export.hms",
		Skip: false,
	},
	{
		Name: "PrimeNumbers",
		File: "./test/programs/primes.hms",
		Skip: false,
	},
	{
		Name: "FizzBuzz",
		File: "./test/programs/fizzbuzz.hms",
		Skip: false,
	},
	{
		Name: "Box",
		File: "./test/programs/box.hms",
		Skip: false,
	},
	{
		Name: "Lists",
		File: "./test/programs/lists.hms",
		Skip: false,
	},
	{
		Name: "Binary",
		File: "./test/programs/binary.hms",
		Skip: false,
	},
	{
		Name: "JSON",
		File: "./test/programs/json.hms",
		Skip: false,
	},
	{
		Name: "Dev",
		File: "./test/programs/dev.hms",
		Skip: false,
	},
}

func TestAnalyzer(t *testing.T) {
	for idx, test := range analysisTasks {
		t.Run(fmt.Sprintf("(%d/%d): %s", idx, len(tests), test.Name), func(t *testing.T) {
			if test.Skip {
				t.SkipNow()
				return
			}
			program, err := os.ReadFile(test.File)
			if err != nil {
				t.Error(err.Error())
			}
			moduleName := strings.ReplaceAll(strings.Split(test.File, "/")[len(strings.Split(test.File, "/"))-1], ".hms", "")
			diagnostics, _, _ := homescript.Analyze(
				homescript.AnalyzerDummyExecutor{},
				string(program),
				make(map[string]homescript.Value),
				make([]string, 0),
				moduleName,
				test.File,
			)
			for _, diagnostic := range diagnostics {
				fmt.Printf("\n%s\n", diagnostic.Display(string(program)))
			}
			if len(diagnostics) == 0 {
				fmt.Println("no diagnostics")
			}
		})
	}
}
