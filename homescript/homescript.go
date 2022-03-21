package homescript

import (
	"fmt"
	"os"
	"time"

	"github.com/MikMuellerDev/homescript/homescript/error"
	"github.com/MikMuellerDev/homescript/homescript/interpreter"
)

type DummyExecutor struct{}

func (self DummyExecutor) Exit(code int) {
	os.Exit(code)
}
func (self DummyExecutor) Print(args ...string) {
	output := ""
	for _, arg := range args {
		output += arg
	}
	fmt.Println(output)
}
func (self DummyExecutor) SwitchOn(name string) (bool, *error.Error) {
	if name == "s3" {
		return true, nil
	}
	return false, nil
}
func (self DummyExecutor) Switch(name string, on bool) *error.Error {
	fmt.Printf("Turning switch '%s' %t\n", name, on)
	return nil
}
func (self DummyExecutor) Play(server string, mode string) *error.Error {
	fmt.Printf("Playing '%s' on server '%s'\n", mode, server)
	return nil
}
func (self DummyExecutor) Notify(
	title string,
	description string,
	level interpreter.LogLevel,
) *error.Error {
	fmt.Printf("Sending notification with level %d '%s' -- '%s'\n", level, title, description)
	return nil
}
func (self DummyExecutor) Log(
	title string,
	description string,
	level interpreter.LogLevel,
) *error.Error {
	fmt.Printf("Logging '%s' -- '%s' with level %d\n", title, description, level)
	return nil
}
func (self DummyExecutor) GetUser() string {
	return "admin"
}
func (self DummyExecutor) GetWeather() (string, *error.Error) {
	return "rainy", nil
}
func (self DummyExecutor) GetTemperature() (int, *error.Error) {
	return 42, nil
}
func (self DummyExecutor) GetDate() (int, int, int, int, int, int) {
	now := time.Now()
	return now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second()
}

// Runs a provided homescript file given the source code
// Returns an error slice
func Run(executor interpreter.Executor, filename string, code string) (int, []error.Error) {
	parser := NewParser(NewLexer(filename, code))
	ast, err := parser.Parse()
	if err != nil && len(err) > 0 {
		return 1, err
	}
	homeScriptInterpreter := NewInterpreter(ast, executor)
	exitCode, errRuntime := homeScriptInterpreter.Run()
	if errRuntime != nil {
		return 1, []error.Error{*errRuntime}
	}
	return exitCode, nil
}
