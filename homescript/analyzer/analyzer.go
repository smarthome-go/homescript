package analyzer

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Analyzer
//

type Analyzer struct {
	analyzedModules   map[string]ast.AnalyzedProgram
	scopeAdditions    map[string]Variable
	diagnostics       []diagnostic.Diagnostic
	syntaxErrors      []errors.Error
	modules           map[string]*Module
	currentModuleName string
	currentModule     *Module
	host              HostProvider
}

func NewAnalyzer(host HostProvider, scopeAdditions map[string]Variable) Analyzer {
	scopeAdditions["throw"] = NewBuiltinVar(
		ast.NewFunctionType(
			ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
				ast.NewFunctionTypeParam(pAst.NewSpannedIdent("error", errors.Span{}), ast.NewUnknownType(), nil),
			}),
			errors.Span{},
			ast.NewNeverType(),
			errors.Span{},
		),
	)

	analyzer := Analyzer{
		analyzedModules:   make(map[string]ast.AnalyzedProgram),
		scopeAdditions:    scopeAdditions,
		diagnostics:       make([]diagnostic.Diagnostic, 0),
		syntaxErrors:      make([]errors.Error, 0),
		modules:           make(map[string]*Module),
		currentModuleName: "",
		currentModule:     nil,
		host:              host,
	}
	return analyzer
}

//
// Analyzer helper functions
//

func (self *Analyzer) error(message string, notes []string, span errors.Span) {
	self.diagnostics = append(self.diagnostics, diagnostic.Diagnostic{
		Level:   diagnostic.DiagnosticLevelError,
		Message: message,
		Notes:   notes,
		Span:    span,
	})
}

func (self *Analyzer) warn(message string, notes []string, span errors.Span) {
	self.diagnostics = append(self.diagnostics, diagnostic.Diagnostic{
		Level:   diagnostic.DiagnosticLevelWarning,
		Message: message,
		Notes:   notes,
		Span:    span,
	})
}

func (self *Analyzer) hint(message string, notes []string, span errors.Span) {
	self.diagnostics = append(self.diagnostics, diagnostic.Diagnostic{
		Level:   diagnostic.DiagnosticLevelHint,
		Message: message,
		Notes:   notes,
		Span:    span,
	})
}

func (self *Analyzer) dropScope(remove bool) {
	var scope scope
	if remove {
		scope = self.currentModule.popScope()
	} else {
		scope = self.currentModule.Scopes[len(self.currentModule.Scopes)-1]
	}

	// check for unused type definitions
	for key, typ := range scope.Types {
		if !typ.Used {
			self.warn(
				fmt.Sprintf("Type '%s' is unused", key),
				[]string{fmt.Sprintf("If this is intentional, change the name to '_%s' to hide this message", key)},
				typ.NameSpan,
			)
		}
	}

	// check for unused variables
	for key, variable := range scope.Values {
		if variable.Used || variable.IsPub || strings.HasPrefix(key, "_") {
			continue
		}
		switch variable.Origin {
		case NormalVariableOriginKind, ParameterVariableOriginKind:
			label := "Variable"
			if variable.Origin == ParameterVariableOriginKind {
				label = "Parameter"
			}

			self.warn(
				fmt.Sprintf("%s '%s' is unused", label, key),
				[]string{fmt.Sprintf("If this is intentional, change the name to '_%s' to hide this message", key)},
				variable.Span,
			)
		case ImportedVariableOriginKind:
			self.warn(
				fmt.Sprintf("Import '%s' is unused", key),
				nil,
				variable.Span,
			)
		case BuiltinVariableOriginKind:
			// ignore this variable
		default:
			panic("A new variable origin kind was added without updating this code")
		}
	}
}

//
// Analyzer logic
//

