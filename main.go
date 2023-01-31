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

	_, _, _ = homescript.Analyze(
		homescript.DummyExecutor{},
		string(program),
		map[string]homescript.Value{
			"power_on": homescript.ValueBool{Value: true},
			"PI":       homescript.ValueNumber{Value: 3.14159265},
			"test": homescript.ValueBuiltinFunction{
				Callback: func(_ homescript.Executor, _ errors.Span, _ ...homescript.Value) (homescript.Value, *int, *errors.Error) {
					return homescript.ValueNull{}, nil, nil
				},
			},
		},
		make([]string, 0),
		"main",
	)

	_, code, _, hmsErrors := homescript.Run(
		homescript.DummyExecutor{},
		&sigTerm,
		string(program),
		make(map[string]homescript.Value),
		map[string]homescript.Value{
			"power_on": homescript.ValueBool{Value: true},
			"PI":       homescript.ValueNumber{Value: 3.14159265},
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
	defer fmt.Printf("Program terminated with exit-code %d\n", code)

	for _, err := range hmsErrors {
		fmt.Println(err.Display(string(program), os.Args[1]))
	}
}
