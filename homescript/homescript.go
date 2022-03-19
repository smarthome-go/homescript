package homescript

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/MikMuellerDev/homescript-dev/homescript/interpreter"
)

type DummyExecutor struct{}

func (self DummyExecutor) Exit(code int) {
	os.Exit(code)
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
	level interpreter.NotificationLevel,
) error {
	fmt.Printf("Sending notification with level %d '%s' -- '%s'\n", level, title, description)
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

func Test() {
	start := time.Now()
	content, err1 := ioutil.ReadFile("demo.hms")
	fmt.Printf("File Read: %v\n", time.Since(start))
	if err1 != nil {
		panic(err1.Error())
	}
	fmt.Printf("Parsing: %v\n", time.Since(start))
	parser := NewParser(NewLexer(string(content)))
	res, err := parser.Parse()
	if len(err) > 0 {
		for i := 0; i < len(err); i += 1 {
			fmt.Println(err[i].Error())
		}
		return
	}
	runner := NewInterpreter(res, DummyExecutor{})
	startRun := time.Now()
	errRuntime := runner.Run()
	if errRuntime != nil {
		fmt.Println(errRuntime.Error())
	}
	fmt.Printf("Execution: %v\n", time.Since(startRun))
	fmt.Printf("TOTAL: %v\n", time.Since(start))
}
