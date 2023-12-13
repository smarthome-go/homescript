package homescript

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/runtime"
	"github.com/stretchr/testify/assert"
)

//
// Tests
//

type Test struct {
	Name           string
	Path           string
	IsGlob         bool
	Debug          bool
	ExpectedOutput string
	ValidateOutput bool
	Skip           bool
}

func TestScripts(t *testing.T) {
	tests := []Test{
		// Can be used for manual override
		// {
		// 	Name:  "TestAllBuiltinMembers",
		// 	Path:  "../tests/builtin_members.hms",
		// 	Debug: false,
		// },
		// {
		// 	Name:           "Fuzz",
		// 	Path:           "../prime_fuzz/*.hms",
		// 	IsGlob:         true,
		// 	Debug:          false,
		// 	ExpectedOutput: "2\n3\n5\n7\n11\n13\n17\n19\n23\n29\n31\n37\n41\n43\n47\n53\n",
		// 	ValidateOutput: true,
		// },
		// TODO: fizzbuzz is broken in the VM
		{
			Name:           "FizzBuzzFuzz",
			Path:           "../fizz_fuzz/*.hms",
			IsGlob:         true,
			Debug:          false,
			ExpectedOutput: "2\n3\n5\n7\n11\n13\n17\n19\n23\n29\n31\n37\n41\n43\n47\n53\n",
			ValidateOutput: true,
		},
		{
			Name:           "scoping_regression",
			Path:           "../tests/regression_scoping.hms",
			IsGlob:         false,
			Debug:          false,
			ExpectedOutput: "69\n123\n69\n42\n",
			ValidateOutput: true,
		},
	}

	outputTests := make([]Test, 0)

	// Prepare the tests
	for _, test := range tests {
		if test.Skip {
			t.Logf("Skipping test `%s` and all its expansions\n", test.Name)
			continue
		}

		if test.IsGlob {
			files, err := filepath.Glob(test.Path)
			if err != nil {
				panic(err.Error())
			}

			for _, file := range files {
				for _, test := range tests {
					if test.Path == file {
						continue
					}
				}

				outputTests = append(outputTests, Test{
					Name:           fmt.Sprintf("%s: %s", test.Name, file),
					Path:           file,
					IsGlob:         false,
					Debug:          test.Debug,
					ExpectedOutput: test.ExpectedOutput,
					ValidateOutput: test.ValidateOutput,
				})
			}
		} else {
			outputTests = append(outputTests, test)
		}

	}

	// After preparation, run the tests
	for idx, test := range outputTests {
		t.Run(fmt.Sprintf("%d: %s", idx, test.Name), func(t *testing.T) {
			execTest(test, t)
		})
	}
}

func execTest(test Test, t *testing.T) {
	code, err := os.ReadFile(test.Path)
	if err != nil {
		t.Error(err.Error())
		return
	}

	modules, diagnostics, syntax := Analyze(
		InputProgram{
			Filename:    test.Path,
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

	if test.Debug {
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
	compiled := compiler.Compile(modules, test.Path)

	if test.Debug {
		i := 0
		for name, function := range compiled.Functions {
			fmt.Printf("%03d ===> func: %s\n", i, name)

			for idx, inst := range function {
				fmt.Printf("%03d | %s\n", idx, inst)
			}

			i++
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	executor := vmExecutor{
		PrintToStdout: test.Debug,
		PrintBuf:      new(string),
		PintBufMutex:  &sync.Mutex{},
	}

	vm := runtime.NewVM(compiled, executor, &ctx, &cancel, vmiScopeAdditions(), runtime.CoreLimits{
		CallStackMaxSize: 10024,
		StackMaxSize:     10024,
		MaxMemorySize:    10024,
	})

	// TODO: how to handle the debugger at this point?
	vm.Spawn(compiled.EntryPoint, nil)
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
			fmt.Println(err.Error())
			wd, err2 := os.Getwd()
			if err2 != nil {
				panic(err2.Error())
			}
			panic(fmt.Sprintf("Could not read file `%s`: %s\nDIR=%s\n", i.GetSpan().Filename, err.Error(), wd))
		}

		fmt.Printf("%s\n", d.Display(string(file)))
		panic(fmt.Sprintf("Core %d crashed", coreNum))
	}

	if test.ValidateOutput {
		assert.Equal(t, test.ExpectedOutput, *executor.PrintBuf, "Generated output does match expected output.")
	}
}
