package fuzzer

import (
	"math/rand"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
)

// TODO: instead of the current, hacky implementation for wrapping statements inside of loops, detect for each statement that is going to be transformed, if it contains a break.
// If so, then do not transform it using these methods any more.

// NOTE: on spans:
// Most spans will be completely broken after the transformation.
// However, this is not relevant as the ast is only serialized to string before it is being compiled.
// Then, after a new parse, the spanss will be correct.
type Transformer struct {
	// Random source
	randSource rand.Source

	// Keeps track of how many ast nodes the transformewr already changed.
	modifications uint

	Out string
}

func NewTransformer(seed int64) Transformer {
	source := rand.NewSource(seed)

	return Transformer{
		randSource:    source,
		modifications: 0,
	}
}

func (self *Transformer) TransformPasses(tree ast.AnalyzedProgram, passes int) []ast.AnalyzedProgram {
	output := make([]ast.AnalyzedProgram, 0)

	for i := 0; i < passes; i++ {
		tree = self.Transform(tree)
		output = append(output, tree)
	}

	return output
}

func (self *Transformer) Transform(tree ast.AnalyzedProgram) ast.AnalyzedProgram {
	output := ast.AnalyzedProgram{
		Imports:   make([]ast.AnalyzedImport, 0),
		Types:     make([]ast.AnalyzedTypeDefinition, 0), // Should not transform these, stuff will break
		Globals:   make([]ast.AnalyzedLetStatement, 0),
		Functions: make([]ast.AnalyzedFunctionDefinition, 0),
		// TODO: remove this
		// Events:    make([]ast.AnalyzedFunctionDefinition, 0),
	}

	// Iterate over the imports and shuffle the order around
	ShuffleSlice(tree.Imports, self.randSource)

	// Iterate over the globals and shuffle their order around
	ShuffleSlice(tree.Globals, self.randSource)

	// Iterate over the functions and shuffle their order around
	ShuffleSlice(tree.Functions, self.randSource)

	// TODO: remove this
	// Iterate over the events and shuffle their order around
	// ShuffleSlice(tree.Events, self.randSource)

	// Iterate over the ast's functions in order to transform each one
	for _, fn := range tree.Functions {
		output.Functions = append(output.Functions, self.Function(fn))
	}

	// TODO: remove this
	// Iterate over the ast's events and transform each one
	// for _, eventFn := range tree.Events {
	// 	output.Events = append(output.Events, self.Function(eventFn))
	// }

	output.Types = tree.Types
	output.Imports = tree.Imports

	for _, glob := range tree.Globals {
		newGlob := ast.AnalyzedLetStatement{
			Ident:                      glob.Ident,
			Expression:                 self.Expression(glob.Expression, true),
			VarType:                    glob.VarType,
			NeedsRuntimeTypeValidation: glob.NeedsRuntimeTypeValidation,
			OptType:                    glob.OptType,
			Range:                      glob.Range,
		}

		output.Globals = append(output.Globals, newGlob)
	}

	return output
}

// Returns a random element from the input slice
func ChoseRandom[T any](input []T, randSource rand.Source) T {
	r := rand.New(randSource)
	chosenIndex := r.Intn(len(input))
	return input[chosenIndex]
}

// Returns the new, shuffled slice and the number of modifications applied
func ShuffleSlice[T any](input []T, randSource rand.Source) {
	if len(input) <= 1 {
		return
	}

	r := rand.New(randSource)

	r.Shuffle(len(input), func(i, j int) {
		input[i], input[j] = input[j], input[i]
	})
}

func (self *Transformer) Function(node ast.AnalyzedFunctionDefinition) ast.AnalyzedFunctionDefinition {
	return ast.AnalyzedFunctionDefinition{
		Ident:      node.Ident,
		Parameters: node.Parameters,
		ReturnType: node.ReturnType,
		Body:       self.Block(node.Body),
		Modifier:   node.Modifier,
		Range:      node.Range,
	}
}
