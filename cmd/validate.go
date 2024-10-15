package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/fuzzer"
)

const minRenderDelay = time.Millisecond * 250
const minProgressDelay = time.Second * 1

type brokenOutput struct {
	wrongStdout *string
	error       *diagnostic.Diagnostic
}

func validateFuzzDB(filename string) error {
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

	// Read previous progress and filter out programs which were already checked.
	progFileName := fmt.Sprintf("%s.prog.json", filename)
	prog, err := readProgress(progFileName)
	if err != nil {
		return err
	}

	brokenMap := make(map[string]brokenOutput)

	successCnt, errsCnt := 0, 0

	inputFile := make([]*zip.File, 0)
	for _, file := range archive.File {
		outcome, found := prog.Completed[file.Name]
		if found {
			if outcome == nil {
				successCnt++
				continue
			}

			errsCnt++
			brokenMap[file.Name] = *outcome
			continue
		}

		inputFile = append(inputFile, file)
	}

	// Workers.
	// wg := sync.WaitGroup{}

	numCpu := runtime.NumCPU()
	inputSize := len(archive.File)
	chunkSize := (inputSize + numCpu - 1) / numCpu
	chunks := fuzzer.ChunkInput[*zip.File](inputFile, uint(chunkSize))
	progressChans := make([]chan workerProgress, len(chunks))
	numChunks := len(chunks)

	success := make([]uint, len(chunks))
	errs := make([]uint, len(chunks))

	// Distribute successCnt and errsCnt across the workers.
	fracSuccess := uint(successCnt / numChunks)
	fracErr := uint(errsCnt / numChunks)

	for idx := range chunks {
		success[idx] += fracSuccess
		errs[idx] += fracErr
	}

	success[0] += uint(successCnt - (numChunks * int(fracSuccess)))
	errs[0] += uint(errsCnt - (numChunks * int(fracErr)))

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

	start := time.Now()

	lastRender := time.Time{}
	lastWrite := time.Time{}

	// Allocate screen space for the progress buffer.
	for i := 0; i < numChunks+3; i++ {
		fmt.Println(strings.Repeat(" ", 100))
	}

	for {
		for idx, ch := range progressChans {
			select {
			case m := <-ch:
				if m.done {
					if success[m.idx]+errs[m.idx] < uint(len(archive.File)) {
						success[m.idx] = uint(len(archive.File) / numChunks)
					}

					progressChans = remove(progressChans, idx)
					continue
				}

				if m.success {
					success[m.idx]++
					prog.Completed[m.filename] = nil
				} else {
					errs[m.idx]++
					brokenMap[m.filename] = m.err
					prog.Completed[m.filename] = &m.err
				}

				// Only print if there is enough (minRefresh) time elapsed.
				if time.Since(lastRender) > minRenderDelay {
					lastRender = time.Now()
					recordProgress(
						numChunks,
						chunkSize,
						&success,
						&errs,
						len(archive.File),
						start,
					)
				}

				if time.Since(lastWrite) > minProgressDelay {
					lastWrite = time.Now()
					if err := writeProgress(&prog, progFileName); err != nil {
						return err
					}
				}
			default:
			}
		}

		if len(progressChans) == 0 {
			break
		}
	}

	recordProgress(
		numChunks,
		chunkSize,
		&success,
		&errs,
		len(archive.File),
		start,
	)

	if err := writeProgress(&prog, progFileName); err != nil {
		return err
	}

	errMsg := fmt.Sprintf("Found %d broken program(s)\n", len(brokenMap))
	log.Println(errMsg)
	for key, output := range brokenMap {
		if output.wrongStdout != nil {
			log.Printf("- `%s` created wrong output `%s`\n", key, *output.wrongStdout)
		} else if output.error != nil {
			file, err := archive.Open(key)
			if err != nil {
				return err
			}

			fileBuf := bytes.NewBuffer(make([]byte, 0))
			_, err = io.Copy(fileBuf, file)
			if err != nil {
				return err
			}

			log.Printf("- `%s` created error `%s`\n", key, output.error.Display(fileBuf.String()))
		}
	}

	if len(brokenMap) > 0 {
		return errors.New(errMsg)
	}

	return nil
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

func recordProgress(numChunks int, chunkSize int, success *[]uint, errs *[]uint, numFiles int, start time.Time) {
	printProgress(numChunks, chunkSize, success, errs, numFiles, start)
}

func printProgress(numChunks int, chunkSize int, success *[]uint, errs *[]uint, numFiles int, start time.Time) {
	// Erase lines.
	for i := 0; i < numChunks+2; i++ {
		fmt.Printf("\x1b[1A")
	}

	accAll := 0

	padding := strings.Repeat(" ", 30)

	// Print state.
	for i := 0; i < numChunks; i++ {
		thisAcc := (*success)[i] + (*errs)[i]
		accAll += int(thisAcc)

		if thisAcc > uint(chunkSize) {
			chunkSize = int(thisAcc)
		}
		fmt.Printf("Worker %2d: (%5d / %d) \x1b[1;32msuccess\x1b[1;0m:%5d; \x1b[1;31mfailed\x1b[1;0m:%5d%s\n", i, thisAcc, chunkSize, (*success)[i], (*errs)[i], padding)
	}

	// Percentage of work left to do multiplied by the time it took to do one percent.
	timeUntilNow := time.Since(start)
	percentNow := (float64(accAll) / float64(numFiles)) * 100.0

	if percentNow > 100 {
		percentNow = 100
	}

	timeOncepercent := time.Duration(float64(timeUntilNow) / percentNow)
	remainingPercent := 100.0 - percentNow
	remainingTime := timeOncepercent * time.Duration(int(remainingPercent))
	remaining := numFiles - accAll
	if remaining < 0 {
		remaining = 0
	}

	fmt.Printf(
		"\x1b[1;35m%3d%% completed\x1b[1;0m | processed: %5d; remaining: %5d\n=> \x1b[1;30mElapsed: %s\x1b[1;0m, ETA %s\n",
		int(percentNow),
		accAll,
		remaining,
		fmtDuration(time.Since(start)),
		fmtDuration(remainingTime),
	)
}

func remove[T any](s []T, idx int) []T {
	s[idx] = s[len(s)-1]
	return s[:len(s)-1]
}

type workerProgress struct {
	done      bool
	success   bool
	err       brokenOutput
	filename  string
	chunkSize int
	idx       int
}

//
// Progress file.
//

type progressFile struct {
	// Maps a filename to an outcome.
	// Filenames which are not in the map have not yet been processed.
	Completed map[string]*brokenOutput `json:"completed"`
}

func writeProgress(file *progressFile, filename string) error {
	data, err := json.Marshal(*file)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return err
	}

	return nil
}

