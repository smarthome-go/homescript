package homescript

import (
	"fmt"
	"time"

	customError "github.com/smarthome-go/homescript/homescript/error"
	"github.com/smarthome-go/homescript/homescript/interpreter"
)

type DummyExecutor struct{}

func (self DummyExecutor) Print(args ...string) {
	output := ""
	for _, arg := range args {
		output += arg
	}
	fmt.Println(output)
}

func (self DummyExecutor) CheckArg(toCheck string) bool {
	return toCheck == "ok"
}

func (self DummyExecutor) GetArg(toGet string) (string, error) {
	if toGet == "ok" {
		return "ok", nil
	}
	return "", fmt.Errorf("No such argument provided: the argument '%s' was not found", toGet)
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
func (self DummyExecutor) Exec(homescriptId string, args map[string]string) (string, error) {
	fmt.Printf("Executing script: '%s'\n", homescriptId)
	return "", nil
}
func (self DummyExecutor) AddUser(username string, password string, forename string, surname string) error {
	fmt.Printf("Created user '%s'.\n", username)
	return nil
}
func (self DummyExecutor) DelUser(username string) error {
	fmt.Printf("Deleted user '%s'.\n", username)
	return nil
}
func (self DummyExecutor) AddPerm(username string, permission string) error {
	fmt.Printf("Added permission '%s' to user '%s'.\n", permission, username)
	return nil
}
func (self DummyExecutor) DelPerm(username string, permission string) error {
	fmt.Printf("Removed permission '%s' from user '%s'.\n", permission, username)
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
func (self DummyExecutor) Get(url string) (string, error) {
	return "response", nil
}

func (self DummyExecutor) Http(url string, method string, contentType string, body string) (string, error) {
	return "response", nil
}

// Runs provided Homescript code given the source code
// Returns an error slice
func Run(executor interpreter.Executor, filename string, code string) (int, []customError.Error) {
	parser := NewParser(NewLexer(filename, code))
	ast, errs := parser.Parse()
	if len(errs) > 0 {
		return 1, errs
	}
	homeScriptInterpreter := NewInterpreter(ast, executor)
	exitCode, err := homeScriptInterpreter.Run()
	if err != nil {
		return 1, []customError.Error{*err}
	}
	return exitCode, nil
}
