package runtime

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

const NUM_INSTRUCTIONS_EXECUTE_IN_VCYCLE = 50

type CallFrame struct {
	Function           string
	InstructionPointer uint
}

type Core struct {
	CallStack []CallFrame
	// FIXME: this is really bad, fix it
	// Replace with continuous memory, implement a stack pointer
	Memory          []*value.Value
	Stack           []*value.Value
	Program         *map[string][]compiler.Instruction
	Labels          map[string]uint // each index is relative to the function, but doesn't matter
	hostCall        func(*VM, string, []*value.Value) (*value.Value, *value.Interrupt)
	parent          *VM
	isAssignmentLhs bool
	Executor        value.Executor
	Corenum         uint
	Verbose         bool
	Handle          chan *value.Interrupt

	ExceptionCatchLabels []uint

	// TODO: write documentation
	MemoryPointer int64
	CancelCtx     *context.Context
	CancelFunc    *context.CancelFunc
}

func NewCore(
	program *map[string][]compiler.Instruction,
	hostCall func(*VM, string, []*value.Value) (*value.Value, *value.Interrupt),
	executor value.Executor,
	vm *VM,
	coreNum uint,
	verbose bool,
	handle chan *value.Interrupt,
	ctx *context.Context,
) Core {
	return Core{
		CallStack:       make([]CallFrame, 0),
		Memory:          make([]*value.Value, 1024), // TODO: configure memory
		Stack:           make([]*value.Value, 0),
		Program:         program,
		hostCall:        hostCall,
		parent:          vm,
		Labels:          make(map[string]uint),
		isAssignmentLhs: false,
		Executor:        executor,
		Corenum:         coreNum,
		Verbose:         verbose,
		Handle:          handle,
		CancelCtx:       ctx,
	}
}

func (self *Core) push(v *value.Value) {
	self.Stack = append(self.Stack, v)
}

func (self *Core) pop() *value.Value {
	v := self.Stack[len(self.Stack)-1]
	self.Stack = self.Stack[:len(self.Stack)-1]
	return v
}

func (self *Core) getStackTop() *value.Value {
	v := self.Stack[len(self.Stack)-1]
	return v
}

func (self *Core) pushCallStack(function string) {
	self.CallStack = append(self.CallStack, CallFrame{
		Function:           function,
		InstructionPointer: 0,
	})
}

func (self *Core) popCallStack() {
	self.CallStack = self.CallStack[:len(self.CallStack)-1]
}

func (self *Core) callFrame() *CallFrame {
	return &self.CallStack[len(self.CallStack)-1]
}

func (self *Core) absolute(rel int64) int {
	return int(rel + self.MemoryPointer)
}

func (self *Core) checkCancelation() *value.Interrupt {
	select {
	case <-(*self.CancelCtx).Done():
		span := self.parent.SourceMap(*self.callFrame())
		return value.NewTerminationInterrupt(context.Cause((*self.CancelCtx)).Error(), span)
	default:
		// do nothing, this should not block the entire interpreter
		return nil
	}
}

