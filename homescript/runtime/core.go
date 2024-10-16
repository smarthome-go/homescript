package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

// How many instructions the VM should execute "at once".
// After such a cycle is over, the VM checks if there is a cancelation request.
// This way, there is only a minimal performance impact.
const NUM_INSTRUCTIONS_EXECUTE_PER_VCYCLE = 50

// Whether the VM should print its current state for each cycle.
type VMVerboseMode uint8

const (
	VMNotVerbose VMVerboseMode = iota
	VMVerbose
	VMVeryVerbose
)

const vmVerbose = VMNotVerbose

const VM_DEBUGGER = false

// const VM_DEBUGGER_SLEEP = 1000 * time.Millisecond

type CallFrame struct {
	Function           string
	InstructionPointer uint
}

type Core struct {
	CallStack []CallFrame
	// Replace with continuous memory, implement a stack pointer
	Memory  []*value.Value
	Stack   []*value.Value
	Program *map[string][]compiler.Instruction
	// Each index is relative to the function, but doesn't matter
	Labels map[string]uint
	// TODO: maybe remove hostCall entirely
	hostCall     func(*VM, string, errors.Span, []*value.Value) (*value.Value, *value.VmInterrupt)
	parent       *VM
	Executor     value.Executor
	Corenum      uint
	SignalHandle chan *value.VmInterrupt

	// A `stack` of labels to jump to if an exception is raised
	ExceptionCatchLabels []CallFrame

	// Points to the start of the current stackframe
	// Then, the absolute index can be computed by adding the value of mp and the relative offset of the memory location.
	MemoryPointer int64
	// If this is triggered, execution is terminated
	CancelCtx *context.Context
	// Describes some resource limits for the current core
	Limits CoreLimits
}

type CoreLimits struct {
	CallStackMaxSize uint
	StackMaxSize     uint
	MaxMemorySize    uint
}

func NewCore(
	program *map[string][]compiler.Instruction,
	hostCall func(*VM, string, errors.Span, []*value.Value) (*value.Value, *value.VmInterrupt),
	executor value.Executor,
	vm *VM,
	coreNum uint,
	handle chan *value.VmInterrupt,
	ctx *context.Context,
	limits CoreLimits,
) Core {
	return Core{
		CallStack:            make([]CallFrame, 0),
		Memory:               make([]*value.Value, limits.MaxMemorySize),
		Stack:                make([]*value.Value, 0),
		Program:              program,
		Labels:               make(map[string]uint),
		hostCall:             hostCall,
		parent:               vm,
		Executor:             executor,
		Corenum:              coreNum,
		SignalHandle:         handle,
		ExceptionCatchLabels: []CallFrame{},
		MemoryPointer:        0,
		CancelCtx:            ctx,
		Limits:               limits,
	}
}

func (core *Core) push(v *value.Value) {
	core.Stack = append(core.Stack, v)
}

func (core *Core) pop() *value.Value {
	v := core.Stack[len(core.Stack)-1]
	core.Stack = core.Stack[:len(core.Stack)-1]
	return v
}

func (core *Core) getStackTop() *value.Value {
	v := core.Stack[len(core.Stack)-1]
	return v
}

func (core *Core) pushCallStack(function string) {
	core.CallStack = append(core.CallStack, CallFrame{
		Function:           function,
		InstructionPointer: 0,
	})
}

func (core *Core) popCallStack() {
	core.CallStack = core.CallStack[:len(core.CallStack)-1]
}

func (core *Core) callFrame() *CallFrame {
	return &core.CallStack[len(core.CallStack)-1]
}

func (core *Core) absolute(rel int64) int {
	return int(core.MemoryPointer - rel)
}

func (core *Core) checkCancelation() *value.VmInterrupt {
	select {
	case <-(*core.CancelCtx).Done():
		span := core.parent.SourceMap(*core.callFrame())
		return value.NewVMTerminationInterrupt(context.Cause((*core.CancelCtx)).Error(), span)
	default:
		// Do nothing, this should not block the entire VM
		return nil
	}
}

type DebugOutput struct {
	CurrentInstruction compiler.Instruction
	CurrentSpan        errors.Span
	CurrentCallFrame   CallFrame
}

