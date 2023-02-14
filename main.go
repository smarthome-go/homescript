package main

import (
	"fmt"
	"os"

	"github.com/smarthome-go/homescript/v2/homescript"
	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

func main() {
	program, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err.Error())
	}
	sigTerm := make(chan int)

	diagnostics, _, _ := homescript.Analyze(
		homescript.AnalyzerDummyExecutor{},
		string(program),
		map[string]homescript.Value{
			"power_on": homescript.ValueBool{Value: true, IsProtected: true},
			"PI":       homescript.ValueNumber{Value: 3.14159265, IsProtected: true},
			"test": homescript.ValueBuiltinFunction{
				Callback: func(_ homescript.Executor, _ errors.Span, _ ...homescript.Value) (homescript.Value, *int, *errors.Error) {
					return homescript.ValueNull{}, nil, nil
				},
			},
		},
		make([]string, 0),
		"main",
	)

	hasError := false
	for _, err := range diagnostics {
		fmt.Println(err.Display(string(program), os.Args[1]))
		if err.Severity == homescript.Error {
			hasError = true
		}
	}

	if hasError {
		fmt.Println("Analyzer detected errors")
		return
	}

	_, code, _, hmsErrors := homescript.Run(
		homescript.DummyExecutor{},
		&sigTerm,
		string(program),
		make(map[string]homescript.Value),
		map[string]homescript.Value{
			"power_on": homescript.ValueBool{Value: true, IsProtected: true},
			"PI":       homescript.ValueNumber{Value: 3.14159265, IsProtected: true},
			"test": homescript.ValueBuiltinFunction{
				Callback: func(_ homescript.Executor, _ errors.Span, _ ...homescript.Value) (homescript.Value, *int, *errors.Error) {
					return homescript.ValueNull{}, nil, nil
				},
			},
		},
		false,
		1000,
		make([]string, 0),
		"main",
	)

	for _, err := range hmsErrors {
		fmt.Println(err.Display(string(program), os.Args[1]))
	}

	fmt.Printf("Program terminated with exit-code %d\n", code)
}
