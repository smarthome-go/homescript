package interpreter

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

func (self *Interpreter) addVar(ident string, val value.Value) {
	self.currentModule.scopes[len(self.currentModule.scopes)-1][ident] = &val
}

func (self *Interpreter) getVar(ident string) *value.Value {

	for i := len(self.currentModule.scopes) - 1; i >= 0; i-- {
		val, found := self.currentModule.scopes[i][ident]
		if found {
			return val
		}
	}

	panic(fmt.Sprintf("CurrModuleName: %s | ModuleName: %v | Variable '%s' not found", self.currentModuleName, self.currentModule, ident))
}

func (self *Interpreter) pushScope() {
	self.currentModule.scopes = append(self.currentModule.scopes, make(map[string]*value.Value))
}

func (self *Interpreter) popScope() map[string]*value.Value {
	last := self.currentModule.scopes[len(self.currentModule.scopes)-1]
	self.currentModule.scopes = self.currentModule.scopes[:len(self.currentModule.scopes)-1]
	return last
}

func (self Interpreter) clearScope() {
	self.currentModule.scopes[len(self.currentModule.scopes)-1] = make(map[string]*value.Value)
}

func (self *Interpreter) switchModule(moduleName string) {
	self.currentModule = self.modules[moduleName]
	self.currentModuleName = moduleName
}

func (self *Interpreter) execModule(moduleName string, restorePrev bool) *value.Interrupt {
	// save previous current module
	prevCurrName := self.currentModuleName

	// initialize module
	self.modules[moduleName] = &Module{
		scopes: make([]map[string]*value.Value, 0),
	}
	sourceModule := self.sourceModules[moduleName]
	self.switchModule(moduleName)

	// add the root scope
	self.pushScope()

	// add all scope additions to the root scope
	for key, val := range self.scopeAdditions {
		self.addVar(key, val)
	}

	// Instantiate all singletons
	for _, singleton := range sourceModule.Singletons {
		if err := self.instantiateSingleton(singleton); err != nil {
			return err
		}
	}

	// bring all imports into the new scope
	for _, item := range sourceModule.Imports {
		if err := self.importItem(item); err != nil {
			return err
		}
	}

	// visit all global let statements
	for _, item := range sourceModule.Globals {
		if i := self.letStatement(item); i != nil {
			return i
		}
	}

	// visit all function definitions
	for _, fn := range sourceModule.Functions {
		self.functionDefinition(fn)
	}

	// visit all event definitions
	for _, event := range sourceModule.Events {
		self.eventFunctionDefinition(event)
	}

	// restore previous current module
	if restorePrev {
		self.switchModule(prevCurrName)
	}

	return nil
}
