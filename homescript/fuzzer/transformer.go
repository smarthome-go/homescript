package fuzzer

import (
	"math/rand"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
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
		Events:    make([]ast.AnalyzedFunctionDefinition, 0),
	}

	// Iterate over the imports and shuffle the order around
	ShuffleSlice(tree.Imports, self.randSource)

	// Iterate over the globals and shuffle their order around
	ShuffleSlice(tree.Globals, self.randSource)

	// Iterate over the functions and shuffle their order around
	ShuffleSlice(tree.Functions, self.randSource)

	// Iterate over the events and shuffle their order around
	ShuffleSlice(tree.Events, self.randSource)

	// Iterate over the ast's functions in order to transform each one
	for _, fn := range tree.Functions {
		output.Functions = append(output.Functions, self.Function(fn))
	}

	// Iterate over the ast's events and transform each one
	for _, eventFn := range tree.Events {
		output.Events = append(output.Events, self.Function(eventFn))
	}

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
		variants = append(variants, node)
	case ast.FunctionLiteralExpressionKind:
		panic("TODO")
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

		// TODO: implement using loops
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
