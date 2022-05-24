package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/smarthome-go/homescript/homescript"
	"github.com/smarthome-go/homescript/homescript/error"
)

func printError(err error.Error, program string) {
	lines := strings.Split(program, "\n")

	line1 := ""
	if err.Location.Line > 1 {
		line1 = fmt.Sprintf("\n \x1b[90m%- 3d | \x1b[0m%s", err.Location.Line-1, lines[err.Location.Line-2])
	}
	line2 := fmt.Sprintf(" \x1b[90m%- 3d | \x1b[0m%s", err.Location.Line, lines[err.Location.Line-1])
	line3 := ""
	if int(err.Location.Line) < len(lines) {
		line3 = fmt.Sprintf("\n \x1b[90m%- 3d | \x1b[0m%s", err.Location.Line+1, lines[err.Location.Line])
	}

	marker := fmt.Sprintf("%s\x1b[1;31m^\x1b[0m", strings.Repeat(" ", int(err.Location.Column+6)))

	fmt.Printf(
		"\x1b[1;36m%s\x1b[39m at %s:%d:%d\x1b[0m\n%s\n%s\n%s%s\n\n\x1b[1;31m%s\x1b[0m\n",
		err.TypeName,
		err.Location.Filename,
		err.Location.Line,
		err.Location.Column,
		line1,
		line2,
		marker,
		line3,
		err.Message,
	)
}

func main() {
	program := `
print('hello\nthere');
switch('s1', on);
exec('coming_home');
print(3.14);
addUser('test','test','','')
addPerm('test', 'test')
delPerm('test', 'test')
delUser('test')
print(get('http://localhost:8082'))
print(http('http://localhost:8082', 'POST', ''))
exit(0);
print('unreachable');
`
	code, errors := homescript.Run(homescript.DummyExecutor{}, "<demo>", program)
	for _, err := range errors {
		printError(err, program)
		fmt.Println()
	}
	os.Exit(code)
}
