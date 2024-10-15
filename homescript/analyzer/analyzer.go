package analyzer

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Analyzer
//

type Analyzer struct {
	analyzedModules                 map[string]ast.AnalyzedProgram
	scopeAdditions                  map[string]Variable
	diagnostics                     []diagnostic.Diagnostic
	syntaxErrors                    []errors.Error
	modules                         map[string]*Module
	currentModuleName               string
	currentModule                   *Module
	host                            HostProvider
	knownObjectTypeFieldAnnotations []string
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
		analyzedModules:                 make(map[string]ast.AnalyzedProgram),
		scopeAdditions:                  scopeAdditions,
		diagnostics:                     make([]diagnostic.Diagnostic, 0),
		syntaxErrors:                    make([]errors.Error, 0),
		modules:                         make(map[string]*Module),
		currentModuleName:               "",
		currentModule:                   nil,
		host:                            host,
		knownObjectTypeFieldAnnotations: []string{},
	}

	// Precompute this list as this could be expensive (depens on the host).
	knownAnnotationsWithoutPrefix := analyzer.host.GetKnownObjectTypeFieldAnnotations()
	analyzer.knownObjectTypeFieldAnnotations = make([]string, len(knownAnnotationsWithoutPrefix))

	for idx, annotation := range knownAnnotationsWithoutPrefix {
		analyzer.knownObjectTypeFieldAnnotations[idx] = fmt.Sprintf("%s%s", lexer.TYPE_ANNOTATION_TOKEN, annotation)
	}

	return analyzer
}

//
// Analyzer helper functions.
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

	// Check for unused type definitions.
	for key, typ := range scope.Types {
		if !typ.Used {
			self.warn(
				fmt.Sprintf("Type '%s' is unused", key),
				[]string{fmt.Sprintf("If this is intentional, change the name to '_%s' to hide this message", key)},
				typ.NameSpan,
			)
		}
	}

	// Check for unused variables.
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
				fmt.Sprintf("Import `%s` is unused", key),
				nil,
				variable.Span,
			)
		case BuiltinVariableOriginKind:
			// Ignore this variable.
		default:
			panic("A new variable origin kind was added without updating this code")
		}
	}
}

//
// Analyzer logic.
//

