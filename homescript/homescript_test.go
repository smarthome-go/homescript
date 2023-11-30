package homescript

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	herrors "github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
	"github.com/smarthome-go/homescript/v3/homescript/runtime"
	"github.com/stretchr/testify/assert"
)

//
// Analyzer
//

func analyzerScopeAdditions() map[string]analyzer.Variable {
	return make(map[string]analyzer.Variable)
}

type analyzerHost struct{}

func (self analyzerHost) ResolveCodeModule(moduleName string) (code string, moduleFound bool, err error) {
	path := fmt.Sprintf("%s.hms", moduleName)

	file, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}

	return string(file), true, nil
}

func (self analyzerHost) GetBuiltinImport(moduleName string, valueName string, span herrors.Span) (valueType ast.Type, moduleFound bool, valueFound bool) {
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

//
// Interpreter
//

func interpreterScopeAdditions() map[string]value.Value {
	return make(map[string]value.Value)
}

type executor struct{}

func (self executor) GetUser() string { return "<unknown>" }

func (self executor) GetBuiltinImport(moduleName string, toImport string) (val value.Value, found bool) {
	switch moduleName {
	case "testing":
		switch toImport {
		case "assert_eq":
			return *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
				lhsDisp, i := args[0].Display()
				if i != nil {
					return nil, i
				}
				rhsDisp, i := args[1].Display()
				if i != nil {
					return nil, i
				}

				if args[0].Kind() != args[1].Kind() {
					return nil, value.NewThrowInterrupt(span, fmt.Sprintf("`%s` is not equal to `%s`", lhsDisp, rhsDisp))
				}

				isEqual, i := args[0].IsEqual(args[1])
				if i != nil {
					return nil, i
				}

				if !isEqual {
					return nil, value.NewThrowInterrupt(span, fmt.Sprintf("`%s` is not equal to `%s`", lhsDisp, rhsDisp))
				}

				return value.NewValueNull(), nil
			}), true
		case "any_func":
			return *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
				return value.NewValueInt(42), nil
			}), true
		case "any_list":
			return *value.NewValueList([]*value.Value{value.NewValueString("Test")}), true
		}
		return nil, false
	default:
		return nil, false

	}
}

func (self executor) ResolveModuleCode(moduleName string) (code string, found bool, err error) {
	return "", false, nil
}

func (self executor) WriteStringTo(input string) error {
	fmt.Print(input)
	return nil
}

//
// Tests
//

type Test struct {
	Name           string
	Path           string
	ExpectedOutput string
}

func TestScripts(t *testing.T) {
	tests := []Test{
		{
			Name:           "TestAllBuiltinMembers",
			Path:           "../tests/builtin_members.hms",
			ExpectedOutput: "",
		},
		{
			Name:           "TestAllNormalCasts",
			Path:           "../tests/normal_casts.hms",
			ExpectedOutput: "",
		},
		{
			Name:           "TestAllAnyCasts",
			Path:           "../tests/any_casts.hms",
			ExpectedOutput: "",
		},
		{
			Name:           "TestSringConversion",
			Path:           "../tests/string_conversion.hms",
			ExpectedOutput: "",
		},
		{
			Name:           "TestStatements",
			Path:           "../tests/statements.hms",
			ExpectedOutput: "",
		},
	}

	for idx, test := range tests {
		t.Run(fmt.Sprintf("program-%d-%s", idx, test.Name), func(t *testing.T) {
			runScript(test.Path, t)
		})
	}
}

func runScript(path string, t *testing.T) {
	code, err := os.ReadFile(path)
	if err != nil {
		t.Error(err.Error())
		return
	}

	modules, diagnostics, syntax := Analyze(
		InputProgram{
			Filename:    path,
			ProgramText: string(code),
		}, analyzerScopeAdditions(), analyzerHost{})
	hasErr := false
	if len(syntax) > 0 {
		for _, s := range syntax {
			file, err := os.ReadFile(s.Span.Filename)
			assert.NoError(t, err)
			t.Error(s.Display(string(file)))
		}
		hasErr = true
	}

	for _, d := range diagnostics {
		file, err := os.ReadFile(d.Span.Filename)
		assert.NoError(t, err)

		t.Error(d.Display(string(file)))
		if d.Level == diagnostic.DiagnosticLevelError {
			hasErr = true
		}
	}

	if hasErr {
		return
	}

	compiler := compiler.NewCompiler()
	compiled := compiler.Compile(modules)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	vm := runtime.NewVM(compiled, executor{}, os.Args[2] == "1", &ctx, &cancel, interpreterScopeAdditions())

	vm.Spawn(compiled.EntryPoints[path])
	if coreNum, i := vm.Wait(); i != nil {
		i := *i

		d := diagnostic.Diagnostic{
			Level:   diagnostic.DiagnosticLevelError,
			Message: i.Message(),
			Notes:   []string{fmt.Sprintf("Exception occurred on core %d", coreNum)},
			Span:    i.GetSpan(), // this call might panic
		}

		fmt.Printf("Reading: %s...\n", i.GetSpan().Filename)

		file, err := os.ReadFile(fmt.Sprintf("%s.hms", i.GetSpan().Filename))
		if err != nil {
			panic(fmt.Sprintf("Could not read file `%s`: %s\n", i.GetSpan().Filename, err.Error()))
		}

		fmt.Printf("%s\n", d.Display(string(file)))
		panic(fmt.Sprintf("Core %d crashed", coreNum))
	}
}
