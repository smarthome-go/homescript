package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/smarthome-go/homescript/homescript"
	"github.com/smarthome-go/homescript/homescript/errors"
	"github.com/stretchr/testify/assert"
)

type dummyExecutor struct{}

func (self dummyExecutor) ResolveModule(id string) (string, bool, error) {
	path := "test/programs/" + id + ".hms"
	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read file: %s", err.Error())
	}
	return string(file), true, nil
}

func (self dummyExecutor) Sleep(sleepTime float64) {
	time.Sleep(time.Duration(sleepTime * 1000 * float64(time.Millisecond)))
}

func (self dummyExecutor) Print(args ...string) {
	fmt.Printf("%s", strings.Join(args, " "))
}

func (self dummyExecutor) Println(args ...string) {
	fmt.Println(strings.Join(args, " "))
}

func (self dummyExecutor) SwitchOn(name string) (bool, error) {
	return false, nil
}

func (self dummyExecutor) Switch(name string, power bool) error {
	return nil
}

func (self dummyExecutor) Notify(title string, description string, level homescript.NotificationLevel) error {
	return nil
}

func (self dummyExecutor) Log(title string, description string, level homescript.LogLevel) error {
	return nil
}

func (self dummyExecutor) Exec(id string, args map[string]string) (homescript.ExecResponse, error) {
	return homescript.ExecResponse{
		Output:      "homescript test output",
		RuntimeSecs: 0.2,
		ReturnValue: homescript.ValueNull{},
	}, nil
}

func (self dummyExecutor) Get(url string) (homescript.HttpResponse, error) {
	return homescript.HttpResponse{
		Status:     "OK",
		StatusCode: 200,
		Body:       "{\"foo\": \"bar\"}",
	}, nil
}

func (self dummyExecutor) Http(url string, method string, body string, headers map[string]string) (homescript.HttpResponse, error) {
	return homescript.HttpResponse{
		Status:     "Internal Server Error",
		StatusCode: 500,
		Body:       "{\"error\": \"the server is currently running on JavaScript\"}",
	}, nil
}

func (self dummyExecutor) GetUser() string {
	return "john_doe"
}

func (self dummyExecutor) GetWeather() (homescript.Weather, error) {
	return homescript.Weather{
		WeatherTitle:       "Rain",
		WeatherDescription: "light rain",
		Temperature:        17.0,
		FeelsLike:          16.0,
		Humidity:           87,
	}, nil
}

func (self dummyExecutor) GetTime() homescript.Time {
	return homescript.Time{
		Year:         2022,
		Month:        10,
		CalendarWeek: 42,
		CalendarDay:  17,
		WeekDayText:  "Monday",
		WeekDay:      1,
		Hour:         19,
		Minute:       9,
		Second:       32,
	}
}

type testError struct {
	Kind    errors.ErrorKind
	Message string
}

type test struct {
	Name              string
	File              string
	Debug             bool
	ExpectedCode      int
	ExpectedValueType homescript.ValueType
	ExpectedErrors    []testError
}

var tests = []test{
	{
		Name:              "Main",
		File:              "./test/programs/main.hms",
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "Fibonacci",
		File:              "./test/programs/fibonacci.hms",
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeFunction,
		ExpectedErrors:    nil,
	},
	{
		Name:              "StackOverFlow",
		File:              "./test/programs/stack_overflow.hms",
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
		Debug:             false,
		ExpectedCode:      1,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors: []testError{
			{
				Kind:    errors.RuntimeError,
				Message: "Illegal import: circular import detected",
			},
		},
	},
	{
		Name:              "Analyzer",
		File:              "./test/programs/analyzer.hms",
		Debug:             false,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
	{
		Name:              "PrimeNumbers",
		File:              "./test/programs/primes.hms",
		Debug:             true,
		ExpectedCode:      0,
		ExpectedValueType: homescript.TypeNull,
		ExpectedErrors:    nil,
	},
}

func TestHomescripts(t *testing.T) {
	for idx, test := range tests {
		t.Run(fmt.Sprintf("(%d/%d): %s", idx, len(tests), test.Name), func(t *testing.T) {
			program, err := os.ReadFile(test.File)
			assert.NoError(t, err)
			sigTerm := make(chan int)
			moduleName := strings.ReplaceAll(strings.Split(test.File, "/")[len(strings.Split(test.File, "/"))-1], ".hms", "")
			value, code, _, hmsErrors := homescript.Run(
				dummyExecutor{},
				&sigTerm,
				string(program),
				//map[string]homescript.Value{
				//"foo": homescript.ValueString{Value: "bar"},
				//},
				make(map[string]homescript.Value),
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
				valueStr, displayErr := value.Display(dummyExecutor{}, errors.Span{})
				if displayErr != nil {
					panic(fmt.Sprintf("Display error: %v: %s", displayErr.Kind, displayErr.Message))
				}
				t.Errorf("Unexpected value: Type: %v | Value: %s\n", value.Type(), valueStr)
				return
			}

			if code != test.ExpectedCode {
				t.Errorf("Unexpected exit-code: expected %d, found %d", test.ExpectedCode, code)
			}
		})
	}
}

func TestAnalyzer(t *testing.T) {
	for idx, test := range tests {
		t.Run(fmt.Sprintf("(%d/%d): %s", idx, len(tests), test.Name), func(t *testing.T) {
			program, err := os.ReadFile(test.File)
			if err != nil {
				t.Error(err.Error())
			}
			diagnostics := homescript.Analyze(
				dummyExecutor{},
				string(program),
				make(map[string]homescript.Value),
			)
			for _, diagnostic := range diagnostics {
				fmt.Printf("%s\n_\n_\n", diagnostic.Display(string(program), "analyzer.hms"))
			}
			if len(diagnostics) == 0 {
				fmt.Println("no diagnostics")
			}
		})
	}
}
