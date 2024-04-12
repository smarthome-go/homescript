package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/evaluator"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
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
	TriggerConnective pAst.TriggerDispatchKeywordKind
	TriggerSource     string
	TriggerArgs       []value.Value
}

func (self TriggerCompiledAnnotation) Kind() CompiledAnnotationKind {
	return CompiledAnnotationKindTrigger
}

func (self *Compiler) compileFn(node ast.AnalyzedFunctionDefinition) *CompiledAnnotations {
	self.currFn = node.Ident.Ident()
	self.pushScope()
	defer self.popScope()

	// Compile annotations.
	var annotations *CompiledAnnotations
	if node.Annotation != nil {
		compiledItems := make([]CompiledAnnotation, len(node.Annotation.Items))

		for idx, annotation := range node.Annotation.Items {
			switch ann := annotation.(type) {
			case ast.AnalyzedAnnotationItemTrigger:
				// Evaluate arguments.
				eval := evaluator.NewInterpreter(self.analyzedSource, self.entryPointModule)

				values := make([]value.Value, len(ann.TriggerArgs.List))

				for idx, arg := range ann.TriggerArgs.List {
					argV, err := eval.Expression(arg.Expression)
					if err != nil {
						panic(fmt.Sprintf("Unreachable: comptime evaluation failed for trigger arg: %s", (*err).Message()))
					}

					argVUpgrade := upgradeValue(argV)
					values[idx] = *argVUpgrade
				}

				compiledItems[idx] = TriggerCompiledAnnotation{
					TriggerConnective: ann.TriggerConnective,
					TriggerSource:     ann.TriggerSource.Ident(),
					TriggerArgs:       values,
				}
			case ast.AnalyzedAnnotationItem:
				panic("TODO: this is not yet implemented")
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

	return annotations
}
