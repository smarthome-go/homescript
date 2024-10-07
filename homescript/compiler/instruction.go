package compiler

import (
	"fmt"
	"slices"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

// This maps the source module's identifiers to their mangled versions.
type MangleMappings struct {
	Functions  map[string]string
	Globals    map[string]string
	Singletons map[string]string
}

type CompileOutput struct {
	Functions map[string][]Instruction
	// Associates a mangled function with its instruction-spans.
	SourceMap map[string][]errors.Span
	// NOTE: this type is returned by the compiler so that the execution environment
	// is still able to interact with the runtime through function calls and global variable access.
	Mappings    MangleMappings
	Annotations ModuleAnnotations
}

func (self CompileOutput) AsmStringHighlight(color bool, activeFunc *string, lineIdx *int) string {
	functionNames := make([]string, 0)
	for ident := range self.Functions {
		functionNames = append(functionNames, ident)
	}
	slices.Sort(functionNames)

	functionsStr := make([]string, 0)

	for _, fnName := range functionNames {
		instructions := self.Functions[fnName]

		if activeFunc != nil && *activeFunc != fnName {
			continue
		}

		fnHeaderStr := fmt.Sprintf("FUNCTION %s (%s)", fnName, self.Mappings.Functions[fnName])
		if color && activeFunc != nil && *activeFunc == fnName {
			fnHeaderStr = fmt.Sprintf("\x1b[1;32m%s\x1b[0m | ACTIVE", fnHeaderStr)
		}
		fnInstructions := make([]string, 0)

		for idx, instruction := range instructions {
			line := fmt.Sprintf("\t%04d | %s\n", idx, instruction.Display(color))

			if color {
				line = fmt.Sprintf("\x1b[1;90m%s\x1b[1;0m", line)
			}

			// Highlight active line
			if activeFunc != nil && *activeFunc == fnName {
				if lineIdx != nil {
					if idx == *lineIdx {
						line = fmt.Sprintf("\x1b[4m%s\x1b[0m", line)
					}
				}
			}

			fnInstructions = append(fnInstructions, line)
		}

		functionsStr = append(functionsStr, fmt.Sprintf("%s\n%s", fnHeaderStr, strings.Join(fnInstructions, "")))
	}

	return strings.Join(functionsStr, "END FUNCTION\n")
}

func (self CompileOutput) AsmString(color bool) string {
	// functionsStr := make([]string, 0)
	//
	// for fnName, instructions := range self.Functions {
	// 	fnHeaderStr := fmt.Sprintf("FUNCTION %s (%s)", fnName, self.Mappings.Functions[fnName])
	// 	fnInstructions := make([]string, 0)
	//
	// 	for idx, instruction := range instructions {
	// 		line := fmt.Sprintf("\t%04d | %s\n", idx, instruction.Display(color))
	//
	// 		if color {
	// 			line = fmt.Sprintf("\x1b[1;90m%s\x1b[1;0m", line)
	// 		}
	//
	// 		fnInstructions = append(fnInstructions, line)
	// 	}
	//
	// 	functionsStr = append(functionsStr, fmt.Sprintf("%s\n%s", fnHeaderStr, strings.Join(fnInstructions, "")))
	// }
	//
	// return strings.Join(functionsStr, "END FUNCTION\n")
	return self.AsmStringHighlight(color, nil, nil)
}

type Opcode uint8

const (
	Opcode_Nop Opcode = iota
	Opcode_Copy_Push
	Opcode_Clone
	Opcode_Cloning_Push
	Opcode_Drop
	Opcode_Spawn
	Opcode_Call_Val
	Opcode_Call_Imm
	Opcode_Return
	Opcode_Load_Singleton
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
	Opcode_Member_Anyobj
	Opcode_Member_Unwrap
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
	case Opcode_Clone:
		return "Clone"
	case Opcode_Copy_Push:
		return "CopyPush"
	case Opcode_Cloning_Push:
		return "CloningPush"
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
	case Opcode_Load_Singleton:
		return "LoadSingleton"
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
	case Opcode_Member_Anyobj:
		return "MemberAnyobj"
	case Opcode_Member_Unwrap:
		return "Unwrap"
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

const opcodeColor = "\x1b[1;39m"
const argumentColor = "\x1b[1;31m"
const colorReset = "\x1b[1;0m"

type Instruction interface {
	Opcode() Opcode
	String() string
	Display(color bool) string
}

// Primitive Instruction

type PrimitiveInstruction struct {
	opCode Opcode
}

func (self PrimitiveInstruction) Opcode() Opcode { return self.opCode }
func (self PrimitiveInstruction) String() string { return self.opCode.String() }
func (self PrimitiveInstruction) Display(color bool) string {
	if !color {
		return self.opCode.String()
	} else {
		return fmt.Sprintf("%s%s%s", opcodeColor, self.opCode.String(), colorReset)
	}
}

func newPrimitiveInstruction(opCode Opcode) Instruction {
	return PrimitiveInstruction{opCode: opCode}
}

// OneBool Instruction

type OneBoolInstruction struct {
	opCode    Opcode
	ValueBool bool
}

func (self OneBoolInstruction) Opcode() Opcode { return self.opCode }
func (self OneBoolInstruction) String() string {
	return fmt.Sprintf("%s(%v)", self.opCode, self.ValueBool)
}

func (self OneBoolInstruction) Display(color bool) string {
	if !color {
		return self.String()
	}
	return fmt.Sprintf("%s%s%s(%s%v%s)", opcodeColor, self.opCode, colorReset, argumentColor, self.ValueBool, colorReset)
}

func newOneBoolInstruction(opCode Opcode, valueBool bool) OneBoolInstruction {
	return OneBoolInstruction{
		opCode:    opCode,
		ValueBool: valueBool,
	}
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
func (self OneIntOneStringInstruction) Display(color bool) string {
	if !color {
		return self.String()
	}
	return fmt.Sprintf("%s%s%s(%s%s:%d%s)", opcodeColor, self.opCode, colorReset, argumentColor, self.ValueString, self.ValueInt, colorReset)
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
func (self OneIntInstruction) Display(color bool) string {
	return fmt.Sprintf("%s%s%s(%s%d%s)", opcodeColor, self.opCode, colorReset, argumentColor, self.Value, colorReset)
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
func (self OneStringInstruction) Display(color bool) string {
	if !color {
		return self.String()
	}
	return fmt.Sprintf("%s%s%s(%s%s%s)", opcodeColor, self.opCode, colorReset, argumentColor, self.Value, colorReset)
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
	return fmt.Sprintf("%s(%s, %s)", self.opCode, self.Values[0], self.Values[1])
}
func (self TwoStringInstruction) Display(color bool) string {
	if !color {
		return self.String()
	}
	return fmt.Sprintf(
		"%s%s%s(%s%s%s, %s%s%s)",
		opcodeColor,
		self.opCode,
		colorReset,
		argumentColor,
		self.Values[0],
		colorReset,
		argumentColor,
		self.Values[1],
		colorReset,
	)
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
	typeStr := strings.ReplaceAll(self.Type.String(), "\n", "\n        ")
	return fmt.Sprintf("%v(as_type=%s; perform_cast=%t)", self.Opcode(), typeStr, self.AllowCast)
}
func (self CastInstruction) Display(color bool) string {
	if !color {
		return self.String()
	}
	typeStr := strings.ReplaceAll(self.Type.String(), "\n", "\n        ")
	return fmt.Sprintf(
		"%s%v%s(as_type=%s%s%s; perform_cast=%s%t%s)",
		opcodeColor,
		self.Opcode(),
		colorReset,
		argumentColor,
		typeStr,
		colorReset,
		argumentColor,
		self.AllowCast,
		colorReset,
	)
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

func (self ValueInstruction) Display(color bool) string {
	if !color {
		return self.String()
	}

	str, i := self.Value.Display()
	if i != nil {
		panic(*i)
	}

	if self.Value.Kind() == value.StringValueKind {
		str = "\"" + str + "\""
	}

	str = strings.ReplaceAll(strings.ReplaceAll(str, "\n    ", ""), "\n", "")
	return fmt.Sprintf("%s%v%s(%s%s%s)", opcodeColor, self.Opcode(), colorReset, argumentColor, str, colorReset)
}

func newValueInstruction(opCode Opcode, value value.Value) ValueInstruction {
	return ValueInstruction{
		opCode: opCode,
		Value:  value,
	}
}
