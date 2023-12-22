package interpreter

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

func (self *Interpreter) instantiateSingleton(node ast.AnalyzedSingletonTypeDefinition) *value.Interrupt {
	singletonValue, usethisValue, err := self.Executor.LoadSingleton(node.Ident.Ident(), node.TypeDef.RhsType)
	if err != nil {
		return err
	}

	// If there is no value to be provided by the host, create a default value.
	if !usethisValue {
		singletonValue = value.CreateDefault(node.TypeDef.RhsType)
	}

	// Use a global variable internally
	self.addVar(node.Ident.Ident(), *singletonValue)
	return nil
}

func (self *Interpreter) implBlock(node ast.AnalyzedImplBlock) {
	for _, fn := range node.Methods {
		self.functionDefinition(fn)
	}
}

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
		val, found := self.Executor.GetBuiltinImport(node.FromModule.Ident(), toImport.Ident.Ident())
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
	extractions := make([]value.SingletonExtraction, 0)
	for _, param := range node.Parameters {
		if !param.IsSingletonExtractor {
			break
		}

		extractions = append(extractions, value.SingletonExtraction{
			ParameterIdent: param.Ident.Ident(),
			SingletonIdent: param.SingletonIdent,
		})
	}

	self.addVar(node.Ident.Ident(), *value.NewValueFunction(
		self.currentModuleName,
		node.Body,
		extractions,
	))
}

func (self *Interpreter) eventFunctionDefinition(node ast.AnalyzedFunctionDefinition) {
	self.addVar(fmt.Sprintf("@event_%s", node.Ident.Ident()), *value.NewValueFunction(
		self.currentModuleName,
		node.Body,
		make([]value.SingletonExtraction, 0),
	))
}
