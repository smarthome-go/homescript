package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	syntaxErrors "github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
	"github.com/smarthome-go/homescript/v3/homescript/runtime"
)

type Executor struct {
}

func (self Executor) GetUser() string { return "<unknown>" }

func createTimeObject(t time.Time) *value.Value {
	return value.NewValueObject(
		map[string]*value.Value{
			"year":       value.NewValueInt(int64(t.Year())),
			"month":      value.NewValueInt(int64(t.Month())),
			"year_day":   value.NewValueInt(int64(t.YearDay())),
			"hour":       value.NewValueInt(int64(t.Hour())),
			"minute":     value.NewValueInt(int64(t.Minute())),
			"second":     value.NewValueInt(int64(t.Second())),
			"month_day":  value.NewValueInt(int64(t.Day())),
			"week_day":   value.NewValueInt(int64(t.Weekday())),
			"unix_milli": value.NewValueInt(t.UnixMilli()),
		},
	)
}

func createTimeStructFromObject(t value.Value) time.Time {
	tObj := t.(value.ValueObject)
	fields, i := tObj.Fields()
	if i != nil {
		panic(i)
	}
	millis := (*fields["unix_milli"]).(value.ValueInt).Inner
	return time.UnixMilli(millis)
}

func checkCancelation(ctx *context.Context, span syntaxErrors.Span) *value.Interrupt {
	select {
	case <-(*ctx).Done():
		return value.NewTerminationInterrupt((*ctx).Err().Error(), span)
	default:
		// do nothing, this should not block the entire interpreter
		return nil
	}
}

func (self Executor) GetBuiltinImport(moduleName string, toImport string) (val value.Value, found bool) {
	if moduleName != "sys" {
		return nil, false
	}

	switch toImport {
	case "any_list":
		return *value.NewValueList([]*value.Value{
			value.NewValueInt(42),
		}), true
	case "any_list2":
		return *value.NewValueList([]*value.Value{
			value.NewValueList([]*value.Value{value.NewValueString("Hello World")}),
		}), true
	case "any_func":
		return *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span syntaxErrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
			return value.NewValueInt(42), nil
		}), true
	case "time":
		return *value.NewValueObject(map[string]*value.Value{
			"sleep": value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span syntaxErrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
				durationSecs := args[0].(value.ValueFloat).Inner

				for i := 0; i < int(durationSecs*1000); i += 10 {
					if i := checkCancelation(cancelCtx, span); i != nil {
						return nil, i
					}
					time.Sleep(time.Millisecond * 10)
				}

				return nil, nil
			},
			),
			"add_days": value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span syntaxErrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
				base := createTimeStructFromObject(args[0])
				days := args[1].(value.ValueInt).Inner
				return createTimeObject(base.Add(time.Hour * 24 * time.Duration(days))), nil
			}),
			"now": value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span syntaxErrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
				now := time.Now()

				return createTimeObject(now), nil
			}),
		}), true
	default:
		return nil, false

	}
}

// returns the Homescript code of the requested module
func (self Executor) ResolveModuleCode(moduleName string) (code string, found bool, err error) {
	return "", false, nil
}

func (self Executor) WriteStringTo(input string) error {
	fmt.Print(input)
	return nil
}

type Host struct {
}

func (self Host) ResolveCodeModule(moduleName string) (code string, moduleFound bool, err error) {
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

func (self Host) GetBuiltinImport(moduleName string, valueName string, span syntaxErrors.Span) (valueType ast.Type, moduleFound bool, valueFound bool) {
	if moduleName != "sys" {
		return nil, false, false
	}

	switch valueName {
	case "any_func":
		return ast.NewFunctionType(
			ast.NewNormalFunctionTypeParamKind(make([]ast.FunctionTypeParam, 0)),
			span,
			ast.NewAnyType(span),
			span,
		), true, true
	// case "obj_func":
	// 	return ast.NewFunctionType(
	// 		ast.NewNormalFunctionTypeParamKind(make([]ast.FunctionTypeParam, 0)),
	// 		span,
	// 		ast.NewObjectType([]ast.ObjectTypeField{ast.NewObjectTypeField(pAst.NewSpannedIdent("foo", span), ast.NewNumType(span), span)}, span),
	// 		span,
	// 	), true, true
	case "any_list":
		return ast.NewListType(ast.NewAnyType(span), span), true, true
	case "any_list2":
		return ast.NewListType(
			ast.NewListType(ast.NewAnyType(span), span), span), true, true
	case "obj_any":
		return ast.NewObjectType(
			[]ast.ObjectTypeField{
				ast.NewObjectTypeField(pAst.NewSpannedIdent("has_any", span), ast.NewAnyType(span), span),
			},
			span,
		), true, true
	case "just_any":
		return ast.NewAnyType(span), true, true
	// case "string":
	// 	return ast.NewStringType(span), true, true
	// case "object":
	// 	return ast.NewObjectType([]ast.ObjectTypeField{ast.NewObjectTypeField(pAst.NewSpannedIdent("foo", span), ast.NewNumType(span), span)}, span), true, true
	case "time":
		return ast.NewObjectType(
			[]ast.ObjectTypeField{
				ast.NewObjectTypeField(
					pAst.NewSpannedIdent("sleep", span),
					ast.NewFunctionType(
						ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
							ast.NewFunctionTypeParam(pAst.NewSpannedIdent("seconds", span), ast.NewIntType(span)),
						}),
						span,
						ast.NewNullType(span),
						span,
					),
					span,
				),
				ast.NewObjectTypeField(
					pAst.NewSpannedIdent("now", span),
					ast.NewFunctionType(
						ast.NewNormalFunctionTypeParamKind(make([]ast.FunctionTypeParam, 0)),
						span,
						timeObjType(span),
						span,
					),
					span,
				),
				ast.NewObjectTypeField(
					pAst.NewSpannedIdent("add_days", span),
					ast.NewFunctionType(
						ast.NewNormalFunctionTypeParamKind([]ast.FunctionTypeParam{
							ast.NewFunctionTypeParam(pAst.NewSpannedIdent("time", span), timeObjType(span)),
							ast.NewFunctionTypeParam(pAst.NewSpannedIdent("days", span), ast.NewIntType(span)),
						}),
						span,
						timeObjType(span),
						span,
					),
					span,
				),
			},
			span,
		), true, true
	default:
		return nil, true, false
	}
}

