package homescript

import (
	"context"
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
	"github.com/smarthome-go/homescript/v3/homescript/lexer"
	"github.com/smarthome-go/homescript/v3/homescript/parser"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

func Parse(code string, filename string) (pAst.Program, []errors.Error, *errors.Error) {
	lex := lexer.NewLexer(code, filename)
	parser := parser.NewParser(lex, filename)
	return parser.Parse()
}

type InputProgram struct {
	ProgramText string
	Filename    string
}

func Analyze(
	input InputProgram,
	scopeAdditions map[string]analyzer.Variable,
	host analyzer.HostProvider,
	mainFunctionShallExist bool,
) (modules map[string]ast.AnalyzedProgram, diagnostics []diagnostic.Diagnostic, syntaxErrors []errors.Error) {
	lex := lexer.NewLexer(input.ProgramText, input.Filename)
	parser := parser.NewParser(lex, input.Filename)
	parsedTree, nonCriticalErrors, criticalError := parser.Parse()

	syntaxErrors = append(syntaxErrors, nonCriticalErrors...)
	if criticalError != nil {
		syntaxErrors = append(syntaxErrors, *criticalError)
		return nil, nil, syntaxErrors
	}

	analyzer := analyzer.NewAnalyzer(host, scopeAdditions)
	analyzedModules, diagnostics, analyzedSyntaxErrors := analyzer.Analyze(parsedTree, mainFunctionShallExist)
	syntaxErrors = append(syntaxErrors, analyzedSyntaxErrors...)
	return analyzedModules, diagnostics, syntaxErrors
}

func Run(
	callStackLimitSize uint,
	inputModules map[string]ast.AnalyzedProgram,
	entryModule string,
	executor value.Executor,
	scopeAdditions map[string]value.Value,
	cancelCtx *context.Context,
) *value.Interrupt {
	interpreter := interpreter.NewInterpreter(
		callStackLimitSize,
		executor,
		inputModules,
		scopeAdditions,
		cancelCtx,
	)
	if _, found := inputModules[entryModule]; !found {
		panic(fmt.Sprintf("Entry module `%s` is not among input modules", entryModule))
	}
	return interpreter.Execute(entryModule)
}
