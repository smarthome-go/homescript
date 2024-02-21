package fuzzer

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

func (self *Transformer) infixExpr(node ast.AnalyzedInfixExpression, needsToBeStatic bool) []ast.AnalyzedExpression {
	variants := make([]ast.AnalyzedExpression, 0)
	variants = append(variants, node)

	switch node.Operator {
	case pAst.PlusInfixOperator, pAst.MinusInfixOperator:
		var topLevelOp pAst.InfixOperator

		if node.Operator == pAst.PlusInfixOperator {
			topLevelOp = pAst.MinusInfixOperator
		} else {
			topLevelOp = pAst.PlusInfixOperator
		}

		switch node.Lhs.Type().Kind() {
		case ast.IntTypeKind, ast.FloatTypeKind:
			// If the operator is plus or mul / swap the operands
			if node.Operator == pAst.PlusInfixOperator {
				variants = append(variants, ast.AnalyzedInfixExpression{
					Lhs:        node.Rhs,
					Rhs:        node.Lhs,
					Operator:   node.Operator,
					ResultType: node.ResultType,
					Range:      node.Range,
				})
			}

			variants = append(variants, ast.AnalyzedInfixExpression{
				Lhs: node.Lhs,
				Rhs: ast.AnalyzedPrefixExpression{
					Operator:   ast.MinusPrefixOperator,
					Base:       node.Rhs,
					ResultType: ast.NewIntType(node.Range),
					Range:      node.Range,
				},
				Operator:   topLevelOp,
				ResultType: ast.NewIntType(node.Range),
				Range:      node.Range,
			})
		default:
			variants = append(variants, node)
		}
	case pAst.MultiplyInfixOperator:
		// Swap the operands
		if node.Lhs.Type().Kind() == ast.IntTypeKind || node.Lhs.Type().Kind() == ast.FloatTypeKind {
			variants = append(variants, ast.AnalyzedInfixExpression{
				Lhs:        node.Rhs,
				Rhs:        node.Lhs,
				Operator:   node.Operator,
				ResultType: node.ResultType,
				Range:      node.Range,
			})
		}

		// If the expression is required to be static, terminate here
		if needsToBeStatic {
			return variants
		}

		// These `unrolliing` hacks only work for integers.
		if node.Lhs.Type().Kind() != ast.IntTypeKind || node.Rhs.Type().Kind() != ast.IntTypeKind {
			return variants
		}

		// Method 1: automatic unrolling during runtime.
		lhsInitIdent := pAst.NewSpannedIdent("lhs_init", node.Range)
		resultIdent := pAst.NewSpannedIdent("mul_res", node.Range)
		countIdent := pAst.NewSpannedIdent("mul_count", node.Range)

		var resultInitExpr ast.AnalyzedExpression
		if node.Lhs.Type().Kind() == ast.IntTypeKind {
			resultInitExpr = ast.AnalyzedExpression(
				ast.AnalyzedIntLiteralExpression{
					Value: 0,
					Range: node.Range,
				},
			)
		} else {
			resultInitExpr = ast.AnalyzedExpression(
				ast.AnalyzedFloatLiteralExpression{
					Value: 0.0,
					Range: node.Range,
				},
			)
		}

		variants = append(variants, ast.AnalyzedBlockExpression{
			Block: ast.AnalyzedBlock{
				Statements: []ast.AnalyzedStatement{
					ast.AnalyzedLetStatement{
						Ident:                      lhsInitIdent,
						Expression:                 node.Lhs,
						VarType:                    node.Lhs.Type(),
						NeedsRuntimeTypeValidation: false,
						OptType:                    nil,
						Range:                      node.Range,
					},

					ast.AnalyzedLetStatement{
						Ident:                      resultIdent,
						Expression:                 resultInitExpr,
						VarType:                    node.ResultType,
						NeedsRuntimeTypeValidation: false,
						OptType:                    nil,
						Range:                      node.Range,
					},

					ast.AnalyzedLetStatement{
						Ident: countIdent,
						Expression: ast.AnalyzedIntLiteralExpression{
							Value: 0,
							Range: node.Range,
						},
						VarType:                    ast.NewIntType(node.Range),
						NeedsRuntimeTypeValidation: false,
						OptType:                    nil,
						Range:                      node.Range,
					},

					ast.AnalyzedWhileStatement{
						Condition: ast.AnalyzedInfixExpression{
							Lhs: ast.AnalyzedIdentExpression{
								Ident:      countIdent,
								ResultType: ast.NewNullType(node.Range),
								IsGlobal:   false,
								IsFunction: false,
							},
							Rhs:        node.Rhs,
							Operator:   pAst.LessThanInfixOperator,
							ResultType: ast.NewBoolType(node.Range),
							Range:      node.Range,
						},
						Body: ast.AnalyzedBlock{
							Statements: []ast.AnalyzedStatement{
								ast.AnalyzedExpressionStatement{
									Expression: ast.AnalyzedAssignExpression{
										Lhs: ast.AnalyzedIdentExpression{
											Ident:      resultIdent,
											ResultType: node.ResultType,
											IsGlobal:   false,
											IsFunction: false,
										},
										Rhs: ast.AnalyzedIdentExpression{
											Ident:      lhsInitIdent,
											ResultType: node.ResultType,
											IsGlobal:   false,
											IsFunction: false,
										},
										Operator:   pAst.PlusAssignOperatorKind,
										ResultType: ast.NewNullType(node.Range),
										Range:      node.Range,
									},
									Range: node.Range,
								},
								ast.AnalyzedExpressionStatement{
									Expression: ast.AnalyzedAssignExpression{
										Lhs: ast.AnalyzedIdentExpression{
											Ident:      countIdent,
											ResultType: ast.NewIntType(node.Range),
											IsGlobal:   false,
											IsFunction: false,
										},
										Rhs: ast.AnalyzedIntLiteralExpression{
											Value: 1,
											Range: node.Range,
										},
										Operator:   pAst.PlusAssignOperatorKind,
										ResultType: ast.NewNullType(node.Range),
										Range:      node.Range,
									},
									Range: node.Range,
								},
							},
							Expression: nil,
							Range:      node.Range,
							ResultType: ast.NewNullType(node.Range),
						},
						NeverTerminates: false,
						Range:           node.Range,
					},
				},
				Expression: ast.AnalyzedIdentExpression{
					Ident:      resultIdent,
					ResultType: node.ResultType,
					IsGlobal:   false,
					IsFunction: false,
				},
				Range:      node.Range,
				ResultType: node.ResultType,
			},
		})

		// Method 2: static unrolling during compile time.
		if !node.Rhs.Constant() {
			return variants
		}

		// TODO: implement using loops which are automatically unrolled at this point
	case pAst.DivideInfixOperator:
		variants = append(variants, node)
	case pAst.ModuloInfixOperator:
		variants = append(variants, node)
	case pAst.PowerInfixOperator, pAst.ShiftLeftInfixOperator, pAst.ShiftRightInfixOperator, pAst.BitOrInfixOperator, pAst.BitAndInfixOperator, pAst.BitXorInfixOperator:
		// Cannot really display these operations differently
		variants = append(variants, node)
	case pAst.LogicalOrInfixOperator:
		variants = append(variants, node)
		// TODO: transpile into if-else
	case pAst.LogicalAndInfixOperator:
		variants = append(variants, node)
		// TODO: transpile into if-else
	case pAst.EqualInfixOperator, pAst.NotEqualInfixOperator:
		variants = append(variants, node)

		var innerOp pAst.InfixOperator
		if node.Operator == pAst.EqualInfixOperator {
			innerOp = pAst.NotEqualInfixOperator
		} else {
			innerOp = pAst.EqualInfixOperator
		}

		variants = append(variants, ast.AnalyzedPrefixExpression{
			Operator: ast.NegatePrefixOperator,
			Base: ast.AnalyzedGroupedExpression{
				Inner: ast.AnalyzedInfixExpression{
					Lhs:        self.Expression(node.Lhs, needsToBeStatic),
					Rhs:        self.Expression(node.Rhs, needsToBeStatic),
					Operator:   innerOp,
					ResultType: ast.NewBoolType(node.Range),
					Range:      node.Range,
				},
				Range: node.Range,
			},
			ResultType: ast.NewBoolType(node.Range),
			Range:      node.Range,
		})

		variants = append(variants, ast.AnalyzedPrefixExpression{
			Operator: ast.NegatePrefixOperator,
			Base: ast.AnalyzedGroupedExpression{
				Inner: ast.AnalyzedGroupedExpression{
					Inner: ast.AnalyzedInfixExpression{
						Lhs:        self.Expression(node.Lhs, needsToBeStatic),
						Rhs:        self.Expression(node.Rhs, needsToBeStatic),
						Operator:   innerOp,
						ResultType: ast.NewBoolType(node.Range),
						Range:      node.Range,
					},
					Range: node.Range,
				},
				Range: node.Range,
			},
			ResultType: ast.NewBoolType(node.Range),
			Range:      node.Range,
		})
	case pAst.LessThanInfixOperator, pAst.GreaterThanInfixOperator, pAst.LessThanEqualInfixOperator, pAst.GreaterThanEqualInfixOperator:
		variants = append(variants, node)

		reversed := map[pAst.InfixOperator]pAst.InfixOperator{
			pAst.LessThanInfixOperator:         pAst.GreaterThanInfixOperator,
			pAst.GreaterThanInfixOperator:      pAst.LessThanInfixOperator,
			pAst.LessThanEqualInfixOperator:    pAst.GreaterThanEqualInfixOperator,
			pAst.GreaterThanEqualInfixOperator: pAst.LessThanEqualInfixOperator,
		}

		variants = append(variants, ast.AnalyzedInfixExpression{
			Lhs:        self.Expression(node.Rhs, needsToBeStatic),
			Rhs:        self.Expression(node.Lhs, needsToBeStatic),
			Operator:   reversed[node.Operator],
			ResultType: node.ResultType,
			Range:      node.Range,
		})
	}

	return variants
}
