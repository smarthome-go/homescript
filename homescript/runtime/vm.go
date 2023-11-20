package runtime

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

type VM struct {
	Program  compiler.Program
	Globals  map[string]value.Value
	Cores    []Core
	Executor value.Executor
}

func NewVM(program compiler.Program, executor value.Executor) VM {
	return VM{
		Program:  program,
		Globals:  make(map[string]value.Value),
		Cores:    make([]Core, 0),
		Executor: executor,
	}
}

func (self VM) SourceMap(frame CallFrame) errors.Span {
	return self.Program.SourceMap[frame.Function][frame.InstructionPointer]
}

func hostcall(self *VM, function string, args []*value.Value) (*value.Value, *value.Interrupt) {
	switch function {
	case "println":
		output := make([]string, 0)
		for _, arg := range args {
			disp, i := (*arg).Display()
			if i != nil {
				return nil, i
			}
			output = append(output, disp)
		}

		outStr := strings.Join(output, " ") + "\n"

		fmt.Print(outStr)

		return value.NewValueNull(), nil
	case "print":
		output := make([]string, 0)
		for _, arg := range args {
			disp, i := (*arg).Display()
			if i != nil {
				return nil, i
			}
			output = append(output, disp)
		}

		outStr := strings.Join(output, " ")

		fmt.Print(outStr)

		return value.NewValueNull(), nil
	}

	panic("INVALID HOSTCALL: " + function)
}

func (self *VM) spawnCore() *Core {
	core := NewCore(&self.Program.Functions, hostcall, self.Executor, self)
	self.Cores = append(self.Cores, core)
	return &core
}

func (self *VM) Run(function string, verbose bool) {
	core := self.spawnCore()
	(*core).Run(function, verbose)
}
