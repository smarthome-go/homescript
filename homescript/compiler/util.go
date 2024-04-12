package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	evalValue "github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

func (self Compiler) CurrFn() *Function { return self.modules[self.currModule][self.currFn] }

func (self Compiler) currLoop() Loop { return self.loops[len(self.loops)-1] }
func (self *Compiler) pushLoop(l Loop) {
	self.loops = append(self.loops, l)
}
func (self *Compiler) popLoop() {
	self.loops = self.loops[:len(self.loops)-1]
}

func (self *Compiler) pushScope() {
	self.varScopes = append(self.varScopes, make(map[string]string))
	self.currScope = &self.varScopes[len(self.varScopes)-1]
}

func (self *Compiler) popScope() {
	self.varScopes = self.varScopes[:len(self.varScopes)-1]
	if len(self.varScopes) == 0 {
		self.currScope = nil
		return
	}
	self.currScope = &self.varScopes[len(self.varScopes)-1]
}

func (self *Compiler) mangleFn(input string) string {
	mangled := fmt.Sprintf("@%s_%s", self.currModule, input)
	return mangled
}

func (self *Compiler) addFn(srcIdent string, mangledName string) {
	fn := Function{
		MangledName:  mangledName,
		Instructions: make([]Instruction, 0),
		SourceMap:    make([]errors.Span, 0),
		CntVariables: 0,
		CleanupLabel: "",
	}
	self.modules[self.currModule][srcIdent] = &fn
}

func (self *Compiler) mangleVar(input string) string {
	self.CurrFn().CntVariables++
	cnt, exists := self.varNameMangle[input]
	if !exists {
		// The next time this variable is mangled, 0 MUST NOT be used as the counter.
		self.varNameMangle[input] = 1
		cnt = 0
	} else {
		self.varNameMangle[input]++
	}

	mangled := fmt.Sprintf("@%s_%s%d", self.currModule, input, cnt)
	(*self.currScope)[input] = mangled

	return mangled
}

func (self *Compiler) mangleLabel(input string) string {
	cnt, exists := self.labelNameMangle[input]
	if !exists {
		self.labelNameMangle[input]++
		cnt = 0
	} else {
		self.labelNameMangle[input]++
	}

	mangled := fmt.Sprintf("%s_%s%d", self.currModule, input, cnt)
	return mangled
}

func (self Compiler) getMangledFn(input string) (string, bool) {
	for key, fn := range self.modules[self.currModule] {
		if key == input {
			return fn.MangledName, true
		}
	}

	// TODO: i don't think that this is really reliable
	for _, module := range self.modules {
		for key, fn := range module {
			if key == input {
				return fn.MangledName, true
			}
		}
	}

	return "", false
}

func (self Compiler) getMangled(input string) (string, bool) {
	// Iterate over the scopes backwards: Most current scope should be considered first.
	for i := len(self.varScopes) - 1; i >= 0; i-- {
		scope := self.varScopes[i]
		name, found := scope[input]
		if found {
			return name, true
		}
	}

	return "", false
}

//
// This function is applied to convert eval values to runtime values that are understood by the compiler.
//

func upgradeValue(from *evalValue.Value) *value.Value {
	switch (*from).Kind() {
	case evalValue.NullValueKind:
		return value.NewValueNull()
	case evalValue.IntValueKind:
		return value.NewValueInt((*from).(evalValue.ValueInt).Inner)
	case evalValue.FloatValueKind:
		return value.NewValueFloat((*from).(evalValue.ValueFloat).Inner)
	case evalValue.BoolValueKind:
		return value.NewValueBool((*from).(evalValue.ValueBool).Inner)
	case evalValue.StringValueKind:
		return value.NewValueString((*from).(evalValue.ValueString).Inner)
	case evalValue.AnyObjectValueKind:
		any := (*from).(evalValue.ValueAnyObject).FieldsInternal

		newFields := make(map[string]*value.Value)

		for key, fromV := range any {
			newFields[key] = upgradeValue(fromV)
		}

		return value.NewValueAnyObject(newFields)
	case evalValue.ObjectValueKind:
		obj := (*from).(evalValue.ValueAnyObject).FieldsInternal

		newFields := make(map[string]*value.Value)

		for key, fromV := range obj {
			newFields[key] = upgradeValue(fromV)
		}

		return value.NewValueObject(newFields)
	case evalValue.OptionValueKind:
		opt := (*from).(evalValue.ValueOption)
		if opt.Inner == nil {
			return value.NewNoneOption()
		}

		return value.NewValueOption(upgradeValue(opt.Inner))
	case evalValue.ListValueKind:
		list := *(*from).(evalValue.ValueList).Values

		newList := make([]*value.Value, len(list))

		for idx, fromV := range list {
			newList[idx] = upgradeValue(fromV)
		}

		return value.NewValueList(newList)
	case evalValue.RangeValueKind:
		rng := (*from).(evalValue.ValueRange)

		return value.NewValueRange(
			*upgradeValue(rng.Start),
			*upgradeValue(rng.End),
			rng.EndIsInclusive,
		)
	case evalValue.FunctionValueKind, evalValue.ClosureValueKind, evalValue.VmFunctionValueKind,
		evalValue.BuiltinFunctionValueKind, evalValue.PointerValueKind, evalValue.IteratorValueKind:
		panic("Cannot upgrade this value")
	default:
		panic("A new value kind was added without updating this code")
	}
}
