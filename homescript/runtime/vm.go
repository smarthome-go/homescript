package runtime

import (
	"fmt"
	"strings"
	"sync"

	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

const CATCH_PANIC = false

type Globals struct {
	Data  map[string]value.Value
	Mutex sync.RWMutex
}

func newGlobals() Globals {
	return Globals{
		Data:  make(map[string]value.Value),
		Mutex: sync.RWMutex{},
	}

}

type Cores struct {
	Cores []Core
	Lock  sync.RWMutex
}

func newCores() Cores {
	return Cores{
		Cores: make([]Core, 0),
		Lock:  sync.RWMutex{},
	}
}

type VM struct {
	Program  compiler.Program
	Globals  Globals
	Cores    Cores
	Executor value.Executor
	Lock     sync.RWMutex
	coreCnt  uint
	Verbose  bool
}

func NewVM(program compiler.Program, executor value.Executor, verbose bool) VM {
	return VM{
		Program:  program,
		Globals:  newGlobals(),
		Cores:    newCores(),
		Executor: executor,
		Lock:     sync.RWMutex{},
		coreCnt:  0,
		Verbose:  verbose,
	}
}

func (self *VM) SourceMap(frame CallFrame) errors.Span {
	return self.Program.SourceMap[frame.Function][frame.InstructionPointer]
}

func hostcall(self *VM, function string, args []*value.Value) (*value.Value, *value.Interrupt) {
	// TODO: this is extremely bad!!!
	self.Lock.Lock()
	defer self.Lock.Unlock()

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
	self.Lock.Lock()
	defer self.Lock.Unlock()

	ch := make(chan *value.Interrupt)
	core := NewCore(&self.Program.Functions, hostcall, self.Executor, self, self.coreCnt, self.Verbose, ch)

	self.Cores.Lock.Lock()
	defer self.Cores.Lock.Unlock()

	self.Cores.Cores = append(self.Cores.Cores, core)
	self.coreCnt++
	return &core
}

func (self *VM) Spawn(function string, verbose bool) {
	core := self.spawnCore()
	catchPanic := func() {
		if err := recover(); err != nil {
			fmt.Printf("Panic occured in core %d: %s\n", core.Corenum, err)
		}
	}

	if CATCH_PANIC {
		defer catchPanic()
	}

	go (*core).Run(function)
}

func (self *VM) Wait() *value.Interrupt {
	for {
		self.Cores.Lock.RLock()
		for _, core := range self.Cores.Cores {
			select {
			case i := <-core.Handle:
				return i
			default:
			}
		}
		self.Cores.Lock.RUnlock()
	}
}