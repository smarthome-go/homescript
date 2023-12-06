package homescript

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	vmValue "github.com/smarthome-go/homescript/v3/homescript/runtime/value"
	"github.com/stretchr/testify/assert"
)

//
// Analyzer
//

func analyzerScopeAdditions() map[string]analyzer.Variable {
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

type analyzerHost struct{}

func (self analyzerHost) ResolveCodeModule(moduleName string) (code string, moduleFound bool, err error) {
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

func interpeterScopeAdditions() map[string]value.Value {
	return map[string]value.Value{
		"print": *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
			output := make([]string, 0)
			for _, arg := range args {
				disp, i := arg.Display()
				if i != nil {
					return nil, i
				}
				output = append(output, disp)
			}

			outStr := strings.Join(output, " ")

			if err := executor.WriteStringTo(outStr); err != nil {
				return nil, value.NewRuntimeErr(
					err.Error(),
					value.HostErrorKind,
					span,
				)
			}

			return value.NewValueNull(), nil
		},
		),
		"println": *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
			output := make([]string, 0)
			for _, arg := range args {
				disp, i := arg.Display()
				if i != nil {
					return nil, i
				}
				output = append(output, disp)
			}

			outStr := strings.Join(output, " ") + "\n"

			if err := executor.WriteStringTo(outStr); err != nil {
				return nil, value.NewRuntimeErr(
					err.Error(),
					value.HostErrorKind,
					span,
				)
			}

			return value.NewValueNull(), nil
		},
		),
		"debug": *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
			output := make([]string, 0)
			for _, arg := range args {
				disp, i := arg.Display()
				if i != nil {
					return nil, i
				}
				output = append(output, disp)
			}

			outStr := "DEBUG: " + strings.Join(output, " ") + "\n"

			if err := executor.WriteStringTo(outStr); err != nil {
				return nil, value.NewRuntimeErr(
					err.Error(),
					value.HostErrorKind,
					span,
				)
			}

			return value.NewValueNull(), nil
		},
		),
		"assert": *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
			if !args[0].(value.ValueBool).Inner {
				return nil, value.NewRuntimeErr(
					"Assert failed",
					value.HostErrorKind,
					span,
				)
			}
			return value.NewValueNull(), nil
		}),
		"assert_eq": *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
			eq, i := args[0].IsEqual(args[1])
			if i != nil {
				return nil, i
			}

			if !eq {
				a, i := args[0].Display()
				if i != nil {
					return nil, i
				}

				b, i := args[1].Display()
				if i != nil {
					return nil, i
				}

				return nil, value.NewRuntimeErr(
					fmt.Sprintf("Assertion failed: `%s` is not equal to `%s`", a, b),
					value.HostErrorKind,
					span,
				)
			}
			return value.NewValueNull(), nil
		}),
	}
}

//
// Vm
//

func vmiScopeAdditions() map[string]vmValue.Value {
	return map[string]vmValue.Value{
		"print": *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
			output := make([]string, 0)
			for _, arg := range args {
				disp, i := arg.Display()
				if i != nil {
					return nil, i
				}
				output = append(output, disp)
			}

			outStr := strings.Join(output, " ")

			if err := executor.WriteStringTo(outStr); err != nil {
				return nil, vmValue.NewVMFatalException(
					err.Error(),
					vmValue.Vm_HostErrorKind,
					span,
				)
			}

			return vmValue.NewValueNull(), nil
		},
		),
		"println": *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
			output := make([]string, 0)
			for _, arg := range args {
				disp, i := arg.Display()
				if i != nil {
					return nil, i
				}
				output = append(output, disp)
			}

			outStr := strings.Join(output, " ") + "\n"

			if err := executor.WriteStringTo(outStr); err != nil {
				return nil, vmValue.NewVMFatalException(
					err.Error(),
					vmValue.Vm_HostErrorKind,
					span,
				)
			}

			return vmValue.NewValueNull(), nil
		},
		),
		"debug": *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
			output := make([]string, 0)
			for _, arg := range args {
				disp, i := arg.Display()
				if i != nil {
					return nil, i
				}
				output = append(output, disp)
			}

			outStr := "DEBUG: " + strings.Join(output, " ") + "\n"

			if err := executor.WriteStringTo(outStr); err != nil {
				return nil, vmValue.NewVMFatalException(
					err.Error(),
					vmValue.Vm_HostErrorKind,
					span,
				)
			}

			return vmValue.NewValueNull(), nil
		},
		),
		"assert": *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
			if !args[0].(vmValue.ValueBool).Inner {
				return nil, vmValue.NewVMFatalException(
					"Assert failed",
					vmValue.Vm_HostErrorKind,
					span,
				)
			}
			return vmValue.NewValueNull(), nil
		}),
		"assert_eq": *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
			if args[0].Kind() != args[1].Kind() {
				a, i := args[0].Display()
				if i != nil {
					return nil, i
				}

				b, i := args[1].Display()
				if i != nil {
					return nil, i
				}
				return nil, vmValue.NewVMThrowInterrupt(
					span,
					fmt.Sprintf("Assertion failed: `%s` is not equal to `%s`", a, b),
				)
			}

			eq, i := args[0].IsEqual(args[1])
			if i != nil {
				return nil, i
			}

			if !eq {
				a, i := args[0].Display()
				if i != nil {
					return nil, i
				}

				b, i := args[1].Display()
				if i != nil {
					return nil, i
				}

				return nil, vmValue.NewVMThrowInterrupt(
					span,
					fmt.Sprintf("Assertion failed: `%s` is not equal to `%s`", a, b),
				)
			}
			return vmValue.NewValueNull(), nil
		}),
	}
}

//
// Tree-walking interpreter
//

type treeExecutor struct{}

func (self treeExecutor) GetUser() string { return "<unknown>" }