func timeObjType(span syntaxErrors.Span) ast.Type {
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

func scopeAdditions() map[string]analyzer.Variable {
	return map[string]analyzer.Variable{
		"print": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewVarArgsFunctionTypeParamKind([]ast.Type{}, ast.NewUnknownType()),
				syntaxErrors.Span{},
				ast.NewNullType(syntaxErrors.Span{}),
				syntaxErrors.Span{},
			),
		),
		"println": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewVarArgsFunctionTypeParamKind([]ast.Type{}, ast.NewUnknownType()),
				syntaxErrors.Span{},
				ast.NewNullType(syntaxErrors.Span{}),
				syntaxErrors.Span{},
			),
		),
		"debug": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewVarArgsFunctionTypeParamKind([]ast.Type{}, ast.NewUnknownType()),
				syntaxErrors.Span{},
				ast.NewNullType(syntaxErrors.Span{}),
				syntaxErrors.Span{},
			),
		),
		"assert": analyzer.NewBuiltinVar(
			ast.NewFunctionType(
				ast.NewVarArgsFunctionTypeParamKind([]ast.Type{}, ast.NewBoolType(syntaxErrors.Span{})),
				syntaxErrors.Span{},
				ast.NewNullType(syntaxErrors.Span{}),
				syntaxErrors.Span{},
			),
		),
	}
}

func iScopeAdditions() map[string]value.Value {
	return map[string]value.Value{
		"print": *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span syntaxErrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
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
		"println": *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span syntaxErrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
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
		"debug": *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span syntaxErrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
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
		"assert": *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span syntaxErrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
			if !args[0].(value.ValueBool).Inner {
				return nil, value.NewRuntimeErr(
					"Assert failed",
					value.HostErrorKind,
					span,
				)
			}
			return value.NewValueNull(), nil
		}),
	}
}

func main() {
	programRaw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err.Error())
	}
	program := string(programRaw)
	filename := strings.Split(os.Args[1], ".")[0]

	analyzed, diagnostics, syntaxErrors := homescript.Analyze(homescript.InputProgram{
		ProgramText: program,
		Filename:    filename,
	}, scopeAdditions(), Host{})

	if len(syntaxErrors) != 0 {
		for _, syntaxErr := range syntaxErrors {
			fmt.Printf("Reading: %s...\n", syntaxErr.Span.Filename)

			file, err := os.ReadFile(fmt.Sprintf("%s.hms", syntaxErr.Span.Filename))
			if err != nil {
				panic(err.Error())
			}

			fmt.Println(syntaxErr.Display(string(file)))
		}
		os.Exit(2)
	}

	abort := false
	fmt.Println("=== DIAGNOSTICS ===")
	for _, item := range diagnostics {
		if item.Level == diagnostic.DiagnosticLevelError {
			abort = true
		}

		fmt.Printf("Reading: %s...\n", item.Span.Filename)

		file, err := os.ReadFile(fmt.Sprintf("%s.hms", item.Span.Filename))
		if err != nil {
			panic(err.Error())
		}

		fmt.Println(item.Display(string(file)))
	}

	if abort {
		os.Exit(1)
	}

	fmt.Println("=== ANALYZED ===")
	for name, module := range analyzed {
		fmt.Printf("=== MODULE: %s ===\n", name)
		fmt.Println(module)
	}

	fmt.Println("=== COMPILED ===")

	compiler := compiler.NewCompiler()
	compiled := compiler.Compile(analyzed[filename])

	i := 0
	for name, function := range compiled.Functions {
		fmt.Printf("%03d ===> func: %s\n", i, name)

		for idx, inst := range function {
			fmt.Printf("%03d | %s\n", idx, inst)
		}

		i++
	}

	start := time.Now()
	vm := runtime.NewVM(compiled, Executor{})
	vm.Run("main0", os.Args[2] == "1")
	fmt.Printf("VM elapsed: %v\n", time.Since(start))

	// return

	fmt.Println("=== BEGIN INTERPRET ===")

	start = time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	blocking := make(chan struct{})
	go func() {
		defer func() { blocking <- struct{}{} }()

		if i := homescript.Run(
			20_000,
			analyzed,
			filename,
			Executor{},
			iScopeAdditions(),
			&ctx,
		); i != nil {
			switch (*i).Kind() {
			case value.RuntimeErrorInterruptKind:
				runtimErr := (*i).(value.RuntimeErr)
				program, err := os.ReadFile(fmt.Sprintf("%s.hms", runtimErr.Span.Filename))
				if err != nil {
					panic(err.Error())
				}

				fmt.Println(diagnostic.Diagnostic{
					Level:   diagnostic.DiagnosticLevelError,
					Message: fmt.Sprintf("%s: %s", runtimErr.ErrKind, runtimErr.MessageInternal),
					Span:    runtimErr.Span,
				}.Display(string(program)))
			default:
				fmt.Printf("%s: %s\n", (*i).Kind(), (*i).Message())
			}
		}

		fmt.Printf("Tree elapsed: %v\n", time.Since(start))
	}()

	<-blocking
	cancel()
}
