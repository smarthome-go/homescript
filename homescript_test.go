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

func (self dummyExecutor) Sleep(sleepTime float64) {
	time.Sleep(time.Duration(sleepTime * 1000 * float64(time.Millisecond)))
}

func (self dummyExecutor) Print(args ...string) {
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

func TestHomescripts(t *testing.T) {
	type testError struct {
		Kind    errors.ErrorKind
		Message string
	}

	type test struct {
		Name              string
		File              string
		ExpectedCode      int
		ExpectedValueType homescript.ValueType
		ExpectedError     *testError
	}

	tests := []test{
		{
			Name:              "Main",
			File:              "./test/programs/main.hms",
			ExpectedCode:      0,
			ExpectedValueType: homescript.TypeNull,
			ExpectedError:     nil,
		},
		{
			Name:              "Fibonacci",
			File:              "./test/programs/fibonacci.hms",
			ExpectedCode:      0,
			ExpectedValueType: homescript.TypeNull,
			ExpectedError:     nil,
		},
		{
			Name:              "StackOverFlow",
			File:              "./test/programs/stack_overflow.hms",
			ExpectedCode:      1,
			ExpectedValueType: homescript.TypeNull,
			ExpectedError: &testError{
				Kind:    errors.StackOverflow,
				Message: "Maximum stack size of",
			},
		},
	}

	for idx, test := range tests {
		t.Run(fmt.Sprintf("(%d/%d): %s", idx, len(tests), test.Name), func(t *testing.T) {
			program, err := os.ReadFile(test.File)
			assert.NoError(t, err)
			sigTerm := make(chan int)
			value, code, hmsError := homescript.Run(
				dummyExecutor{},
				&sigTerm,
				"foo.hms",
				string(program),
				map[string]homescript.Value{
					"foo": homescript.ValueString{Value: "bar"},
				},
				false,
				1_000,
			)
			defer fmt.Printf("Program terminated with exit-code %d\n", code)

			if hmsError != nil && test.ExpectedError == nil {
				t.Errorf("Unexpected HMS error")
				fmt.Println(hmsError.Display(string(program)))
				return
			}
			if hmsError == nil && test.ExpectedError != nil {
				t.Error("Expected HMS error, got none")
				return
			}

			if hmsError != nil && test.ExpectedError != nil {
				if hmsError.Kind != test.ExpectedError.Kind {
					t.Errorf("Expected %v, got %v", test.ExpectedError.Kind, hmsError.Kind)
					fmt.Println(hmsError.Display(string(program)))
					return
				}
				if !strings.Contains(hmsError.Message, test.ExpectedError.Message) {
					t.Errorf("Expected to find error-message `%s` inside error", test.ExpectedError.Message)
					fmt.Println(hmsError.Display(string(program)))
					return
				}
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
