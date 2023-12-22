package analyzer

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser"
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
	host              HostDependencies
}

func NewAnalyzer(host HostDependencies, scopeAdditions map[string]Variable) Analyzer {
	scopeAdditions["throw"] = NewBuiltinVar(
		ast.NewFunctionType(
			ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
				ast.NewFunctionTypeParam(pAst.NewSpannedIdent("error", errors.Span{}), ast.NewUnknownType(), false),
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
		Singletons:               make(map[string]ast.Type),
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

	// add all function signatures first (renders order of definition irrelevant)
	for _, fn := range module.Functions {
		newParams := make([]ast.AnalyzedFnParam, 0)
		for _, param := range fn.Parameters {
			newParams = append(newParams, ast.AnalyzedFnParam{
				Ident:                param.Ident,
				Type:                 self.ConvertType(param.Type, false), // errors are only reported in the `self.functionDefinition` method
				Span:                 param.Span,
				IsSingletonExtractor: param.Type.Kind() == pAst.SingletonReferenceParserTypeKind,
			})
		}

		// add function to current module
		if prev, exists := self.currentModule.getFunc(fn.Ident.Ident()); exists {
			// check if the identifier conflicts with another function
			self.error(
				fmt.Sprintf("Duplicate function definition of '%s'", fn.Ident.Ident()),
				[]string{"Consider changing the name of this function"},
				fn.Ident.Span(),
			)
			self.hint(
				fmt.Sprintf("Function '%s' previously defined here", fn.Ident.Ident()),
				nil,
				(*prev).FnType.(normalFunction).Ident.Span(),
			)
		}

		self.currentModule.addFunc(newFunction(
			newNormalFunction(fn.Ident),
			newParams,
			fn.ParamSpan,
			self.ConvertType(fn.ReturnType, false), // errors are only reported in the `self.functionDefinition` method
			fn.ReturnType.Span(),                   // is the return span really required?
			fn.Modifier,
		))
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

//
// Import statement
//

func (self *Analyzer) importDummyFields(node pAst.ImportStatement) ast.AnalyzedImport {
	dummyFields := make([]ast.AnalyzedImportValue, 0)
	for _, toImport := range node.ToImport {
		// type imports are filtered out completely (the have no relevance during runtime)
		if toImport.IsTypeImport {
			if prev := self.currentModule.addType(toImport.Ident, newTypeWrapper(ast.NewUnknownType(), false, toImport.Span, true)); prev != nil {
				self.error(fmt.Sprintf("Type '%s' already exists in current scope", toImport.Ident), nil, toImport.Span)
			}
			continue
		}

		dummyFields = append(dummyFields, ast.AnalyzedImportValue{
			Ident: pAst.NewSpannedIdent(toImport.Ident, toImport.Span),
			Type:  ast.NewUnknownType(),
		})

		if prev := self.currentModule.addVar(toImport.Ident, NewVar(ast.NewUnknownType(), toImport.Span, ImportedVariableOriginKind, false), false); prev != nil {
			self.error(fmt.Sprintf("Name '%s' already exists in current scope", toImport.Ident), nil, toImport.Span)
		}
	}

	return ast.AnalyzedImport{
		ToImport:   dummyFields,
		FromModule: node.FromModule,
		Range:      node.Range,
	}
}

func (self *Analyzer) importItem(node pAst.ImportStatement) ast.AnalyzedImport {
	toImport := make([]ast.AnalyzedImportValue, 0)

	codeModule, found, err := self.host.ResolveCodeModule(node.FromModule.Ident())
	if err != nil {
		self.error(
			fmt.Sprintf("Host error: could not resolve module '%s': %s", node.FromModule.Ident(), err.Error()),
			nil,
			node.Span(),
		)

		return self.importDummyFields(node)
	}

	if found {
		self.currentModule.ImportsModules = append(self.currentModule.ImportsModules, node.FromModule.Ident())

		lexer := parser.NewLexer(codeModule, node.FromModule.Ident())
		parser := parser.NewParser(lexer, node.FromModule.Ident())
		parsed, softErrors, err := parser.Parse()
		self.syntaxErrors = append(self.syntaxErrors, softErrors...)
		if err != nil {
			self.syntaxErrors = append(self.syntaxErrors, *err)
			return self.importDummyFields(node)
		}

		module, alreadyAnalyzed := self.modules[node.FromModule.Ident()]

		if !alreadyAnalyzed {
			self.analyzeModule(node.FromModule.Ident(), parsed)

			// analyze if this import causes a cyclic dependency
			if path, isCyclic := self.importGraphIsCyclic(self.currentModuleName); isCyclic {
				importInner := strings.Join(path, " -> ")

				self.error(
					fmt.Sprintf("Illegal cyclic import: module %s", importInner),
					nil,
					node.Span(),
				)
			}

			module = self.modules[node.FromModule.Ident()]
		}

		// add values to current scope
		for _, item := range node.ToImport {
			// type imports need special action: only add the type and filter out this import
			if item.IsTypeImport {
				typ, found := module.getType(item.Ident)
				if !found {
					self.error(
						fmt.Sprintf("No type named '%s' found in module '%s'", item.Ident, node.FromModule),
						nil,
						item.Span,
					)

					if prev := self.currentModule.addType(item.Ident, newTypeWrapper(ast.NewUnknownType(), false, item.Span, true)); prev != nil {
						self.error(fmt.Sprintf("Type '%s' already exists in current scope", item.Ident), nil, item.Span)
					}
					continue
				}

				// cannot import this type
				if !typ.IsPub {
					self.error(
						fmt.Sprintf("Cannot import private type: '%s' is not declared as 'pub'", item.Ident),
						nil,
						item.Span,
					)
					self.hint(
						"This type is not declared as 'pub'",
						[]string{"A type can be declared as 'pub' like this: `pub type = ...`"},
						typ.NameSpan,
					)
				}

				if prev := self.currentModule.addType(item.Ident, newTypeWrapper(typ.Type.SetSpan(item.Span), false, item.Span, false)); prev != nil {
					self.error(fmt.Sprintf("Type '%s' already exists in current scope", item.Ident), nil, item.Span)
				}
				continue
			}

			fn, found := module.getFunc(item.Ident)
			if found {
				toImport = append(toImport, ast.AnalyzedImportValue{
					Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
					Type:  fn.Type(item.Span),
				})

				// cannot import this function
				if fn.Modifier != pAst.FN_MODIFIER_PUB {
					self.error(
						fmt.Sprintf("Cannot import private function: '%s' is not declared as 'pub'", item.Ident),
						nil,
						item.Span,
					)
					self.hint(
						"This function is not declared as 'pub'",
						[]string{"A function can be declared as 'pub' like this: `pub fn name(...) { ... }`"},
						fn.FnType.(normalFunction).Ident.Span(),
					)
				}

				if prev := self.currentModule.addVar(item.Ident, NewVar(fn.Type(item.Span), item.Span, ImportedVariableOriginKind, false), false); prev != nil {
					self.error(fmt.Sprintf("Name '%s' already exists in current scope", item.Ident), nil, item.Span)
				}
				continue
			}

			val, found := module.Scopes[0].Values[item.Ident]
			if !found {
				self.error(
					fmt.Sprintf("No variable or function named '%s' found in module '%s'", item.Ident, node.FromModule),
					nil,
					item.Span,
				)
				toImport = append(toImport, ast.AnalyzedImportValue{
					Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
					Type:  ast.NewUnknownType(),
				})

				if prev := self.currentModule.addVar(item.Ident, NewVar(ast.NewUnknownType(), item.Span, ImportedVariableOriginKind, false), false); prev != nil {
					self.error(fmt.Sprintf("Name '%s' already exists in current scope", item.Ident), nil, item.Span)
				}
			} else {
				if !val.IsPub {
					// cannot import this variable
					self.error(
						fmt.Sprintf("Cannot import private variable: '%s' is not declared as 'pub'", item.Ident),
						nil,
						item.Span,
					)
					self.hint(
						"This variable is not declared as 'pub'",
						[]string{"A variable can be declared as 'pub' like this: `pub let name = ...`"},
						val.Span,
					)
				}

				toImport = append(toImport, ast.AnalyzedImportValue{
					Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
					Type:  val.Type.SetSpan(item.Span),
				})
				if prev := self.currentModule.addVar(item.Ident, NewVar(val.Type.SetSpan(item.Span), item.Span, ImportedVariableOriginKind, false), false); prev != nil {
					self.error(fmt.Sprintf("Name '%s' already exists in current scope", item.Ident), nil, item.Span)
				}
			}
		}

		return ast.AnalyzedImport{
			ToImport:    toImport,
			FromModule:  node.FromModule,
			Range:       node.Range,
			TargetIsHMS: true,
		}
	}

	for _, item := range node.ToImport {
		typ, moduleFound, valueFound := self.host.GetBuiltinImport(node.FromModule.Ident(), item.Ident, item.Span)
		if !moduleFound {
			self.error(
				fmt.Sprintf("Module '%s' not found", node.FromModule),
				nil,
				node.FromModule.Span(),
			)
			return self.importDummyFields(node)
		} else if !valueFound {
			self.error(
				fmt.Sprintf("No variable or function named '%s' found in module '%s'", item.Ident, node.FromModule),
				nil,
				item.Span,
			)
			toImport = append(toImport, ast.AnalyzedImportValue{
				Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
				Type:  ast.NewUnknownType(),
			})
			if prev := self.currentModule.addVar(item.Ident, NewVar(ast.NewUnknownType(), item.Span, ImportedVariableOriginKind, false), false); prev != nil {
				self.error(fmt.Sprintf("Name '%s' already exists in current scope", item.Ident), nil, item.Span)
			}
		} else {
			// type imports need special action: only add the type and filter out this import
			if item.IsTypeImport {
				if prev := self.currentModule.addType(item.Ident, newTypeWrapper(typ.SetSpan(item.Span), false, item.Span, false)); prev != nil {
					self.error(fmt.Sprintf("Type '%s' already exists in current scope", item.Ident), nil, item.Span)
				}
				continue
			}

			toImport = append(toImport, ast.AnalyzedImportValue{
				Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
				Type:  typ.SetSpan(item.Span),
			})
			if prev := self.currentModule.addVar(item.Ident, NewVar(typ.SetSpan(item.Span), item.Span, ImportedVariableOriginKind, false), false); prev != nil {
				self.error(fmt.Sprintf("Name '%s' already exists in current scope", item.Ident), nil, item.Span)
			}
		}
	}

	return ast.AnalyzedImport{
		ToImport:    toImport,
		FromModule:  node.FromModule,
		Range:       node.Range,
		TargetIsHMS: false,
	}
}

//
// Function definition
//

// TODO: handle singletons (sort of compile them out)
func (self *Analyzer) functionDefinition(node pAst.FunctionDefinition) ast.AnalyzedFunctionDefinition {
	fnReturnType := self.ConvertType(node.ReturnType, true).SetSpan(node.ReturnType.Span())

	// analyze params
	newParams := make([]ast.AnalyzedFnParam, 0)

	self.currentModule.pushScope()

	if node.Ident.Ident() == "main" || node.Modifier == pAst.FN_MODIFIER_EVENT {
		modifierErrMsg := ""
		if node.Modifier != pAst.FN_MODIFIER_NONE {
			modifierErrMsg = node.Modifier.String() + " "
		}

		// for this function, only check thate there are NO params
		if len(node.Parameters) > 0 {
			errMsgVerb := "are"
			if len(node.Parameters) == 1 {
				errMsgVerb = "is"
			}

			self.error(
				fmt.Sprintf("The '%s%s' function must have 0 parameters, however %d %s defined", modifierErrMsg, node.Ident.Ident(), len(node.Parameters), errMsgVerb),
				nil,
				node.ParamSpan,
			)
		}

		// the return type of the `main` function is always `null`
		if fnReturnType.Kind() != ast.UnknownTypeKind && fnReturnType.Kind() != ast.NullTypeKind {
			self.error(
				fmt.Sprintf("The return type of the '%s%s' function must be '%s', but is declared as '%s'", modifierErrMsg, node.Ident.Ident(), ast.NewNullType(errors.Span{}).Kind(), fnReturnType.Kind()),
				[]string{fmt.Sprintf("Remove the return type: `fn %s%s() { ... }`", modifierErrMsg, node.Ident.Ident())},
				fnReturnType.Span(),
			)
			fnReturnType = ast.NewUnknownType()
		}
	} else {
		newParams = self.analyzeParams(node.Parameters)
	}

	// set current function
	self.currentModule.setCurrentFunc(node.Ident.Ident())

	// analyze function body
	analyzedBlock := self.block(node.Body, false)

	// analyze return type
	if err := self.TypeCheck(analyzedBlock.Type(), fnReturnType, true); err != nil {
		self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
		if err.ExpectedDiagnostic != nil {
			self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
		}
	}

	// drop scope when finished
	self.dropScope(true)

	// unset current function
	self.currentModule.CurrentFunction = nil

	return ast.AnalyzedFunctionDefinition{
		Ident:      node.Ident,
		Parameters: newParams,
		ReturnType: fnReturnType,
		Body:       analyzedBlock,
		Modifier:   node.Modifier,
		Range:      node.Range,
	}
}

func (self *Analyzer) analyzeParams(params []pAst.FnParam) []ast.AnalyzedFnParam {
	newParams := make([]ast.AnalyzedFnParam, 0)

	// analyze params (no doubles, valid types)
	existentParams := make(map[string]struct{}) // to keep track of duplicate param names

	existentSingletons := make(map[pAst.SingletonReferenceType]struct{}) // keep track of duplicate singleton params

	encounteredNonSingletonParam := false

	for _, param := range params {
		isSingletonExtractor := false

		if param.Type.Kind() == pAst.SingletonReferenceParserTypeKind {
			isSingletonExtractor = true

			if encounteredNonSingletonParam {
				newParams = append(newParams,
					ast.AnalyzedFnParam{
						Ident:                param.Ident,
						Type:                 ast.NewUnknownType(),
						Span:                 param.Span,
						IsSingletonExtractor: true,
					},
				)

				// add this parameter to the new scope
				self.currentModule.addVar(param.Ident.Ident(), NewVar(
					ast.NewUnknownType(),
					param.Ident.Span(),
					ParameterVariableOriginKind,
					false,
				), false)

				self.error(
					fmt.Sprintf("Extraction of singleton '%s' follows normal parameter", param.Ident.Ident()),
					[]string{"Singletons are to be extracted as the first parameters of a function."},
					param.Span,
				)

				continue
			}

			singleton := param.Type.(pAst.SingletonReferenceType)

			if _, duplicate := existentSingletons[singleton]; duplicate {
				self.error(
					fmt.Sprintf("Duplicate extraction of singleton '%s'", param.Ident.Ident()),
					[]string{"Every unique singleton can only be extracted once per function"},
					param.Span,
				)
			}

			// add this singleton to the set of existent singletons
			existentSingletons[singleton] = struct{}{}
		} else {
			encounteredNonSingletonParam = true

			if _, duplicate := existentParams[param.Ident.Ident()]; duplicate {
				self.error(
					fmt.Sprintf("Duplicate declaration of parameter '%s'", param.Ident.Ident()),
					nil,
					param.Span,
				)
			}

			// add this param to the set of existent params
			existentParams[param.Ident.Ident()] = struct{}{}
		}

		newType := self.ConvertType(param.Type, true)

		// add param to new params
		newParams = append(newParams,
			ast.AnalyzedFnParam{
				Ident:                param.Ident,
				Type:                 newType,
				Span:                 param.Span,
				IsSingletonExtractor: isSingletonExtractor,
			},
		)

		// add this parameter to the new scope
		self.currentModule.addVar(param.Ident.Ident(), NewVar(
			newType,
			param.Ident.Span(),
			ParameterVariableOriginKind,
			false,
		), false)
	}

	return newParams
}

//
// Block
//

func (self *Analyzer) block(node pAst.Block, pushNewScope bool) ast.AnalyzedBlock {
	// push a new scope if required
	if pushNewScope {
		self.currentModule.pushScope()
	}

	// analyze statements
	statements := make([]ast.AnalyzedStatement, 0)
	var unreachableSpan *errors.Span = nil
	warnedUnreachable := false

	for _, statement := range node.Statements {
		newStatement := self.statement(statement)

		// if the previous statement had the never type, warn that this statement is unreachable
		if unreachableSpan != nil && !warnedUnreachable {
			warnedUnreachable = true
			self.warn(
				"Unreachable statement",
				nil,
				newStatement.Span(),
			)
			self.hint(
				"Any code following this statement is unreachable",
				nil,
				*unreachableSpan,
			)
		}

		// detect if this statement renders all following unreachable
		if unreachableSpan == nil && newStatement.Type().Kind() == ast.NeverTypeKind {
			span := newStatement.Span()
			unreachableSpan = &span
		}

		statements = append(statements, newStatement)
	}

	// analyze optional trailing expression
	var trailingExpr ast.AnalyzedExpression = nil

	if node.Expression != nil {
		trailingExpr = self.expression(node.Expression)

		if unreachableSpan != nil && !warnedUnreachable {
			self.warn(
				"Unreachable expression",
				nil,
				node.Expression.Span(),
			)
			self.hint(
				"Any code following this statement is unreachable",
				nil,
				*unreachableSpan,
			)
		}
	}

	// if this block diverges, the entrire block has the never type
	// otherwise, use the type of the trailing expression
	var resultType ast.Type
	if unreachableSpan != nil {
		resultType = ast.NewNeverType()
	} else if trailingExpr != nil {
		resultType = trailingExpr.Type()
	} else {
		resultType = ast.NewNullType(node.Range)
	}

	// pop scope if one was pushed at the beginning
	if pushNewScope {
		self.dropScope(true)
	}

	return ast.AnalyzedBlock{
		Statements: statements,
		Expression: trailingExpr,
		ResultType: resultType,
		Range:      node.Range,
	}
}
