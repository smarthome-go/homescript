package fuzzer

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

// Output is only a linear array, as the actual combination is chosen randomly.
// Otherwise, the size of the output tree would explode.
func (self *Transformer) Statements(input []ast.AnalyzedStatement) []ast.AnalyzedStatement {
	outStmts := make([]ast.AnalyzedStatement, 0)

	for _, stmt := range input {
		outStmts = append(outStmts, self.Statement(stmt))
	}

	return outStmts
}

func (self *Transformer) Block(node ast.AnalyzedBlock) ast.AnalyzedBlock {
	stmts := self.Statements(node.Statements)

	var outputExpression ast.AnalyzedExpression = nil

	if node.Expression != nil {
		outputExpression = self.Expression(node.Expression, false)
	}

	return ast.AnalyzedBlock{
		Statements: stmts,
		Expression: outputExpression,
		Range:      node.Range,
		ResultType: node.ResultType,
	}
}

func (self *Transformer) Statement(node ast.AnalyzedStatement) ast.AnalyzedStatement {
	variants := self.stmtVariants(node)
	selected := ChoseRandom[ast.AnalyzedStatement](variants, self.randSource)
	return selected
}

func (self *Transformer) stmtVariants(node ast.AnalyzedStatement) []ast.AnalyzedStatement {
	output := make([]ast.AnalyzedStatement, 0)

	switch node.Kind() {
	case ast.TypeDefinitionStatementKind:
		panic("TODO")
	case ast.LetStatementKind:
		node := node.(ast.AnalyzedLetStatement)
		// TODO: make two definitions out of one
		output = append(output, ast.AnalyzedLetStatement{
			Ident:                      node.Ident,
			Expression:                 self.Expression(node.Expression, false),
			VarType:                    node.VarType,
			NeedsRuntimeTypeValidation: node.NeedsRuntimeTypeValidation,
			OptType:                    node.OptType,
			Range:                      node.Range,
		})
	case ast.ReturnStatementKind:
		nodeTemp := node.(ast.AnalyzedReturnStatement)

		var expr ast.AnalyzedExpression
		if nodeTemp.ReturnValue != nil {
			expr = self.Expression(nodeTemp.ReturnValue, false)
		}

		node := ast.AnalyzedReturnStatement{
			ReturnValue: expr,
			Range:       nodeTemp.Range,
		}

		// output = append(output, node)
		output = append(output, ast.AnalyzedStatement(
			ast.AnalyzedExpressionStatement{
				Expression: ast.AnalyzedBlockExpression{
					Block: ast.AnalyzedBlock{
						Statements: []ast.AnalyzedStatement{node},
						Expression: nil,
						Range:      node.Span(),
						ResultType: ast.NewNeverType(),
					},
				},
				Range: node.Span(),
			}))
		output = append(output, ast.AnalyzedWhileStatement{
			Condition: ast.AnalyzedBlockExpression{
				Block: ast.AnalyzedBlock{
					Statements: []ast.AnalyzedStatement{node},
					Expression: nil,
					Range:      node.Span(),
					ResultType: ast.NewNeverType(),
				},
			},
			Body: ast.AnalyzedBlock{
				Statements: make([]ast.AnalyzedStatement, 0),
				Expression: nil,
				Range:      node.Span(),
				ResultType: ast.NewNullType(node.Span()),
			},
			NeverTerminates: false,
			Range:           node.Span(),
		})
	case ast.BreakStatementKind:
		output = append(output, node)
		output = append(output, ast.AnalyzedStatement(
			ast.AnalyzedExpressionStatement{
				Expression: ast.AnalyzedBlockExpression{
					Block: ast.AnalyzedBlock{
						Statements: []ast.AnalyzedStatement{node},
						Expression: nil,
						Range:      node.Span(),
						ResultType: ast.NewNeverType(),
					},
				},
				Range: node.Span(),
			}))
		output = append(output, ast.AnalyzedWhileStatement{
			Condition: ast.AnalyzedBlockExpression{
				Block: ast.AnalyzedBlock{
					Statements: []ast.AnalyzedStatement{node},
					Expression: nil,
					Range:      node.Span(),
					ResultType: ast.NewNeverType(),
				},
			},
			Body: ast.AnalyzedBlock{
				Statements: make([]ast.AnalyzedStatement, 0),
				Expression: nil,
				Range:      node.Span(),
				ResultType: ast.NewNeverType(),
			},
			NeverTerminates: true,
			Range:           node.Span(),
		})
	case ast.ContinueStatementKind:
		// output = append(output, node)
		output = append(output, ast.AnalyzedStatement(
			ast.AnalyzedExpressionStatement{
				Expression: ast.AnalyzedBlockExpression{
					Block: ast.AnalyzedBlock{
						Statements: []ast.AnalyzedStatement{node},
						Expression: nil,
						Range:      node.Span(),
						ResultType: ast.NewNeverType(),
					},
				},
				Range: node.Span(),
			}))
		output = append(output, ast.AnalyzedWhileStatement{
			Condition: ast.AnalyzedBlockExpression{
				Block: ast.AnalyzedBlock{
					Statements: []ast.AnalyzedStatement{node},
					Expression: nil,
					Range:      node.Span(),
					ResultType: ast.NewNeverType(),
				},
			},
			Body: ast.AnalyzedBlock{
				Statements: make([]ast.AnalyzedStatement, 0),
				Expression: nil,
				Range:      node.Span(),
				ResultType: ast.NewNeverType(),
			},
			NeverTerminates: true,
			Range:           node.Span(),
		})
	case ast.LoopStatementKind:
		node := node.(ast.AnalyzedLoopStatement)

		output = append(output, ast.AnalyzedLoopStatement{
			Body:            self.Block(node.Body),
			NeverTerminates: false,
			Range:           node.Span(),
		})
		output = append(output, ast.AnalyzedWhileStatement{
			Condition: ast.AnalyzedBoolLiteralExpression{
				Value: true,
				Range: node.Range,
			},
			Body:            self.Block(node.Body),
			NeverTerminates: false,
			Range:           node.Span(),
		})
	case ast.WhileStatementKind:
		node := node.(ast.AnalyzedWhileStatement)
		output = append(output, self.WhileStmtAsLoop(node)...)
		output = append(output, ast.AnalyzedWhileStatement{
			Condition:       self.Expression(node.Condition, false),
			Body:            self.Block(node.Body),
			NeverTerminates: false,
			Range:           node.Range,
		})
	case ast.ForStatementKind:
		node := node.(ast.AnalyzedForStatement)
		output = append(output, ast.AnalyzedForStatement{
			Identifier:      node.Identifier,
			IterExpression:  self.Expression(node.IterExpression, false),
			IterVarType:     node.IterVarType,
			Body:            self.Block(node.Body),
			NeverTerminates: node.NeverTerminates,
			Range:           node.Range,
		})
	case ast.ExpressionStatementKind:
		exprOut := ast.AnalyzedExpressionStatement{
			Expression: self.Expression(node.(ast.AnalyzedExpressionStatement).Expression, false),
			Range:      node.Span(),
		}
		output = append(output, exprOut)
	}

	output = append(output, node)

	// The following transformations will create a new scope for the statement, rendering let-statements useless.
	if node.Kind() == ast.LetStatementKind {
		return output
	}

	// Always true `if`

	// TODO: maybe check if the node is NEVER? (if this breaks)
	output = append(output, ast.AnalyzedExpressionStatement{
		Expression: ast.AnalyzedIfExpression{
			Condition: ast.AnalyzedBoolLiteralExpression{
				Value: true,
				Range: node.Span(),
			},
			ThenBlock: ast.AnalyzedBlock{
				Statements: []ast.AnalyzedStatement{node},
				Expression: nil,
				Range:      node.Span(),
				ResultType: node.Type(),
			},
			ElseBlock:  nil,
			ResultType: node.Type(),
			Range:      node.Span(),
		},
		Range: node.Span(),
	})

	// TODO: also include matches and maybe tru-catch to obfuscate

	// If the current statement is something like `continue` / `break`, do not wrap it in a loop
	// if node.Kind() != ast.ReturnStatementKind && node.Type().Kind() == ast.NeverTypeKind {
	// 	self.Out += fmt.Sprintf("Not further transforming stmt with !: `%s`\n", node.String())
	// 	return output
	// }

	// Detect whether the current statement is able to control a parent loop
	// If so, do not apply loop obfuscations on top of this node.
	if self.stmtCanControlLoop(node) {
		self.Out += "Statement contains loop control code, not applying loop obfuscations"
		return output
	}

	output = append(output, self.IterOnceWhileLoop(node))

	// Iter-once `for` loop using ranges
	output = append(output, ast.AnalyzedForStatement{
		Identifier: pAst.NewSpannedIdent("_i", node.Span()),
		IterExpression: ast.AnalyzedRangeLiteralExpression{
			Start: ast.AnalyzedIntLiteralExpression{
				Value: 0,
				Range: node.Span(),
			},
			End: ast.AnalyzedIntLiteralExpression{
				Value: 1,
				Range: node.Span(),
			},
			Range: node.Span(),
		},
		IterVarType: ast.NewRangeType(node.Span()),
		Body: ast.AnalyzedBlock{
			Statements: []ast.AnalyzedStatement{node},
			Expression: nil,
			Range:      node.Span(),
			ResultType: ast.NewNullType(node.Span()),
		},
		NeverTerminates: false,
		Range:           node.Span(),
	})

	return output
}

