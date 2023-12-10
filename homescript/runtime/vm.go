package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

const CATCH_PANIC = false

type Globals struct {
	Data  map[string]value.Value
	Mutex sync.RWMutex
}

func newGlobals(scopeAdditions map[string]value.Value) Globals {
	return Globals{
		Data:  scopeAdditions,
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
	Program       compiler.Program
	Globals       Globals
	Cores         Cores
	Executor      value.Executor
	Lock          sync.RWMutex
	coreCnt       uint
	CancelCtx     *context.Context
	CancelFunc    *context.CancelFunc
	Interrupts    map[uint]value.VmInterrupt
	LimitsPerCore CoreLimits
}

func NewVM(
	program compiler.Program,
	executor value.Executor,
	ctx *context.Context,
	cancelFunc *context.CancelFunc,
	scopeAdditions map[string]value.Value,
	limits CoreLimits,
) VM {
	return VM{
		Program:       program,
		Globals:       newGlobals(scopeAdditions),
		Cores:         newCores(),
		Executor:      executor,
		Lock:          sync.RWMutex{},
		coreCnt:       0,
		CancelCtx:     ctx,
		CancelFunc:    cancelFunc,
		Interrupts:    make(map[uint]value.VmInterrupt),
		LimitsPerCore: limits,
	}
}

func hostcall(self *VM, function string, args []*value.Value) (*value.Value, *value.VmInterrupt) {
	// TODO: this is extremely bad!!!
	self.Lock.Lock()
	defer self.Lock.Unlock()

	switch function {
	case "__internal_list_push":
		elem := args[0]
		list := (*args[1]).(value.ValueList)

		(*list.Values) = append((*list.Values), elem)
		return args[1], nil
	}

	panic("Invalid hostcall: " + function)
}

func (self *VM) spawnCore() *Core {
	self.Lock.Lock()
	defer self.Lock.Unlock()

	ch := make(chan *value.VmInterrupt)
	core := NewCore(&self.Program.Functions, hostcall, self.Executor, self, self.coreCnt, ch, self.CancelCtx, self.LimitsPerCore)

	self.Cores.Lock.Lock()
	defer self.Cores.Lock.Unlock()

	self.Cores.Cores = append(self.Cores.Cores, core)
	self.coreCnt++
	return &core
}

// Returns the corenum of the newly spawned process
func (self *VM) Spawn(function string, debuggerOut *chan DebugOutput) uint {
	return self.spawnCoreInternal(function, make([]value.Value, 0), debuggerOut)
}

// Returns the corenum of the newly spawned process
func (self *VM) spawnCoreInternal(function string, addToStack []value.Value, debuggerOutput *chan DebugOutput) uint {
	core := self.spawnCore()
	for _, elem := range addToStack {
		// TODO: However, the VM should not do this implicitly,
		// Smarter would be to insert clones manually?
		core.push(&elem) // Implement a deep copy? Or clone?
	}
	go (*core).Run(function, debuggerOutput)
	return core.Corenum
}

func (self *VM) WaitNonConsuming() {
	for {
		self.Cores.Lock.RLock()

		if len(self.Cores.Cores) == 0 {
			fmt.Printf("breakout..")
			break
		}

		self.Cores.Lock.RUnlock()
	}
}

func (self *VM) Wait() (uint, *value.VmInterrupt) {
	for {
		self.Cores.Lock.RLock()
		for _, core := range self.Cores.Cores {
			// fmt.Printf("checking core: %d | %v\n", core.Corenum, time.Now())

			select {
			case i := <-core.SignalHandle:
				if i == nil {
					newCores := make([]Core, 0)

					for _, coreIter := range self.Cores.Cores {
						if coreIter.Corenum == core.Corenum {
							continue
						}

						newCores = append(newCores, coreIter)
					}

					self.Cores.Lock.RUnlock()

					self.Cores.Lock.Lock()
					self.Cores.Cores = newCores
					self.Cores.Lock.Unlock()

					self.Cores.Lock.RLock()
				} else {

					self.Cores.Lock.RUnlock()

					// TODO: is this OK?
					self.Cores.Lock.Lock()

					(*self.CancelFunc)()

					self.Cores.Cores = make([]Core, 0)
					self.Cores.Lock.Unlock()

					self.Cores.Lock.RLock()

					return core.Corenum, i
				}
			default:
			}
		}

		if len(self.Cores.Cores) == 0 {
			break
		}

		self.Cores.Lock.RUnlock()
	}

	return 0, nil
}
