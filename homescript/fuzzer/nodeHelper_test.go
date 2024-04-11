package fuzzer

import (
	"testing"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/stretchr/testify/assert"
)

// Previously, there was stuff going on
func TestRegressionIfNode(t *testing.T) {
	node := ast.AnalyzedIfExpression{
		Condition: ast.AnalyzedBoolLiteralExpression{
			Value: true,
			Range: errors.Span{},
		},
		ThenBlock: ast.AnalyzedBlock{},
		ElseBlock: &ast.AnalyzedBlock{
			Statements: []ast.AnalyzedStatement{
				ast.AnalyzedExpressionStatement{
					Expression: ast.AnalyzedBlockExpression{
						Block: ast.AnalyzedBlock{
							Statements: []ast.AnalyzedStatement{
								ast.AnalyzedBreakStatement{
									Range: errors.Span{},
								},
							},
							Expression: nil,
							Range:      errors.Span{},
							ResultType: nil,
						},
					},
					Range: errors.Span{},
				},
			},
			Expression: nil,
			Range:      errors.Span{},
			ResultType: nil,
		},
		ResultType: nil,
		Range:      errors.Span{},
	}

	// node = ast.AnalyzedIfExpression{
	// 	Condition: ast.AnalyzedBoolLiteralExpression{
	// 		Value: true,
	// 		Range: errors.Span{},
	// 	},
	// 	ThenBlock: ast.AnalyzedBlock{},
	// 	ElseBlock: &ast.AnalyzedBlock{
	// 		Statements: []ast.AnalyzedStatement{
	// 			ast.AnalyzedBreakStatement{
	// 				Range: errors.Span{},
	// 			},
	// 		},
	// 		Expression: nil,
	// 		Range:      errors.Span{},
	// 		ResultType: nil,
	// 	},
	// 	ResultType: nil,
	// 	Range:      errors.Span{},
	// }

	trans := NewTransformer(0)
	assert.True(t, trans.exprCanControlLoop(node))
}
