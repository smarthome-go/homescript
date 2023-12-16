package homescript

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
		"time": *value.NewValueObject(map[string]*value.Value{
			"sleep": value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
				durationSecs := args[0].(value.ValueFloat).Inner

				for i := 0; i < int(durationSecs*1000); i += 10 {
					if i := checkCancelationTree(cancelCtx, span); i != nil {
						return nil, i
					}
					time.Sleep(time.Millisecond * 10)
				}

				return nil, nil
			},
			),
			"add_days": value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
				base := createTimeStructFromObjectTree(args[0])
				days := args[1].(value.ValueInt).Inner
				return createTimeObjectTree(base.Add(time.Hour * 24 * time.Duration(days))), nil
			}),
			"now": value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
				now := time.Now()

				return createTimeObjectTree(now), nil
			}),
		}),
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

func createTimeObjectTree(t time.Time) *value.Value {
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

func createTimeStructFromObjectTree(t value.Value) time.Time {
	tObj := t.(value.ValueObject)
	fields, i := tObj.Fields()
	if i != nil {
		panic(i)
	}
	millis := (*fields["unix_milli"]).(value.ValueInt).Inner
	return time.UnixMilli(millis)
}

func checkCancelationTree(ctx *context.Context, span herrors.Span) *value.Interrupt {
	select {
	case <-(*ctx).Done():
		return value.NewTerminationInterrupt((*ctx).Err().Error(), span)
	default:
		// do nothing, this should not block the entire interpreter
		return nil
	}
}
