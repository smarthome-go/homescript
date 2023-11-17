package main

import "github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"

type Opcode uint8

const (
	Nop Opcode = iota
	Push
	Drop
	Spawn
	Call
	Ret
	HostCall
	Jump
	JumpIfFalse
	GetVarImm
	GetGlobImm
	SetVatImm
	SetGlobImm
	Cast
	Neg
	Not
	Add
	Sub
	Mul
	Pow
	Div
	Rem
	Eq
	Ne
	Lt
	Gt
	Le
	Ge
	Shl
	Shr
	BitOr
	BitAnd
	BitXor
	Index
	SetTryLabel
	Member
	Import
)

type Instruction interface {
	Kind() Opcode
	String() string
}

type OneStringInstruction struct {
	Operand string
}

type TwoStringInstruction struct {
	Operand string
}

type CastInstruction struct {
	Type ast.Type
}
