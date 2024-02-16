package fuzzer

import (
	"crypto/md5"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

type Generator struct {
	onOutput             func(tree ast.AnalyzedProgram, treeStr string, hashSum string) error
	seed                 int64
	input                ast.AnalyzedProgram
	passes               uint
	terminateAfterNTries uint
	outputLimitPerPass   uint
	verbose              bool
}

func NewGenerator(
	input ast.AnalyzedProgram,
	onOutput func(ast.AnalyzedProgram, string, string) error,
	seed int64,
	passes uint,
	terminateAfterNTries uint,
	outputSizeLimit uint,
	verbose bool,
) Generator {
	return Generator{
		onOutput:             onOutput,
		seed:                 seed,
		input:                input,
		passes:               passes,
		terminateAfterNTries: terminateAfterNTries,
		outputLimitPerPass:   outputSizeLimit,
		verbose:              verbose,
	}
}

func ChunkInput[T any](input []T, chunkSize uint) [][]T {
	var chunks [][]T
	for {
		if len(input) == 0 {
			break
		}

		// Necessary check to avoid slicing beyond slice capacity
		if uint(len(input)) < chunkSize {
			chunkSize = uint(len(input))
		}

		chunks = append(chunks, input[0:chunkSize])
		input = input[chunkSize:]
	}

	return chunks
}

func (self *Generator) Gen() {
	startAll := time.Now()

	passResults := make([][]ast.AnalyzedProgram, self.passes+1)
	passResults[0] = []ast.AnalyzedProgram{self.input}

	for passIndex := 0; passIndex < int(self.passes); passIndex++ {
		start := time.Now()

		if len(passResults[passIndex]) < 100 {
			log.Printf("Executing pass %d single threaded.\n", passIndex)
			wg := sync.WaitGroup{}
			wg.Add(1)
			self.Pass(
				&wg,
				&passResults[passIndex],
				&passResults[passIndex+1],
				self.outputLimitPerPass,
			)

			log.Printf("\n=== Pass %d duration: %v for input size %d\n", passIndex, time.Since(start), len(passResults[passIndex]))
			continue
		}

		inputSize := uint(len(passResults[passIndex]))
		numCpu := uint(runtime.NumCPU())
		chunkSize := (inputSize + numCpu - 1) / numCpu

		inputChunks := ChunkInput[ast.AnalyzedProgram](passResults[passIndex], chunkSize)
		numChunks := uint(len(inputChunks))
		outputChunks := make([][]ast.AnalyzedProgram, numChunks)

		if numChunks > numCpu {
			panic("Illegal state")
		}

		wg := sync.WaitGroup{}

		for chunkIndex := 0; uint(chunkIndex) < numChunks; chunkIndex++ {
			log.Printf("Spawning worker %d for pass %d...\n", chunkIndex, passIndex)

			wg.Add(1)
			go self.Pass(
				&wg,
				&inputChunks[chunkIndex],
				&outputChunks[chunkIndex],
				self.outputLimitPerPass/numChunks,
			)
		}

		log.Println("Waiting...")
		wg.Wait()
		log.Println("Wait finished")

		// Slice the output chunks together again
		for chunkIndex := 0; uint(chunkIndex) < numChunks; chunkIndex++ {
			passResults[passIndex+1] = append(passResults[passIndex+1], outputChunks[chunkIndex]...)
		}

		log.Printf("\n=== Pass %d duration: %v for input size %d\n", passIndex, time.Since(start), len(passResults[passIndex]))
	}

	if self.verbose {
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

		log.Printf("\nFuzz: %v, generated: %d\n", time.Since(startAll), generated)
	}
}

func (self *Generator) Pass(
	wg *sync.WaitGroup,
	inputTrees *[]ast.AnalyzedProgram,
	outputTrees *[]ast.AnalyzedProgram,
	outputSizeLimit uint,
) {
	defer wg.Done()

	hashset := make(map[string]struct{})
	var countNoNew uint = 0

	trans := NewTransformer(self.seed)

	lastLen := len(*inputTrees)

	for (countNoNew < self.terminateAfterNTries || self.terminateAfterNTries <= 0) && uint(len(hashset)) <= outputSizeLimit {
		if self.verbose {
			if countNoNew > 11 {
				log.Printf("%d (%d remaining) ", countNoNew, self.terminateAfterNTries-countNoNew)
			} else if countNoNew > 10 {
				log.Printf("No new outputs since %d iterations (%d remaining).\n", countNoNew, self.terminateAfterNTries-countNoNew)
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

			if err := self.onOutput(newTree, newTreeStr, sum); err != nil {
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
