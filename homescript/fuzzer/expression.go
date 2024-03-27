package fuzzer

import (
	"math/rand"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

func (self *Transformer) Expression(node ast.AnalyzedExpression, needsToBeStatic bool) ast.AnalyzedExpression {
	variants := self.expressionVariants(node, needsToBeStatic)
	selected := ChoseRandom[ast.AnalyzedExpression](variants, self.randSource)
	return selected
}

func (self *Transformer) expressionVariants(node ast.AnalyzedExpression, needsToBeStatic bool) []ast.AnalyzedExpression {
	variants := make([]ast.AnalyzedExpression, 0) // TODO: replace this by actual transformations being applied

	switch node.Kind() {
	case ast.UnknownExpressionKind:
		panic("Unsupported expression kind")
	case ast.IntLiteralExpressionKind:
		variants = append(variants, node)

		operators := []pAst.InfixOperator{pAst.PlusInfixOperator, pAst.MinusInfixOperator, pAst.MultiplyInfixOperator}
		inverseOperators := []pAst.InfixOperator{pAst.MinusInfixOperator, pAst.PlusInfixOperator, pAst.DivideInfixOperator}

		randomValues := []int64{42, 69, 4711}

		for idx, operator := range operators {
			r := rand.New(self.randSource)
			uselessValue := randomValues[r.Intn(len(randomValues))]

			variants = append(variants,
				ast.AnalyzedGroupedExpression{
					Inner: ast.AnalyzedInfixExpression{
						Lhs: ast.AnalyzedInfixExpression{
							Lhs: node,
							Rhs: ast.AnalyzedIntLiteralExpression{
								Value: int64(uselessValue),
								Range: node.Span(),
							},
							Operator:   operator,
							ResultType: ast.NewIntType(node.Span()),
							Range:      node.Span(),
						},
						Rhs: ast.AnalyzedIntLiteralExpression{
							Value: int64(uselessValue),
							Range: node.Span(),
						},
						Operator:   inverseOperators[idx],
						ResultType: ast.NewIntType(node.Span()),
						Range:      node.Span(),
					},
					Range: node.Span(),
				})
		}
	case ast.FloatLiteralExpressionKind:
		variants = append(variants, node)

		operators := []pAst.InfixOperator{pAst.PlusInfixOperator, pAst.MinusInfixOperator, pAst.MultiplyInfixOperator}
		inverseOperators := []pAst.InfixOperator{pAst.MinusInfixOperator, pAst.PlusInfixOperator, pAst.DivideInfixOperator}

		randomValues := []float64{42, 69, 4711}

		for idx, operator := range operators {
			r := rand.New(self.randSource)
			uselessValue := randomValues[r.Intn(len(randomValues))]

			variants = append(variants,
				ast.AnalyzedGroupedExpression{
					Inner: ast.AnalyzedInfixExpression{
						Lhs: ast.AnalyzedInfixExpression{
							Lhs: node,
							Rhs: ast.AnalyzedFloatLiteralExpression{
								Value: uselessValue,
								Range: node.Span(),
							},
							Operator:   operator,
							ResultType: ast.NewFloatType(node.Span()),
							Range:      node.Span(),
						},
						Rhs: ast.AnalyzedFloatLiteralExpression{
							Value: uselessValue,
							Range: node.Span(),
						},
						Operator:   inverseOperators[idx],
						ResultType: ast.NewFloatType(node.Span()),
						Range:      node.Span(),
					},
					Range: node.Span(),
				})
		}
	case ast.BoolLiteralExpressionKind:
		variants = append(variants, node)
		variants = append(variants, ast.AnalyzedPrefixExpression{
			Operator: ast.NegatePrefixOperator,
			Base: ast.AnalyzedPrefixExpression{
				Operator: ast.NegatePrefixOperator,
				Base: ast.AnalyzedGroupedExpression{
					Inner: node,
					Range: node.Span(),
				},
				ResultType: ast.NewBoolType(node.Span()),
				Range:      node.Span(),
			},
			ResultType: ast.NewBoolType(node.Span()),
			Range:      node.Span(),
		})
	case ast.StringLiteralExpressionKind:
		// TODO: use list, then `.join` the list
		variants = append(variants, node)
	case ast.IdentExpressionKind:
		// TODO: load then store in another variable
		variants = append(variants, node)
	case ast.NullLiteralExpressionKind:
		panic("TODO")
	case ast.NoneLiteralExpressionKind:
		panic("TODO")
	case ast.RangeLiteralExpressionKind:
		// TODO: also add a block in which there are two variables (lower upper)
		variants = append(variants, node)
	case ast.ListLiteralExpressionKind:
		// TODO: this can be extremely obfuscated.
		variants = append(variants, node)
	case ast.AnyObjectLiteralExpressionKind:
		panic("TODO")
	case ast.ObjectLiteralExpressionKind:
		// TODO: this can be obfuscated
		// For instance: swap around the order
		variants = append(variants, node)
	case ast.FunctionLiteralExpressionKind:
		node := node.(ast.AnalyzedFunctionLiteralExpression)
		newBody := self.Block(node.Body)
		variants = append(variants, ast.AnalyzedFunctionLiteralExpression{
			Parameters: node.Parameters,
			ParamSpan:  node.ParamSpan,
			ReturnType: node.ReturnType,
			Body:       newBody,
			Range:      node.Range,
		})
	case ast.GroupedExpressionKind:
		variants = append(variants, node)
		variants = append(variants, ast.AnalyzedGroupedExpression{
			Inner: node,
			Range: node.Span(),
		})
		variants = append(variants, ast.AnalyzedGroupedExpression{
			Inner: ast.AnalyzedBlockExpression{
				Block: ast.AnalyzedBlock{
					Statements: make([]ast.AnalyzedStatement, 0),
					Expression: node,
					Range:      node.Span(),
					ResultType: node.Type(),
				},
			},
			Range: node.Span(),
		})
	case ast.PrefixExpressionKind:
		node := node.(ast.AnalyzedPrefixExpression)
		variants = append(variants, node)
	case ast.InfixExpressionKind:
		// TODO: add transformers here
		node := node.(ast.AnalyzedInfixExpression)
		variants = append(variants, self.infixExpr(node, needsToBeStatic)...)
	case ast.AssignExpressionKind:
		variants = append(variants, node)
	case ast.CallExpressionKind:
		variants = append(variants, node)
	case ast.IndexExpressionKind:
		variants = append(variants, node)
	case ast.MemberExpressionKind:
		variants = append(variants, node)
	case ast.CastExpressionKind:
		node := node.(ast.AnalyzedCastExpression)
		variants = append(variants, node)
		variants = append(variants, ast.AnalyzedCastExpression{
			Base:   node,
			AsType: node.AsType, // Completely redundant cast
			Range:  node.Range,
		})
	case ast.BlockExpressionKind:
		variants = append(variants, node)
	case ast.IfExpressionKind:
		node := node.(ast.AnalyzedIfExpression)
		variants = append(variants, node)
		variants = append(variants, self.ifExpression(node)...)
	case ast.MatchExpressionKind:
		variants = append(variants, node)
	case ast.TryExpressionKind:
		variants = append(variants, node)
	}

	// TODO: add call-obfuscations

	return variants
}

func (self *Transformer) ifExpression(node ast.AnalyzedIfExpression) []ast.AnalyzedExpression {
	variants := make([]ast.AnalyzedExpression, 0)

	// Case one, just transform the expression
	var elseBlock1 *ast.AnalyzedBlock = nil
	if node.ElseBlock != nil {
		b := self.Block(*node.ElseBlock)
		elseBlock1 = &b
	}

	then := self.Block(node.ThenBlock)
	variants = append(variants, ast.AnalyzedIfExpression{
		Condition:  node.Condition,
		ThenBlock:  then,
		ElseBlock:  elseBlock1,
		ResultType: node.ResultType,
		Range:      node.Range,
	})

	// Case two, invert the condition
	elseBlock := self.Block(node.ThenBlock)

	thenblock := ast.AnalyzedBlock{
		Statements: make([]ast.AnalyzedStatement, 0),
		Expression: nil,
		Range:      node.Range,
		ResultType: ast.NewNullType(node.Range),
	}
	if node.ElseBlock != nil {
		thenblock = self.Block(*node.ElseBlock)
	}

	variants = append(variants, ast.AnalyzedIfExpression{
		Condition: ast.AnalyzedPrefixExpression{
			Operator: ast.NegatePrefixOperator,
			Base: ast.AnalyzedGroupedExpression{
				Inner: self.Expression(node.Condition, false),
				Range: node.Range,
			},
			ResultType: ast.NewBoolType(node.Range),
			Range:      node.Range,
		},
		ThenBlock:  thenblock,
		ElseBlock:  &elseBlock,
		ResultType: node.ResultType,
		Range:      node.Range,
	})

	return variants
}
