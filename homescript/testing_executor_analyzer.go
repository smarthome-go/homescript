package homescript

import (
	"errors"
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	herrors "github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Analyzer
//

func TestingAnalyzerScopeAdditions() map[string]analyzer.Variable {
	return map[string]analyzer.Variable{
		"log": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
					ast.NewFunctionTypeParam(
						pAst.NewSpannedIdent("base", herrors.Span{}),
						ast.NewFloatType(herrors.Span{}), nil,
					),
					ast.NewFunctionTypeParam(
						pAst.NewSpannedIdent("value", herrors.Span{}),
						ast.NewFloatType(herrors.Span{}), nil,
					),
				}),
				herrors.Span{},
				ast.NewFloatType(herrors.Span{}),
				herrors.Span{},
			),
		),
		"print": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewVarArgsFunctionTypeParamKind([]ast.Type{}, ast.NewUnknownType()),
				herrors.Span{},
				ast.NewNullType(herrors.Span{}),
				herrors.Span{},
			),
		),
		"println": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewVarArgsFunctionTypeParamKind([]ast.Type{}, ast.NewUnknownType()),
				herrors.Span{},
				ast.NewNullType(herrors.Span{}),
				herrors.Span{},
			),
		),
		"fmt": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewVarArgsFunctionTypeParamKind([]ast.Type{ast.NewStringType(herrors.Span{})}, ast.NewUnknownType()),
				herrors.Span{},
				ast.NewStringType(herrors.Span{}),
				herrors.Span{},
			),
		),
		"time": analyzer.NewBuiltinVar(
			ast.NewObjectType(
				[]ast.ObjectTypeField{
					ast.NewObjectTypeField(
						pAst.NewSpannedIdent("sleep", herrors.Span{}),
						ast.NewFunctionType(
							ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
								ast.NewFunctionTypeParam(pAst.NewSpannedIdent("seconds", herrors.Span{}), ast.NewFloatType(herrors.Span{}), nil),
							}),
							herrors.Span{},
							ast.NewNullType(herrors.Span{}),
							herrors.Span{},
						),
						herrors.Span{},
					),
					ast.NewObjectTypeField(
						pAst.NewSpannedIdent("now", herrors.Span{}),
						ast.NewFunctionType(
							ast.NewNormalFunctionTypeParamKind(make([]ast.FunctionTypeParam, 0)),
							herrors.Span{},
							timeObjType(herrors.Span{}),
							herrors.Span{},
						),
						herrors.Span{},
					),
					ast.NewObjectTypeField(
						pAst.NewSpannedIdent("add_days", herrors.Span{}),
						ast.NewFunctionType(
							ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
								ast.NewFunctionTypeParam(pAst.NewSpannedIdent("time", herrors.Span{}), timeObjType(herrors.Span{}), nil),
								ast.NewFunctionTypeParam(pAst.NewSpannedIdent("days", herrors.Span{}), ast.NewIntType(herrors.Span{}), nil),
							}),
							herrors.Span{},
							timeObjType(herrors.Span{}),
							herrors.Span{},
						),
						herrors.Span{},
					),
				},
				herrors.Span{},
			),
		),
		"debug": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewVarArgsFunctionTypeParamKind([]ast.Type{}, ast.NewUnknownType()),
				herrors.Span{},
				ast.NewNullType(herrors.Span{}),
				herrors.Span{},
			),
		),
		"assert": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
					{
						Name: pAst.NewSpannedIdent("t", herrors.Span{}),
						Type: ast.NewBoolType(herrors.Span{}),
					},
				}),
				herrors.Span{},
				ast.NewNullType(herrors.Span{}),
				herrors.Span{},
			),
		),
		// "assert_eq": analyzer.NewBuiltinVar(
		// 	ast.NewFunctionType(
		// 		ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
		// 			{
		// 				Name: pAst.NewSpannedIdent("l", herrors.Span{}),
		// 				Type: ast.NewUnknownType(),
		// 			},
		// 			{
		// 				Name: pAst.NewSpannedIdent("r", herrors.Span{}),
		// 				Type: ast.NewUnknownType(),
		// 			},
		// 		}),
		// 		herrors.Span{},
		// 		ast.NewNullType(herrors.Span{}),
		// 		herrors.Span{},
		// 	),
		// ),
	}
}

type TestingAnalyzerHost struct {
	IsInvokedInTests bool
}

func (TestingAnalyzerHost) GetKnownObjectTypeFieldAnnotations() []string {
	return []string{}
}

func (self TestingAnalyzerHost) PostValidationHook(analyzedModules map[string]ast.AnalyzedProgram, mainModule string, _ *analyzer.Analyzer, _ bool) []diagnostic.Diagnostic {
	fmt.Println("=== Begin functions ===")

	for _, fn := range analyzedModules[mainModule].Functions {
		annotations := "NIL"

		if fn.Annotation != nil {
			annotations = spew.Sdump(*fn.Annotation)
		}

		fmt.Printf("function `%s` with annotations: %s\n", fn.Ident, annotations)
	}

	fmt.Println("=== END functions ===")

	return nil
}

