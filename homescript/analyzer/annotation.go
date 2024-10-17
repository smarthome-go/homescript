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
		const allowUnusedAnnotation = "allow_unused"

		switch ann.Ident.Ident() {
		case allowUnusedAnnotation:
			fn, found := self.currentModule.getFunc(fnIdent)
			if !found {
				panic("impossible")
			}

			fn.Used = true
		default:
			self.error(
				fmt.Sprintf("Illegal annotation: `%s`", ann.Ident),
				[]string{},
				ann.Span(),
			)
		}

		return ast.AnalyzedAnnotationItemIdent{
			Ident: ann.Ident,
		}
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

		if callbackFn.Modifier != pAst.FN_MODIFIER_EVENT {
			message := fmt.Sprintf("Target function has wrong modifier `%s`", callbackFn.Modifier)
			if callbackFn.Modifier == pAst.FN_MODIFIER_NONE {
				message = "Target function misses the `event` modifier"
			}

			self.error(
				message,
				[]string{
					"Functions invoked through a `trigger` will run once an *event* takes place",
					fmt.Sprintf("The correct modifier `event` can be implemented like this: `event fn %s(...)`", fnIdent),
				},
				callbackFn.IdentSpan,
			)
		}

		args := ast.AnalyzedCallArgs{
			Span: ann.TriggerArgs.Span,
			List: []ast.AnalyzedCallArgument{},
		}

		if triggerFound {
			if trigger.CallbackFnType.ReturnType == nil {
				panic("trigger return type is <nil>")
			}

			if err := self.TypeCheck(
				callbackFnType.SetSpan(callbackFn.IdentSpan),
				trigger.CallbackFnType.SetSpanAdvanced(ann.TriggerSource.Span(), ann.TriggerSource.Span()),
				TypeCheckOptions{
					AllowFunctionTypes:          true,
					IgnoreFnParamNameMismatches: true,
				},
			); err != nil {
				self.diagnostics = append(
					self.diagnostics,
					err.GotDiagnostic.WithContext("Regarding callback function"),
				)
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

		callbackFn.Used = true

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