func (self *Transformer) IterOnceWhileLoop(node ast.AnalyzedStatement) ast.AnalyzedStatement {
	// Iter-once while-loop
	whileLoopObfuscateIdent := "count_once"
	return ast.AnalyzedExpressionStatement{
		Expression: ast.AnalyzedBlockExpression{
			Block: ast.AnalyzedBlock{
				Statements: []ast.AnalyzedStatement{
					ast.AnalyzedLetStatement{
						Ident: pAst.NewSpannedIdent(whileLoopObfuscateIdent, node.Span()),
						Expression: ast.AnalyzedIntLiteralExpression{
							Value: 0,
							Range: node.Span(),
						},
						VarType:                    ast.NewIntType(node.Span()),
						OptType:                    nil,
						NeedsRuntimeTypeValidation: false,
						Range:                      node.Span(),
					},
					ast.AnalyzedWhileStatement{
						Condition: ast.AnalyzedInfixExpression{
							Lhs: ast.AnalyzedIdentExpression{
								Ident:      pAst.NewSpannedIdent(whileLoopObfuscateIdent, node.Span()),
								ResultType: ast.NewIntType(node.Span()),
								IsGlobal:   true,
								IsFunction: false,
							},
							Rhs: ast.AnalyzedIntLiteralExpression{
								Value: 1,
								Range: node.Span(),
							},
							Operator:   pAst.LessThanInfixOperator,
							ResultType: ast.NewBoolType(node.Span()),
							Range:      node.Span(),
						},
						Body: ast.AnalyzedBlock{
							Statements: []ast.AnalyzedStatement{
								// Increment the counter by one at the start of the loop
								ast.AnalyzedExpressionStatement{
									Expression: ast.AnalyzedAssignExpression{
										Lhs: ast.AnalyzedIdentExpression{
											Ident:      pAst.NewSpannedIdent(whileLoopObfuscateIdent, node.Span()),
											ResultType: ast.NewIntType(node.Span()),
											IsGlobal:   false,
											IsFunction: false,
										},
										Rhs: ast.AnalyzedIntLiteralExpression{
											Value: 1,
											Range: node.Span(),
										},
										Operator:   pAst.PlusAssignOperatorKind,
										ResultType: ast.NewNullType(node.Span()),
										Range:      node.Span(),
									},
									Range: node.Span(),
								},
								// The actual statement
								node,
							},
							Expression: nil,
							Range:      node.Span(),
							ResultType: ast.NewNullType(node.Span()),
						},
						NeverTerminates: false,
						Range:           node.Span(),
					},
				},
				Expression: nil,
				Range:      node.Span(),
				ResultType: ast.NewNullType(node.Span()),
			},
		},
		Range: node.Span(),
	}
}

