package homescript

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/MikMuellerDev/homescript-dev/homescript/interpreter"
)

type DummyExecutor struct{}

func (self *DummyExecutor) Exit(code int) {
	os.Exit(code)
}
func (self *DummyExecutor) Print(args ...string) {
	for i, arg := range args {
		fmt.Print(arg)
		if i == len(args)-1 {
			fmt.Print(" ")
		}
	}
	fmt.Println()
}
func (self *DummyExecutor) SwitchOn(name string) bool {
	if name == "s3" {
		return true
	}
	return false
}
func (self *DummyExecutor) Switch(name string, on bool) {
	fmt.Printf("Turning switch '%s' %t\n", name, on)
}
func (self *DummyExecutor) Play(server string, mode string) {
	fmt.Printf("Playing '%s' on server '%s'\n", mode, server)
}
func (self *DummyExecutor) Notify(
	title string,
	description string,
	level interpreter.NotificationLevel,
) {
	fmt.Printf("Sending notification with level %d '%s' -- '%s'\n", level, title, description)
}
func (self *DummyExecutor) GetUser() string {
	return "admin"
}
func (self *DummyExecutor) GetWeather() string {
	return "rainy"
}
func (self *DummyExecutor) GetTemperature() int {
	return 42
}
func (self *DummyExecutor) GetDate() (int, int, int, int, int, int) {
	now := time.Now()
	return now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second()
}

func Test() {
	content, err1 := ioutil.ReadFile("demo.hms")
	if err1 != nil {
		panic(err1.Error())
	}

	parser := NewParser(NewLexer(string(content)))
	res, err := parser.Parse()
	if len(err) > 0 {
		for i := 0; i < len(err); i += 1 {
			fmt.Println(err[i].Error())
		}
		return
	}
	fmt.Println(res)

	// lexer := NewLexer(program)
	// for {
	// 	res, err := lexer.Scan()
	// 	if err != nil {
	// 		panic(err.Error())
	// 	}
	// 	fmt.Println(res.Value)
	// 	if res.TokenType == EOF {
	// 		return
	// 	}
	// }
}