func (self treeExecutor) GetBuiltinImport(moduleName string, toImport string) (val value.Value, found bool) {
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

// returns the Homescript code of the requested module
func (self treeExecutor) ResolveModuleCode(moduleName string) (code string, found bool, err error) {
	path := fmt.Sprintf("tests/%s.hms", moduleName)

	file, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}

	return string(file), true, nil
}

func (self treeExecutor) WriteStringTo(input string) error {
	fmt.Print(input)
	return nil
}

//
// Vm interpreter
//

type vmExecutor struct{}

func (self vmExecutor) GetUser() string { return "<unknown>" }

func (self vmExecutor) GetBuiltinImport(moduleName string, toImport string) (val vmValue.Value, found bool) {
	switch moduleName {
	case "testing":
		switch toImport {
		case "assert_eq":
			return *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
				lhsDisp, i := args[0].Display()
				if i != nil {
					return nil, i
				}
				rhsDisp, i := args[1].Display()
				if i != nil {
					return nil, i
				}

				if args[0].Kind() != args[1].Kind() {
					return nil, vmValue.NewVMThrowInterrupt(span, fmt.Sprintf("`%s` is not equal to `%s`", lhsDisp, rhsDisp))
				}

				isEqual, i := args[0].IsEqual(args[1])
				if i != nil {
					return nil, i
				}

				if args[0].Kind() != args[1].Kind() {
					return nil, vmValue.NewVMThrowInterrupt(span, fmt.Sprintf("`%s` is not equal to `%s`", lhsDisp, rhsDisp))
				}
				if !isEqual {
					return nil, vmValue.NewVMThrowInterrupt(span, fmt.Sprintf("`%s` is not equal to `%s`", lhsDisp, rhsDisp))
				}

				return vmValue.NewValueNull(), nil
			}), true
		case "any_func":
			return *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
				return vmValue.NewValueInt(42), nil
			}), true
		case "any_list":
			return *vmValue.NewValueList([]*vmValue.Value{vmValue.NewValueString("Test")}), true
		}
		return nil, false
	default:
		return nil, false

	}
}

// returns the Homescript code of the requested module
func (self vmExecutor) ResolveModuleCode(moduleName string) (code string, found bool, err error) {
	path := fmt.Sprintf("tests/%s.hms", moduleName)

	file, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}

	return string(file), true, nil
}

func (self vmExecutor) WriteStringTo(input string) error {
	fmt.Print(input)
	return nil
}

//
// Tests
//

type Test struct {
	Name  string
	Path  string
	Debug bool
}

func TestScripts(t *testing.T) {
	tests := []Test{
		{
			Name:  "TestAllBuiltinMembers",
			Path:  "../tests/builtin_members.hms",
			Debug: false,
		},
		{
			Name:  "TestAllNormalCasts",
			Path:  "../tests/normal_casts.hms",
			Debug: false,
		},
		{
			Name:  "TestAllAnyCasts",
			Path:  "../tests/any_casts.hms",
			Debug: false,
		},
		{
			Name:  "TestSringConversion",
			Path:  "../tests/string_conversion.hms",
			Debug: false,
		},
		{
			Name:  "TestStatements",
			Path:  "../tests/statements.hms",
			Debug: false,
		},
	}

	files, err := filepath.Glob("../tests/*.hms")
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		for _, test := range tests {
			if test.Path == file {
				continue
			}
		}

		split := strings.Split(file, "/")
		tests = append(tests, Test{
			Name:  split[len(split)-1],
			Path:  file,
			Debug: false,
		})
	}

	for idx, test := range tests {
		t.Run(fmt.Sprintf("program-%d-%s", idx, test.Name), func(t *testing.T) {
			runScript(test.Path, test.Debug, t)
		})
	}
}

func runScript(path string, debug bool, t *testing.T) {
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

	if debug {
		for _, d := range diagnostics {
			file, err := os.ReadFile(d.Span.Filename)
			assert.NoError(t, err)

			t.Error(d.Display(string(file)))
			if d.Level == diagnostic.DiagnosticLevelError {
				hasErr = true
			}
		}
	} else if len(diagnostics) > 0 {
		fmt.Printf("Program `%s` generated %d diagnostics.\n", t.Name(), len(diagnostics))
	}

	if hasErr {
		return
	}

	compiler := compiler.NewCompiler()
	compiled := compiler.Compile(modules, path)

	if debug {
		i := 0
		for name, function := range compiled.Functions {
			fmt.Printf("%03d ===> func: %s\n", i, name)

			for idx, inst := range function {
				fmt.Printf("%03d | %s\n", idx, inst)
			}

			i++
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	vm := runtime.NewVM(compiled, vmExecutor{}, debug, &ctx, &cancel, vmiScopeAdditions(), runtime.CoreLimits{
		CallStackMaxSize: 10024,
		StackMaxSize:     10024,
		MaxMemorySize:    10024,
	})

	vm.Spawn(compiled.EntryPoint)
	if coreNum, i := vm.Wait(); i != nil {
		i := *i

		d := diagnostic.Diagnostic{
			Level:   diagnostic.DiagnosticLevelError,
			Message: i.Message(),
			Notes:   []string{fmt.Sprintf("Exception occurred on core %d", coreNum)},
			Span:    i.GetSpan(), // this call might panic
		}

		fmt.Printf("Reading: %s...\n", i.GetSpan().Filename)

		file, err := os.ReadFile(i.GetSpan().Filename)
		if err != nil {
			panic(fmt.Sprintf("Could not read file `%s`: %s\n", i.GetSpan().Filename, err.Error()))
		}

		fmt.Printf("%s\n", d.Display(string(file)))
		panic(fmt.Sprintf("Core %d crashed", coreNum))
	}
}