func (self TestingAnalyzerHost) ResolveCodeModule(moduleName string) (code string, moduleFound bool, err error) {
	var path string

	if self.IsInvokedInTests {
		path = fmt.Sprintf("../tests/%s.hms", moduleName)
	} else {
		path = fmt.Sprintf("%s.hms", moduleName)
	}

	file, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}

	return string(file), true, nil
}

func (self TestingAnalyzerHost) GetBuiltinImport(
	moduleName string,
	valueName string,
	span herrors.Span,
	kind pAst.IMPORT_KIND,
) (res analyzer.BuiltinImport, moduleFound bool, valueFound bool) {
	switch moduleName {
	case "net":
		newHttpResponse := func() ast.Type {
			return ast.NewObjectType(
				[]ast.ObjectTypeField{
					ast.NewObjectTypeField(pAst.NewSpannedIdent("status", span), ast.NewStringType(span), span),
					ast.NewObjectTypeField(pAst.NewSpannedIdent("status_code", span), ast.NewIntType(span), span),
					ast.NewObjectTypeField(pAst.NewSpannedIdent("body", span), ast.NewStringType(span), span),
					ast.NewObjectTypeField(pAst.NewSpannedIdent("cookies", span), ast.NewAnyObjectType(span), span),
				},
				span,
			)
		}

		switch valueName {
		case "ping":
			return analyzer.BuiltinImport{
				Type: ast.NewFunctionType(
					ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
						ast.NewFunctionTypeParam(pAst.NewSpannedIdent("ip", span), ast.NewStringType(span), nil),
						ast.NewFunctionTypeParam(pAst.NewSpannedIdent("timeout", span), ast.NewFloatType(span), nil),
					}),
					span,
					ast.NewBoolType(span),
					span,
				),
				Template: &ast.TemplateSpec{},
			}, true, true
		case "HttpResponse":
			return analyzer.BuiltinImport{
				Type:     newHttpResponse(),
				Template: &ast.TemplateSpec{},
			}, true, true
		case "http":
			return analyzer.BuiltinImport{
				Type: ast.NewObjectType([]ast.ObjectTypeField{
					ast.NewObjectTypeField(pAst.NewSpannedIdent("get", span), ast.NewFunctionType(
						ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{ast.NewFunctionTypeParam(pAst.NewSpannedIdent("url", span), ast.NewStringType(span), nil)}),
						span,
						newHttpResponse(),
						span,
					), span),
					ast.NewObjectTypeField(pAst.NewSpannedIdent("generic", span), ast.NewFunctionType(
						ast.NewNormalFunctionTypeParamKind(
							[]ast.FunctionTypeParam{
								ast.NewFunctionTypeParam(pAst.NewSpannedIdent("url", span), ast.NewStringType(span), nil),
								ast.NewFunctionTypeParam(pAst.NewSpannedIdent("method", span), ast.NewStringType(span), nil),
								ast.NewFunctionTypeParam(pAst.NewSpannedIdent("body", span), ast.NewOptionType(ast.NewStringType(span), span), nil),
								ast.NewFunctionTypeParam(pAst.NewSpannedIdent("headers", span), ast.NewAnyObjectType(span), nil),
								ast.NewFunctionTypeParam(pAst.NewSpannedIdent("cookies", span), ast.NewAnyObjectType(span), nil),
							},
						),
						span,
						newHttpResponse(),
						span,
					), span),
				}, span),
				Template: &ast.TemplateSpec{},
			}, true, true
		default:
			return analyzer.BuiltinImport{}, true, false
		}
	case "triggers":
		if kind != pAst.IMPORT_KIND_TRIGGER {
			return analyzer.BuiltinImport{}, true, false
		}

		switch valueName {
		case "minute":
			return analyzer.BuiltinImport{
				Trigger: &analyzer.TriggerFunction{
					TriggerFnType: ast.NewFunctionType(
						ast.NewNormalFunctionTypeParamKind(
							[]ast.FunctionTypeParam{ast.NewFunctionTypeParam(
								pAst.NewSpannedIdent("minutes", span),
								ast.NewIntType(span),
								nil,
							)},
						),
						span,
						ast.NewNullType(span),
						span,
					).(ast.FunctionType),
					CallbackFnType: ast.NewFunctionType(
						ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
							ast.NewFunctionTypeParam(
								pAst.NewSpannedIdent("elapsed", span),
								ast.NewIntType(span),
								nil,
							),
						}),
						span,
						ast.NewNullType(span),
						span,
					).(ast.FunctionType),
					Connective: pAst.AtTriggerDispatchKeyword,
					ImportedAt: span,
				},
				Type:     nil,
				Template: nil,
			}, true, true
		default:
			return analyzer.BuiltinImport{}, true, false
		}
	case "testing":
		if kind != pAst.IMPORT_KIND_NORMAL {
			return analyzer.BuiltinImport{}, true, false
		}

		switch valueName {
		case "assert_eq":
			return analyzer.BuiltinImport{
					Type: ast.NewFunctionType(
						ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
							ast.NewFunctionTypeParam(pAst.NewSpannedIdent("lhs", herrors.Span{}), ast.NewUnknownType(), nil),
							ast.NewFunctionTypeParam(pAst.NewSpannedIdent("rhs", herrors.Span{}), ast.NewUnknownType(), nil),
						}),
						herrors.Span{},
						ast.NewNullType(herrors.Span{}),
						herrors.Span{},
					),
					Template: nil,
				},
				true, true
		case "any_func":
			return analyzer.BuiltinImport{
					Type: ast.NewFunctionType(
						ast.NewNormalFunctionTypeParamKind(make([]ast.FunctionTypeParam, 0)),
						span,
						ast.NewAnyType(span),
						span,
					),
					Template: nil,
				},
				true, true
		case "any_list":
			return analyzer.BuiltinImport{
					Type:     ast.NewListType(ast.NewAnyType(span), span),
					Template: nil,
				},
				true, true
		}
		return analyzer.BuiltinImport{}, false, false
	case "templates":
		if kind != pAst.IMPORT_KIND_TEMPLATE {
			return analyzer.BuiltinImport{}, true, false
		}

		switch valueName {
		case "FooFeature":
			return analyzer.BuiltinImport{
				Type: nil,
				Template: &ast.TemplateSpec{
					BaseMethods: map[string]ast.TemplateMethod{
						"dim": {
							Signature: ast.FunctionType{
								Params: ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
									{
										Name:                 pAst.NewSpannedIdent("percent", span),
										Type:                 ast.NewIntType(span),
										IsSingletonExtractor: false,
										SingletonIdent:       "",
									},
								}),
								ParamsSpan: span,
								ReturnType: ast.NewBoolType(span),
								Range:      span,
							},
							Modifier: pAst.FN_MODIFIER_NONE,
						},
						"set_temp": {
							Signature: ast.FunctionType{
								Params: ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
									{
										Name:                 pAst.NewSpannedIdent("celsius", span),
										Type:                 ast.NewFloatType(span),
										IsSingletonExtractor: false,
										SingletonIdent:       "",
									},
								}),
								ParamsSpan: span,
								ReturnType: ast.NewNullType(span),
								Range:      span,
							},
							Modifier: pAst.FN_MODIFIER_NONE,
						},
					},
					Capabilities: map[string]ast.TemplateCapability{
						"light": {
							RequiresMethods: []string{"dim"},
							ConflictsWithCapabilities: []ast.TemplateConflict{
								{
									ConflictingCapability: "temperature",
									ConflictReason:        "",
								},
							},
						},
						"temperature": {
							RequiresMethods: []string{"set_temp"},
							ConflictsWithCapabilities: []ast.TemplateConflict{
								{
									ConflictingCapability: "light",
									ConflictReason:        "",
								},
							},
						},
					},
					DefaultCapabilities: []string{},
					Span:                span,
				},
			}, true, true
		default:
			return analyzer.BuiltinImport{}, true, false
		}
	default:
		return analyzer.BuiltinImport{}, false, false

	}
}

func timeObjType(span herrors.Span) ast.Type {
	return ast.NewObjectType(
		[]ast.ObjectTypeField{
			ast.NewObjectTypeField(pAst.NewSpannedIdent("year", span), ast.NewIntType(span), span),
			ast.NewObjectTypeField(pAst.NewSpannedIdent("month", span), ast.NewIntType(span), span),
			ast.NewObjectTypeField(pAst.NewSpannedIdent("year_day", span), ast.NewIntType(span), span),
			ast.NewObjectTypeField(pAst.NewSpannedIdent("hour", span), ast.NewIntType(span), span),
			ast.NewObjectTypeField(pAst.NewSpannedIdent("minute", span), ast.NewIntType(span), span),
			ast.NewObjectTypeField(pAst.NewSpannedIdent("second", span), ast.NewIntType(span), span),
			ast.NewObjectTypeField(pAst.NewSpannedIdent("month_day", span), ast.NewIntType(span), span),
			ast.NewObjectTypeField(pAst.NewSpannedIdent("week_day", span), ast.NewIntType(span), span),
			ast.NewObjectTypeField(pAst.NewSpannedIdent("unix_milli", span), ast.NewIntType(span), span),
		},
		span,
	)
}
