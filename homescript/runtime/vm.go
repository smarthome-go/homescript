package runtime

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

type VM struct {
	Program map[string][]compiler.Instruction
	Globals map[string]value.Value
	Cores   []Core
}

func NewVM(program map[string][]compiler.Instruction) VM {
	return VM{
		Program: program,
		Globals: make(map[string]value.Value),
		Cores:   make([]Core, 0),
	}
}

func hostcall(self *VM, function string, args []value.Value) (*value.Value, *value.Interrupt) {
	switch function {
	case "println":
		output := make([]string, 0)
		for _, arg := range args {
			disp, i := arg.Display()
			if i != nil {
				return nil, i
			}
			output = append(output, disp)
		}

		outStr := strings.Join(output, " ") + "\n"

		fmt.Print(outStr)

		return value.NewValueNull(), nil
	}

	panic("INVALID HOSTCALL: " + function)
}

func (self *VM) spawnCore() *Core {
	core := NewCore(&self.Program, hostcall, self)
	self.Cores = append(self.Cores, core)
	return &core
}

func (self *VM) Run(function string) {
	core := self.spawnCore()
	(*core).Run(function)
}
