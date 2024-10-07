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

const DEFAULT_TIMEOUT = time.Second * 5

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
	OverrideTimeout    time.Duration
	UseOverrideTimeout bool
}

func TestScripts(t *testing.T) {
	tests := []Test{
		// Can be used for manual override
		// {
		// 	Name:  "TestAllBuiltinMembers",
		// 	Path:  "../tests/tests/builtin_members.hms",
		// 	Debug: false,
		// },
		{
			Name:               "PrimeFuzz",
			Path:               "../tests/prime_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/primes.hms.out",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_FILE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "FizzBuzzFuzz",
			Path:               "../tests/fizz_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/fizzbuzz.hms.out",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_FILE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "BoxFuzz",
			Path:               "../tests/box_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/box.hms.out",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_FILE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "BinaryFuzz",
			Path:               "../tests/binary_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/binary.hms.out",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_NONE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "DevFuzz",
			Path:               "../tests/dev_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "../examples/dev.hms.out",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_FILE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "PowFuzz",
			Path:               "../tests/pow_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_NONE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "PiFuzz",
			Path:               "../tests/pi_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_NONE,
			Skip:               false,
			OverrideTimeout:    time.Second * 60,
			UseOverrideTimeout: true,
		},
		{
			Name:               "EFuzz",
			Path:               "../tests/e_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_NONE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "AperyFuzz",
			Path:               "../tests/apery_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_NONE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "MatrixFuzz",
			Path:               "../tests/matrix_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_NONE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "scoping_regression",
			Path:               "../tests/regression_scoping.hms",
			IsGlob:             false,
			Debug:              false,
			ExpectedOutputFile: "",
			ExpectedOutputRaw:  "69\n123\n69\n42\n",
			ValidateOutput:     OUTPUT_VALIDATION_RAW,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "iterators_regression",
			Path:               "../tests/regression_iterators.hms",
			IsGlob:             false,
			Debug:              false,
			ExpectedOutputFile: "",
			ExpectedOutputRaw:  "OUTER\nA\nB\nOUTER\nA\n[]\n",
			ValidateOutput:     OUTPUT_VALIDATION_RAW,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "Linear Gradient",
			Path:               "../tests/linear_gradient_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_NONE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
		{
			Name:               "Linear Gradient 2D",
			Path:               "../tests/linear_gradient_2d_fuzz/*.hms",
			IsGlob:             true,
			Debug:              false,
			ExpectedOutputFile: "",
			ExpectedOutputRaw:  "",
			ValidateOutput:     OUTPUT_VALIDATION_NONE,
			Skip:               false,
			OverrideTimeout:    0,
			UseOverrideTimeout: false,
		},
	}

	outputTests := make([]Test, 0)

	fileCache := make(map[string]string)

	// Prepare the tests
	for _, test := range tests {
		fmt.Println(test.Path)

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
				fmt.Printf("FILE GLOB=%s\n", file)
				for _, test := range tests {
					if test.Path == file {
						fmt.Println("cont")
						continue
					}
				}

				testName := fmt.Sprintf("%s: %s", test.Name, file)

				outputTests = append(outputTests, Test{
					Name:               testName,
					Path:               file,
					IsGlob:             false,
					Debug:              test.Debug,
					ExpectedOutputFile: test.ExpectedOutputFile,
					ExpectedOutputRaw:  test.ExpectedOutputRaw,
					ValidateOutput:     test.ValidateOutput,
					Skip:               false,
					OverrideTimeout:    test.OverrideTimeout,
					UseOverrideTimeout: test.UseOverrideTimeout,
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
		}, TestingAnalyzerScopeAdditions(), TestingAnalyzerHost{
			IsInvokedInTests: true,
		})

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

		for _, d := range diagnostics {
			if d.Level == diagnostic.DiagnosticLevelError {
				file, err := os.ReadFile(d.Span.Filename)
				assert.NoError(t, err)

				t.Error(d.Display(string(file)))
				hasErr = true
			}
		}
	}

	if hasErr {
		t.Error("Homescript contains errors")
		return
	}

	compilerStruct := compiler.NewCompiler(modules, test.Path)
	compiled, err := compilerStruct.Compile()
	if err != nil {
		panic(fmt.Sprintf("compiler failed: %s", err.Error()))
	}

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

	timeout := DEFAULT_TIMEOUT
	if test.UseOverrideTimeout {
		timeout = test.OverrideTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	executor := TestingVmExecutor{
		PrintToStdout: test.Debug,
		PrintBuf:      new(string),
		PintBufMutex:  &sync.Mutex{},
	}

	start := time.Now()

	vm := runtime.NewVM(compiled, executor, &ctx, &cancel, TestingVmScopeAdditions(), runtime.CoreLimits{
		CallStackMaxSize: 10024,
		StackMaxSize:     10024,
		MaxMemorySize:    10024,
	})

	// TODO: how to handle the debugger at this point?

	vm.SpawnAsync(runtime.MainFn(), nil, nil, nil)

	if coreNum, i := vm.Wait(); i != nil {
		i := *i

		d := diagnostic.Diagnostic{
			Level:   diagnostic.DiagnosticLevelError,
			Message: i.Message(),
			Notes:   []string{fmt.Sprintf("Exception occurred on core %d (timeout = %v (override: %v), runtime = %v)", coreNum, timeout, test.UseOverrideTimeout, time.Since(start))},
			Span:    i.GetSpan(), // this call might panic
		}

		fmt.Printf("Reading: %s../tests.\n", i.GetSpan().Filename)

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

	buf := *executor.PrintBuf

	switch test.ValidateOutput {
	case OUTPUT_VALIDATION_NONE:
		break
	case OUTPUT_VALIDATION_FILE:
		assert.Equal(t, expectedOutputCache, buf, "Generated output does match expected output.")
	case OUTPUT_VALIDATION_RAW:
		assert.Equal(t, test.ExpectedOutputRaw, buf, "Generated output does match expected output.")
	}

}