func (self *Core) Run(function string) {
	catchPanic := func() {
		if err := recover(); err != nil {
			fmt.Printf("Panic occured in core %d at (%s:%d): `%s`\n", self.Corenum, self.callFrame().Function, self.callFrame().InstructionPointer, err)
		}
	}

	if CATCH_PANIC {
		defer catchPanic()
	}

	self.pushCallStack(function)

outer:
	for len(self.CallStack) > 0 {
		fmt.Printf("======== CHECK CANCEL ==========\n")
		if i := self.checkCancelation(); i != nil {
			self.Handle <- i
			return
		}

		for c := 0; c < NUM_INSTRUCTIONS_EXECUTE_IN_VCYCLE; c++ {
			callFrame := *self.callFrame()
			fn, found := (*self.Program)[callFrame.Function]
			if !found {
				panic(fmt.Sprintf("Canot execute instructions of non-existent routine: %s", callFrame.Function))
			}

			if callFrame.InstructionPointer >= uint(len(fn)) { // TODO: len can be shortened
				fmt.Printf("Terminating from fn `%s` with ip=%d\n", callFrame.Function, callFrame.InstructionPointer)
				self.popCallStack()
				continue outer
			}

			i := fn[callFrame.InstructionPointer]

			// TODO: remove the verbose mode if possible
			if self.Verbose {
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

					disp, i := (*elem).Display()
					if i != nil {
						panic(*i)
					}
					mem = append(mem, fmt.Sprintf("%d=%s", key, strings.ReplaceAll(disp, "\n", " ")))
				}

				fmt.Printf("Corenum %d | I: %v | IP: %d | FP: %s | CLSTCK: %v | STCKSS=%d | STCK: %s | MEM: [%s]\n", self.Corenum, i, self.callFrame().InstructionPointer, self.callFrame().Function, self.CallStack, len(self.Stack), stack, strings.Join(mem, ", "))
				time.Sleep(10 * time.Millisecond)
			}

			if i := self.runInstruction(i); i != nil {
				switch (*i).Kind() {
				case value.ThrowInterruptKind:
					throwError := (*i).(value.ThrowInterrupt)

					if len(self.ExceptionCatchLabels) == 0 {
						self.Handle <- value.NewRuntimeErr(throwError.Message(), value.UncaughtThrowKind, throwError.Span)
						return
					}

					catchIndex := self.ExceptionCatchLabels[len(self.ExceptionCatchLabels)-1]
					self.callFrame().InstructionPointer = catchIndex

					self.push(
						value.NewValueObject(map[string]*value.Value{
							"message":  value.NewValueString(throwError.Message()),
							"line":     value.NewValueInt(int64(throwError.Span.Start.Line)),
							"column":   value.NewValueInt(int64(throwError.Span.Start.Column)),
							"filename": value.NewValueString(throwError.Span.Filename),
						}))
				default:
					self.Handle <- i
					return
				}
			}
		}
	}

	self.Handle <- nil
}

