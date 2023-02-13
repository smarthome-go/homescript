package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/smarthome-go/homescript/v2/homescript"
	"github.com/smarthome-go/homescript/v2/homescript/errors"
	"github.com/stretchr/testify/assert"
)

type testError struct {
	Kind    errors.ErrorKind
	Message string
}

type test struct {
	Name              string
	File              string
	Skip              bool
	Debug             bool
	ExpectedCode      int
	ExpectedValueType homescript.ValueType
	ExpectedErrors    []testError
}

var tests = []test{
	{
		Name:              "Main",
		File:              "./test/programs/main.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "Fibonacci",
		File:              "./test/programs/fibonacci.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "StackOverFlow",
		File:              "./test/programs/stack_overflow.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      1,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors: []testError{
			{
				Kind:    errors.StackOverflow,
				Message: "Maximum stack size of",
			},
		},
	},
	{
		Name:              "ImportExport",
		File:              "./test/programs/import_export.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      1,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors: []testError{
			{
				Kind:    errors.ImportError,
				Message: "illegal import: circular import detected",
			},
		},
	},
	{
		Name:              "PrimeNumbers",
		File:              "./test/programs/primes.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "FizzBuzz",
		File:              "./test/programs/fizzbuzz.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "Box",
		File:              "./test/programs/box.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "Lists",
		File:              "./test/programs/lists.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "Binary",
		File:              "./test/programs/binary.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "JSON",
		File:              "./test/programs/json.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "Import",
		File:              "./test/programs/import.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors: []testError{
			{
				Kind:    errors.Warning,
				Message: "Using the `as` keyword is currently unstable",
			},
		},
	},
	{
		Name:              "Iterators",
		File:              "./test/programs/iterators.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "Dev",
		File:              "./test/programs/dev.hms",
		Skip:              false,
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
}

func TestHomescripts(t *testing.T) {
	for idx, test := range tests {
		t.Run(fmt.Sprintf("(%d/%d): %s", idx, len(tests), test.Name), func(t *testing.T) {
			if test.Skip {
				t.SkipNow()
				return
			}
			program, err := os.ReadFile(test.File)
			assert.NoError(t, err)
			sigTerm := make(chan int)
			moduleName := strings.ReplaceAll(strings.Split(test.File, "/")[len(strings.Split(test.File, "/"))-1], ".hms", "")
			value, code, _, hmsErrors := homescript.Run(
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
				test.Debug,
				1000,
				make([]string, 0),
				moduleName,
			)
			defer fmt.Printf("Program terminated with exit-code %d\n", code)

			if len(hmsErrors) > 0 && len(test.ExpectedErrors) == 0 {
				t.Errorf("Unexpected HMS error(s)")
				for _, err := range hmsErrors {
					fmt.Println(err.Display(string(program), test.File))
				}
				return
			}
			if len(hmsErrors) == 0 && len(test.ExpectedErrors) > 0 {
				t.Error("Expected HMS error(s), got none")
				return
			}

			if len(hmsErrors) > 0 && len(test.ExpectedErrors) > 0 {
				if len(hmsErrors) != len(test.ExpectedErrors) {
					t.Errorf("Expected %d error(s), got %d", len(test.ExpectedErrors), len(hmsErrors))
					return
				}
				for idx, err := range test.ExpectedErrors {
					if err.Kind != hmsErrors[idx].Kind {
						t.Errorf("Expected %v, got %v", err.Kind, hmsErrors[idx].Kind)
						fmt.Println(hmsErrors[idx].Display(string(program), test.File))
						return
					}
					if !strings.Contains(hmsErrors[idx].Message, err.Message) {
						t.Errorf("Expected to find error-message `%s` inside error", err.Message)
						fmt.Println(hmsErrors[idx].Display(string(program), test.File))
						return
					}
				}
				return
			}
			if value.Type() != test.ExpectedValueType {
				valueStr, displayErr := value.Display(homescript.DummyExecutor{}, errors.Span{})
				if displayErr != nil {
					panic(fmt.Sprintf("Display error: %v: %s", displayErr.Kind, displayErr.Message))
				}
				t.Errorf("Unexpected value: (Type: %v | Value: %s), expected: (Type: %v)\n", value.Type(), valueStr, test.ExpectedValueType)
				return
			}

			if code != test.ExpectedCode {
				t.Errorf("Unexpected exit-code: expected %d, found %d", test.ExpectedCode, code)
			}
		})
	}
}

func TestRunDev(t *testing.T) {
	path := "./test/programs/dev.hms"
	program, err := os.ReadFile(path)
	assert.NoError(t, err)
	sigTerm := make(chan int)
	moduleName := strings.ReplaceAll(strings.Split(path, "/")[len(strings.Split(path, "/"))-1], ".hms", "")
	value, code, _, hmsErrors := homescript.Run(
		homescript.DummyExecutor{},
		&sigTerm,
		string(program),
		make(map[string]homescript.Value),
		make(map[string]homescript.Value),
		false,
		10,
		make([]string, 0),
		moduleName,
	)
	for _, err := range hmsErrors {
		fmt.Println(err.Display(string(program), moduleName))
	}
	if value != nil {
		fmt.Printf(">>> %v\n", value.Type())
	}
	fmt.Printf("Program terminated with exit-code %d\n", code)
}

func TestSigTerm(t *testing.T) {
	path := "./test/programs/sig_term.hms"
	program, err := os.ReadFile(path)
	assert.NoError(t, err)
	sigTerm := make(chan int)
	moduleName := strings.ReplaceAll(strings.Split(path, "/")[len(strings.Split(path, "/"))-1], ".hms", "")
	go func() {
		value, code, _, hmsErrors := homescript.Run(
			homescript.DummyExecutor{},
			&sigTerm,
			string(program),
			make(map[string]homescript.Value),
			make(map[string]homescript.Value),
			false,
			10,
			make([]string, 0),
			moduleName,
		)
		for _, err := range hmsErrors {
			fmt.Println(err.Display(string(program), moduleName))
		}
		if value != nil {
			fmt.Printf(">>> %v\n", value.Type())
		}
		fmt.Printf("Program terminated with exit-code %d\n", code)
	}()
	time.Sleep(3 * time.Second)
	fmt.Println("Sending SigTerm with exit-code 10")
	sigTerm <- 10
}
