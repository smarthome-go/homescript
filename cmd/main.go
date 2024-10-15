package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/smarthome-go/homescript/v3/homescript"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/fuzzer"
	"github.com/smarthome-go/homescript/v3/homescript/optimizer"
	"github.com/urfave/cli/v2"
)

const programName = "homescript"
const version = "latest"
const satisfiedAfterDefault = 50
const expectedFile = "expected.out"
const outputDir = "output"

func fileValidator(ctx *cli.Context) error {
	if ctx.Args().Len() != 1 {
		return fmt.Errorf("Expected exactly one argument <file>")
	}
	return nil
}

func analyzeFile(
	program string,
	pathS string,
	printAnalyzed bool,
	printDiagnostics bool,
	fileReader func(path string) (string, error),
) (analyzed map[string]ast.AnalyzedProgram, entryModule string, err error) {
	analyzed, diagnostics, syntaxErrors := homescript.Analyze(
		homescript.InputProgram{
			ProgramText: program,
			Filename:    pathS,
		},
		homescript.TestingAnalyzerScopeAdditions(),
		homescript.TestingAnalyzerHost{
			IsInvokedInTests: false,
		},
		true,
	)

	if len(syntaxErrors) != 0 {
		for _, syntaxErr := range syntaxErrors {
			file, err := fileReader(syntaxErr.Span.Filename)
			if err != nil {
				return nil, "", err
			}

			fmt.Println(syntaxErr.Display(string(file)))
		}

		return nil, "", errors.New("Encountered syntax error(s)")
	}

	abort := false

	for _, item := range diagnostics {
		if item.Level == diagnostic.DiagnosticLevelError {
			abort = true
		}

		if item.Span.Filename == "" {
			panic(spew.Sdump(item))
		}

		if !printDiagnostics {
			continue
		}

		file, err := fileReader(item.Span.Filename)
		if err != nil {
			return nil, "", fmt.Errorf("Could not read file `%s`: %s\n%s | %v", item.Span.Filename, err.Error(), item.Message, item.Span)
		}

		fmt.Println(item.Display(string(file)))
	}

	if abort {
		return nil, "", errors.New("Encountered semantic error(s)")
	}

	if !printAnalyzed {
		return analyzed, pathS, nil
	}

	log.Println("=== ANALYZED ===")
	for name, module := range analyzed {
		log.Printf("=== MODULE: %s ===\n", name)
		fmt.Println(module)
	}

	log.Println("Optimizing...")
	optStart := time.Now()
	optimizer := optimizer.NewOptimizer()
	optimized, diagnostics := optimizer.Optimize(analyzed)
	log.Printf("Finished optimization: elapsed: %v\n", time.Since(optStart))

	for _, item := range diagnostics {
		if item.Level == diagnostic.DiagnosticLevelError {
			abort = true
		}

		if item.Span.Filename == "" {
			panic(spew.Sdump(item))
		}

		file, err := fileReader(item.Span.Filename)
		if err != nil {
			return nil, "", fmt.Errorf("Could not read file `%s`: %s\n%s | %v", item.Span.Filename, err.Error(), item.Message, item.Span)
		}

		fmt.Println(item.Display(string(file)))
	}

	log.Println("=== OPTIMIZED ===")
	for name, module := range optimized {
		log.Printf("=== (OPTIMIZED) MODULE: %s ===\n", name)
		fmt.Println(module)
	}

	return optimized, pathS, nil
}