func (self *Core) runInstruction(instruction compiler.Instruction) *value.Interrupt {
	switch instruction.Opcode() {
	case compiler.Opcode_Nop:
		break
	case compiler.Opcode_AddMempointer:
		i := instruction.(compiler.OneIntInstruction)
		self.MemoryPointer += i.Value
	case compiler.Opcode_Push:
		i := instruction.(compiler.ValueInstruction)
		v := i.Value
		self.push(&v)
	case compiler.Opcode_Drop:
		self.pop()
	case compiler.Opcode_Duplicate:
		// TODO: does this break? when copying the pointer?
		self.push(self.getStackTop())
	case compiler.Opcode_Spawn:
		i := instruction.(compiler.OneStringInstruction)

		// TODO: implement deepcopy

		args := make([]value.Value, 0)
		numArgs := (*self.pop()).(value.ValueInt).Inner
		for i := 0; i < int(numArgs); i++ {
			args = append([]value.Value{*self.pop()}, args...) // TODO: implement deepcopy
		}

		self.parent.spawnCoreInternal(i.Value, args)
		// TODO: get thread handle
		self.push(value.NewValueNull())
	case compiler.Opcode_Call_Val:
		n := *self.pop()
		numArgs := n.(value.ValueInt).Inner

		v := *self.pop()
		switch v.Kind() {
		// TODO: support arguments
		case value.VmFunctionValueKind:
			function := v.(value.ValueVMFunction)

			self.callFrame().InstructionPointer++
			self.pushCallStack(function.Ident)

			fmt.Printf("calling: %s\n", function.Ident)

			return nil
		case value.BuiltinFunctionValueKind:
			fn := v.(value.ValueBuiltinFunction)

			args := make([]value.Value, 0)
			for i := 0; i < int(numArgs); i++ {
				v := *self.pop()
				args = append(args, v)
			}

			res, i := fn.Callback(
				self.Executor,
				self.CancelCtx,
				self.parent.SourceMap(*self.callFrame()),
				args...,
			)
			if i != nil {
				return i
			}

			if (*res).Kind() != value.NullValueKind {
				self.push(res)
			}
		default:
			panic(fmt.Sprintf("Values of kind %s cannot be called", v.Kind()))
		}
	case compiler.Opcode_Call_Imm:
		i := instruction.(compiler.OneStringInstruction)
		self.callFrame().InstructionPointer++
		self.pushCallStack(i.Value)
		return nil
	case compiler.Opcode_Return:
		self.popCallStack()
	case compiler.Opcode_HostCall:
		i := instruction.(compiler.OneStringInstruction)

		raw := self.pop()
		argc := int((*raw).(value.ValueInt).Inner)
		args := make([]*value.Value, 0)
		for i := 0; i < argc; i++ {
			args = append(args, self.pop())
		}
		v, interrupt := self.hostCall(self.parent, i.Value, args)
		if interrupt != nil {
			return interrupt
		}

		self.push(v)
	case compiler.Opcode_Jump:
		i := instruction.(compiler.OneIntInstruction)
		self.callFrame().InstructionPointer = uint(i.Value)
		return nil // do not increment the new instruction
	case compiler.Opcode_JumpIfFalse:
		v := *self.pop()

		if !v.(value.ValueBool).Inner {
			i := instruction.(compiler.OneIntInstruction)
			self.callFrame().InstructionPointer = uint(i.Value)
			return nil // do not increment the new instruction
		}
	case compiler.Opcode_GetVarImm:
		i := instruction.(compiler.OneIntInstruction)
		self.push(self.Memory[self.absolute(i.Value)])
	case compiler.Opcode_GetGlobImm:
		i := instruction.(compiler.OneStringInstruction)
		self.parent.Globals.Mutex.RLock()
		defer self.parent.Globals.Mutex.RUnlock()

		v := self.parent.Globals.Data[i.Value]
		self.push(&v)
	case compiler.Opcode_SetVarImm:
		i := instruction.(compiler.OneIntInstruction)
		v := self.pop()

		self.Memory[self.absolute(i.Value)] = v
	case compiler.Opcode_SetGlobImm:
		i := instruction.(compiler.OneStringInstruction)
		self.parent.Globals.Mutex.Lock()
		defer self.parent.Globals.Mutex.Unlock()

		v := self.pop()
		self.parent.Globals.Data[i.Value] = *v
	case compiler.Opcode_Assign: // assigns pointers on the stack???
		src := self.pop()
		dest := self.pop()

		// perform assignment here
		*dest = *src
	case compiler.Opcode_Cast:
		i := instruction.(compiler.CastInstruction)
		v := self.pop()

		casted, interrupt := value.DeepCast(*v, i.Type, self.parent.SourceMap(*self.callFrame()))
		if interrupt != nil {
			return interrupt
		}
		self.push(casted)
	case compiler.Opcode_Neg:
		v := *self.pop()

		switch v.Kind() {
		case value.IntValueKind:
			intV := v.(value.ValueInt)
			self.push(value.NewValueInt(-intV.Inner))
		case value.FloatValueKind:
			floatV := v.(value.ValueFloat)
			self.push(value.NewValueFloat(-floatV.Inner))
		default:
			panic("Unsupported value kind: " + v.Kind().String())
		}
	case compiler.Opcode_Some: // ?foo -> converts foo to a Option<foo>
		v := *self.pop()
		self.push(value.NewValueOption(&v))
	case compiler.Opcode_Not:
		v := *self.pop()
		boolV := v.(value.ValueBool)
		self.push(value.NewValueBool(!boolV.Inner))
	case compiler.Opcode_Add:
		l := *self.pop()
		r := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner + rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			self.push(value.NewValueFloat(lFloat.Inner + rFloat.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Sub:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner - rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			self.push(value.NewValueFloat(lFloat.Inner - rFloat.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Mul:
		l := *self.pop()
		r := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner * rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			self.push(value.NewValueFloat(lFloat.Inner * rFloat.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Pow:
		// TODO: improve performance here
		r := (*self.pop()).(value.ValueInt).Inner
		l := (*self.pop()).(value.ValueInt).Inner
		res := math.Pow(float64(l), float64(r))
		self.push(value.NewValueInt(int64(res)))
	case compiler.Opcode_Div:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner / rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			self.push(value.NewValueFloat(lFloat.Inner / rFloat.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Rem:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner % rInt.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Eq:
		l := *self.pop()
		r := *self.pop()

		eq, i := l.IsEqual(r)
		if i != nil {
			return i
		}

		self.push(value.NewValueBool(eq))
	// Only pops the stack once, the other value is left untouched
	case compiler.Opcode_Eq_PopOnce:
		l := *self.pop()
		r := *self.getStackTop()

		eq, i := l.IsEqual(r)
		if i != nil {
			return i
		}

		self.push(value.NewValueBool(eq))
	case compiler.Opcode_Lt:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueBool(lInt.Inner < rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			self.push(value.NewValueBool(lFloat.Inner < rFloat.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Gt:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueBool(lInt.Inner > rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			self.push(value.NewValueBool(lFloat.Inner > rFloat.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Le:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueBool(lInt.Inner <= rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			self.push(value.NewValueBool(lFloat.Inner <= rFloat.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Ge:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueBool(lInt.Inner >= rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			self.push(value.NewValueBool(lFloat.Inner >= rFloat.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Shl:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner << rInt.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Shr:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner >> rInt.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_BitOr:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner | rInt.Inner))
		case value.BoolValueKind:
			lBool := l.(value.ValueBool)
			rBool := r.(value.ValueBool)
			self.push(value.NewValueBool(lBool.Inner || rBool.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_BitAnd:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner & rInt.Inner))
		case value.BoolValueKind:
			lBool := l.(value.ValueBool)
			rBool := r.(value.ValueBool)
			self.push(value.NewValueBool(lBool.Inner && rBool.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_BitXor:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner ^ rInt.Inner))
		case value.BoolValueKind:
			lBool := l.(value.ValueBool)
			rBool := r.(value.ValueBool)
			self.push(value.NewValueBool(lBool.Inner != rBool.Inner))
		default:
			panic("This value combination is unsupported")
		}
	case compiler.Opcode_Index:
		indexV := self.pop()
		baseV := self.pop()

		// Only used for performance reasons
		span := func() errors.Span {
			return self.parent.SourceMap(*self.callFrame())
		}

		indexed, interrupt := value.IndexValue(baseV, indexV, span)
		if interrupt != nil {
			return interrupt
		}
		self.push(indexed)
	case compiler.Opcode_Throw:
		v := *self.pop()

		display, i := v.Display()
		if i != nil {
			return i
		}

		self.callFrame().InstructionPointer++

		return value.NewThrowInterrupt(
			self.parent.SourceMap(*self.callFrame()),
			display,
		)
	case compiler.Opcode_SetTryLabel:
		i := instruction.(compiler.OneIntInstruction)
		self.ExceptionCatchLabels = append(self.ExceptionCatchLabels, uint(i.Value))
	case compiler.Opcode_PopTryLabel:
		self.ExceptionCatchLabels = self.ExceptionCatchLabels[:len(self.ExceptionCatchLabels)-1]
	case compiler.Opcode_Member:
		i := instruction.(compiler.OneStringInstruction)

		v := *self.pop()
		fields, interrupt := v.Fields()
		if interrupt != nil {
			return interrupt
		}

		field, found := fields[i.Value]
		if !found {
			span := self.parent.SourceMap(*self.callFrame())
			panic(fmt.Sprintf("Field `%s` not found on `%v`: %s:%d:%d", i.Value, v, span.Filename, span.Start.Index, span.Start.Column))
		}
		self.push(field)
	case compiler.Opcode_Import:
		i := instruction.(compiler.TwoStringInstruction)
		v := self.importItem(i.Values[0], i.Values[1])
		self.push(&v)
	case compiler.Opcode_Into_Range:
		end := *self.pop()
		start := *self.pop()
		self.push(value.NewValueRange(start, end))
	case compiler.Opcode_IntoIter:
		v := *self.pop()
		self.push(value.NewValueIter(v))
	case compiler.Opcode_IteratorAdvance:
		// Get the iterator from the stack.
		iterator := (*self.getStackTop()).(value.ValueIterator).Func
		val, shallContinue := iterator()
		self.push(&val)
		self.push(value.NewValueBool(shallContinue))
	default:
		panic(fmt.Sprintf("Illegal instruction erorr: %v", instruction))
	}

	self.callFrame().InstructionPointer++
	return nil
}
