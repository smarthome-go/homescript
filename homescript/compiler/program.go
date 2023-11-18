package compiler

type Function []Instruction

type Program struct {
	Functions map[string]Function
}
