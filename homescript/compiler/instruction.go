package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

type Opcode uint8

const (
	Opcode_Nop Opcode = iota
	Opcode_Push
	Opcode_Drop
	Opcode_Spawn
	Opcode_Call
	Opcode_Return
	Opcode_HostCall
	Opcode_Jump
	Opcode_JumpIfFalse
	Opcode_GetVarImm
	Opcode_GetGlobImm
	Opcode_SetVatImm
	Opcode_SetGlobImm
	Opcode_Cast
	Opcode_Neg
	Opcode_Not
	Opcode_Add
	Opcode_Sub
	Opcode_Mul
	Opcode_Pow
	Opcode_Div
	Opcode_Rem
	Opcode_Eq
	Opcode_Ne
	Opcode_Lt
	Opcode_Gt
	Opcode_Le
	Opcode_Ge
	Opcode_Shl
	Opcode_Shr
	Opcode_BitOr
	Opcode_BitAnd
	Opcode_BitXor
	Opcode_Index
	Opcode_SetTryLabel
	Opcode_Member
	Opcode_Import
	Opcode_Label
)

func (self Opcode) String() string {
	switch self {
	case Opcode_Nop:
		return "Nop"
	case Opcode_Push:
		return "Push"
	case Opcode_Drop:
		return "Drop"
	case Opcode_Spawn:
		return "Spawn"
	case Opcode_Call:
		return "Call"
	case Opcode_Return:
		return "Return"
	case Opcode_HostCall:
		return "HostCall"
	case Opcode_Jump:
		return "Jump"
	case Opcode_JumpIfFalse:
		return "JumpIfFalse"
	case Opcode_GetVarImm:
		return "GetVarImm"
	case Opcode_GetGlobImm:
		return "GetGlobImm"
	case Opcode_SetVatImm:
		return "SetVatImm"
	case Opcode_SetGlobImm:
		return "SetGlobImm"
	case Opcode_Cast:
		return "Cast"
	case Opcode_Neg:
		return "Neg"
	case Opcode_Not:
		return "Not"
	case Opcode_Add:
		return "Add"
	case Opcode_Sub:
		return "Sub"
	case Opcode_Mul:
		return "Mul"
	case Opcode_Pow:
		return "Pow"
	case Opcode_Div:
		return "Div"
	case Opcode_Rem:
		return "Rem"
	case Opcode_Eq:
		return "Eq"
	case Opcode_Ne:
		return "Ne"
	case Opcode_Lt:
		return "Lt"
	case Opcode_Gt:
		return "Gt"
	case Opcode_Le:
		return "Le"
	case Opcode_Ge:
		return "Ge"
	case Opcode_Shl:
		return "Shl"
	case Opcode_Shr:
		return "Shr"
	case Opcode_BitOr:
		return "BitOr"
	case Opcode_BitAnd:
		return "BitAnd"
	case Opcode_BitXor:
		return "BitXor"
	case Opcode_Index:
		return "Index"
	case Opcode_SetTryLabel:
		return "SetTryLabel"
	case Opcode_Member:
		return "Member"
	case Opcode_Import:
		return "Import"
	default:
		panic("Invalid instruction")
	}
}

type Instruction interface {
	Opcode() Opcode
	String() string
}

// Primitive Instruction

type PrimitiveInstruction struct {
	opCode Opcode
}

func (self PrimitiveInstruction) Opcode() Opcode { return self.opCode }
func (self PrimitiveInstruction) String() string { return self.opCode.String() }

func newPrimitiveInstruction(opCode Opcode) Instruction {
	return PrimitiveInstruction{opCode: opCode}
}

// OneString Instruction

type OneStringInstruction struct {
	opCode Opcode
	value  string
}

func (self OneStringInstruction) Opcode() Opcode { return self.opCode }
func (self OneStringInstruction) String() string {
	return fmt.Sprintf("%s(%s)", self.opCode, self.value)
}

func newOneStringInstruction(opCode Opcode, value string) OneStringInstruction {
	return OneStringInstruction{
		opCode: opCode,
		value:  value,
	}
}

// TwoString Instruction

type TwoStringInstruction struct {
	Operand string
}

// Cast Instruction

type CastInstruction struct {
	Type ast.Type
}

// Value Instruction

type ValueInstruction struct {
	Value Value
}

func (self ValueInstruction) Opcode() Opcode { return self.Opcode() }
func (self ValueInstruction) String() string { return fmt.Sprintf("%v(%v)", self.Opcode(), self.Value) }

func newValueInstruction(opCode Opcode, value Value) ValueInstruction {
	return ValueInstruction{
		Value: value,
	}
}
