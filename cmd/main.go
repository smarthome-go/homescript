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

	"github.com/smarthome-go/homescript/v3/homescript"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/fuzzer"
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

func analyzeFile(program string, pathS string, printAnalyzed bool) (analyzed map[string]ast.AnalyzedProgram, entryModule string, err error) {
	// hmsFilename := path.Base(pathS)

	analyzed, diagnostics, syntaxErrors := homescript.Analyze(homescript.InputProgram{
		ProgramText: program,
		Filename:    pathS,
	}, homescript.TestingAnalyzerScopeAdditions(), homescript.TestingAnalyzerHost{})

	if len(syntaxErrors) != 0 {
		for _, syntaxErr := range syntaxErrors {
			log.Printf("Reading: %s...\n", syntaxErr.Span.Filename)
			file, err := os.ReadFile(syntaxErr.Span.Filename)
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

		log.Printf("Reading: %s...\n", item.Span.Filename)

		file, err := os.ReadFile(fmt.Sprintf("%s.hms", item.Span.Filename))
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
		log.Println(module)
	}

	return analyzed, pathS, nil
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

					analyzed, entryModule, err := analyzeFile(string(file), filename, true)
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

					analyzed, entryModule, err := analyzeFile(string(file), filename, true)
					if err != nil {
						return err
					}

					code := CompileVm(analyzed, entryModule)

					if emitAsm {
						fmt.Println(code.AsmString())
					}

					TestingRunVm(code, true)

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

							verbose := ctx.Bool("verbose")

							filename := ctx.Args().First()

							file, err := os.ReadFile(filename)
							if err != nil {
								return err
							}

							analyzed, entryModule, err := analyzeFile(string(file), filename, true)
							if err != nil {
								return err
							}

							// Generate reference output.
							code := CompileVm(analyzed, entryModule)
							referenceOutput := TestingRunVm(code, true)

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
							)

							doneChan := make(chan struct{})

							go func() {
								gen.Gen()
								doneChan <- struct{}{}
							}()

							// Spawn writer.
							bufferFlush(
								newJobChan,
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

							archive, err := zip.OpenReader(filename)
							if err != nil {
								return err
							}

							expectedOutFile, err := archive.Open(expectedFile)
							if err != nil {
								return err
							}

							expectedBuf := bytes.NewBuffer(make([]byte, 0))
							_, err = io.Copy(expectedBuf, expectedOutFile)
							if err != nil {
								return err
							}

							expectedFileContents := expectedBuf.String()

							// Workers.
							// wg := sync.WaitGroup{}

							inputSize := len(archive.File)
							numCpu := runtime.NumCPU()
							chunkSize := (inputSize + numCpu - 1) / numCpu
							chunks := fuzzer.ChunkInput[*zip.File](archive.File, uint(chunkSize))
							brokenMap := make(map[string]string)
							progressChans := make([]chan workerProgress, len(chunks))
							numChunks := len(chunks)

							for idx, chunk := range chunks {
								ch := make(chan workerProgress)

								go worker(
									chunk,
									idx,
									expectedFileContents,
									ch,
								)

								progressChans[idx] = ch
							}

							success := make([]uint, len(chunks))
							errs := make([]uint, len(chunks))
							start := time.Now()

							const minRenderDelay = time.Millisecond * 100
							lastRender := time.Time{}

							for {
								for idx, ch := range progressChans {
									select {
									case m := <-ch:
										if m.done {
											if success[m.idx]+errs[m.idx] < uint(len(archive.File)) {
												success[m.idx] = uint(chunkSize)
											}

											progressChans = remove(progressChans, idx)
											continue
										}

										if m.success {
											success[m.idx]++
										} else {
											errs[m.idx]++
											brokenMap[m.mismatchID] = m.mismatchOutput
										}

										// Only print if there is enough (minRefresh) time elapsed.
										if time.Since(lastRender) > minRenderDelay {
											lastRender = time.Now()
											printProgress(
												numChunks,
												chunkSize,
												&success,
												&errs,
												len(archive.File),
												start,
											)
										}
									default:
									}
								}

								if len(progressChans) == 0 {
									break
								}
							}

							printProgress(
								numChunks,
								chunkSize,
								&success,
								&errs,
								len(archive.File),
								start,
							)

							log.Printf("Found %d broken program(s)\n", len(brokenMap))
							for key, output := range brokenMap {
								log.Printf("- `%s` created output `%s`\n", key, output)
							}

							return nil
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

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)

	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func printProgress(numChunks int, chunkSize int, success *[]uint, errs *[]uint, numFiles int, start time.Time) {
	// Erase lines.
	for i := 0; i < numChunks+2; i++ {
		fmt.Printf("\x1b[1A")
	}

	accAll := 0

	// Print state.
	for i := 0; i < numChunks; i++ {
		thisAcc := (*success)[i] + (*errs)[i]
		accAll += int(thisAcc)
		fmt.Printf("Worker %2d: (%5d / %d) \x1b[1;32mworking\x1b[1;0m:%5d; \x1b[1;31mfailed\x1b[1;0m:%5d\n", i, thisAcc, chunkSize, (*success)[i], (*errs)[i])
	}

	// Percentage of work left to do multiplied by the time it took to do one percent.
	timeUntilNow := time.Since(start)
	percentNow := (float64(accAll) / float64(numFiles)) * 100.0
	timeOncepercent := time.Duration(float64(timeUntilNow) / percentNow)
	remainingPercent := 100.0 - percentNow
	remainingTime := timeOncepercent * time.Duration(int(remainingPercent))
	fmt.Printf(
		"\x1b[1;35m%3d%% completed\x1b[1;0m | processed: %5d; remaining: %5d\n=> \x1b[1;30mElapsed: %s\x1b[1;0m, ETA %s\n",
		int(percentNow),
		accAll,
		numFiles-accAll,
		fmtDuration(time.Since(start)),
		fmtDuration(remainingTime),
	)
}

func remove[T any](s []T, idx int) []T {
	s[idx] = s[len(s)-1]
	return s[:len(s)-1]
}

type workerProgress struct {
	done           bool
	success        bool
	mismatchID     string
	mismatchOutput string
	chunkSize      int
	idx            int
}

func worker(chunk []*zip.File, workerIndex int, expectedFileContents string, resultChan chan workerProgress) {
	for _, file := range chunk {
		if file.Name == expectedFile {
			continue
		}

		// log.Printf("\x1b[2m\x1b[1;30mRUNNING:\x1b[1;0m (5%d / %d) testing file `%s`...\n", idx, chunkSize, file.Name)

		contents, err := file.Open()
		if err != nil {
			log.Panic(err.Error())
		}

		buf := bytes.NewBuffer(make([]byte, 0))
		_, err = io.Copy(buf, contents)
		if err != nil {
			log.Panic(err.Error())
		}

		analyzed, entryModule, err := analyzeFile(buf.String(), file.Name, false)
		if err != nil {
			log.Panic(err.Error())
		}

		code := CompileVm(analyzed, entryModule)
		output := TestingRunVm(code, false)

		// Erase previous line.
		// fmt.Printf("\x1b[1A")

		if output != expectedFileContents {
			// log.Printf("\x1b[1;31mFAIL:   \x1b[1;0m expected `%s`, got: `%s`\n", expectedFileContents, output)

			resultChan <- workerProgress{
				done:           false,
				success:        false,
				mismatchID:     file.Name,
				mismatchOutput: output,
				chunkSize:      len(chunk),
				idx:            workerIndex,
			}

			continue
		}

		// log.Printf("\x1b[1;32mPASS:   \x1b[1;0m output matches reference.\n")

		resultChan <- workerProgress{
			done:           false,
			success:        true,
			mismatchID:     "",
			mismatchOutput: "",
			chunkSize:      len(chunk),
			idx:            workerIndex,
		}
	}

	resultChan <- workerProgress{
		done:           true,
		success:        false,
		mismatchID:     "",
		mismatchOutput: "",
		chunkSize:      len(chunk),
		idx:            workerIndex,
	}
}
