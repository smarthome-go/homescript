package runtime

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

type CallFrame struct {
	Function           string
	InstructionPointer uint
}

type Core struct {
	CallStack       []CallFrame
	Memory          map[string]*value.Value
	Stack           []*value.Value
	Program         *map[string][]compiler.Instruction
	Labels          map[string]uint // each index is relative to the function, but doesn't matter
	hostCall        func(*VM, string, []*value.Value) (*value.Value, *value.Interrupt)
	parent          *VM
	isAssignmentLhs bool
}

func NewCore(program *map[string][]compiler.Instruction, hostCall func(*VM, string, []*value.Value) (*value.Value, *value.Interrupt), vm *VM) Core {
	return Core{
		CallStack:       make([]CallFrame, 0),
		Memory:          make(map[string]*value.Value),
		Stack:           make([]*value.Value, 0),
		Program:         program,
		hostCall:        hostCall,
		parent:          vm,
		Labels:          make(map[string]uint),
		isAssignmentLhs: false,
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

func (self *Core) Run(function string) {
	self.pushCallStack(function)

	for len(self.CallStack) > 0 {
		callFrame := self.callFrame()
		fn := (*self.Program)[callFrame.Function]
		if callFrame.InstructionPointer >= uint(len(fn)) {
			fmt.Printf("Terminating from fn `%s` with ip=%d\n", callFrame.Function, callFrame.InstructionPointer)
			self.popCallStack()
			continue
		}

		i := fn[callFrame.InstructionPointer]

		fmt.Printf("I: %v | IP: %d | FP: %s | CLSTCK: %v | STCK: %v | MEM: %v\n", i, self.callFrame().InstructionPointer, self.callFrame().Function, self.CallStack, self.Stack, self.Memory)

		self.runInstruction(i)
	}
}

func (self *Core) runInstruction(instruction compiler.Instruction) *value.Interrupt {
	switch instruction.Opcode() {
	case compiler.Opcode_Nop:
		break
	case compiler.Opcode_Push:
		i := instruction.(compiler.ValueInstruction)
		v := i.Value
		self.push(&v)
	case compiler.Opcode_Drop:
		self.pop()
	case compiler.Opcode_Spawn:
		panic("TODO")
	case compiler.Opcode_Call_Val:
		panic("TODO")
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
		self.hostCall(self.parent, i.Value, args)
	case compiler.Opcode_Jump:
		i := instruction.(compiler.OneIntInstruction)
		self.callFrame().InstructionPointer = uint(i.Value)
		return nil // do not increment the new instruction
	case compiler.Opcode_JumpIfFalse:
		v := self.pop()

		// TODO: this check can be done more efficiently!!!
		isEqual, interrupt := (*v).IsEqual(*value.NewValueBool(false))
		if interrupt != nil {
			return interrupt
		}

		if isEqual {
			i := instruction.(compiler.OneIntInstruction)
			self.callFrame().InstructionPointer = uint(i.Value)
			return nil // do not increment the new instruction
		}
	case compiler.Opcode_GetVarImm:
		i := instruction.(compiler.OneStringInstruction)
		self.push(self.Memory[i.Value])
	case compiler.Opcode_GetGlobImm:
		panic("TODO")
	case compiler.Opcode_SetVatImm:
		i := instruction.(compiler.OneStringInstruction)
		v := self.pop()

		self.Memory[i.Value] = v
	case compiler.Opcode_SetGlobImm:
		panic("TODO")
	case compiler.Opcode_Assign: // assigns pointers on the stack???
		dest := self.pop()
		src := self.pop()

		// perform assignment here
		*dest = *src
	case compiler.Opcode_Cast:
		panic("TODO")
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
		panic("TODO")
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
			panic("Unsupported")
		}
	case compiler.Opcode_Sub:
		l := *self.pop()
		r := *self.pop()

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
			panic("Unsupported")
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
			panic("Unsupported")
		}
	case compiler.Opcode_Pow:
		panic("TODO")
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
			panic("Unsupported")
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
			panic("Unsupported")
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
			panic("Unsupported")
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
			panic("Unsupported")
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
			panic("Unsupported")
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
			panic("Unsupported")
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
			panic("Unsupported")
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
			panic("Unsupported")
		}
	case compiler.Opcode_BitOr:
		panic("TODO")
	case compiler.Opcode_BitAnd:
		panic("TODO")
	case compiler.Opcode_BitXor:
		panic("TODO")
	case compiler.Opcode_Index:
		panic("TODO")
	case compiler.Opcode_SetTryLabel:
		panic("TODO")
	case compiler.Opcode_Member:
		i := instruction.(compiler.OneStringInstruction)

		v := *self.pop()
		fields, interrupt := v.Fields()
		if interrupt != nil {
			return interrupt
		}

		self.push(value.NewValuePointer(fields[i.Value]))
	case compiler.Opcode_Import:
		panic("TODO")
	case compiler.Opcode_Label:
		panic("This should not happen")
	case compiler.Opcode_Into_Range:
		panic("TODO")
	}

	self.callFrame().InstructionPointer++
	return nil
}
