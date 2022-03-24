package homescript

import (
	"fmt"
	"time"

	customError "github.com/MikMuellerDev/homescript/homescript/error"
	"github.com/MikMuellerDev/homescript/homescript/interpreter"
)

type DummyExecutor struct{}

func (self DummyExecutor) Print(args ...string) {
	output := ""
	for _, arg := range args {
		output += arg
	}
	fmt.Println(output)
}
func (self DummyExecutor) SwitchOn(name string) (bool, error) {
	if name == "s3" {
		return true, nil
	}
	return false, nil
}
func (self DummyExecutor) Switch(name string, on bool) error {
	fmt.Printf("Turning switch '%s' %t\n", name, on)
	return nil
}
func (self DummyExecutor) Play(server string, mode string) error {
	fmt.Printf("Playing '%s' on server '%s'\n", mode, server)
	return nil
}
func (self DummyExecutor) Notify(
	title string,
	description string,
	level interpreter.LogLevel,
) error {
	fmt.Printf("Sending notification with level %d '%s' -- '%s'\n", level, title, description)
	return nil
}
func (self DummyExecutor) Log(
	title string,
	description string,
	level interpreter.LogLevel,
) error {
	fmt.Printf("Logging '%s' -- '%s' with level %d\n", title, description, level)
	return nil
}
func (self DummyExecutor) Exec(homescriptId string) (string, error) {
	fmt.Printf("Executing script: '%s'\n", homescriptId)
	return "", nil
}
func (self DummyExecutor) GetUser() string {
	return "admin"
}
func (self DummyExecutor) GetWeather() (string, error) {
	return "rainy", nil
}
func (self DummyExecutor) GetTemperature() (int, error) {
	return 42, nil
}
func (self DummyExecutor) GetDate() (int, int, int, int, int, int) {
	now := time.Now()
	return now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second()
}
func (self DummyExecutor) GetDebugInfo() (string, error) {
	return `
―――――――――――――――――――――――――――――――――――――――――――――
 Smarthome Server Version:      │ v0.0.5
 Database Online:               │ YES
 Compiled with:                 │ go1.18
 CPU Cores:                     │ 12
 Current Goroutines:            │ 8
 Current Memory Usage:          │ 0
 Current Power Jobs:            │ 0
 Last Power Job Error Count:    │ 0
―――――――――――――――――――――――――――――――――――――――――――――
`, nil
}

// Runs a provided homescript file given the source code
// Returns an error slice
func Run(executor interpreter.Executor, filename string, code string) (int, []customError.Error) {
	parser := NewParser(NewLexer(filename, code))
	ast, errs := parser.Parse()
	if errs != nil && len(errs) > 0 {
		return 1, errs
	}
	homeScriptInterpreter := NewInterpreter(ast, executor)
	exitCode, err := homeScriptInterpreter.Run()
	if err != nil {
		return 1, []customError.Error{*err}
	}
	return exitCode, nil
}
