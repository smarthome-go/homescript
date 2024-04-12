package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

func (self *Compiler) insert(instruction Instruction, span errors.Span) int {
	self.CurrFn().Instructions = append(self.CurrFn().Instructions, instruction)
	self.CurrFn().SourceMap = append(self.CurrFn().SourceMap, span)
	return len(self.CurrFn().Instructions) - 1
}

func (self *Compiler) relocateLabels() {
	for moduleName, module := range self.modules {
		for name, fn := range module {
			labels := make(map[string]int64)

			fnOut := make([]Instruction, 0)
			sourceMapOut := make([]errors.Span, 0)

			index := 0
			for idx, inst := range fn.Instructions {
				if inst.Opcode() == Opcode_Label {
					i := inst.(OneStringInstruction).Value
					labels[i] = int64(index)
				} else {
					fnOut = append(fnOut, inst)
					sourceMapOut = append(sourceMapOut, fn.SourceMap[idx])
					index++
				}
			}

			for idx, inst := range fnOut {
				switch inst.Opcode() {
				case Opcode_Jump, Opcode_JumpIfFalse:
					i := inst.(OneStringInstruction)

					ip, found := labels[i.Value]
					if !found {
						panic(fmt.Sprintf("Every label needs to appear in the code: %s", i.Value))
					}

					fnOut[idx] = newOneIntInstruction(inst.Opcode(), ip)
				case Opcode_SetTryLabel:
					i := inst.(TwoStringInstruction)

					ip, found := labels[i.Values[1]]
					if !found {
						panic("Every label needs to appear in the code")
					}

					fnOut[idx] = newOneIntOneStringInstruction(inst.Opcode(), i.Values[0], ip)
				case Opcode_Label:
					panic("This should not happen")
				}
			}

			self.modules[moduleName][name].Instructions = fnOut
			self.modules[moduleName][name].SourceMap = sourceMapOut
		}
	}
}

func (self *Compiler) renameVariables() {
	slot := make(map[string]int64, 0)

	for _, module := range self.modules {
		for name, fn := range module {
			cnt := 0
			for idx, inst := range fn.Instructions {
				switch inst.Opcode() {
				case Opcode_GetVarImm:
					i := inst.(OneStringInstruction)
					if _, found := slot[i.Value]; !found {
						slot[i.Value] = int64(cnt)
						cnt++
					}
					module[name].Instructions[idx] = newOneIntInstruction(Opcode_GetVarImm, slot[i.Value])
				case Opcode_SetVarImm:
					i := inst.(OneStringInstruction)
					if _, found := slot[i.Value]; !found {
						slot[i.Value] = int64(cnt)
						cnt++
					}
					module[name].Instructions[idx] = newOneIntInstruction(Opcode_SetVarImm, slot[i.Value])
				default:
					continue
				}
			}
		}
	}
}
