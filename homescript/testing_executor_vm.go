package homescript

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	herrors "github.com/smarthome-go/homescript/v3/homescript/errors"
	vmValue "github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

//
// Vm scope additions
//

func TestingVmScopeAdditions() map[string]vmValue.Value {
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
		"time": *vmValue.NewValueObject(map[string]*vmValue.Value{
			"sleep": vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
				durationSecs := args[0].(vmValue.ValueFloat).Inner

				for i := 0; i < int(durationSecs*1000); i += 10 {
					if i := checkCancelationVM(cancelCtx, span); i != nil {
						return nil, i
					}
					time.Sleep(time.Millisecond * 10)
				}

				return nil, nil
			},
			),
			"add_days": vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
				base := createTimeStructFromObjectVM(args[0])
				days := args[1].(vmValue.ValueInt).Inner
				return createTimeObjectVM(base.Add(time.Hour * 24 * time.Duration(days))), nil
			}),
			"now": vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
				now := time.Now()

				return createTimeObjectVM(now), nil
			}),
		}),
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
// Vm interpreter
//

type TestingVmExecutor struct {
	PrintToStdout bool
	PrintBuf      *string
	PintBufMutex  *sync.Mutex
}

func (self TestingVmExecutor) GetUser() string { return "<unknown>" }

func (self TestingVmExecutor) GetBuiltinImport(moduleName string, toImport string) (val vmValue.Value, found bool) {
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
func (self TestingVmExecutor) ResolveModuleCode(moduleName string) (code string, found bool, err error) {
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

func (self TestingVmExecutor) WriteStringTo(input string) error {
	if self.PrintToStdout {
		fmt.Print(input)
	}
	self.PintBufMutex.Lock()
	*self.PrintBuf += input
	self.PintBufMutex.Unlock()
	return nil
}

func createTimeObjectVM(t time.Time) *vmValue.Value {
	return vmValue.NewValueObject(
		map[string]*vmValue.Value{
			"year":       vmValue.NewValueInt(int64(t.Year())),
			"month":      vmValue.NewValueInt(int64(t.Month())),
			"year_day":   vmValue.NewValueInt(int64(t.YearDay())),
			"hour":       vmValue.NewValueInt(int64(t.Hour())),
			"minute":     vmValue.NewValueInt(int64(t.Minute())),
			"second":     vmValue.NewValueInt(int64(t.Second())),
			"month_day":  vmValue.NewValueInt(int64(t.Day())),
			"week_day":   vmValue.NewValueInt(int64(t.Weekday())),
			"unix_milli": vmValue.NewValueInt(t.UnixMilli()),
		},
	)
}

func createTimeStructFromObjectVM(t vmValue.Value) time.Time {
	tObj := t.(vmValue.ValueObject)
	fields, i := tObj.Fields()
	if i != nil {
		panic(i)
	}
	millis := (*fields["unix_milli"]).(vmValue.ValueInt).Inner
	return time.UnixMilli(millis)
}

func checkCancelationVM(ctx *context.Context, span herrors.Span) *vmValue.VmInterrupt {
	select {
	case <-(*ctx).Done():
		return vmValue.NewVMTerminationInterrupt((*ctx).Err().Error(), span)
	default:
		// do nothing, this should not block the entire interpreter
		return nil
	}
}
