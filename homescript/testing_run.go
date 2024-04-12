package homescript

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
	"github.com/smarthome-go/homescript/v3/homescript/runtime"
	vmValue "github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

var testingLimits = runtime.CoreLimits{
	CallStackMaxSize: 100,
	StackMaxSize:     500,
	MaxMemorySize:    100 * 1000,
}

func TestingRunVm(analyzed map[string]ast.AnalyzedProgram, filename string, printToStdout bool) string {
	compilerStruct := compiler.NewCompiler(analyzed, filename)
	compiled := compilerStruct.Compile()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	rawExecutor := TestingVmExecutor{
		PrintToStdout: printToStdout,
		PrintBuf:      new(string),
		PintBufMutex:  &sync.Mutex{},
	}

	executor := vmValue.Executor(rawExecutor)

	vm := runtime.NewVM(compiled, executor, &ctx, &cancel, TestingVmScopeAdditions(), testingLimits)

	debuggerOut := make(chan runtime.DebugOutput)
	vm.SpawnAsync(runtime.FunctionInvocation{
		Function: compiler.EntryPointFunctionIdent,
		Args:     make([]vmValue.Value, 0),
		// TODO: is this allowed?
		FunctionSignature: runtime.FunctionInvocationSignature{},
	}, &debuggerOut)

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
			panic(fmt.Sprintf("Could not read file `%s`: %s | %s\n", i.GetSpan().Filename, err.Error(), i.Message()))
		}

		fmt.Printf("%s\n", d.Display(string(file)))
		panic(fmt.Sprintf("Core %d crashed", coreNum))
	}

	return *rawExecutor.PrintBuf
}

func TestingRunInterpreter(analyzed map[string]ast.AnalyzedProgram, filename string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	blocking := make(chan struct{})
	go func() {
		defer func() { blocking <- struct{}{} }()

		executor := TestingTreeExecutor{
			Output: new(string),
		}

		if i := Run(
			20_000,
			analyzed,
			filename,
			executor,
			TestingInterpreterScopeAdditions(),
			&ctx,
		); i != nil {
			switch (*i).Kind() {
			case value.FatalExceptionInterruptKind:
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
	}()

	<-blocking
	cancel()
}
