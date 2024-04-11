package analyzer

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

func (self *Analyzer) analyzeFnAnnotations(annotations pAst.FunctionAnnotationInner, ident string) ast.AnalyzedFunctionAnnotation {
	items := make([]ast.AnalyzedAnnotationItem, len(annotations.Items))

	for idx, annotation := range annotations.Items {
		items[idx] = self.analyzeFnAnnotation(annotation, ident)
	}

	return ast.AnalyzedFunctionAnnotation{
		Items: items,
		Span:  annotations.Span,
	}
}

func (self *Analyzer) analyzeFnAnnotation(annotation pAst.AnnotationItem, fnIdent string) ast.AnalyzedAnnotationItem {
	switch ann := annotation.(type) {
	case pAst.AnnotationItemIdent:
		panic("TODO: implement this kind of validation")
		return ast.AnalyzedAnnotationItemIdent{}
	case pAst.AnnotationItemTrigger:
		// Analyze event trigger
		trigger, triggerFound := self.currentModule.getTrigger(ann.TriggerSource.Ident())
		// triggerType, found, connectiveCorrect := self.host.GetTriggerEvent(node.EventIdent.Ident(), node.DispatchKeyword)
		if !triggerFound {
			self.error(
				fmt.Sprintf("Use of undefined trigger function '%s'", ann.TriggerSource.Ident()),
				[]string{
					fmt.Sprintf("Trigger functions can be imported like this: `import { trigger %s } from ... ;`", ann.TriggerSource.Ident()),
				},
				ann.TriggerSource.Span(),
			)
		}

		// Analyze callback function.
		callbackFn, _ := self.currentModule.getFunc(fnIdent)

		// Analyze callback function compatibility with event trigger.
		callbackFnType := callbackFn.Type(callbackFn.IdentSpan).(ast.FunctionType)

		args := ast.AnalyzedCallArgs{
			Span: ann.TriggerArgs.Span,
			List: []ast.AnalyzedCallArgument{},
		}

		if triggerFound {
			if err := self.TypeCheck(
				callbackFnType.SetSpan(callbackFn.IdentSpan),
				trigger.CallbackFnType.SetSpanAdvanced(ann.TriggerSource.Span(), ann.TriggerSource.Span()),
				true,
			); err != nil {
				self.diagnostics = append(self.diagnostics, err.GotDiagnostic.WithContext("Regarding callback function"))
				if err.ExpectedDiagnostic != nil {
					self.diagnostics = append(
						self.diagnostics,
						err.ExpectedDiagnostic.WithContext(
							fmt.Sprintf("Regarding callback function for trigger `%s`", ann.TriggerSource),
						),
					)
					err.ExpectedDiagnostic.Notes = append(
						err.ExpectedDiagnostic.Notes,
						fmt.Sprintf("Function `%s` used as callback for trigger `%s`", fnIdent, ann.TriggerSource),
					)
				}
				self.hint(
					fmt.Sprintf("This function is used as a callback for trigger `%s`", ann.TriggerSource),
					make([]string, 0),
					callbackFn.IdentSpan,
				)
			}
			args = self.callArgs(
				trigger.TriggerFnType,
				ann.TriggerArgs,
				false,
			)
		}

		return ast.AnalyzedAnnotationItemTrigger{
			TriggerConnective: ann.TriggerConnective,
			TriggerSource:     ann.TriggerSource,
			TriggerArgs:       args,
			Range:             ann.Range,
		}
	default:
		panic("TODO: a new kind of annotation item was added without updating this code")
	}
}