func (self *Transformer) WhileStmtAsLoop(node ast.AnalyzedWhileStatement) []ast.AnalyzedStatement {
	if node.Condition.Type().Kind() == ast.NeverTypeKind {
		// This is required in order to prevent putting a `break` into a new loop, which defeats the purpose
		return []ast.AnalyzedStatement{node}
	}

	body := self.Block(node.Body)

	stmts := append([]ast.AnalyzedStatement{
		ast.AnalyzedExpressionStatement{
			Expression: ast.AnalyzedIfExpression{
				Condition: ast.AnalyzedPrefixExpression{
					Operator: ast.NegatePrefixOperator,
					Base: ast.AnalyzedGroupedExpression{
						Inner: node.Condition,
						Range: node.Range,
					},
					ResultType: ast.NewBoolType(node.Range),
					Range:      node.Range,
				},
				ThenBlock: ast.AnalyzedBlock{
					Statements: []ast.AnalyzedStatement{
						ast.AnalyzedBreakStatement{
							Range: node.Range,
						},
					},
					Expression: nil,
					Range:      node.Range,
					ResultType: ast.NewNeverType(),
				},
				ElseBlock:  nil,
				ResultType: ast.NewNullType(node.Range),
				Range:      node.Range,
			},
			Range: node.Range,
		},
	}, body.Statements...)

	if body.Expression != nil {
		stmts = append(stmts, ast.AnalyzedExpressionStatement{
			Expression: body.Expression,
			Range:      node.Range,
		})
	}

	loopS0 := ast.AnalyzedLoopStatement{
		Body: ast.AnalyzedBlock{
			Statements: stmts,
			Expression: nil,
			Range:      node.Range,
			ResultType: ast.NewNullType(node.Range),
		},
		NeverTerminates: false,
		Range:           node.Range,
	}

	loopS1 := ast.AnalyzedLoopStatement{
		Body: ast.AnalyzedBlock{
			Statements: []ast.AnalyzedStatement{
				ast.AnalyzedExpressionStatement{
					Expression: ast.AnalyzedIfExpression{
						Condition: node.Condition,
						ThenBlock: self.Block(node.Body),
						ElseBlock: &ast.AnalyzedBlock{
							Statements: []ast.AnalyzedStatement{
								ast.AnalyzedBreakStatement{
									Range: node.Range,
								},
							},
							Expression: nil,
							Range:      node.Range,
							ResultType: ast.NewNeverType(),
						},
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
	}

	return []ast.AnalyzedStatement{
		loopS0,
		loopS1,
	}
}