func readProgress(filename string) (progressFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Progress file `%s` does not exist, creating new one...\n", filename)
			return progressFile{
				Completed: make(map[string]*brokenOutput),
			}, nil
		}
		return progressFile{}, err
	}

	var prog progressFile

	if err := json.Unmarshal(data, &prog); err != nil {
		return progressFile{}, err
	}

	fmt.Printf("Progress file `%s` read successfully.\n", filename)

	return prog, nil
}

//
// Worker.
//

func worker(chunk []*zip.File, workerIndex int, expectedFileContents string, resultChan chan workerProgress) {
	for _, file := range chunk {
		if file.Name == expectedFile {
			continue
		}

		// log.Printf("\x1b[2m\x1b[1;30mRUNNING:\x1b[1;0m (5%d / %d) testing file `%s`...\n", idx, chunkSize, file.Name)

		zipFileReader := func(path string) (string, error) {
			for _, file := range chunk {
				if file.Name == path {
					buf := bytes.NewBuffer(make([]byte, 0))
					file, err := file.Open()
					if err != nil {
						return "", err
					}
					_, err = io.Copy(buf, file)
					if err != nil {
						return "", err
					}
					return buf.String(), nil
				}
			}

			return "", fmt.Errorf("File not found in chunk")
		}

		contents, err := file.Open()
		if err != nil {
			log.Panic(err.Error())
		}

		buf := bytes.NewBuffer(make([]byte, 0))
		_, err = io.Copy(buf, contents)
		if err != nil {
			log.Panic(err.Error())
		}

		// TODO: also record these errors
		analyzed, entryModule, err := analyzeFile(buf.String(), file.Name, false, false, zipFileReader)
		if err != nil {
			log.Panic(err.Error())
		}

		code := CompileVm(analyzed, entryModule)
		output, d := TestingRunVm(code, false, zipFileReader)

		// Erase previous line.
		// fmt.Printf("\x1b[1A")

		if output != expectedFileContents {
			// log.Printf("\x1b[1;31mFAIL:   \x1b[1;0m expected `%s`, got: `%s`\n", expectedFileContents, output)

			resultChan <- workerProgress{
				done:     false,
				success:  false,
				filename: file.Name,
				err: brokenOutput{
					wrongStdout: &output,
					error:       nil,
				},
				chunkSize: len(chunk),
				idx:       workerIndex,
			}

			continue
		}

		if d != nil {
			resultChan <- workerProgress{
				done:     false,
				success:  false,
				filename: file.Name,
				err: brokenOutput{
					wrongStdout: nil,
					error:       d,
				},
				chunkSize: len(chunk),
				idx:       workerIndex,
			}

			continue
		}

		// log.Printf("\x1b[1;32mPASS:   \x1b[1;0m output matches reference.\n")

		resultChan <- workerProgress{
			done:      false,
			success:   true,
			filename:  file.Name,
			err:       brokenOutput{},
			chunkSize: len(chunk),
			idx:       workerIndex,
		}
	}

	resultChan <- workerProgress{
		done:      true,
		success:   false,
		filename:  "",
		err:       brokenOutput{},
		chunkSize: len(chunk),
		idx:       workerIndex,
	}
}