func (self *Core) Run(function string, debuggerOut *chan DebugOutput, debuggerResume *chan struct{}) {
	if debuggerOut != nil {
		defer close(*debuggerOut)
		defer close(*debuggerResume)
	}

	catchPanic := func() {
		if err := recover(); err != nil {
			span := self.parent.SourceMap(*self.callFrame())
			fmt.Printf("Panic occurred in core %d at (%s:%d => l.%d): `%s`\n", self.Corenum, self.callFrame().Function, self.callFrame().InstructionPointer, span.Start.Line, err)
		}
	}

	if shouldCatchPanic {
		defer catchPanic()
	}

	self.pushCallStack(function)

outer:
	for len(self.CallStack) > 0 {
		// Check cancelation
		if i := self.checkCancelation(); i != nil {
			self.SignalHandle <- i
			return
		}

		// Check for stack overflow
		if len(self.Stack) > int(self.Limits.StackMaxSize) {
			self.SignalHandle <- self.fatalErr(
				fmt.Sprintf("Runtime stack limit of %d was exceeded by %d", self.Limits.StackMaxSize, len(self.Stack)-int(self.Limits.StackMaxSize)),
				value.VMFatalExceptionKind(value.Vm_StackOverFlowErrorKind),
				self.parent.SourceMap(self.CallStack[len(self.CallStack)-2]),
			)
			return
		}

		// Check for callstack overflows
		if len(self.CallStack) > int(self.Limits.CallStackMaxSize) {
			self.SignalHandle <- self.fatalErr(
				fmt.Sprintf("Runtime callstack limit of %d was exceeded by %d", self.Limits.CallStackMaxSize, len(self.CallStack)-int(self.Limits.CallStackMaxSize)),
				value.Vm_StackOverFlowErrorKind,
				self.parent.SourceMap(*self.callFrame()),
			)
			return
		}

		for c := 0; c < NUM_INSTRUCTIONS_EXECUTE_PER_VCYCLE; c++ {
			// TODO: this can be implemented nicer.
			if len(self.CallStack) == 0 {
				break outer
			}

			callFrame := self.callFrame()
			fn, found := (*self.Program)[callFrame.Function]
			if !found || len(fn) == 0 {
				panic(fmt.Sprintf("Cannot execute instructions of non-existent routine: %s", callFrame.Function))
			}

			if callFrame.InstructionPointer >= uint(len(fn)) { // TODO: len can be shortened
				self.popCallStack() // TODO: is this really required?
				continue outer
			}

			i := fn[callFrame.InstructionPointer]

			// Because this condition is constant, it should be optimized away
			if vmVerbose != VMNotVerbose {
				stack := make([]string, 0)
				for _, elem := range self.Stack {
					if elem == nil || *elem == nil {
						stack = append(stack, "<nil>")
					} else {
						disp, i := (*elem).Display()
						if i != nil {
							panic(*i)
						}
						stack = append(stack, strings.ReplaceAll(disp, "\n", ""))
					}
				}

				mem := make([]string, 0)
				for key, elem := range self.Memory {
					if elem == nil {
						continue
					}

					var disp string
					if *elem == nil {
						// This can occur if an iterator is terminated
						disp = "<nil>"
					} else {
						dispTemp, i := (*elem).Display()
						if i != nil {
							panic(*i)
						}
						disp = dispTemp
					}

					mem = append(mem, fmt.Sprintf("%d=%s", key, strings.ReplaceAll(disp, "\n", " ")))
				}

				globals := make([]string, 0)
				for key, elem := range self.parent.globals.Data {
					if elem == nil {
						continue
					}

					var disp string
					if elem == nil {
						disp = "<nil>"
					} else {
						if elem.Kind() == value.ObjectValueKind {
							disp = "<obj>"
						} else {
							dispTemp, i := elem.Display()
							if i != nil {
								panic(*i)
							}
							disp = dispTemp
						}
					}

					globals = append(globals, fmt.Sprintf("%s=%s", key, strings.ReplaceAll(disp, "\n", " ")))
				}

				switch vmVerbose {
				case VMVerbose:
					fmt.Printf("Corenum %d | I: %v | IP: %d | FP: %s\n", self.Corenum, i, self.callFrame().InstructionPointer, self.callFrame().Function)
				case VMVeryVerbose:
					fmt.Printf("Corenum %d | I: %v | IP: %d | FP: %s MP=%d | CLSTCK: %v | STCKSS=%d | STCK: [%s] | MEM: [%s] | GLOB:  [%s]\n", self.Corenum, i, self.callFrame().InstructionPointer, self.callFrame().Function, self.MemoryPointer, self.CallStack, len(self.Stack), strings.Join(stack, ", "), strings.Join(mem, ", "), strings.Join(globals, ", "))
				default:
					panic("New VM verbose mode added without updating this code.")
				}

				time.Sleep(10 * time.Millisecond)
			}

			if VM_DEBUGGER {
				// If there is a debugger attached, send it information
				if debuggerOut != nil && debuggerResume != nil {
					*debuggerOut <- DebugOutput{
						CurrentInstruction: i,
						CurrentSpan:        self.parent.SourceMap(*self.callFrame()),
						CurrentCallFrame:   *self.callFrame(),
					}

					<-*debuggerResume
				}
			}

			if i := self.runInstruction(i); i != nil {
				switch (*i).Kind() {
				// Only non-fatal exceptions can be handled
				case value.Vm_NormalExceptionInterruptKind:
					throwError := (*i).(value.Vm_NormalException)

					// If there is no catch-block, terminate this core
					if len(self.ExceptionCatchLabels) == 0 {
						self.SignalHandle <- self.fatalErr(throwError.Message(), value.Vm_UncaughtThrowKind, throwError.Span)
						return
					}

					// If the exception occurred in another function, also pop the call frame of this function
					// If this was not the case, a function would basically "return twice",
					// as the jump to the error-handling code would not pop the most current call frame.
					catchLocation := self.ExceptionCatchLabels[len(self.ExceptionCatchLabels)-1]
					if self.callFrame().Function != catchLocation.Function {
						self.popCallStack()
					}
					*self.callFrame() = catchLocation

					self.push(
						value.NewValueObject(map[string]*value.Value{
							"message":  value.NewValueString(throwError.Message()),
							"line":     value.NewValueInt(int64(throwError.Span.Start.Line)),
							"column":   value.NewValueInt(int64(throwError.Span.Start.Column)),
							"filename": value.NewValueString(throwError.Span.Filename),
						}))
				default:
					self.SignalHandle <- i // TODO: add universal stacktrace
					return
				}
			}
		}
	}

	self.SignalHandle <- nil
}
