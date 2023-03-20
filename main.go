package main

import (
	"fmt"
	"os"

	"github.com/smarthome-go/homescript/v2/homescript"
)

func main() {
	program, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err.Error())
	}
	sigTerm := make(chan int)

	analyzerExecutor := homescript.AnalyzerDummyExecutor{}

	diagnostics, _, _ := homescript.Analyze(
		analyzerExecutor,
		string(program),
		map[string]homescript.Value{
			"PI": homescript.ValueNumber{Value: 3.14159265, IsProtected: true},
		},
		make([]string, 0),
		"main",
		os.Args[1], // Filename
	)

	hasError := false
	for _, err := range diagnostics {
		// Read the file of the error
		code, fileErr := analyzerExecutor.ReadFile(err.Span.Filename)
		if fileErr != nil {
			panic(fileErr.Error())
		}

		fmt.Println(err.Display(string(code)))
		if err.Severity == homescript.Error {
			hasError = true
		}
	}

	if hasError {
		fmt.Println("Analyzer detected errors")
		return
	}

	interpreterExecutor := homescript.DummyExecutor{}

	_, code, _, hmsErrors := homescript.Run(
		interpreterExecutor,
		&sigTerm,
		string(program),
		make(map[string]homescript.Value),
		map[string]homescript.Value{
			"PI": homescript.ValueNumber{Value: 3.14159265, IsProtected: true},
		},
		false,
		1000,
		make([]string, 0),
		"main",
		os.Args[1], // Filename
	)

	for _, err := range hmsErrors {
		// Read the file of the error
		code, fileErr := analyzerExecutor.ReadFile(err.Span.Filename)
		if fileErr != nil {
			panic(fileErr.Error())
		}

		fmt.Println(err.Display(string(code)))
	}

	fmt.Printf("Program terminated with exit-code %d\n", code)
}
