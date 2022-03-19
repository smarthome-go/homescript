package homescript

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/MikMuellerDev/homescript-dev/homescript/interpreter"
)

type DummyExecutor struct{}

func (self *DummyExecutor) Switch(name string, on bool) {
	fmt.Printf("Turning switch '%s' %t\n", name, on)
}
func (self *DummyExecutor) SwitchOn(name string) bool {
	if name == "s3" {
		return true
	}
	return false
}
func (self *DummyExecutor) Sleep(seconds int) {
	time.Sleep(time.Second * time.Duration(seconds))
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
func (self *DummyExecutor) Notify(
	title string,
	description string,
	level interpreter.NotificationLevel,
) {
	fmt.Printf("Sending notification with level %d '%s' -- '%s'\n", level, title, description)
}
func (self *DummyExecutor) Play(server string, mode string) {
	fmt.Printf("Playing '%s' on server '%s'\n", mode, server)
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
