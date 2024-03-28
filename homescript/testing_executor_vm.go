package homescript

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	herrors "github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
	vmValue "github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

//
// Vm scope additions
//

func TestingVmScopeAdditions() map[string]vmValue.Value {
	return map[string]vmValue.Value{
		"log": *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
			base := args[0].(value.ValueFloat).Inner
			x := args[1].(value.ValueFloat).Inner

			// Change of base: https://stackoverflow.com/questions/52917461/how-to-calculate-log16-of-a-256-bit-integer-in-golang
			res := math.Log(x) / math.Log(base)
			return value.NewValueFloat(res), nil
		}),
		"fmt": *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
			displays := make([]any, 0)

			for idx, arg := range args {
				if idx == 0 {
					continue
				}

				var out any

				switch arg.Kind() {
				case vmValue.NullValueKind:
					out = "null"
				case vmValue.IntValueKind:
					out = arg.(value.ValueInt).Inner
				case vmValue.FloatValueKind:
					out = arg.(value.ValueFloat).Inner
				case vmValue.BoolValueKind:
					out = arg.(value.ValueBool).Inner
				case vmValue.StringValueKind:
					out = arg.(value.ValueString).Inner
				default:
					display, i := arg.Display()
					if i != nil {
						return nil, i
					}
					out = display
				}

				displays = append(displays, out)
			}

			out := fmt.Sprintf(args[0].(value.ValueString).Inner, displays...)

			return vmValue.NewValueString(out), nil
		}),
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
		// "assert_eq": *vmValue.NewValueBuiltinFunction(func(executor vmValue.Executor, cancelCtx *context.Context, span herrors.Span, args ...vmValue.Value) (*vmValue.Value, *vmValue.VmInterrupt) {
		// 	if args[0].Kind() != args[1].Kind() {
		// 		a, i := args[0].Display()
		// 		if i != nil {
		// 			return nil, i
		// 		}
		//
		// 		b, i := args[1].Display()
		// 		if i != nil {
		// 			return nil, i
		// 		}
		// 		return nil, vmValue.NewVMThrowInterrupt(
		// 			span,
		// 			fmt.Sprintf("Assertion failed: `%s` is not equal to `%s`", a, b),
		// 		)
		// 	}
		//
		// 	eq, i := args[0].IsEqual(args[1])
		// 	if i != nil {
		// 		return nil, i
		// 	}
		//
		// 	if !eq {
		// 		a, i := args[0].Display()
		// 		if i != nil {
		// 			return nil, i
		// 		}
		//
		// 		b, i := args[1].Display()
		// 		if i != nil {
		// 			return nil, i
		// 		}
		//
		// 		return nil, vmValue.NewVMThrowInterrupt(
		// 			span,
		// 			fmt.Sprintf("Assertion failed: `%s` is not equal to `%s`", a, b),
		// 		)
		// 	}
		// 	return vmValue.NewValueNull(), nil
		// }),
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

func (self TestingVmExecutor) LoadSingleton(singletonIdent, moduleName string) (value.Value, bool, error) {
	// Rely on the default values inserted by the compiler.
	return nil, false, nil
}

func (self TestingVmExecutor) Free() error { return nil }

func (self TestingVmExecutor) GetBuiltinImport(moduleName string, toImport string) (val vmValue.Value, found bool) {
	switch moduleName {
	case "net":
		switch toImport {
		case "http":
			return *value.NewValueObject(map[string]*value.Value{
				"get": value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.VmInterrupt) {
					spew.Dump(args)

					return value.NewValueObject(map[string]*value.Value{
						"status":      value.NewValueString("OK"),
						"status_code": value.NewValueInt(int64(200)),
						"body":        value.NewValueString("test"),
						"cookies":     value.NewValueAnyObject(make(map[string]*vmValue.Value)),
					}), nil
				}),
				"generic": value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span herrors.Span, args ...value.Value) (*value.Value, *value.VmInterrupt) {
					url := args[0].(value.ValueString).Inner
					method := args[1].(value.ValueString).Inner
					body := args[2].(value.ValueOption)
					headers := args[3].(value.ValueAnyObject).FieldsInternal
					cookies := args[4].(value.ValueAnyObject).FieldsInternal

					fmt.Printf("url=%s,method=%s,body=%s,headers=%v,cookies=%v\n", url, method, body, headers, cookies)

					return value.NewValueObject(map[string]*value.Value{
						"status":      value.NewValueString("ok"),
						"status_code": value.NewValueInt(int64(200)),
						"body":        value.NewValueString("TEST"),
						"cookies":     value.NewValueAnyObject(make(map[string]*vmValue.Value)),
					}), nil
				}),
			}), true
		}
		return nil, false
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

func (self TestingVmExecutor) RegisterTrigger(
	callbackFunctionIdent string,
	eventTriggerIdent string,
	span herrors.Span,
	args []value.Value,
) error {
	fmt.Println(callbackFunctionIdent, eventTriggerIdent, span)

	switch eventTriggerIdent {
	case "minute":
		stringArgs := make([]string, len(args))
		for idx, arg := range args {
			argVString, err := arg.Display()
			if err != nil {
				panic((*err).Message())
			}
			stringArgs[idx] = argVString
		}
		fmt.Printf("PLACEHOLDER: Would register trigger `minute` (with args: `[%s]`) now", strings.Join(stringArgs, ", "))
	default:
		panic(fmt.Sprintf("Unknown event trigger ident: `%s`", eventTriggerIdent))
	}

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
