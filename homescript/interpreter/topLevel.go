package interpreter

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

func (self *Interpreter) importItem(node ast.AnalyzedImport) *value.Interrupt {
	_, moduleFound := self.sourceModules[node.FromModule.Ident()]

	if moduleFound {
		// visit the module so that the root scope is populated
		if i := self.execModule(node.FromModule.Ident(), true); i != nil {
			return i
		}

		for _, importItem := range node.ToImport {
			val := self.modules[node.FromModule.Ident()].scopes[0][importItem.Ident.Ident()]
			self.addVar(importItem.Ident.Ident(), *val)
		}

		return nil
	}

	// since the module was not found, source the imports from the builtin modules
	for _, toImport := range node.ToImport {
		val, found := self.executor.GetBuiltinImport(node.FromModule.Ident(), toImport.Ident.Ident())
		if !found {
			return value.NewRuntimeErr(
				fmt.Sprintf("Unknown import '%s' in module '%s'", toImport, node.FromModule),
				value.ImportErrorKind,
				toImport.Ident.Span(),
			)
		}
		// add the imported value to the current scope
		self.addVar(toImport.Ident.Ident(), val)
	}

	return nil
}

func (self *Interpreter) functionDefinition(node ast.AnalyzedFunctionDefinition) {
	self.addVar(node.Ident.Ident(), *value.NewValueFunction(
		self.currentModuleName,
		node.Body,
	))
}
