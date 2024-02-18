package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
	"github.com/smarthome-go/homescript/v3/homescript/runtime"
	vmValue "github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

var vmLimits = runtime.CoreLimits{
	CallStackMaxSize: 100,
	StackMaxSize:     500,
	MaxMemorySize:    100 * 1000,
}

func CompileVm(analyzed map[string]ast.AnalyzedProgram, filename string) compiler.Program {
	compilerStruct := compiler.NewCompiler()
	compiled := compilerStruct.Compile(analyzed, filename)
	return compiled
}

func TestingRunVm(compiled compiler.Program, printToStdout bool) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	executor := homescript.TestingVmExecutor{
		PrintToStdout: printToStdout,
		PrintBuf:      new(string),
		PintBufMutex:  &sync.Mutex{},
	}

	start := time.Now()
	vm := runtime.NewVM(compiled, executor, &ctx, &cancel, homescript.TestingVmScopeAdditions(), vmLimits)

	debuggerOut := make(chan runtime.DebugOutput)
	core := vm.SpawnAsync(runtime.FunctionInvocation{
		Function: compiler.EntryPointFunctionIdent,
		Args:     make([]vmValue.Value, 0),
	}, &debuggerOut)

	go TestingDebugConsumer(&debuggerOut, core)

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

	if printToStdout {
		fmt.Printf("VM elapsed: %v\n", time.Since(start))
	}
	return *executor.PrintBuf
}

func TestingRunInterpreter(analyzed map[string]ast.AnalyzedProgram, filename string) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	blocking := make(chan struct{})
	go func() {
		defer func() { blocking <- struct{}{} }()

		executor := homescript.TestingTreeExecutor{
			Output: new(string),
		}

		if i := homescript.Run(
			20_000,
			analyzed,
			filename,
			executor,
			homescript.TestingInterpreterScopeAdditions(),
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

		fmt.Printf("Tree elapsed: %v\n", time.Since(start))
	}()

	<-blocking
	cancel()
}

func TestingDebugConsumer(debuggerOutput *chan runtime.DebugOutput, core *runtime.Core) {
	hits := make(map[uint]int)
	colors := []int{0, 10, 2, 12, 4, 14, 3, 11, 1}

	for {
		select {
		case msg, open := <-*debuggerOutput:
			if !open {
				return
			}

			// Read input file
			program, err := os.ReadFile(fmt.Sprintf("%s.hms", msg.CurrentSpan.Filename))
			if err != nil {
				fmt.Printf("Debugger: cannot open input file `%s.hms`: %s\n", msg.CurrentSpan.Filename, err.Error())
				return
			}

			programStr := string(program)
			lines := strings.Split(programStr, "\n")

			lineIdx := msg.CurrentSpan.Start.Line - 1
			hits[lineIdx]++

			// Highlight active line
			for idx := range lines {
				lineHit := hits[uint(idx)]
				sumHits := 0
				for _, lineHitsI := range hits {
					sumHits += lineHitsI
				}

				cpuTimePercent := (float64(lineHit) / float64(sumHits))

				color := colors[int(cpuTimePercent*float64(len(colors)-1))]

				if idx == int(lineIdx) {
					lines[idx] = fmt.Sprintf("\x1b[4m\x1b[1;3%dm%s\x1b[0m       (%s)", color, lines[lineIdx], msg.CurrentInstruction)
				} else {
					lines[idx] = fmt.Sprintf("\x1b[1;3%dm%s\x1b[1;0m", color, lines[idx])
				}
			}

			fmt.Printf("\033[2J\033[H%s\n---------------------------\n%s\n", *core.Executor.(homescript.TestingVmExecutor).PrintBuf, strings.Join(lines, "\n"))
		}
	}
}
