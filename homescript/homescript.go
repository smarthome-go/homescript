package homescript

import (
	"fmt"
	"time"

	"github.com/MikMuellerDev/homescript/homescript/interpreter"
)

type DummyExecutor struct{}

func (self DummyExecutor) Exit(code int) {
	// os.Exit(code)
}

func (self DummyExecutor) Print(args ...string) {
	for i, arg := range args {
		fmt.Print(arg)
		if i == len(args)-1 {
			fmt.Print(" ")
		}
	}
	fmt.Println()
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

// Runs a provided homescript file given the source code
// Returns an error slice
func Run(executor interpreter.Executor, code string) []error {
	parser := NewParser(NewLexer(code))
	ast, err := parser.Parse()
	homeScriptInterpreter := NewInterpreter(ast, executor)
	if err != nil && len(err) > 0 {
		return err
	}
	errRuntime := homeScriptInterpreter.Run()
	if errRuntime != nil {
		return []error{errRuntime}
	}
	return nil
}
