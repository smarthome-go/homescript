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

type OUTPUT_VALIDATION uint

const (
	OUTPUT_VALIDATION_NONE OUTPUT_VALIDATION = iota
	OUTPUT_VALIDATION_FILE
	OUTPUT_VALIDATION_RAW
)

type Test struct {
	Name               string
	Path               string
	IsGlob             bool
	Debug              bool
	ExpectedOutputFile string
	ExpectedOutputRaw  string
	ValidateOutput     OUTPUT_VALIDATION
	Skip               bool
}

func TestScripts(t *testing.T) {
	tests := []Test{
		// Can be used for manual override
		// {
		// 	Name:  "TestAllBuiltinMembers",
		// 	Path:  "../tests/builtin_members.hms",
		// 	Debug: false,
		// },
		{
			Name:               "PrimeFuzz",
			Path:               "../prime_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/primes.hms.out",
			ValidateOutput:     OUTPUT_VALIDATION_FILE,
			Skip:               false,
		},
		{
			Name:               "FizzBuzzFuzz",
			Path:               "../fizz_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/fizzbuzz.hms.out",
			ValidateOutput:     OUTPUT_VALIDATION_FILE,
			Skip:               false,
		},
		{
			Name:               "BoxFuzz",
			Path:               "../box_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/box.hms.out",
			ValidateOutput:     OUTPUT_VALIDATION_FILE,
			Skip:               false,
		},
		{
			Name:               "BinaryFuzz",
			Path:               "../binary_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/binary.hms.out",
			Skip:               false,
			ValidateOutput:     OUTPUT_VALIDATION_FILE,
		},
		{
			Name:               "DevFuzz",
			Path:               "../dev_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/dev.hms.out",
			ValidateOutput:     OUTPUT_VALIDATION_FILE,
			Skip:               false,
		},
		{
			Name:           "PowFuzz",
			Path:           "../pow_fuzz/*.hms",
			IsGlob:         true,
			Debug:          false,
			ValidateOutput: OUTPUT_VALIDATION_NONE,
			Skip:           false,
		},
		{
			Name:           "PiFuzz",
			Path:           "../pi_fuzz/*.hms",
			IsGlob:         true,
			Debug:          false,
			ValidateOutput: OUTPUT_VALIDATION_NONE,
			Skip:           false,
		},
		{
			Name:           "EFuzz",
			Path:           "../e_fuzz/*.hms",
			IsGlob:         true,
			Debug:          false,
			ValidateOutput: OUTPUT_VALIDATION_NONE,
			Skip:           false,
		},
		{
			Name:           "AperyFuzz",
			Path:           "../apery_fuzz/*.hms",
			IsGlob:         true,
			Debug:          false,
			ValidateOutput: OUTPUT_VALIDATION_NONE,
			Skip:           false,
		},
		{
			Name:              "scoping_regression",
			Path:              "../tests/regression_scoping.hms",
			IsGlob:            false,
			Debug:             false,
			ExpectedOutputRaw: "69\n123\n69\n42\n",
			ValidateOutput:    OUTPUT_VALIDATION_RAW,
			Skip:              false,
		},
	}

	outputTests := make([]Test, 0)

	fileCache := make(map[string]string)

	// Prepare the tests
	for _, test := range tests {
		if test.Skip {
			t.Logf("Skipping test `%s` and all its expansions\n", test.Name)
			continue
		}

		var fileContent string
		if test.ValidateOutput == OUTPUT_VALIDATION_FILE {
			fileCont, err := os.ReadFile(test.ExpectedOutputFile)
			if err != nil {
				panic(err.Error())
			}
			fileContent = string(fileCont)
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

				testName := fmt.Sprintf("%s: %s", test.Name, file)

				outputTests = append(outputTests, Test{
					Name:               testName,
					Path:               file,
					IsGlob:             false,
					Debug:              test.Debug,
					ExpectedOutputRaw:  test.ExpectedOutputRaw,
					ExpectedOutputFile: test.ExpectedOutputFile,
					ValidateOutput:     test.ValidateOutput,
				})

				fileCache[testName] = fileContent
			}
		} else {
			outputTests = append(outputTests, test)
		}

	}

	// After preparation, run the tests
	for idx, test := range outputTests {
		execTestWrapper(idx, test, fileCache[test.Name], t)
	}
}

func execTestWrapper(idx int, test Test, expectedOutputCache string, t *testing.T) {
	t.Run(fmt.Sprintf("%d: %s", idx, test.Name), func(t *testing.T) {
		t.Parallel()
		execTest(test, expectedOutputCache, t)
	})
}

func execTest(test Test, expectedOutputCache string, t *testing.T) {
	code, err := os.ReadFile(test.Path)
	if err != nil {
		t.Error(err.Error())
		return
	}

	modules, diagnostics, syntax := Analyze(
		InputProgram{
			Filename:    test.Path,
			ProgramText: string(code),
		}, TestingAnalyzerScopeAdditions(), TestingAnalyzerHost{})
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

	executor := TestingVmExecutor{
		PrintToStdout: test.Debug,
		PrintBuf:      new(string),
		PintBufMutex:  &sync.Mutex{},
	}

	vm := runtime.NewVM(compiled, executor, &ctx, &cancel, TestingVmScopeAdditions(), runtime.CoreLimits{
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

	switch test.ValidateOutput {
	case OUTPUT_VALIDATION_NONE:
		break
	case OUTPUT_VALIDATION_FILE:
		assert.Equal(t, expectedOutputCache, *executor.PrintBuf, "Generated output does match expected output.")
	case OUTPUT_VALIDATION_RAW:
		assert.Equal(t, test.ExpectedOutputRaw, *executor.PrintBuf, "Generated output does match expected output.")
	}

}
