package compiler

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

type Program struct {
	Functions   map[string][]Instruction
	SourceMap   map[string][]errors.Span
	EntryPoints map[string]string
}

type Opcode uint8

const (
	Opcode_Nop Opcode = iota
	Opcode_Push
	Opcode_Drop
	Opcode_Spawn
	Opcode_Call_Val
	Opcode_Call_Imm
	Opcode_Return
	Opcode_HostCall
	Opcode_Jump
	Opcode_JumpIfFalse
	Opcode_GetVarImm
	Opcode_GetGlobImm
	Opcode_SetVarImm
	Opcode_SetGlobImm
	Opcode_Assign // assigns pointers on the stack???
	Opcode_Cast
	Opcode_Neg
	Opcode_Some // ?foo -> converts foo to a Option<foo>
	Opcode_Not
	Opcode_Add
	Opcode_Sub
	Opcode_Mul
	Opcode_Pow
	Opcode_Div
	Opcode_Rem
	Opcode_Eq
	Opcode_Eq_PopOnce // Only pops the stack once, the other value is left untouched
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
	Opcode_PopTryLabel
	Opcode_Throw
	Opcode_Member
	Opcode_Import
	Opcode_Label
	Opcode_Into_Range
	Opcode_Duplicate
	Opcode_AddMempointer
	Opcode_IteratorAdvance
	Opcode_IntoIter
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
	case Opcode_Call_Imm:
		return "Call_Imm"
	case Opcode_Call_Val:
		return "Call_Val"
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
	case Opcode_SetVarImm:
		return "SetVarImm"
	case Opcode_SetGlobImm:
		return "SetGlobImm"
	case Opcode_Assign:
		return "Assign"
	case Opcode_Cast:
		return "Cast"
	case Opcode_Neg:
		return "Neg"
	case Opcode_Some:
		return "Some"
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
	case Opcode_Eq_PopOnce:
		return "Eq_PopOnce"
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
	case Opcode_PopTryLabel:
		return "PopTryLabel"
	case Opcode_Throw:
		return "Throw"
	case Opcode_Member:
		return "Member"
	case Opcode_Import:
		return "Import"
	case Opcode_Label:
		return "Label"
	case Opcode_Into_Range:
		return "Into_Range"
	case Opcode_Duplicate:
		return "Duplicate"
	case Opcode_AddMempointer:
		return "AddMempointer"
	case Opcode_IteratorAdvance:
		return "IterAdvance"
	case Opcode_IntoIter:
		return "IntoIter"
	default:
		panic(fmt.Sprintf("Invalid instruction: %d", self))
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

// OneInt OneString Instruction

type OneIntOneStringInstruction struct {
	opCode      Opcode
	ValueInt    int64
	ValueString string
}

func (self OneIntOneStringInstruction) Opcode() Opcode { return self.opCode }
func (self OneIntOneStringInstruction) String() string {
	return fmt.Sprintf("%s(%s:%d)", self.opCode, self.ValueString, self.ValueInt)
}

func newOneIntOneStringInstruction(opCode Opcode, valueString string, valueInt int64) OneIntOneStringInstruction {
	return OneIntOneStringInstruction{
		opCode:      opCode,
		ValueInt:    valueInt,
		ValueString: valueString,
	}
}

// OneInt Instruction

type OneIntInstruction struct {
	opCode Opcode
	Value  int64
}

func (self OneIntInstruction) Opcode() Opcode { return self.opCode }
func (self OneIntInstruction) String() string {
	return fmt.Sprintf("%s(%d)", self.opCode, self.Value)
}

func newOneIntInstruction(opCode Opcode, value int64) OneIntInstruction {
	return OneIntInstruction{
		opCode: opCode,
		Value:  value,
	}
}

// OneString Instruction

type OneStringInstruction struct {
	opCode Opcode
	Value  string
}

func (self OneStringInstruction) Opcode() Opcode { return self.opCode }
func (self OneStringInstruction) String() string {
	return fmt.Sprintf("%s(%s)", self.opCode, self.Value)
}

func newOneStringInstruction(opCode Opcode, value string) OneStringInstruction {
	return OneStringInstruction{
		opCode: opCode,
		Value:  value,
	}
}

// TwoString Instruction

type TwoStringInstruction struct {
	opCode Opcode
	Values [2]string
}

func (self TwoStringInstruction) Opcode() Opcode { return self.opCode }
func (self TwoStringInstruction) String() string {
	return fmt.Sprintf("%s(%s, %s)", self.opCode, self.Values[0], self.Values[0])
}

func newTwoStringInstruction(opCode Opcode, value0 string, value1 string) TwoStringInstruction {
	return TwoStringInstruction{
		opCode: opCode,
		Values: [2]string{value0, value1},
	}
}

// Cast Instruction

type CastInstruction struct {
	opCode    Opcode
	Type      ast.Type
	AllowCast bool
}

func (self CastInstruction) Opcode() Opcode { return self.opCode }
func (self CastInstruction) String() string {
	return fmt.Sprintf("%v(%v; %t)", self.Opcode(), self.Type, self.AllowCast)
}

func newCastInstruction(type_ ast.Type, allowCast bool) CastInstruction {
	return CastInstruction{
		opCode:    Opcode_Cast,
		Type:      type_,
		AllowCast: allowCast,
	}
}

// Value Instruction

type ValueInstruction struct {
	opCode Opcode
	Value  value.Value
}

func (self ValueInstruction) Opcode() Opcode { return self.opCode }
func (self ValueInstruction) String() string {
	str, i := self.Value.Display()
	if i != nil {
		panic(*i)
	}

	if self.Value.Kind() == value.StringValueKind {
		str = "\"" + str + "\""
	}

	str = strings.ReplaceAll(strings.ReplaceAll(str, "\n    ", ""), "\n", "")
	return fmt.Sprintf("%v(%s)", self.Opcode(), str)
}

func newValueInstruction(opCode Opcode, value value.Value) ValueInstruction {
	return ValueInstruction{
		opCode: opCode,
		Value:  value,
	}
}
