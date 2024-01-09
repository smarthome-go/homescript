package analyzer

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Function signatures
//

func (self *Analyzer) functionSignature(node pAst.FunctionDefinition) {
	newParams := make([]ast.AnalyzedFnParam, 0)

	// This set is used to prevent singletons from being extracted multiple times
	extractedSet := make(map[string]struct{})

	for _, param := range node.Parameters {
		singletonIdent := ""
		isSingletonExtractor := false
		if param.Type.Kind() == pAst.SingletonReferenceParserTypeKind {
			isSingletonExtractor = true
			singletonIdent = param.Type.(pAst.SingletonReferenceType).Ident.Ident()

			_, alreadyExtracted := extractedSet[singletonIdent]
			if alreadyExtracted {
				self.error(
					fmt.Sprintf("Singleton `%s` is already being extracted", singletonIdent),
					[]string{
						"Singletons can only be extracted once at the start of the parameter list",
						fmt.Sprintf("Remove this parameter `%s`", param.String()),
					},
					param.Span,
				)
				continue
			}

			extractedSet[singletonIdent] = struct{}{}
		}

		newParams = append(newParams, ast.AnalyzedFnParam{
			Ident:                param.Ident,
			Type:                 self.ConvertType(param.Type, false), // errors are only reported in the `self.functionDefinition` method
			Span:                 param.Span,
			IsSingletonExtractor: isSingletonExtractor,
			SingletonIdent:       singletonIdent,
		})
	}

	// add function to current module
	if prev, exists := self.currentModule.getFunc(node.Ident.Ident()); exists {
		// check if the identifier conflicts with another function
		self.error(
			fmt.Sprintf("Duplicate function definition of '%s'", node.Ident.Ident()),
			[]string{"Consider changing the name of this function"},
			node.Ident.Span(),
		)
		self.hint(
			fmt.Sprintf("Function '%s' previously defined here", node.Ident.Ident()),
			nil,
			(*prev).FnType.(normalFunction).Ident.Span(),
		)
	}

	self.currentModule.addFunc(newFunction(
		newNormalFunction(node.Ident),
		newParams,
		node.ParamSpan,
		self.ConvertType(node.ReturnType, false), // errors are only reported in the `self.functionDefinition` method
		node.ReturnType.Span(),                   // is the return span really required?
		node.Modifier,
	))
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
		Ident: node.Ident,
		Parameters: ast.AnalyzedFunctionParams{
			List: newParams,
			Span: node.ParamSpan,
		},
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
		singletonIdent := ""

		if param.Type.Kind() == pAst.SingletonReferenceParserTypeKind {
			typ := param.Type.(pAst.SingletonReferenceType)
			singletonIdent = typ.Ident.Ident()
			isSingletonExtractor = true

			if encounteredNonSingletonParam {
				newParams = append(newParams,
					ast.AnalyzedFnParam{
						Ident:                param.Ident,
						Type:                 ast.NewUnknownType(),
						Span:                 param.Span,
						IsSingletonExtractor: true,
						SingletonIdent:       singletonIdent,
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
				SingletonIdent:       singletonIdent,
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
// Import
//

func (self *Analyzer) importDummyFields(node pAst.ImportStatement) ast.AnalyzedImport {
	dummyFields := make([]ast.AnalyzedImportValue, 0)
	for _, toImport := range node.ToImport {
		// type imports are filtered out completely (the have no relevance during runtime)
		if toImport.Kind == pAst.IMPORT_KIND_TYPE {
			if prev := self.currentModule.addType(toImport.Ident, newTypeWrapper(ast.NewUnknownType(), false, toImport.Span, true)); prev != nil {
				self.error(fmt.Sprintf("Type '%s' already exists in current scope", toImport.Ident), nil, toImport.Span)
			}
			continue
		}

		// TODO: also filter out template imports (no relevance during runtime)
		panic("TODO")

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
			// TODO: template imports also need special action
			panic("TODO: templates")

			// type imports need special action: only add the type and filter out this import
			if item.Kind == pAst.IMPORT_KIND_TYPE {
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
		imported, moduleFound, valueFound := self.host.GetBuiltinImport(node.FromModule.Ident(), item.Ident, item.Span, item.Kind)
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
			// Type and templ imports need special action: only add the type and filter out this import
			switch item.Kind {
			case pAst.IMPORT_KIND_TYPE:
				if prev := self.currentModule.addType(item.Ident, newTypeWrapper(imported.Type.SetSpan(item.Span), false, item.Span, false)); prev != nil {
					self.error(fmt.Sprintf("Type '%s' already exists in current scope", item.Ident), nil, item.Span)
				}
				continue
			case pAst.IMPORT_KIND_TEMPLATE:
				// TODO: where to get the template spec?
				prev, prevFound := self.currentModule.addTemplate(item.Ident, *imported.Template)
				if prevFound {
					self.error(fmt.Sprintf("Template '%s' already exists in current module", item.Ident), nil, item.Span)
					self.hint(fmt.Sprintf("Template `%s` previously imported here", item.Ident), make([]string, 0), prev.Span)
				}

				continue
			}

			toImport = append(toImport, ast.AnalyzedImportValue{
				Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
				Type:  imported.Type.SetSpan(item.Span),
			})
			if prev := self.currentModule.addVar(item.Ident, NewVar(imported.Type.SetSpan(item.Span), item.Span, ImportedVariableOriginKind, false), false); prev != nil {
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
// Impl block
//

func (self *Analyzer) implBlock(node pAst.ImplBlock) ast.AnalyzedImplBlock {
	// TODO: check that the constraints of the template are satisfied!

	singletonType := ast.NewUnknownType()

	// Check that this singleton exists
	singleton, exists := self.currentModule.Singletons[node.SingletonIdent.Ident()]
	if !exists {
		self.error(
			fmt.Sprintf("Undeclared singleton `%s`", node.SingletonIdent.Ident()),
			[]string{
				"Cannot implement methods for non-existent singleton",
				fmt.Sprintf("Singleton types can be declared like this: `@%s\n type %sFoo = ...;`", node.SingletonIdent.Ident(), node.SingletonIdent.Ident()),
			},
			node.SingletonIdent.Span(),
		)
	} else {
		singletonType = singleton.Type.SetSpan(node.SingletonIdent.Span())
	}

	// Analyze each method
	methods := make([]ast.AnalyzedFunctionDefinition, 0)
	for _, fn := range node.Methods {
		methods = append(methods, self.functionDefinition(fn))
	}

	// If this impl uses a template, analyze that it is valid
	if node.TemplateIdent != nil {
		// Check if the template exists and retrieve it
		tmpl, templateFound := self.currentModule.getTemplate(node.TemplateIdent.Ident())

		if !templateFound {
			self.error(
				fmt.Sprintf("Template `%s` not found", node.TemplateIdent.Ident()),
				[]string{fmt.Sprintf("Templates can be imported like this: `import templ %s;`", node.TemplateIdent.Ident())},
				node.TemplateIdent.Span(),
			)
		} else {
			// TODO: do template analysis
			fmt.Printf("Would analyze template now: %v\n", tmpl)
			self.validateTemplateConstraints(
				singletonType,
				node.SingletonIdent,
				tmpl,
				*node.TemplateIdent,
				methods,
				node.Span,
			)
		}
	}

	return ast.AnalyzedImplBlock{
		SingletonIdent: node.SingletonIdent,
		SingletonType:  singletonType,
		TemplateIdent:  node.TemplateIdent,
		Methods:        methods,
		Span:           node.Span,
	}
}

// This method must validate the following constraints:
// - Validate that all required methods (and no more) exist with the correct signatures
// - Ignore any singleton extractions, just make sure that the singleton on which the template is implemented is also extracted
func (self *Analyzer) validateTemplateConstraints(
	singletonType ast.Type,
	singletonIdent pAst.SpannedIdent,
	template ast.TemplateSpec,
	templateIdent pAst.SpannedIdent,
	methods []ast.AnalyzedFunctionDefinition,
	span errors.Span,
) {
	// Validate that all required methods exist with their correct signatures
	for reqName, reqSignature := range template.RequiredMethods {
		isImplemented := false

		for _, method := range methods {
			if method.Ident.Ident() == reqName {
				isImplemented = true

				fmt.Printf("got: %v | expected: %v\n", method.Type(), reqSignature)

				// TODO: validate correct implementation
				if err := self.TypeCheck(method.Type(), reqSignature.SetSpan(span), true); err != nil {
					self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
					if err.ExpectedDiagnostic != nil {
						self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
					}
				} else {
					// Validate that this method also extracts the singleton.
					// If not, it is not really useful
					extractsSingleton := false

					for _, param := range method.Parameters.List {
						if param.IsSingletonExtractor && param.SingletonIdent == singletonIdent.Ident() {
							extractsSingleton = true
							break
						}
					}

					if !extractsSingleton {
						self.error(
							fmt.Sprintf("Method does not extract singleton `%s`", singletonIdent.Ident()),
							[]string{
								fmt.Sprintf("Since the method is implemented for `%s`, it should also extract it", singletonIdent.Ident()),
								fmt.Sprintf("Singletons can be extracted like this: fn %s(ident: %s, ...)", method.Ident.Ident(), singletonIdent.Ident()),
							},
							method.Range,
						)
					}
				}

				break
			}
		}

		if !isImplemented {
			returnType := ""
			if reqSignature.ReturnType.Kind() != ast.NullTypeKind {
				returnType = fmt.Sprintf(" -> %s", reqSignature.ReturnType.String())
			}

			self.error(
				fmt.Sprintf("Not all methods implemented: implementation `%s` is is missing", reqName),
				[]string{
					"Template is not satisfied",
					fmt.Sprintf("It can be implemented like this: `fn %s(%s)%s { ... }", reqName, reqSignature.Params.String(), returnType),
				},
				span,
			)
		}
	}

	// Validate that there are no excess methods
	for _, method := range methods {
		isRequired := false

		for reqName := range template.RequiredMethods {
			if reqName == method.Ident.Ident() {
				isRequired = true
				break
			}
		}

		if !isRequired {
			self.error(
				fmt.Sprintf("Additional method `%s` implemented: this method is not part of the template `%s`", method.Ident, templateIdent.Ident()),
				[]string{"Remove this function definition"},
				method.Range,
			)
		}
	}
}
