package fuzzer

import (
	"crypto/md5"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

const GEN_VERBOSE = true

type Generator struct {
	outputDir            string
	seed                 int64
	input                ast.AnalyzedProgram
	passes               int
	terminateAfterNTries int
	outputLimitPerPass   int
}

func NewGenerator(
	input ast.AnalyzedProgram,
	outputDir string,
	seed int64,
	passes int,
	terminateAfterNTries int,
	outputSizeLimit int,
) Generator {
	return Generator{
		outputDir:            outputDir,
		seed:                 seed,
		input:                input,
		passes:               passes,
		terminateAfterNTries: terminateAfterNTries,
		outputLimitPerPass:   outputSizeLimit,
	}
}

func chunkInput(input []ast.AnalyzedProgram, chunkSize int) [][]ast.AnalyzedProgram {
	var chunks [][]ast.AnalyzedProgram
	for {
		if len(input) == 0 {
			break
		}

		// Necessary check to avoid slicing beyond slice capacity
		if len(input) < chunkSize {
			chunkSize = len(input)
		}

		chunks = append(chunks, input[0:chunkSize])
		input = input[chunkSize:]
	}

	return chunks
}

func (self *Generator) Gen() {
	startAll := time.Now()

	if err := os.MkdirAll(self.outputDir, 0755); err != nil {
		panic(err.Error())
	}

	passResults := make([][]ast.AnalyzedProgram, self.passes+1)
	passResults[0] = []ast.AnalyzedProgram{self.input}

	for passIndex := 0; passIndex < self.passes; passIndex++ {
		start := time.Now()

		if len(passResults[passIndex]) < 100 {
			fmt.Printf("Executing pass %d single threaded.\n", passIndex)
			wg := sync.WaitGroup{}
			wg.Add(1)
			self.Pass(
				&wg,
				&passResults[passIndex],
				&passResults[passIndex+1],
				self.outputLimitPerPass,
			)

			fmt.Printf("\n=== Pass %d duration: %v for input size %d\n", passIndex, time.Since(start), len(passResults[passIndex]))
			continue
		}

		inputSize := len(passResults[passIndex])
		numCpu := runtime.NumCPU()
		chunkSize := (inputSize + numCpu - 1) / numCpu

		inputChunks := chunkInput(passResults[passIndex], chunkSize)
		numChunks := len(inputChunks)
		outputChunks := make([][]ast.AnalyzedProgram, numChunks)

		if numChunks > numCpu {
			panic("Illegal state")
		}

		wg := sync.WaitGroup{}

		for chunkIndex := 0; chunkIndex < numChunks; chunkIndex++ {
			fmt.Printf("Spawning worker %d for pass %d...\n", chunkIndex, passIndex)

			wg.Add(1)
			go self.Pass(
				&wg,
				&inputChunks[chunkIndex],
				&outputChunks[chunkIndex],
				self.outputLimitPerPass/numChunks,
			)
		}

		fmt.Println("Waiting...")
		wg.Wait()
		fmt.Println("Wait finished")

		// Slice the output chunks together again
		for chunkIndex := 0; chunkIndex < numChunks; chunkIndex++ {
			passResults[passIndex+1] = append(passResults[passIndex+1], outputChunks[chunkIndex]...)
		}

		fmt.Printf("\n=== Pass %d duration: %v for input size %d\n", passIndex, time.Since(start), len(passResults[passIndex]))
	}

	if GEN_VERBOSE {
		generated := 0

		hashset := make(map[string]struct{})
		for _, pass := range passResults {
			for _, tree := range pass {
				newTreeStr := tree.String()
				sumRam := md5.Sum([]byte(newTreeStr))
				sum := fmt.Sprintf("%x", sumRam)

				if _, found := hashset[sum]; found {
					continue
				}

				hashset[sum] = struct{}{}
				generated += 1
			}
		}

		fmt.Printf("\nFuzz: %v, generated: %d\n", time.Since(startAll), generated)
	}
}

func (self *Generator) Pass(
	wg *sync.WaitGroup,
	inputTrees *[]ast.AnalyzedProgram,
	outputTrees *[]ast.AnalyzedProgram,
	outputSizeLimit int,
) {
	defer wg.Done()

	hashset := make(map[string]struct{})
	countNoNew := 0

	trans := NewTransformer(100, self.seed)

	lastLen := len(*inputTrees)

	for (countNoNew < self.terminateAfterNTries || self.terminateAfterNTries <= 0) && len(hashset) <= outputSizeLimit {
		if GEN_VERBOSE {
			if countNoNew > 11 {
				fmt.Printf("%d (%d remaining) ", countNoNew, self.terminateAfterNTries-countNoNew)
			} else if countNoNew > 10 {
				fmt.Printf("No new outputs since %d iterations (%d remaining).\n", countNoNew, self.terminateAfterNTries-countNoNew)
			}
		}

		// Iterate over trees
		for treeIdx := 0; treeIdx < lastLen; treeIdx++ {
			treeLen := len(*inputTrees)
			tree := (*inputTrees)[treeIdx]

			if lastLen != treeLen {
				// TODO: is this required
				// Void the exit condition as there is more work todo
				lastLen = treeLen
				countNoNew = 0
			}

			newTree := trans.Transform(tree)
			newTreeStr := newTree.String()
			sumRam := md5.Sum([]byte(newTreeStr))
			sum := fmt.Sprintf("%x", sumRam)

			_, signatureExists := hashset[sum]
			if signatureExists {
				countNoNew++
				continue
			}

			countNoNew = 0
			*outputTrees = append(*outputTrees, newTree)

			hashset[sum] = struct{}{}

			file, err := os.Create(fmt.Sprintf("%s/%s.hms", self.outputDir, sum))
			if err != nil {
				panic(err.Error())
			}

			if _, err := file.WriteString(newTreeStr); err != nil {
				panic(err.Error())
			}
		}
	}
}

// func (self *Generator) GenThread(wg *sync.WaitGroup) {
// 	trans := NewTransformer(100, self.seed)
//
// 	countNoNew := 0
// 	for countNoNew < 100 {
// 		if GEN_VERBOSE {
// 			if countNoNew > 11 {
// 				fmt.Printf("%d ", countNoNew)
// 			} else if countNoNew > 10 {
// 				fmt.Printf("No new outputs since %d iterations.\n", countNoNew)
// 			}
// 		}
//
// 		trans.Out = ""
//
// 		newTrees := make([]ast.AnalyzedProgram, 0)
// 		tree := self.input
//
// 		for i := 0; i < self.passes; i++ {
// 			newTreeStr := tree.String()
// 			sumRam := md5.Sum([]byte(newTreeStr))
// 			sum := fmt.Sprintf("%x", sumRam)
//
// 			self.hashsetMtx.RLock()
// 			_, signatureExists := self.hashset[sum]
// 			self.hashsetMtx.RUnlock()
//
// 			if signatureExists {
// 				continue
// 			}
//
// 			tree = trans.Transform(tree)
// 			newTrees = append(newTrees, tree)
// 		}
//
// 		for _, newTree := range newTrees {
// 			newTreeStr := newTree.String()
// 			sumRam := md5.Sum([]byte(newTreeStr))
// 			sum := fmt.Sprintf("%x", sumRam)
//
// 			self.hashsetMtx.RLock()
// 			_, signatureExists := self.hashset[sum]
// 			self.hashsetMtx.RUnlock()
// 			if signatureExists {
// 				countNoNew++
// 				continue
// 			}
//
// 			countNoNew = 0
// 			self.hashsetMtx.Lock()
// 			self.hashset[sum] = struct{}{}
// 			self.hashsetMtx.Unlock()
//
// 			file, err := os.Create(fmt.Sprintf("%s/%s.hms", self.outputDir, sum))
// 			if err != nil {
// 				panic(err.Error())
// 			}
//
// 			if _, err := file.WriteString(newTreeStr); err != nil {
// 				panic(err.Error())
// 			}
// 		}
// 	}
//
// 	wg.Done()
// }
