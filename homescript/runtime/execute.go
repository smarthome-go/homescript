package runtime

import (
	"fmt"
	"math"

	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

func (self *Core) runInstruction(instruction compiler.Instruction) *value.VmInterrupt {
	switch instruction.Opcode() {
	case compiler.Opcode_Nop:
		break
	case compiler.Opcode_AddMempointer:
		i := instruction.(compiler.OneIntInstruction)
		self.MemoryPointer += i.Value

		if int(self.MemoryPointer) >= int(self.Limits.MaxMemorySize) {
			return self.fatalErr(
				fmt.Sprintf("Memory capacity of %d variables was exceeded (mp=%d)", self.MemoryPointer, self.Limits.MaxMemorySize),
				value.Vm_OutOfMemoryErrorKind,
				self.parent.SourceMap(*self.callFrame()),
			)
		}
	case compiler.Opcode_Push:
		i := instruction.(compiler.ValueInstruction)
		v := i.Value
		self.push(&v)
	case compiler.Opcode_Drop:
		self.pop()
	case compiler.Opcode_Duplicate:
		// TODO: analyze where this instruction is generated and if it could break stuff
		// TODO: does this break? when copying the pointer?
		self.push(self.getStackTop())
	case compiler.Opcode_Spawn:
		i := instruction.(compiler.OneStringInstruction)

		// TODO: implement deepcopy for the arguments which are sent over to the new thread
		// Otherwise, when passing a list as an argument, we will get in trouble

		args := make([]value.Value, 0)
		numArgs := (*self.pop()).(value.ValueInt).Inner
		for i := 0; i < int(numArgs); i++ {
			args = append([]value.Value{*self.pop()}, args...) // TODO: implement deepcopy here
		}

		// TODO: how to handle the debugger
		self.parent.spawnCoreInternal(i.Value, args, nil)
		// TODO: implement a wrapper around the threading model and add it to a std-lib
		// TODO: get thread handle and push it onto the stack
		self.push(value.NewValueNull())
	case compiler.Opcode_Call_Val:
		numberArgsRaw := *self.pop()
		numArgs := numberArgsRaw.(value.ValueInt).Inner
		function := *self.pop()
		switch function.Kind() {
		case value.VmFunctionValueKind:
			function := function.(value.ValueVMFunction)

			self.callFrame().InstructionPointer++
			self.pushCallStack(function.Ident)

			return nil
		case value.BuiltinFunctionValueKind:
			fn := function.(value.ValueBuiltinFunction)

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

			// TODO: do we need to handle `nil` values?
			// TODO: this is bad: fix it
			if (*res).Kind() != value.NullValueKind {
				self.push(res)
			}
		default:
			panic(fmt.Sprintf("Values of kind %s cannot be called", function.Kind()))
		}
	case compiler.Opcode_Call_Imm:
		i := instruction.(compiler.OneStringInstruction)
		self.callFrame().InstructionPointer++
		self.pushCallStack(i.Value)
		return nil
	case compiler.Opcode_Return:
		self.popCallStack()
		// Need to return, otherwise, the callstack would have been popped, instantly skipping the next instruction
		return nil
	case compiler.Opcode_Load_Singleton:
		i := instruction.(compiler.TwoStringInstruction)

		singletonIdent := i.Values[0]
		moduleName := i.Values[1]

		// Load singleton from host
		singletonValue, found, err := self.parent.Executor.LoadSingleton(singletonIdent, moduleName)
		if err != nil {
			return value.NewVMFatalException(
				fmt.Sprintf(
					"Could not load singleton `%s` from module `%s`: %s",
					singletonIdent,
					moduleName,
					err.Error(),
				),
				value.Vm_HostErrorKind,
				self.parent.SourceMap(*self.callFrame()),
			)
		}

		if found {
			// Pop the default value off the stack and use loaded value instead.
			self.pop()
			self.push(&singletonValue)
		}
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
		return nil // Do not increment the new instruction
	case compiler.Opcode_JumpIfFalse:
		v := *self.pop()

		if !v.(value.ValueBool).Inner {
			i := instruction.(compiler.OneIntInstruction)
			self.callFrame().InstructionPointer = uint(i.Value)
			return nil // Do not increment the new instruction
		}
	case compiler.Opcode_GetVarImm:
		i := instruction.(compiler.OneIntInstruction)

		abs := self.absolute(i.Value)

		if VM_VERBOSE {
			fmt.Printf("Memory read access at %x\n", abs)
		}

		self.push(self.Memory[abs])
	case compiler.Opcode_GetGlobImm:
		i := instruction.(compiler.OneStringInstruction)
		self.parent.globals.Mutex.RLock()
		defer self.parent.globals.Mutex.RUnlock()

		v := self.parent.globals.Data[i.Value]
		self.push(&v)
	case compiler.Opcode_SetVarImm:
		i := instruction.(compiler.OneIntInstruction)
		v := self.pop()

		abs := self.absolute(i.Value)

		if VM_VERBOSE {
			fmt.Printf("Memory write access `%v` at %x\n", *v, abs)
		}

		self.Memory[abs] = v
	case compiler.Opcode_SetGlobImm:
		i := instruction.(compiler.OneStringInstruction)
		self.parent.globals.Mutex.Lock()
		defer self.parent.globals.Mutex.Unlock()

		v := self.pop()
		self.parent.globals.Data[i.Value] = *v
	case compiler.Opcode_Assign: // TODO: Assigns pointers on the stack???
		src := self.pop()
		dest := self.pop()

		// Perform actual assignment here
		*dest = *src
	case compiler.Opcode_Cast:
		i := instruction.(compiler.CastInstruction)
		v := self.pop()

		casted, interrupt := value.DeepCast(*v, i.Type, self.parent.SourceMap(*self.callFrame()), i.AllowCast)
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
	case compiler.Opcode_Some:
		v := *self.pop()
		self.push(value.NewValueOption(&v))
	case compiler.Opcode_Not:
		v := *self.pop()

		switch v.Kind() {
		case value.IntValueKind:
			intV := v.(value.ValueInt)
			self.push(value.NewValueInt(^intV.Inner))
		case value.BoolValueKind:
			boolV := v.(value.ValueBool)
			self.push(value.NewValueBool(!boolV.Inner))
		default:
			panic("Unsupported value kind: " + v.Kind().String())
		}
	case compiler.Opcode_Add:
		r := *self.pop()
		l := *self.pop()

		switch l.Kind() {
		case value.IntValueKind:
			lInt := l.(value.ValueInt)
			rInt := r.(value.ValueInt)
			self.push(value.NewValueInt(lInt.Inner + rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			self.push(value.NewValueFloat(lFloat.Inner + rFloat.Inner))
		case value.StringValueKind:
			lStr := l.(value.ValueString)
			rStr := r.(value.ValueString)
			self.push(value.NewValueString(lStr.Inner + rStr.Inner))
		default:
			panic(fmt.Sprintf("This value combination is unsupported: %v", l.Kind()))
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
			if rInt.Inner == 0 {
				return self.fatalErr(
					"Division by zero error: this is operation is illegal",
					value.Vm_ValueErrorKind,
					self.parent.SourceMap(*self.callFrame()),
				)
			}
			self.push(value.NewValueInt(lInt.Inner / rInt.Inner))
		case value.FloatValueKind:
			lFloat := l.(value.ValueFloat)
			rFloat := r.(value.ValueFloat)
			if rFloat.Inner == 0.0 {
				return self.fatalErr(
					"Division by zero error: this is operation is illegal",
					value.Vm_ValueErrorKind,
					self.parent.SourceMap(*self.callFrame()),
				)
			}
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
			panic(fmt.Sprintf("This value combination is unsupported: `%v` `%v`", l, r))
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

		return value.NewVMThrowInterrupt(
			self.parent.SourceMap(*self.callFrame()),
			display,
		)
	case compiler.Opcode_SetTryLabel:
		i := instruction.(compiler.OneIntOneStringInstruction)
		self.ExceptionCatchLabels = append(self.ExceptionCatchLabels, CallFrame{
			Function:           i.ValueString,
			InstructionPointer: uint(i.ValueInt),
		})
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
			disp, interrupt := v.Display()
			if interrupt != nil {
				panic(interrupt)
			}
			panic(fmt.Sprintf("Field `%s` not found on `%s`: %s:%d:%d", i.Value, disp, span.Filename, span.Start.Index, span.Start.Column))
		}
		self.push(field)
	case compiler.Opcode_Import:
		i := instruction.(compiler.TwoStringInstruction)
		self.importItem(i.Values[0], i.Values[1])
	case compiler.Opcode_Into_Range:
		// Used in order to determine whether the end is inclusive.
		i := instruction.(compiler.OneBoolInstruction)
		end := *self.pop()
		start := *self.pop()
		self.push(value.NewValueRange(start, end, i.ValueBool))
	case compiler.Opcode_IntoIter:
		v := *self.pop()
		self.push(value.NewValueIter(v))
	case compiler.Opcode_IteratorAdvance:
		// Get the iterator from the stack.
		iterator := (*self.pop()).(value.ValueIterator).Func
		val, shallContinue := iterator()
		self.push(value.NewValueBool(shallContinue))
		self.push(&val)
	default:
		panic(fmt.Sprintf("Illegal instruction error: %v", instruction))
	}

	self.callFrame().InstructionPointer++
	return nil
}
