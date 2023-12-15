package homescript

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	herrors "github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

//
// Tree-walking interpreter scope additions
//

func TestingInterpeterScopeAdditions() map[string]value.Value {
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
// Tree-walking interpreter
//

type TestingTreeExecutor struct {
	Output *string
}

func (self TestingTreeExecutor) GetUser() string { return "<unknown>" }

func (self TestingTreeExecutor) GetBuiltinImport(moduleName string, toImport string) (val value.Value, found bool) {
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
func (self TestingTreeExecutor) ResolveModuleCode(moduleName string) (code string, found bool, err error) {
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

func (self TestingTreeExecutor) WriteStringTo(input string) error {
	*self.Output += input
	fmt.Print(input)
	return nil
}
