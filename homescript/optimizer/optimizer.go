package main

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

func (o *Optimizer) Analyze(
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

	return ast.AnalyzedProgram{
		Imports:    module.Imports,
		Types:      module.Types,
		Singletons: module.Singletons,
		ImplBlocks: module.ImplBlocks,
		Globals:    module.Globals,
		Functions:  module.Functions,
	}
}

func (o *Optimizer) functions(node ast.AnalyzedFunctionDefinition) ast.AnalyzedFunctionDefinition {
	// TODO: optimize params
	newBlock := o.block(node.Body)

	newParams := ast.AnalyzedFunctionParams{
		List: make([]ast.AnalyzedFnParam, 0),
		Span: errors.Span{},
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

func (o *Optimizer) block(node ast.AnalyzedBlock, pushNewScope bool) ast.AnalyzedBlock {
	// newStatements := make([]ast.AnalyzedStatement, 0)
	//
	// var expression ast.AnalyzedExpression
	//
	// for _, statement := range node.Statements {
	// 	if statement.Type().Kind() == ast.NeverTypeKind {
	//
	// 	}
	// 	newStatements = append(newStatements, o.statement(statement))
	// }
	//
	// return ast.AnalyzedBlock{
	// 	Statements: newStatements,
	// 	Expression: expression,
	// 	Range:      node.Range,
	// 	ResultType: node.ResultType,
	// }

	// push a new scope if required
	if pushNewScope {
		panic("not implemented")
		// self.currentModule.pushScope()
	}

	// analyze statements
	statements := make([]ast.AnalyzedStatement, 0)
	var unreachableSpan *errors.Span = nil
	warnedUnreachable := false

	for _, statement := range node.Statements {
		newStatement := o.statement(statement)

		// if the previous statement had the never type, warn that this statement is unreachable
		if unreachableSpan != nil && !warnedUnreachable {
			warnedUnreachable = true
			self.warn(
				"Unreachable statement",
				nil,
				newStatement.Span(),
			)
			self.hint(
				"Any code following this statement is unreachable",
				nil,
				*unreachableSpan,
			)
		}

		// detect if this statement renders all following unreachable
		if unreachableSpan == nil && newStatement.Type().Kind() == ast.NeverTypeKind {
			span := newStatement.Span()
			unreachableSpan = &span
		}

		statements = append(statements, newStatement)
	}

	// analyze optional trailing expression
	var trailingExpr ast.AnalyzedExpression = nil

	if node.Expression != nil {
		trailingExpr = self.expression(node.Expression)

		if unreachableSpan != nil && !warnedUnreachable {
			self.warn(
				"Unreachable expression",
				nil,
				node.Expression.Span(),
			)
			self.hint(
				"Any code following this statement is unreachable",
				nil,
				*unreachableSpan,
			)
		}
	}

	// if this block diverges, the entrire block has the never type
	// otherwise, use the type of the trailing expression
	var resultType ast.Type
	if unreachableSpan != nil {
		resultType = ast.NewNeverType()
	} else if trailingExpr != nil {
		resultType = trailingExpr.Type()
	} else {
		resultType = ast.NewNullType(node.Range)
	}

	// pop scope if one was pushed at the beginning
	if pushNewScope {
		self.dropScope(true)
	}

	return ast.AnalyzedBlock{
		Statements: statements,
		Expression: trailingExpr,
		ResultType: resultType,
		Range:      node.Range,
	}
}

func (o *Optimizer) statement(ast.AnalyzedStatement) ast.AnalyzedStatement {
	panic("TODO")
	return nil
}
