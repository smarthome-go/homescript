package analyzer

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
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
		node.Ident.Span(),
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

	// TODO: handle event functions somewhere else
	if node.Ident.Ident() == "main" {
		modifierErrMsg := ""
		if node.Modifier != pAst.FN_MODIFIER_NONE {
			modifierErrMsg = node.Modifier.String() + " "
		}

		// Create a slice that does not contain singleton extractions
		filteredWithoutExtractions := make([]pAst.FnParam, 0)
		filteredExtractions := make([]pAst.FnParam, 0)
		for _, param := range node.Parameters {
			if param.Type.Kind() == pAst.SingletonReferenceParserTypeKind {
				filteredExtractions = append(filteredExtractions, param)
				continue
			}

			filteredWithoutExtractions = append(filteredWithoutExtractions, param)
		}

		// Analyze all singleton parameters
		// other parameters are errors anyways
		newParams = self.analyzeParams(filteredExtractions)

		// for this function, only check that there are NO params
		if len(filteredWithoutExtractions) > 0 {
			errMsgVerb := "are"
			if len(filteredWithoutExtractions) == 1 {
				errMsgVerb = "is"
			}

			self.error(
				fmt.Sprintf("The '%s%s' function must have 0 parameters, however %d %s defined", modifierErrMsg, node.Ident.Ident(), len(filteredWithoutExtractions), errMsgVerb),
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

	fmt.Printf("function=%s | expected=%s | got=%s\n", node.Ident, fnReturnType, analyzedBlock.Type())

	// analyze return type
	if err := self.TypeCheck(analyzedBlock.Type(), fnReturnType, TypeCheckOptions{
		AllowFunctionTypes:          true,
		IgnoreFnParamNameMismatches: false,
	}); err != nil {
		self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
		if err.ExpectedDiagnostic != nil {
			self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
		}
	}

	// drop scope when finished
	self.dropScope(true)

	// validate annotations.
	var annotations *ast.AnalyzedFunctionAnnotation
	if node.Annotation != nil {
		annotationsTemp := self.analyzeFnAnnotations(*node.Annotation, node.Ident.Ident())
		annotations = &annotationsTemp
	}

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
		Annotation: annotations,
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
		isSingletonExtractor := param.Type.Kind() == pAst.SingletonReferenceParserTypeKind
		singletonIdent := ""

		if isSingletonExtractor {
			typ := param.Type.(pAst.SingletonReferenceType)
			singletonIdent = typ.Ident.Ident()

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

		// Add param to new list of params
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

func entityErrNameFromImportKind(from pAst.IMPORT_KIND) string {
	// Customize *what* has not been found based on the import kind
	switch from {
	case pAst.IMPORT_KIND_NORMAL:
		return "variable or function"
	case pAst.IMPORT_KIND_TYPE:
		return "type"
	case pAst.IMPORT_KIND_TEMPLATE:
		return "template"
	case pAst.IMPORT_KIND_TRIGGER:
		return "trigger"
	default:
		panic("BUG warning: a new import kind was added without updating this code")
	}
}

func (self *Analyzer) importDummyFields(node pAst.ImportStatement) ast.AnalyzedImport {
	dummyFields := make([]ast.AnalyzedImportValue, 0)
	for _, toImport := range node.ToImport {
		// type and template imports are filtered out completely (the have no relevance during runtime)
		switch toImport.Kind {
		case pAst.IMPORT_KIND_TYPE:
			if prev := self.currentModule.addType(toImport.Ident, newTypeWrapper(ast.NewUnknownType(), false, toImport.Span, true)); prev != nil {
				self.error(fmt.Sprintf("Type '%s' already exists in current scope", toImport.Ident), nil, toImport.Span)
			}
		case pAst.IMPORT_KIND_TEMPLATE:
			templ := ast.TemplateSpec{
				BaseMethods:         make(map[string]ast.TemplateMethod),
				Capabilities:        make(map[string]ast.TemplateCapability),
				DefaultCapabilities: make([]string, 0),
				Span:                toImport.Span,
			}
			if prev, prevFound := self.currentModule.addTemplate(toImport.Ident, templ); prevFound {
				self.error(fmt.Sprintf("Template '%s' already exists in current module", toImport.Ident), nil, toImport.Span)
				self.hint(fmt.Sprintf("Template `%s` previously imported here", toImport.Ident), make([]string, 0), prev.Span)
			}
		case pAst.IMPORT_KIND_TRIGGER:
			// TODO: what to do here?
			// fn := TriggerFunction{
			// 	TriggerFnType:  ast.NewFunctionType(
			// 		ast.NewNormalFunctionTypeParamKind(),
			//
			// 	),
			// 	CallbackFnType: ast.FunctionType{},
			// 	Connective:     0,
			// 	ImportedAt:     toImport.Span,
			// }
			// if _, prevFound := self.currentModule.addTrigger(toImport.Ident, fn); prevFound {
			// 	self.error(fmt.Sprintf("Trigger '%s' already exists in current module", toImport.Ident), nil, toImport.Span)
			// }
		case pAst.IMPORT_KIND_NORMAL:
			dummyFields = append(dummyFields, ast.AnalyzedImportValue{
				Ident: pAst.NewSpannedIdent(toImport.Ident, toImport.Span),
				Type:  ast.NewUnknownType(),
			})

			if prev := self.currentModule.addVar(toImport.Ident, NewVar(ast.NewUnknownType(), toImport.Span, ImportedVariableOriginKind, false), false); prev != nil {
				self.error(fmt.Sprintf("Name '%s' already exists in current scope", toImport.Ident), nil, toImport.Span)
			}
		}
	}

	return ast.AnalyzedImport{
		ToImport:    dummyFields,
		FromModule:  node.FromModule,
		Range:       node.Range,
		TargetIsHMS: false,
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

		lexer := lexer.NewLexer(codeModule, node.FromModule.Ident())
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
			// panic("TODO: templates")
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

			if item.Kind == pAst.IMPORT_KIND_TEMPLATE {
				templ, found := module.getTemplate(item.Ident)
				if !found {
					self.error(
						fmt.Sprintf("No template named '%s' found in module '%s'", item.Ident, node.FromModule),
						nil,
						item.Span,
					)

					if _, prevFound := self.currentModule.addTemplate(item.Ident, templ); prevFound {
						self.error(fmt.Sprintf("Template '%s' already exists in current scope", item.Ident), nil, item.Span)
					}
					continue
				}

				if _, prevFound := self.currentModule.addTemplate(item.Ident, templ); prevFound {
					self.error(fmt.Sprintf("Template '%s' already exists in current scope", item.Ident), nil, item.Span)
				}

				continue
			}

			if item.Kind == pAst.IMPORT_KIND_TRIGGER {
				trigg, found := module.getTrigger(item.Ident)
				if !found {
					self.error(
						fmt.Sprintf("No trigger named '%s' found in module '%s'", item.Ident, node.FromModule),
						nil,
						item.Span,
					)

					if _, prevFound := self.currentModule.addTrigger(item.Ident, trigg); prevFound {
						self.error(fmt.Sprintf("Trigger '%s' already exists in current scope", item.Ident), nil, item.Span)
					}
					continue
				}

				if _, prevFound := self.currentModule.addTrigger(item.Ident, trigg); prevFound {
					self.error(fmt.Sprintf("Trigger '%s' already exists in current scope", item.Ident), nil, item.Span)
				}

				toImport = append(toImport, ast.AnalyzedImportValue{
					Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
					Kind:  pAst.IMPORT_KIND_TRIGGER,
					Type:  nil,
				})

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

				if prev := self.currentModule.addVar(
					item.Ident,
					NewVar(
						fn.Type(item.Span),
						item.Span,
						ImportedVariableOriginKind,
						false,
					),
					false,
				); prev != nil {
					self.error(fmt.Sprintf("Name '%s' already exists in current scope", item.Ident), nil, item.Span)
				}

				continue
			}

			val, found := module.Scopes[0].Values[item.Ident]
			if !found {
				self.error(
					fmt.Sprintf("No %s named '%s' found in module '%s'", entityErrNameFromImportKind(item.Kind), item.Ident, node.FromModule),
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
				fmt.Sprintf("No %s named '%s' found in module '%s'", entityErrNameFromImportKind(item.Kind), item.Ident, node.FromModule),
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

				toImport = append(toImport, ast.AnalyzedImportValue{
					Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
					Type:  imported.Type.SetSpan(item.Span),
					Kind:  pAst.IMPORT_KIND_TYPE,
				})

				continue
			case pAst.IMPORT_KIND_TEMPLATE:
				prev, prevFound := self.currentModule.addTemplate(item.Ident, *imported.Template)
				if prevFound {
					self.error(fmt.Sprintf("Template '%s' already exists in current module", item.Ident), nil, item.Span)
					self.hint(fmt.Sprintf("Template `%s` previously imported here", item.Ident), make([]string, 0), prev.Span)
				}

				toImport = append(toImport, ast.AnalyzedImportValue{
					Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
					Type:  nil,
					Kind:  pAst.IMPORT_KIND_TEMPLATE,
				})

				continue
			case pAst.IMPORT_KIND_TRIGGER:
				prev, prevFound := self.currentModule.addTrigger(item.Ident, *imported.Trigger)
				if prevFound {
					self.error(fmt.Sprintf("Trigger function '%s' already exists in current module", item.Ident), nil, item.Span)
					self.hint(fmt.Sprintf("Trigger function `%s` previously imported here", item.Ident), make([]string, 0), prev.ImportedAt)
				}

				toImport = append(toImport, ast.AnalyzedImportValue{
					Ident: pAst.NewSpannedIdent(item.Ident, item.Span),
					Type:  nil,
					Kind:  pAst.IMPORT_KIND_TRIGGER,
				})

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

	// Check that this singleton singletonExists
	singleton, singletonExists := self.currentModule.Singletons[node.SingletonIdent.Ident()]
	if !singletonExists {
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
		method := self.functionDefinition(fn)
		methods = append(methods, method)
	}

	// Check if the template exists and retrieve it
	tmpl, templateFound := self.currentModule.getTemplate(node.UsingTemplate.Template.Ident())
	if !templateFound {
		self.error(
			fmt.Sprintf("Template `%s` not found", node.UsingTemplate.Template.Ident()),
			[]string{fmt.Sprintf("Templates can be imported like this: `import templ %s;`", node.UsingTemplate.Template.Ident())},
			node.UsingTemplate.Template.Span(),
		)

		return ast.AnalyzedImplBlock{
			SingletonIdent:    node.SingletonIdent,
			SingletonType:     singletonType,
			UsingTemplate:     node.UsingTemplate,
			Methods:           methods,
			Span:              node.Span,
			FinalCapabilities: nil,
		}
	}

	finalCapabilities := self.templateCapabilitiesWithDefault(
		node.UsingTemplate.Template.Ident(),
		tmpl,
		node.UsingTemplate.UserDefinedCapabilities,
	)

	requiredMethods, err := self.WithCapabilities(
		tmpl,
		finalCapabilities,
	)

	// If there are serious errors with the capability config, skip checking other details
	if err {
		return ast.AnalyzedImplBlock{
			SingletonIdent:    node.SingletonIdent,
			SingletonType:     singletonType,
			UsingTemplate:     node.UsingTemplate,
			Methods:           methods,
			Span:              node.Span,
			FinalCapabilities: nil,
		}
	}

	self.validateTemplateConstraints(
		singletonType,
		node.SingletonIdent,
		tmpl,
		node.UsingTemplate,
		methods,
		// Computes span of template header
		node.Span.Start.Until(node.SingletonIdent.Span().End, node.Span.Filename),
		requiredMethods,
	)

	// Update the singleton so that the post-validation hook is aware of this impl
	// NOTE: this must only be done if the singleton exists, otherwise, this is not even required
	// as the post-validation hook will never be called.
	if singletonExists {
		old := self.currentModule.Singletons[node.SingletonIdent.Ident()]
		old.ImplementsTemplates = append(old.ImplementsTemplates, node.UsingTemplate)
		self.currentModule.Singletons[node.SingletonIdent.Ident()] = old
	}

	return ast.AnalyzedImplBlock{
		SingletonIdent:    node.SingletonIdent,
		SingletonType:     singletonType,
		UsingTemplate:     node.UsingTemplate,
		Methods:           methods,
		Span:              node.Span,
		FinalCapabilities: finalCapabilities,
	}
}

func (self *Analyzer) templateTypeFail(method ast.AnalyzedFunctionDefinition, singletonIdent string) {
	// Validate that this method also extracts the singleton.
	// If not, it is not really useful
	extractsSingleton := false

	for _, param := range method.Parameters.List {
		if param.IsSingletonExtractor && param.SingletonIdent == singletonIdent {
			extractsSingleton = true
			break
		}
	}

	if !extractsSingleton {
		self.error(
			fmt.Sprintf("Method does not extract singleton `%s`", singletonIdent),
			[]string{
				fmt.Sprintf("Since the method is implemented for `%s`, it should also extract it", singletonIdent),
				fmt.Sprintf("Singletons can be extracted like this: fn %s(ident: %s, ...)", method.Ident.Ident(), singletonIdent),
			},
			method.Range,
		)
	}
}

// This method must validate the following constraints:
// - Validate that all required methods (and no more) exist with the correct signatures
// - Ignore any singleton extractions, just make sure that the singleton on which the template is implemented is also extracted
func (self *Analyzer) validateTemplateConstraints(
	singletonType ast.Type,
	singletonIdent pAst.SpannedIdent,
	templateSpec ast.TemplateSpec,
	implementedTemplateWithCapabilities pAst.ImplBlockTemplate,
	methods []ast.AnalyzedFunctionDefinition,
	implHeaderSpan errors.Span,
	requiredMethods map[string]ast.TemplateMethod,
) {
	// Validate that all required methods exist with their correct signatures
	for reqName, reqMethod := range requiredMethods {
		isImplemented := false

		for _, method := range methods {
			if method.Ident.Ident() == reqName {
				isImplemented = true

				reqMethodParams := reqMethod.Signature.Params.(ast.NormalFunctionTypeParamKindIdentifier)

				hasParamErrs := false

				filteredReq := make([]ast.FunctionTypeParam, 0)
				for _, reqParam := range reqMethodParams.Params {
					if reqParam.IsSingletonExtractor {
						continue
					}
					filteredReq = append(filteredReq, reqParam)
				}

				filteredMethod := make([]ast.AnalyzedFnParam, 0)
				for _, methParam := range method.Parameters.List {
					if methParam.IsSingletonExtractor {
						continue
					}
					filteredMethod = append(filteredMethod, methParam)
				}

				// Check that the count of params is the same.
				if len(filteredReq) != len(filteredMethod) {
					trailingS := ""
					if len(reqMethodParams.Params) > 1 || len(reqMethodParams.Params) == 0 {
						trailingS = "s"
					}

					// If a singleton is extracted, mention that it is not counted in the parameter list.
					notes := []string{}
					if len(filteredMethod) != len(method.Parameters.List) {
						// TODO: dedicated syntax!
						notes = append(notes, "Singleton extractions do count as a parameter definition.")
					}

					for _, param := range filteredReq {
						notes = append(
							notes,
							fmt.Sprintf("Parameter `%s: %s` is missing but required",
								param.Name,
								param.Type,
							))
					}

					self.error(
						fmt.Sprintf(
							"Expected %d parameter%s, got %d",
							len(filteredReq),
							trailingS,
							len(filteredMethod),
						),
						notes,
						method.Parameters.Span,
					)
					break
				}

				// Iterate over params and check that name and type match.
				for idx, reqParam := range filteredReq {
					gotParam := filteredMethod[idx]

					if reqParam.Name.Ident() != gotParam.Ident.Ident() {
						// TODO: break now or later?
						hasParamErrs = true
						self.error(
							fmt.Sprintf("Expected parameter with name `%s`, got `%s`",
								reqParam.Name,
								gotParam.Ident.Ident(),
							),
							[]string{},
							gotParam.Ident.Span(),
						)
						break
					}

					if err := self.TypeCheck(gotParam.Type, reqParam.Type.SetSpan(implHeaderSpan), TypeCheckOptions{
						AllowFunctionTypes:          true,
						IgnoreFnParamNameMismatches: false,
					}); err != nil {
						self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
						if err.ExpectedDiagnostic != nil {
							self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
						}

						// TODO: BREAK as usual
						hasParamErrs = true
						break
					}
				}

				if !hasParamErrs {
					self.templateTypeFail(method, singletonIdent.Ident())
				}

				// Check that the method return type matches the required signature.
				if err := self.TypeCheck(
					method.ReturnType,
					reqMethod.Signature.ReturnType.SetSpan(implHeaderSpan),
					TypeCheckOptions{
						AllowFunctionTypes:          true,
						IgnoreFnParamNameMismatches: false,
					},
				); err != nil {
					self.diagnostics = append(self.diagnostics, err.GotDiagnostic.WithContext("Regarding function's return type"))
					if err.ExpectedDiagnostic != nil {
						self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
					}
				} else {
					self.templateTypeFail(method, singletonIdent.Ident())
				}

				// Validate that the modifier is identical to the one that is needed
				if method.Modifier != reqMethod.Modifier {
					returnTypeString := ""
					if reqMethod.Signature.ReturnType.Kind() != ast.NullTypeKind {
						returnTypeString = fmt.Sprintf(" -> %s", reqMethod.Signature.ReturnType.String())
					}

					if reqMethod.Modifier == pAst.FN_MODIFIER_NONE {
						self.error(
							fmt.Sprintf("Template method has redundant modifier `%s`", method.Modifier.String()),
							[]string{
								fmt.Sprintf("Remove the modifier like this: `fn %s(...)%s {...}`", reqName, returnTypeString),
							},
							method.Range,
						)
					} else if method.Modifier == pAst.FN_MODIFIER_NONE {
						self.error(
							fmt.Sprintf("Template method lacks required modifier `%s`", reqMethod.Modifier.String()),
							[]string{
								fmt.Sprintf("Add the modifier like this: `%s fn %s(...)%s {...}`", reqMethod.Modifier.String(), reqName, returnTypeString),
							},
							method.Range,
						)
					} else {
						self.error(
							fmt.Sprintf("Expected modifier `%s`, but found `%s`", reqMethod.Modifier.String(), method.Modifier.String()),
							[]string{
								fmt.Sprintf("Change the `%s` modifier to `%s`", method.Modifier.String(), reqMethod.Modifier.String()),
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
			if reqMethod.Signature.ReturnType.Kind() != ast.NullTypeKind {
				returnType = fmt.Sprintf(" -> %s", reqMethod.Signature.ReturnType.String())
			}

			modifier := ""
			switch reqMethod.Modifier {
			case pAst.FN_MODIFIER_NONE:
				break
			case pAst.FN_MODIFIER_PUB:
				modifier = "pub "
			case pAst.FN_MODIFIER_EVENT:
				modifier = "event "
			default:
				panic(fmt.Sprintf("A new modifier kind was added without updating this code: `%s`", reqMethod.Modifier))
			}

			self.error(
				fmt.Sprintf("Not all methods implemented: method `%s` is is missing", reqName),
				[]string{
					"Template is not satisfied",
					fmt.Sprintf(
						"It can be implemented like this: `%sfn %s(self: %s, %s)%s { ... }",
						modifier,
						reqName,
						singletonIdent,
						reqMethod.Signature.Params.String(),
						returnType,
					),
				},
				implHeaderSpan,
			)
		}
	}

	// Validate that there are no excess methods
	for _, method := range methods {
		isRequired := false

		for reqName := range requiredMethods {
			if reqName == method.Ident.Ident() {
				isRequired = true
				break
			}
		}

		if !isRequired {
			self.error(
				fmt.Sprintf("Additional method `%s` implemented: this method is not part of the template `%s`", method.Ident, implementedTemplateWithCapabilities.Template.Ident()),
				[]string{"Remove this function definition"},
				method.Range,
			)
		}
	}
}
