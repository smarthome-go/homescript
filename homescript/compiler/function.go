package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Compiled function annotations.
//

type CompiledAnnotations struct {
	Items []CompiledAnnotation
}

type CompiledAnnotationKind uint8

const (
	CompiledAnnotationKindIdent CompiledAnnotationKind = iota
	CompiledAnnotationKindTrigger
)

type CompiledAnnotation interface {
	Kind() CompiledAnnotationKind
}

type IdentCompiledAnnotation struct {
	Ident string
}

func (self IdentCompiledAnnotation) Kind() CompiledAnnotationKind { return CompiledAnnotationKindIdent }

type TriggerCompiledAnnotation struct {
	CallbackFnIdent   string
	TriggerConnective pAst.TriggerDispatchKeywordKind
	TriggerSource     string
	// Which function to call in order to retrieve the arguments.
	// The compiler generates a 'hidden' function for every trigger annotation.
	ArgumentFunctionIdent string
}

func (self TriggerCompiledAnnotation) Kind() CompiledAnnotationKind {
	return CompiledAnnotationKindTrigger
}

func (self *Compiler) compileFn(node ast.AnalyzedFunctionDefinition) (annotations *CompiledAnnotations, mangledIdent string) {
	mangledFn := self.mangleFn(node.Ident.Ident())
	self.addFn(node.Ident.Ident(), mangledFn)
	self.currFn = node.Ident.Ident()
	self.pushScope()
	defer self.popScope()

	// Compile annotations.
	if node.Annotation != nil {
		compiledItems := make([]CompiledAnnotation, len(node.Annotation.Items))

		for idx, annotation := range node.Annotation.Items {
			switch ann := annotation.(type) {
			case ast.AnalyzedAnnotationItemTrigger:
				//
				// Compile argument function.
				//

				argFnIdent := fmt.Sprintf("TRIGGER_args_for_%s", node.Ident)
				argFnfnRetType := ast.NewListType(ast.NewAnyType(node.Range), node.Range)

				argList := make([]ast.AnalyzedExpression, len(ann.TriggerArgs.List))
				for idx, arg := range ann.TriggerArgs.List {
					argList[idx] = arg.Expression
				}

				// fmt.Printf("OLD = %s\n", self.currFn)
				currFnOld := self.currFn

				self.addFn(argFnIdent, argFnIdent)
				self.currFn = argFnIdent

				_, annotationCallbackIdent := self.compileFn(ast.AnalyzedFunctionDefinition{
					Ident: pAst.NewSpannedIdent(argFnIdent, node.Range),
					Parameters: ast.AnalyzedFunctionParams{
						List: make([]ast.AnalyzedFnParam, 0),
						Span: node.Range,
					},
					ReturnType: argFnfnRetType,
					Body: ast.AnalyzedBlock{
						Statements: []ast.AnalyzedStatement{},
						Expression: ast.AnalyzedListLiteralExpression{
							Values:   argList,
							Range:    node.Range,
							ListType: ast.NewAnyType(node.Range),
						},
						Range:      node.Range,
						ResultType: argFnfnRetType,
					},
					Modifier:   pAst.FN_MODIFIER_NONE,
					Annotation: nil,
					Range:      node.Range,
				})

				self.currFn = currFnOld

				//
				// End argument function.
				//

				compiledItems[idx] = TriggerCompiledAnnotation{
					CallbackFnIdent:       node.Ident.Ident(),
					TriggerConnective:     ann.TriggerConnective,
					TriggerSource:         ann.TriggerSource.Ident(),
					ArgumentFunctionIdent: annotationCallbackIdent,
				}
			case ast.AnalyzedAnnotationItem:
				fmt.Println("======= WARN: TODO: this is not yet implemented")
			default:
				panic("A new trigger kind was added without updating this code")
			}
		}

		annotations = &CompiledAnnotations{
			Items: compiledItems,
		}
	}

	// Value / stack  depth is replaced later
	mpIdx := self.insert(newOneIntInstruction(Opcode_AddMempointer, 0), node.Range)

	singletonExtractors := make([]ast.AnalyzedFnParam, 0)

	// Parameters are pushed in reverse-order, so they can be popped in the correct order.
	for _, param := range node.Parameters.List {
		// TODO: If the current parameter is a singleton extraction, do extra work.
		// Add a name-alias for the singleton so that each time the extracted name is used, the singleton is accessed instead.

		if param.IsSingletonExtractor {
			singletonExtractors = append(singletonExtractors, param)
			continue
		}

		name := self.mangleVar(param.Ident.Ident())
		self.insert(newOneStringInstruction(Opcode_SetVarImm, name), node.Range)
	}

	// NOTE: this is required because the `vm.Spawn` method allows for inserting custom values onto the stack.
	// These values should then be used to construct function parameter variables.
	// However, if singletons are being extracted at the beginning, this might interfere with the normal stack layout,
	// causing runtime panics.
	for _, singletonParam := range singletonExtractors {
		// Load singleton first
		name, found := self.getMangled(singletonParam.SingletonIdent)
		if !found {
			panic("Existing singleton not found")
		}
		self.insert(newOneStringInstruction(Opcode_GetGlobImm, name), node.Range)
		self.insert(newOneStringInstruction(Opcode_GetGlobImm, name), node.Range)

		localName := self.mangleVar(singletonParam.Ident.Ident())
		self.insert(newOneStringInstruction(Opcode_SetVarImm, localName), node.Range)
	}

	// When `return` is encountered, the compiler inserts a jump to this label.
	// Needed for restoring the memory pointer.
	cleanupLabel := self.mangleLabel("cleanup")
	self.CurrFn().CleanupLabel = cleanupLabel

	self.compileBlock(node.Body, false)

	varCnt := int64(self.CurrFn().CntVariables)
	self.CurrFn().Instructions[mpIdx] = newOneIntInstruction(Opcode_AddMempointer, varCnt)

	self.insert(newOneStringInstruction(Opcode_Label, cleanupLabel), node.Range)
	self.insert(newOneIntInstruction(Opcode_AddMempointer, -varCnt), node.Range)
	self.insert(newPrimitiveInstruction(Opcode_Return), node.Span())

	return annotations, mangledFn
}
