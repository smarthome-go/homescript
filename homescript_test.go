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
	fmt.Println(">>> ", strings.Join(args, " "))
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

func TestInterpreter(t *testing.T) {
	program, err := os.ReadFile("./test/interpreter_test.hms")
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
	)

	if hmsError != nil {
		fmt.Println(hmsError.Display(string(program)))
		return
	}

	fmt.Printf("value kind: %v\n", value.Type())

	valueStr, displayErr := value.Display(dummyExecutor{}, errors.Span{})
	if displayErr != nil {
		panic(fmt.Sprintf("Display error: %v: %s", displayErr.Kind, displayErr.Message))
	}
	fmt.Printf("Exit-code %d: Return-value: %s\n", code, valueStr)
}
