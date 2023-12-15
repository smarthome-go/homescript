package homescript

import (
	"errors"
	"fmt"
	"os"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	herrors "github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Analyzer
//

func TestingAnalyzerScopeAdditions() map[string]analyzer.Variable {
	return map[string]analyzer.Variable{
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
		"assert_eq": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
					{
						Name: pAst.NewSpannedIdent("l", herrors.Span{}),
						Type: ast.NewUnknownType(),
					},
					{
						Name: pAst.NewSpannedIdent("r", herrors.Span{}),
						Type: ast.NewUnknownType(),
					},
				}),
				herrors.Span{},
				ast.NewNullType(herrors.Span{}),
				herrors.Span{},
			),
		),
	}
}

type TestingAnalyzerHost struct{}

func (self TestingAnalyzerHost) ResolveCodeModule(moduleName string) (code string, moduleFound bool, err error) {
	path := fmt.Sprintf("../tests/%s.hms", moduleName)

	file, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}

	return string(file), true, nil
}

func (self TestingAnalyzerHost) GetBuiltinImport(moduleName string, valueName string, span herrors.Span) (valueType ast.Type, moduleFound bool, valueFound bool) {
	switch moduleName {
	case "testing":
		switch valueName {
		case "assert_eq":
			return ast.NewFunctionType(
				ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
					ast.NewFunctionTypeParam(pAst.NewSpannedIdent("lhs", herrors.Span{}), ast.NewUnknownType()),
					ast.NewFunctionTypeParam(pAst.NewSpannedIdent("rhs", herrors.Span{}), ast.NewUnknownType()),
				}),
				herrors.Span{},
				ast.NewNullType(herrors.Span{}),
				herrors.Span{},
			), true, true
		case "any_func":
			return ast.NewFunctionType(
				ast.NewNormalFunctionTypeParamKind(make([]ast.FunctionTypeParam, 0)),
				span,
				ast.NewAnyType(span),
				span,
			), true, true
		case "any_list":
			return ast.NewListType(ast.NewAnyType(span), span), true, true
		}
		return nil, false, false
	default:
		return nil, false, false

	}
}
