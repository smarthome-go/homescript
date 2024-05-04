package optimizer

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

//
// Optimizer helper functions.
//

func (o *Optimizer) error(message string, notes []string, span errors.Span) {
	o.diagnostics = append(o.diagnostics, diagnostic.Diagnostic{
		Level:   diagnostic.DiagnosticLevelError,
		Message: message,
		Notes:   notes,
		Span:    span,
	})
}

func (o *Optimizer) warn(message string, notes []string, span errors.Span) {
	o.diagnostics = append(o.diagnostics, diagnostic.Diagnostic{
		Level:   diagnostic.DiagnosticLevelWarning,
		Message: message,
		Notes:   notes,
		Span:    span,
	})
}

func (o *Optimizer) hint(message string, notes []string, span errors.Span) {
	o.diagnostics = append(o.diagnostics, diagnostic.Diagnostic{
		Level:   diagnostic.DiagnosticLevelHint,
		Message: message,
		Notes:   notes,
		Span:    span,
	})
}

//
// END helper.
//

type Optimizer struct {
	diagnostics []diagnostic.Diagnostic
}

func NewOptimizer() Optimizer {
	return Optimizer{
		diagnostics: []diagnostic.Diagnostic{},
	}
}

func (o *Optimizer) Optimize(
	analyzedModule map[string]ast.AnalyzedProgram,
) (
	modules map[string]ast.AnalyzedProgram,
	diagnostics []diagnostic.Diagnostic,
) {
	modulesOut := make(map[string]ast.AnalyzedProgram)

	for moduleName, module := range analyzedModule {
		modulesOut[moduleName] = o.analyzeModule(moduleName, module)
	}

	return modulesOut, o.diagnostics
}

func (o *Optimizer) analyzeModule(moduleName string, module ast.AnalyzedProgram) ast.AnalyzedProgram {
	functionsOut := make([]ast.AnalyzedFunctionDefinition, 0)

	for _, fn := range module.Functions {
		newFn := o.optimizeFn(fn)
		functionsOut = append(functionsOut, newFn)
	}

	return ast.AnalyzedProgram{
		Imports:    module.Imports,
		Types:      module.Types,
		Singletons: module.Singletons,
		ImplBlocks: module.ImplBlocks,
		Globals:    module.Globals,
		Functions:  functionsOut,
	}
}

func (o *Optimizer) optimizeFn(node ast.AnalyzedFunctionDefinition) ast.AnalyzedFunctionDefinition {
	newBlock := o.block(node.Body)

	// TODO: optimize the parameters
	newParams := ast.AnalyzedFunctionParams{
		List: make([]ast.AnalyzedFnParam, len(node.Parameters.List)),
		Span: errors.Span{},
	}

	// TODO: remove unused parameters
	for idx, param := range node.Parameters.List {
		newParams.List[idx] = param
	}

	return ast.AnalyzedFunctionDefinition{
		Ident:      node.Ident,
		Parameters: newParams,
		ReturnType: node.ReturnType,
		Body:       newBlock,
		Modifier:   node.Modifier,
		Annotation: node.Annotation,
		Range:      node.Range,
	}
}

func (o *Optimizer) block(node ast.AnalyzedBlock) ast.AnalyzedBlock {
	statements := make([]ast.AnalyzedStatement, 0)
	var unreachableSpan *errors.Span
	warnedUnreachable := false

	for _, statement := range node.Statements {
		newStatement := o.optStatement(statement)

		// If the previous statement had the never type, warn that this statement is unreachable.
		if unreachableSpan != nil && !warnedUnreachable {
			warnedUnreachable = true
			o.warn(
				"Unreachable statement",
				nil,
				newStatement.Span(),
			)
			o.hint(
				"Any code following this statement is unreachable",
				nil,
				*unreachableSpan,
			)
		}

		// Detect if this statement renders all following unreachable.
		if unreachableSpan == nil && newStatement.Type().Kind() == ast.NeverTypeKind {
			span := newStatement.Span()
			unreachableSpan = &span
		}

		if warnedUnreachable {
			continue
		}

		statements = append(statements, newStatement)
	}

	// Analyze optional trailing expression.
	var trailingExpr ast.AnalyzedExpression

	if node.Expression != nil {
		trailingExpr = o.optExpression(node.Expression)

		if unreachableSpan != nil && !warnedUnreachable {
			o.warn(
				"Unreachable expression",
				nil,
				node.Expression.Span(),
			)
			o.hint(
				"Any code following this statement is unreachable",
				nil,
				*unreachableSpan,
			)
		}
	}

	// If this block diverges, the entrire block has the never type
	// otherwise, use the type of the trailing expression.
	var resultType ast.Type

	switch {
	case unreachableSpan != nil:
		resultType = ast.NewNeverType()
	case trailingExpr != nil:
		resultType = trailingExpr.Type()
	default:
		resultType = ast.NewNullType(node.Range)
	}

	return ast.AnalyzedBlock{
		Statements: statements,
		Expression: trailingExpr,
		ResultType: resultType,
		Range:      node.Range,
	}
}

func (o *Optimizer) optStatement(node ast.AnalyzedStatement) ast.AnalyzedStatement {
	return node
}

func (o *Optimizer) optExpression(node ast.AnalyzedExpression) ast.AnalyzedExpression {
	return node
}