func main() {
	// nolint:exhaustruct
	app := &cli.App{
		Name:     programName,
		Version:  version,
		Compiled: time.Now(),
		Authors: []*cli.Author{
			{
				Name:  "The Smarthome Authors",
				Email: "",
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "tree",
				Usage:     "Run a Homescript file using the tree-walking interpreter",
				ArgsUsage: "[file]",
				Args:      true,
				Flags:     []cli.Flag{
					// &cli.StringFlag{
					// 	Name:        "mode",
					// 	DefaultText: "tree",
					// 	Usage:       "Select backend",
					// 	Aliases:     []string{"m"},
					// 	Action: func(ctx *cli.Context, s string) error {
					// 		switch s {
					// 		case "vm":
					// 			break
					// 		case "tree":
					// 			break
					// 		}
					// 		return fmt.Errorf("Illegal backend `%s`: Valid values are `vm` and `tree`", s)
					// 	},
					// },
				},
				Before: fileValidator,
				Action: func(c *cli.Context) error {
					filename := c.Args().Get(0)

					file, err := os.ReadFile(filename)
					if err != nil {
						return err
					}

					analyzed, entryModule, err := analyzeFile(string(file), filename, true, true, DefaultReadFileProvider)
					if err != nil {
						return err
					}

					homescript.TestingRunInterpreter(analyzed, entryModule)

					return nil
				},
			},
			{
				Name:      "vm",
				Usage:     "Run a Homescript file using the VM interpreter",
				ArgsUsage: "[file]",
				Args:      true,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "emit-asm",
						Usage:   "If set, the VM asm is printed.",
						Aliases: []string{"s"},
					},
				},
				Before: fileValidator,
				Action: func(c *cli.Context) error {
					filename := c.Args().Get(0)
					emitAsm := c.Bool("emit-asm")

					file, err := os.ReadFile(filename)
					if err != nil {
						return err
					}

					analyzedAndOpt, entryModule, err := analyzeFile(string(file), filename, true, true, DefaultReadFileProvider)
					if err != nil {
						return err
					}

					code := CompileVm(analyzedAndOpt, entryModule)

					if emitAsm {
						fmt.Println("========= COMPILED (ASM) ============")
						fmt.Println(code.AsmString(true))

						fmt.Println("=== Function annotations ===")
						for key, annotations := range code.Annotations {
							module := fmt.Sprintf("mod %s", key.Module)

							if key.Module == entryModule {
								module = fmt.Sprintf("mod (MAIN) %s", key.Module)
							}

							fmt.Printf("%s | `fn %s` | annotation=%v\n", module, key.UnmangledFunction, annotations)
						}
					}

					TestingRunVm(code, true, DefaultReadFileProvider)

					return nil
				},
			},
			{
				Name:    "fuzz",
				Aliases: []string{"f"},
				Usage:   "fuzzing subcommand",
				Subcommands: []*cli.Command{
					{
						Name:      "gen",
						Usage:     "generate fuzzing database from file",
						ArgsUsage: "[file]",
						Args:      true,
						Before: func(ctx *cli.Context) error {
							if ctx.Args().Len() != 2 {
								return fmt.Errorf("Expected exactly two arguments <file> <db-output>")
							}
							return nil
						},
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "verbose",
								Usage:   "Enables additional logging during fuzz generation",
								Aliases: []string{"v"},
							},
							&cli.Int64Flag{
								Name:    "seed",
								Usage:   "Random seed for the transformer",
								Aliases: []string{"s"},
							},
							&cli.UintFlag{
								Name:    "passes",
								Usage:   "N passes",
								Aliases: []string{"p"},
							},
							&cli.UintFlag{
								Name:    "pass-limit",
								Usage:   "The maximum number of output trees of each pass",
								Aliases: []string{"l"},
							},
							&cli.UintFlag{
								Name:    "satisfied-after",
								Usage:   "The number of iterations to continue even though no new output was generated",
								Aliases: []string{"a"},
							},
							&cli.UintFlag{
								Name:    "num-workers",
								Usage:   "The number of threads to spawn during fuzz generation. (Default is number of CPUs)",
								Aliases: []string{"n"},
							},
						},
						Action: func(ctx *cli.Context) error {
							fmt.Println("fuzzing database generation: ", ctx.Args().First())

							passes := ctx.Uint("passes")
							if passes == 0 {
								passes = 1
							}

							seed := ctx.Int64("seed")

							passLimit := ctx.Uint("pass-limit")
							satisfiedAfter := ctx.Uint("satisfied-after")
							if satisfiedAfter == 0 {
								satisfiedAfter = satisfiedAfterDefault
							}

							numWorkers := ctx.Uint("num-workers")
							if numWorkers == 0 {
								numWorkers = uint(runtime.NumCPU())
								log.Printf("Using default amount of %d workers.\n", numWorkers)
							}

							verbose := ctx.Bool("verbose")

							filename := ctx.Args().First()

							file, err := os.ReadFile(filename)
							if err != nil {
								return err
							}

							analyzed, entryModule, err := analyzeFile(string(file), filename, true, true, DefaultReadFileProvider)
							if err != nil {
								return err
							}

							// Generate reference output.
							code := CompileVm(analyzed, entryModule)
							referenceOutput, d := TestingRunVm(code, true, DefaultReadFileProvider)
							if d != nil {
								return fmt.Errorf("VM crashed: %s", d.Display(string(file)))
							}

							// Create output zip file.
							outputFile := ctx.Args().Get(1)
							archive, err := os.Create(outputFile)
							if err != nil {
								return err
							}

							zipWriter := zip.NewWriter(archive)

							w1, err := zipWriter.Create(expectedFile)
							if err != nil {
								return err
							}

							reader := bytes.NewReader([]byte(referenceOutput))
							if _, err := io.Copy(w1, reader); err != nil {
								return err
							}

							newJobChan := make(chan tuple)
							gen := fuzzer.NewGenerator(
								analyzed[entryModule],
								func(tree ast.AnalyzedProgram, treeString string, hashSum string) error {
									newJobChan <- tuple{
										hash:  hashSum,
										value: treeString,
									}

									return nil
								},
								seed,
								passes,
								satisfiedAfter,
								passLimit,
								verbose,
								numWorkers,
							)

							doneChan := make(chan struct{})

							go func() {
								gen.Gen()
								doneChan <- struct{}{}
							}()

							// Spawn writer.
							bufferFlush(
								newJobChan,
								filename,
								zipWriter,
								doneChan,
							)

							if err := zipWriter.SetComment(fmt.Sprintf("Fuzzing output of '%s'", filename)); err != nil {
								return err
							}

							zipWriter.Close()
							archive.Close()

							return nil
						},
					},
					{
						Name:   "validate",
						Usage:  "Validate existing fuzzing database",
						Before: fileValidator,
						Args:   true,
						Action: func(ctx *cli.Context) error {
							filename := ctx.Args().First()
							return validateFuzzDB(filename)
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
