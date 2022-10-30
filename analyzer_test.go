package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/smarthome-go/homescript/v2/homescript"
)

type analyzerExecutor struct{}

func (self analyzerExecutor) ResolveModule(id string) (string, bool, error) {
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

func (self analyzerExecutor) Sleep(sleepTime float64) {
}

func (self analyzerExecutor) Print(args ...string) error {
	return nil
}

func (self analyzerExecutor) Println(args ...string) error {
	return nil
}

func (self analyzerExecutor) Switch(name string, power bool) error {
	return nil
}
func (self analyzerExecutor) GetSwitch(id string) (homescript.SwitchResponse, error) {
	return homescript.SwitchResponse{}, nil
}

func (self analyzerExecutor) Ping(ip string, timeout float64) (bool, error) {
	return false, nil
}

func (self analyzerExecutor) Notify(title string, description string, level homescript.NotificationLevel) error {
	return nil
}

func (self analyzerExecutor) Log(title string, description string, level homescript.LogLevel) error {
	return nil
}

func (self analyzerExecutor) Exec(id string, args map[string]string) (homescript.ExecResponse, error) {
	return homescript.ExecResponse{ReturnValue: homescript.ValueNull{}}, nil
}

func (self analyzerExecutor) Get(url string) (homescript.HttpResponse, error) {
	return homescript.HttpResponse{}, nil
}

func (self analyzerExecutor) Http(url string, method string, body string, headers map[string]string) (homescript.HttpResponse, error) {
	return homescript.HttpResponse{}, nil
}

func (self analyzerExecutor) GetUser() string {
	return ""
}

func (self analyzerExecutor) GetWeather() (homescript.Weather, error) {
	return homescript.Weather{}, nil
}

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
		Name: "Analyzer",
		File: "./test/programs/analyzer.hms",
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
			diagnostics, _ := homescript.Analyze(
				analyzerExecutor{},
				string(program),
				make(map[string]homescript.Value),
			)
			for _, diagnostic := range diagnostics {
				fmt.Printf("\n%s\n", diagnostic.Display(string(program), test.File))
			}
			if len(diagnostics) == 0 {
				fmt.Println("no diagnostics")
			}
		})
	}
}