func (self *Analyzer) analyzeModule(moduleName string, module pAst.Program, mainShallExist bool) {
	currModulePrev := self.currentModule // Save previous module.
	currModuleNamePrev := self.currentModuleName

	// Create module.
	self.modules[moduleName] = &Module{
		ImportsModules:           make([]string, 0),
		Functions:                make([]*function, 0),
		Scopes:                   make([]scope, 0),
		Singletons:               make(map[string]*ast.AnalyzedSingleton),
		Templates:                make(map[string]ast.TemplateSpec),
		TriggerFunctions:         make(map[string]TriggerFunction),
		LoopDepth:                0, // `break` and `continue` are legal if > 0.
		CurrentFunction:          nil,
		CurrentLoopIsTerminated:  false,
		CreateErrorIfContainsAny: true,
	}

	// Set the current module to the new name.
	self.setCurrentModule(moduleName)

	output := ast.AnalyzedProgram{
		Imports:    make([]ast.AnalyzedImport, 0),
		Types:      make([]ast.AnalyzedTypeDefinition, 0),
		Globals:    make([]ast.AnalyzedLetStatement, 0),
		Functions:  make([]ast.AnalyzedFunctionDefinition, 0),
		Singletons: make([]ast.AnalyzedSingletonTypeDefinition, 0),
	}

	// Add the root scope.
	self.pushScope()

	// Populate root scope with scope additions.
	for name, val := range self.scopeAdditions {
		self.currentModule.addVar(name, val, false)
	}

	// Analyze all import statements.
	for _, item := range module.Imports {
		output.Imports = append(output.Imports, self.importItem(item))
	}

	// Analyze all type declarations.
	for _, item := range module.Types {
		output.Types = append(output.Types, self.typeDefStatement(item))
	}

	// Analyze all singleton declarations.
	for _, item := range module.Singletons {
		output.Singletons = append(output.Singletons, self.singletonDeclStatement(item))
	}

	// Add all impl block function signatures first.
	for _, impl := range module.ImplBlocks {
		for _, fn := range impl.Methods {
			// TODO: is this OK?
			self.functionSignature(fn)
		}
	}

	// Add all function signatures first (renders order of definition irrelevant).
	for _, fn := range module.Functions {
		self.functionSignature(fn)
	}

	// Analyze all global let statements.
	for _, item := range module.Globals {
		output.Globals = append(output.Globals, self.letStatement(item, true))
	}

	// Analyze all functions / events.
	for _, item := range module.Functions {
		fnOut := self.functionDefinition(item)

		switch item.Modifier {
		case pAst.FN_MODIFIER_EVENT:
			// TODO: merge this, perform any extra actions here?
			output.Functions = append(output.Functions, fnOut)
		default:
			output.Functions = append(output.Functions, fnOut)
		}
	}

	// Analyze all impl blocks.
	for _, impl := range module.ImplBlocks {
		output.ImplBlocks = append(output.ImplBlocks, self.implBlock(impl))
	}

	for singletonName, singleton := range self.currentModule.Singletons {
		for idx, otherSingleton := range output.Singletons {
			// Match found, copy implemented templates.
			if singletonName == otherSingleton.Ident.Ident() {
				output.Singletons[idx].ImplementsTemplates = append(
					output.Singletons[idx].ImplementsTemplates,
					singleton.ImplementsTemplates...,
				)
				output.Singletons[idx].Used = singleton.Used
				break
			}
		}
	}

	// Check if the `main` function exists.
	mainExists := false

	for _, fn := range output.Functions {
		if fn.Ident.Ident() == "main" {
			mainExists = true
			break
		}
	}

	if !mainExists && mainShallExist {
		self.error(
			"Missing 'main' function",
			[]string{"the 'main' function can be implemented like this: `fn main() { ... }`"},
			errors.Span{Filename: module.Filename},
		)
	}

	// Check that all functions are used and ignore them if they are `pub`.
	// NOTE: this only works if `NONE` is the only default modifier.
	for _, fn := range self.currentModule.Functions {
		if fn.Used || fn.Modifier != pAst.FN_MODIFIER_NONE || fn.FnType.Kind() != normalFunctionKind {
			continue
		}
		fnType := fn.FnType.(normalFunction)
		if fnType.Ident.Ident() == "main" || strings.HasPrefix(fnType.Ident.Ident(), "_") { // Ignore the `main` fn.
			continue
		}
		self.warn(
			fmt.Sprintf("Function '%s' is never used", fnType.Ident.Ident()),
			[]string{fmt.Sprintf(
				"If this is intentional, change the name to '_%s' to hide this message",
				fnType.Ident.Ident(),
			)},
			fnType.Ident.Span(),
		)
	}

	// Detect any unused singletons.
	for singletonName, singleton := range self.currentModule.Singletons {
		if !singleton.Used {
			self.warn(
				fmt.Sprintf("Singleton '%s' is never used", singletonName),
				[]string{fmt.Sprintf(
					"If this is intentional, change the name to '_%s' to hide this message",
					singletonName,
				)},
				singleton.Type.Span(),
			)
		}
	}

	// Drop the root scope so that all unused globals are displayed.
	// Do not remove the scope.
	self.dropScope(false)

	if currModulePrev != nil {
		self.currentModule = currModulePrev
		self.currentModuleName = currModuleNamePrev
	}

	self.analyzedModules[moduleName] = output
}

func (self *Analyzer) Analyze(
	parsedEntryModule pAst.Program,
	mainShallExist bool,
) (
	modules map[string]ast.AnalyzedProgram,
	diagnostics []diagnostic.Diagnostic,
	syntaxErrors []errors.Error,
) {
	self.analyzeModule(parsedEntryModule.Filename, parsedEntryModule, mainShallExist)

	// If there are no serious errors found, call the post-validation hook.
	containsErrs := false
	for _, d := range self.diagnostics {
		if d.Level == diagnostic.DiagnosticLevelError {
			containsErrs = true
		}
	}

	diagnosticsPostValidation := self.host.PostValidationHook(
		self.analyzedModules,
		parsedEntryModule.Filename,
		self,
		containsErrs,
	)
	self.diagnostics = append(self.diagnostics, diagnosticsPostValidation...)

	return self.analyzedModules, self.diagnostics, self.syntaxErrors
}