func (self *Analyzer) analyzeModule(moduleName string, module pAst.Program) {
	currModulePrev := self.currentModule // save previous module
	currModuleNamePrev := self.currentModuleName

	// create module
	self.modules[moduleName] = &Module{
		ImportsModules:           make([]string, 0),
		Functions:                make([]*function, 0),
		Scopes:                   make([]scope, 0),
		Singletons:               make(map[string]ast.AnalyzedSingleton),
		Templates:                make(map[string]ast.TemplateSpec),
		LoopDepth:                0, // `break` and `continue` are legal if > 0
		CurrentFunction:          nil,
		CurrentLoopIsTerminated:  false,
		CreateErrorIfContainsAny: true,
	}

	// set the current module to the new name
	self.setCurrentModule(moduleName)

	output := ast.AnalyzedProgram{
		Imports:   make([]ast.AnalyzedImport, 0),
		Types:     make([]ast.AnalyzedTypeDefinition, 0),
		Globals:   make([]ast.AnalyzedLetStatement, 0),
		Functions: make([]ast.AnalyzedFunctionDefinition, 0),
	}

	// add the root scope
	self.pushScope()

	// populate root scope with scope additions
	for name, val := range self.scopeAdditions {
		self.currentModule.addVar(name, val, false)
	}

	// analyze all import statements
	for _, item := range module.Imports {
		output.Imports = append(output.Imports, self.importItem(item))
	}

	// analyze all type declarations
	for _, item := range module.Types {
		output.Types = append(output.Types, self.typeDefStatement(item))
	}

	// analyze all singleton declarations
	for _, item := range module.Singletons {
		output.Singletons = append(output.Singletons, self.singletonDeclStatement(item))
	}

	// add all impl block function signatures first
	for _, impl := range module.ImplBlocks {
		for _, fn := range impl.Methods {
			// TODO: is this OK?
			self.functionSignature(fn)
		}
	}

	// add all function signatures first (renders order of definition irrelevant)
	for _, fn := range module.Functions {
		self.functionSignature(fn)
	}

	// analyze all global let statements
	for _, item := range module.Globals {
		output.Globals = append(output.Globals, self.letStatement(item, true))
	}

	// analyze all functions / events
	for _, item := range module.Functions {
		fnOut := self.functionDefinition(item)

		switch item.Modifier {
		case pAst.FN_MODIFIER_EVENT:
			output.Events = append(output.Events, fnOut)
		default:
			output.Functions = append(output.Functions, fnOut)
		}
	}

	// analyze all impl blocks
	for _, impl := range module.ImplBlocks {
		output.ImplBlocks = append(output.ImplBlocks, self.implBlock(impl))
	}

	// check if the `main` function exists
	mainExists := false
	for _, fn := range output.Functions {
		if fn.Ident.Ident() == "main" {
			mainExists = true
			break
		}
	}
	if !mainExists {
		self.error(
			"Missing 'main' function",
			[]string{"the 'main' function can be implemented like this: `fn main() { ... }`"},
			errors.Span{Filename: module.Filename},
		)
	}

	// check that all functions are used and ignore them if they are `pub`
	// NOTE: this only works if `NONE` is the only default modifier
	for _, fn := range self.currentModule.Functions {
		if fn.Used || fn.Modifier != pAst.FN_MODIFIER_NONE || fn.FnType.Kind() != normalFunctionKind {
			continue
		}
		fnType := fn.FnType.(normalFunction)
		if fnType.Ident.Ident() == "main" || strings.HasPrefix(fnType.Ident.Ident(), "_") { // ignore the `main` fn
			continue
		}
		self.warn(
			fmt.Sprintf("Function '%s' is never used", fnType.Ident.Ident()),
			[]string{fmt.Sprintf("If this is intentional, change the name to '_%s' to hide this message", fnType.Ident.Ident())},
			fnType.Ident.Span(),
		)
	}

	// drop the root scope so that all unused globals are displayed
	// do not remove the scope
	self.dropScope(false)

	if currModulePrev != nil {
		self.currentModule = currModulePrev
		self.currentModuleName = currModuleNamePrev
	}

	self.analyzedModules[moduleName] = output
}

func (self *Analyzer) Analyze(
	parsedEntryModule pAst.Program,
) (
	modules map[string]ast.AnalyzedProgram,
	diagnostics []diagnostic.Diagnostic,
	syntaxErrors []errors.Error,
) {
	self.analyzeModule(parsedEntryModule.Filename, parsedEntryModule)
	return self.analyzedModules, self.diagnostics, self.syntaxErrors
}
