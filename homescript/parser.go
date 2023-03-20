package homescript

import (
	"fmt"
	"strconv"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

// A list of possible tokens which can start an expression
var firstExpr = []TokenKind{
	LParen,
	LBracket,
	LCurly,
	Not,
	Plus,
	Minus,
	If,
	Fn,
	Try,
	For,
	While,
	Loop,
	True,
	False,
	String,
	Number,
	NullType,
	Identifier,
	Enum,
}

type parser struct {
	lexer     lexer
	prevToken Token
	currToken Token
	errors    []errors.Error
	filename  string
}

func newParser(program string, filename string) parser {
	return parser{
		lexer:     newLexer(program, filename),
		prevToken: unknownToken(errors.Location{}),
		currToken: unknownToken(errors.Location{}),
		errors:    make([]errors.Error, 0),
		filename:  filename,
	}
}

func (self *parser) advance() *errors.Error {
	next, err := self.lexer.nextToken()
	if err != nil {
		return err
	}
	self.prevToken = self.currToken
	self.currToken = next
	return nil
}

func (self *parser) expression() (Expression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.andExpr()
	if err != nil {
		return Expression{}, err
	}

	following := make([]AndExpression, 0)
	for self.currToken.Kind == Or {
		// Skip the ||
		if err := self.advance(); err != nil {
			return Expression{}, err
		}
		followingExpr, err := self.andExpr()
		if err != nil {
			return Expression{}, err
		}
		following = append(following, followingExpr)
	}

	return Expression{
		Base:      base,
		Following: following,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) andExpr() (AndExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.eqExpr()
	if err != nil {
		return AndExpression{}, err
	}

	following := make([]EqExpression, 0)
	for self.currToken.Kind == And {
		// Skip &&
		if err := self.advance(); err != nil {
			return AndExpression{}, err
		}
		followingExpr, err := self.eqExpr()
		if err != nil {
			return AndExpression{}, err
		}
		following = append(following, followingExpr)
	}

	return AndExpression{
		Base:      base,
		Following: following,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) eqExpr() (EqExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.relExpr()
	if err != nil {
		return EqExpression{}, err
	}
	if self.currToken.Kind == Equal || self.currToken.Kind == NotEqual {
		isInverted := self.currToken.Kind == NotEqual
		if err := self.advance(); err != nil {
			return EqExpression{}, err
		}
		other, err := self.relExpr()
		if err != nil {
			return EqExpression{}, err
		}
		return EqExpression{
			Base: base,
			Other: &struct {
				Inverted bool
				Node     RelExpression
			}{
				Inverted: isInverted,
				Node:     other,
			},
			Span: errors.Span{
				Start:    startLocation,
				End:      self.prevToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}
	return EqExpression{
		Base:  base,
		Other: nil,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) relExpr() (RelExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.addExpr()
	if err != nil {
		return RelExpression{}, err
	}

	var otherOp RelOperator

	switch self.currToken.Kind {
	case LessThan:
		otherOp = RelLessThan
	case LessThanEqual:
		otherOp = RelLessOrEqual
	case GreaterThan:
		otherOp = RelGreaterThan
	case GreaterThanEqual:
		otherOp = RelGreaterOrEqual
	default:
		return RelExpression{
			Base:  base,
			Other: nil,
			Span: errors.Span{
				Start:    startLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}
	if err := self.advance(); err != nil {
		return RelExpression{}, err
	}

	other, err := self.addExpr()
	if err != nil {
		return RelExpression{}, err
	}

	return RelExpression{
		Base: base,
		Other: &struct {
			RelOperator RelOperator
			Node        AddExpression
		}{
			RelOperator: otherOp,
			Node:        other,
		},
		Span: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) addExpr() (AddExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.mulExpr()
	if err != nil {
		return AddExpression{}, err
	}
	following := make([]struct {
		AddOperator AddOperator
		Other       MulExpression
		Span        errors.Span
	}, 0)
	for self.currToken.Kind == Minus || self.currToken.Kind == Plus {
		otherStart := self.currToken.StartLocation
		var addOp AddOperator
		switch self.currToken.Kind {
		case Minus:
			addOp = AddOpMinus
		case Plus:
			addOp = AddOpPlus
		}
		if err := self.advance(); err != nil {
			return AddExpression{}, err
		}

		other, err := self.mulExpr()
		if err != nil {
			return AddExpression{}, err
		}

		following = append(following, struct {
			AddOperator AddOperator
			Other       MulExpression
			Span        errors.Span
		}{
			AddOperator: addOp,
			Other:       other,
			Span: errors.Span{
				Start:    otherStart,
				End:      self.prevToken.EndLocation,
				Filename: self.filename,
			},
		})
	}

	return AddExpression{
		Base:      base,
		Following: following,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) mulExpr() (MulExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.castExpr()
	if err != nil {
		return MulExpression{}, err
	}
	following := make([]struct {
		MulOperator MulOperator
		Other       CastExpression
		Span        errors.Span
	}, 0)

	for self.currToken.Kind == Multiply || self.currToken.Kind == Divide || self.currToken.Kind == IntDivide || self.currToken.Kind == Reminder {
		otherStart := self.currToken.StartLocation
		var mulOp MulOperator
		switch self.currToken.Kind {
		case Multiply:
			mulOp = MulOpMul
		case Divide:
			mulOp = MulOpDiv
		case IntDivide:
			mulOp = MulOpIntDiv
		case Reminder:
			mulOp = MulOpReminder
		}

		if err := self.advance(); err != nil {
			return MulExpression{}, err
		}

		other, err := self.castExpr()
		if err != nil {
			return MulExpression{}, err
		}

		following = append(following, struct {
			MulOperator MulOperator
			Other       CastExpression
			Span        errors.Span
		}{
			MulOperator: mulOp,
			Other:       other,
			Span: errors.Span{
				Start:    otherStart,
				End:      self.prevToken.EndLocation,
				Filename: self.filename,
			},
		})
	}

	return MulExpression{
		Base:      base,
		Following: following,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) castExpr() (CastExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.unaryExpr()
	if err != nil {
		return CastExpression{}, err
	}

	if self.currToken.Kind != As {
		return CastExpression{
			Base:  base,
			Other: nil,
			Span: errors.Span{
				Start:    startLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}

	// Typecast should be performed
	if err := self.advance(); err != nil {
		return CastExpression{}, err
	}
	var typeCast ValueType
	switch self.currToken.Kind {
	case NullType:
		typeCast = TypeNull
	case NumberType:
		typeCast = TypeNumber
	case StringType:
		typeCast = TypeString
	case BooleanType:
		typeCast = TypeBoolean
	default:
		return CastExpression{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("typecast requires a valid type: expected type name, found %v", self.currToken.Kind),
			Span: errors.Span{
				Start:    self.prevToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return CastExpression{}, err
	}
	return CastExpression{
		Base:  base,
		Other: &typeCast,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) unaryExpr() (UnaryExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	// Make unary expr again (use prefix)
	if self.currToken.Kind == Plus || self.currToken.Kind == Minus || self.currToken.Kind == Not {
		var unaryOp UnaryOp
		switch self.currToken.Kind {
		case Plus:
			unaryOp = UnaryOpPlus
		case Minus:
			unaryOp = UnaryOpMinus
		case Not:
			unaryOp = UnaryOpNot
		default:
			panic("This statement should be unreachable: was a new unary-op introduced?")
		}
		if err := self.advance(); err != nil {
			return UnaryExpression{}, err
		}
		base, err := self.unaryExpr()
		if err != nil {
			return UnaryExpression{}, err
		}
		return UnaryExpression{
			UnaryExpression: &struct {
				UnaryOp         UnaryOp
				UnaryExpression UnaryExpression
			}{
				UnaryOp:         unaryOp,
				UnaryExpression: base,
			},
			ExpExpression: nil,
			Span: errors.Span{
				Start:    startLocation,
				End:      self.prevToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}
	expr, err := self.expExpr()
	if err != nil {
		return UnaryExpression{}, err
	}
	returnValue := UnaryExpression{
		UnaryExpression: nil,
		ExpExpression:   &expr,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}
	return returnValue, nil
}

func (self *parser) expExpr() (ExpExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.assignExpr()
	if err != nil {
		return ExpExpression{}, err
	}
	if self.currToken.Kind == Power {
		if err := self.advance(); err != nil {
			return ExpExpression{}, err
		}
		unary, err := self.unaryExpr()
		if err != nil {
			return ExpExpression{}, err
		}
		return ExpExpression{
			Base:  base,
			Other: &unary,
			Span: errors.Span{
				Start:    startLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}
	return ExpExpression{
		Base:  base,
		Other: nil,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) assignExpr() (AssignExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.callExpr()
	if err != nil {
		return AssignExpression{}, err
	}
	// If an assignment should be made
	var assignOp AssignOperator
	switch self.currToken.Kind {
	case Assign:
		assignOp = OpAssign
	case MultiplyAssign:
		assignOp = OpMulAssign
	case DivideAssign:
		assignOp = OpDivAssign
	case IntDivideAssign:
		assignOp = OpIntDivAssign
	case ReminderAssign:
		assignOp = OpReminderAssign
	case PlusAssign:
		assignOp = OpPlusAssign
	case MinusAssign:
		assignOp = OpMinusAssign
	case PowerAssign:
		assignOp = OpPowerAssign
	default:
		return AssignExpression{
			Base:  base,
			Other: nil,
			Span: errors.Span{
				Start:    startLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}
	if err := self.advance(); err != nil {
		return AssignExpression{}, err
	}
	assignExpr, err := self.expression()
	if err != nil {
		return AssignExpression{}, err
	}
	return AssignExpression{
		Base: base,
		Other: &struct {
			Operator   AssignOperator
			Expression Expression
		}{
			Operator:   assignOp,
			Expression: assignExpr,
		},
		Span: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) callExpr() (CallExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.memberExpr()
	if err != nil {
		return CallExpression{}, err
	}
	// Arguments may follow
	var parts []CallExprPart = nil
	if self.currToken.Kind == LParen {
		// Make call expr parts (includes initial arguments)
		partsTmp, err := self.argsOrCallExprPart()
		if err != nil {
			return CallExpression{}, err
		}
		parts = partsTmp
	}

	return CallExpression{
		Base:  base,
		Parts: parts,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) argsOrCallExprPart() ([]CallExprPart, *errors.Error) {
	var parts []CallExprPart = nil
	for self.currToken.Kind == LParen || self.currToken.Kind == Dot || self.currToken.Kind == LBracket {
		switch self.currToken.Kind {
		case LParen:
			startLocation := self.currToken.StartLocation
			// Make args
			argsItem, err := self.args()
			if err != nil {
				return nil, err
			}
			parts = append(parts, CallExprPart{
				MemberExpressionPart: nil,
				Args:                 &argsItem,
				Span: errors.Span{
					Start:    startLocation,
					End:      self.prevToken.EndLocation,
					Filename: self.filename,
				},
			})
		case Dot:
			// Make call expr part
			if err := self.advance(); err != nil {
				return nil, err
			}
			startLocation := self.currToken.StartLocation
			if self.currToken.Kind != Identifier {
				return nil, &errors.Error{
					Kind:    errors.SyntaxError,
					Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
					Span: errors.Span{
						Start:    self.currToken.StartLocation,
						End:      self.currToken.EndLocation,
						Filename: self.filename,
					},
				}
			}
			identifier := self.currToken.Value
			parts = append(parts, CallExprPart{
				MemberExpressionPart: &identifier,
				Args:                 nil,
				Span: errors.Span{
					Start:    startLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				},
			})
			if err := self.advance(); err != nil {
				return nil, err
			}
		case LBracket:
			if err := self.advance(); err != nil {
				return nil, err
			}
			// Make index
			exprStart := self.prevToken.StartLocation
			isPossible := false
			for _, option := range firstExpr {
				if option == self.currToken.Kind {
					isPossible = true
					break
				}
			}
			if !isPossible {
				return nil, errors.NewError(
					errors.Span{
						Start:    self.currToken.StartLocation,
						End:      self.currToken.EndLocation,
						Filename: self.filename,
					},
					fmt.Sprintf("expected expression, found %v", self.currToken.Kind),
					errors.SyntaxError,
				)
			}
			expression, err := self.expression()
			if err != nil {
				return nil, err
			}
			if self.currToken.Kind != RBracket {
				self.errors = append(self.errors,
					errors.Error{
						Kind:    errors.SyntaxError,
						Message: fmt.Sprintf("unclosed indexing: expected %v, found %v", RBracket, self.currToken.Kind),
						Span: errors.Span{
							Start:    self.currToken.StartLocation,
							End:      self.currToken.EndLocation,
							Filename: self.filename,
						},
					},
				)
			} else {
				if err := self.advance(); err != nil {
					return nil, err
				}
			}
			parts = append(parts, CallExprPart{
				MemberExpressionPart: nil,
				Args:                 nil,
				Index:                &expression,
				Span: errors.Span{
					Start:    exprStart,
					End:      self.prevToken.EndLocation,
					Filename: self.filename,
				},
			})
		default:
			panic("This should be unreachable")
		}
	}
	return parts, nil
}

func (self *parser) args() ([]Expression, *errors.Error) {
	callArgs := make([]Expression, 0)
	// Skip opening brace
	if err := self.advance(); err != nil {
		return nil, err
	}
	// Return early if no args follow
	if self.currToken.Kind == RParen {
		if err := self.advance(); err != nil {
			return nil, err
		}
		return callArgs, nil
	}
	makeExpr := false
	for _, possible := range firstExpr {
		if possible == self.currToken.Kind {
			makeExpr = true
			break
		}
	}

	if !makeExpr {
		return nil, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Unclosed function call: Expected %v, found %v", RParen, self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}

	// Consume first expression
	expr, err := self.expression()
	if err != nil {
		return nil, err
	}
	callArgs = append(callArgs, expr)

	// Consume more expressions
	for self.currToken.Kind == Comma {
		if err := self.advance(); err != nil {
			return nil, err
		}
		if self.currToken.Kind == RParen {
			break
		}
		expr, err := self.expression()
		if err != nil {
			return nil, err
		}
		callArgs = append(callArgs, expr)
	}

	if self.currToken.Kind != RParen {
		return nil, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Unclosed function call: Expected %v, found %v", RParen, self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return nil, err
	}
	return callArgs, nil
}

func (self *parser) memberExpr() (MemberExpression, *errors.Error) {
	startLocation := self.currToken.StartLocation
	base, err := self.atom()
	if err != nil {
		return MemberExpression{}, err
	}

	members := make([]struct {
		Identifier *string
		Index      *Expression
		Span       errors.Span
	}, 0)

	for self.currToken.Kind == Dot || self.currToken.Kind == LBracket {
		if err := self.advance(); err != nil {
			return MemberExpression{}, err
		}
		if self.prevToken.Kind == Dot {
			if self.currToken.Kind != Identifier {
				return MemberExpression{}, &errors.Error{
					Kind:    errors.SyntaxError,
					Message: fmt.Sprintf("expected identifier, found %v", self.currToken.Kind),
					Span: errors.Span{
						Start:    self.currToken.StartLocation,
						End:      self.currToken.EndLocation,
						Filename: self.filename,
					},
				}
			}
			ident := self.currToken.Value
			members = append(members, struct {
				Identifier *string
				Index      *Expression
				Span       errors.Span
			}{
				Identifier: &ident,
				Index:      nil,
				Span: errors.Span{
					Start:    self.currToken.StartLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				}})
			if err := self.advance(); err != nil {
				return MemberExpression{}, err
			}
		} else if self.prevToken.Kind == LBracket {
			exprStart := self.prevToken.StartLocation
			isPossible := false
			for _, option := range firstExpr {
				if option == self.currToken.Kind {
					isPossible = true
					break
				}
			}
			if !isPossible {
				return MemberExpression{}, errors.NewError(
					errors.Span{
						Start:    self.currToken.StartLocation,
						End:      self.currToken.EndLocation,
						Filename: self.filename,
					},
					fmt.Sprintf("expected expression, found %v", self.currToken.Kind),
					errors.SyntaxError,
				)
			}
			expression, err := self.expression()
			if err != nil {
				return MemberExpression{}, err
			}
			if self.currToken.Kind != RBracket {
				self.errors = append(self.errors,
					errors.Error{
						Kind:    errors.SyntaxError,
						Message: fmt.Sprintf("unclosed indexing: expected %v, found %v", RBracket, self.currToken.Kind),
						Span: errors.Span{
							Start:    self.currToken.StartLocation,
							End:      self.currToken.EndLocation,
							Filename: self.filename,
						},
					},
				)
			} else {
				if err := self.advance(); err != nil {
					return MemberExpression{}, err
				}
			}
			members = append(members, struct {
				Identifier *string
				Index      *Expression
				Span       errors.Span
			}{
				Identifier: nil,
				Index:      &expression,
				Span: errors.Span{
					Start:    exprStart,
					End:      self.prevToken.EndLocation,
					Filename: self.filename,
				},
			})
		} else {
			panic("BUG: this should be unreachable")
		}
	}
	return MemberExpression{
		Base:    base,
		Members: members,
		Span: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) atom() (Atom, *errors.Error) {
	startLocation := self.currToken.StartLocation
	switch self.currToken.Kind {
	case Number:
		startTok := self.currToken

		if err := self.advance(); err != nil {
			return nil, err
		}

		if self.currToken.Kind == Range {
			return self.rangeExpr(startTok)
		}

		num, _ := strconv.ParseFloat(startTok.Value, 64)
		return AtomNumber{
			Num: num,
			Range: errors.Span{
				Start:    startLocation,
				End:      self.prevToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	case True, False:
		boolValue := self.currToken.Kind == True
		if err := self.advance(); err != nil {
			return nil, err
		}
		return AtomBoolean{
			Value: boolValue,
			Range: errors.Span{
				Start:    startLocation,
				End:      self.prevToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	case LCurly:
		return self.makeObject()
	case String:
		if err := self.advance(); err != nil {
			return nil, err
		}
		// Atom is not a string but a pair
		if self.currToken.Kind == Arrow {
			pairKey := self.prevToken.Value
			if err := self.advance(); err != nil {
				return nil, err
			}
			pairValueExpr, err := self.expression()
			if err != nil {
				return nil, err
			}
			return AtomPair{
				Key:       pairKey,
				ValueExpr: pairValueExpr,
				Range: errors.Span{
					Start:    startLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				},
			}, nil
		}
		// Atom is not a pair but a string
		return AtomString{
			Content: self.prevToken.Value,
			Range: errors.Span{
				Start:    startLocation,
				End:      self.prevToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	case LBracket:
		return self.listLiteral()
	case Identifier:
		return self.enumVariant()
	case NullType:
		if err := self.advance(); err != nil {
			return nil, err
		}
		return AtomNull{
				Range: errors.Span{
					Start:    startLocation,
					End:      self.prevToken.EndLocation,
					Filename: self.filename,
				},
			},
			nil
	case Enum:
		return self.makeEnum()
	case If:
		return self.ifExpr()
	case For:
		return self.forExpr()
	case While:
		return self.whileExpr()
	case Loop:
		return self.loopExpr()
	case Fn:
		return self.fnExpr()
	case Try:
		return self.tryExpr()
	case LParen:
		// Skip paranthesis
		if err := self.advance(); err != nil {
			return nil, err
		}

		nestedExpr, err := self.expression()
		if err != nil {
			return nil, err
		}
		// Check and  skip closing paranthesis
		if self.currToken.Kind != RParen {
			return nil, &errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Unclosed nested expression: expected %v, found %v", RParen, self.currToken.Kind),
				Span: errors.Span{
					Start:    self.currToken.StartLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				},
			}
		}
		if err := self.advance(); err != nil {
			return nil, err
		}
		return AtomExpression{
			Expression: nestedExpr,
			Range: errors.Span{
				Start:    startLocation,
				End:      self.prevToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}
	self.errors = append(self.errors, errors.Error{
		Span: errors.Span{
			Start:    self.prevToken.EndLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
		Message: "expression expected here",
		Kind:    errors.Warning,
	},
	)
	return nil, errors.NewError(
		errors.Span{
			Start:    self.currToken.StartLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
		fmt.Sprintf("unexpected token %v found here", self.currToken.Kind),
		errors.SyntaxError,
	)
}

func (self *parser) makeEnum() (AtomEnum, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return AtomEnum{}, err
	}

	var enumName string

	if self.currToken.Kind != Identifier {
		return AtomEnum{}, errors.NewError(
			errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
			fmt.Sprintf("expected `%v`, found `%v`", Identifier, self.currToken.Kind),
			errors.SyntaxError,
		)
	}

	enumName = self.currToken.Value

	if err := self.advance(); err != nil {
		return AtomEnum{}, err
	}

	if self.currToken.Kind != LCurly {
		return AtomEnum{}, errors.NewError(
			errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
			fmt.Sprintf("expected `%v`, found `%v`", LCurly, self.currToken.Kind),
			errors.SyntaxError,
		)
	}

	if err := self.advance(); err != nil {
		return AtomEnum{}, err
	}

	variants := make([]EnumVariant, 0)
	for {
		if self.currToken.Kind != Identifier {
			return AtomEnum{}, errors.NewError(
				errors.Span{
					Start:    self.currToken.StartLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				},
				fmt.Sprintf("expected `%v`, found `%v`", Identifier, self.currToken.Kind),
				errors.SyntaxError,
			)
		}

		variants = append(variants, EnumVariant{
			Value: self.currToken.Value,
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		})

		commaSpan := errors.Span{
			Start:    self.currToken.EndLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		}

		if err := self.advance(); err != nil {
			return AtomEnum{}, err
		}

		if err := self.advance(); err != nil {
			return AtomEnum{}, err
		}

		if self.prevToken.Kind == RCurly {
			break
		}

		if self.prevToken.Kind != Comma {
			return AtomEnum{}, errors.NewError(commaSpan, fmt.Sprintf("Expected `%v`, found `%v`", Comma, self.prevToken.Kind), errors.SyntaxError)
		}
	}

	return AtomEnum{
		Name:     enumName,
		Variants: variants,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) makeObject() (AtomObject, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return AtomObject{}, err
	}
	// If there are no fields, return here
	if self.currToken.Kind == RCurly {
		if err := self.advance(); err != nil {
			return AtomObject{}, err
		}
		return AtomObject{Range: errors.Span{
			Start:    self.prevToken.StartLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		}}, nil
	}

	// Make initial field
	fields := make([]AtomObjectField, 0)
	field, err := self.objectField()
	if err != nil {
		return AtomObject{}, err
	}
	fields = append(fields, field)

	// Make other fields
	for self.currToken.Kind == Comma {
		if err := self.advance(); err != nil {
			return AtomObject{}, err
		}
		if self.currToken.Kind == RCurly {
			break
		}
		field, err := self.objectField()
		if err != nil {
			return AtomObject{}, err
		}
		fields = append(fields, field)

	}
	// Expect a RCurly
	if self.currToken.Kind != RCurly {
		return AtomObject{}, errors.NewError(
			errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
			fmt.Sprintf("unclosed object literal: expected %v, found %v", RCurly, self.currToken.Kind),
			errors.SyntaxError,
		)
	}
	if err := self.advance(); err != nil {
		return AtomObject{}, err
	}

	return AtomObject{
		Range: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
		Fields: fields,
	}, nil
}

func (self *parser) objectField() (AtomObjectField, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if self.currToken.Kind != Identifier {
		return AtomObjectField{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("expected identifier, found %v", self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}

	identifier, identSpan := self.currToken.Value, errors.Span{
		Start:    self.currToken.StartLocation,
		End:      self.currToken.EndLocation,
		Filename: self.filename,
	}

	if err := self.advance(); err != nil {
		return AtomObjectField{}, err

	}

	// Expect a `:` colon
	if self.currToken.Kind != Colon {
		return AtomObjectField{}, errors.NewError(errors.Span{
			Start:    self.currToken.StartLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
			fmt.Sprintf("expected %v, found %v", Colon, self.currToken.Kind),
			errors.SyntaxError,
		)
	}
	if err := self.advance(); err != nil {
		return AtomObjectField{}, err
	}

	isValidExpr := false
	for _, possible := range firstExpr {
		if possible == self.currToken.Kind {
			isValidExpr = true
			break
		}
	}
	if !isValidExpr {
		return AtomObjectField{}, errors.NewError(errors.Span{
			Start:    self.currToken.StartLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
			fmt.Sprintf("expected expression, found %v", self.currToken.Kind),
			errors.SyntaxError,
		)
	}
	expression, err := self.expression()
	if err != nil {
		return AtomObjectField{}, err
	}
	return AtomObjectField{
		Span: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
		Identifier: identifier,
		IdentSpan:  identSpan,
		Expression: expression,
	}, nil
}

func (self *parser) listLiteral() (AtomListLiteral, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return AtomListLiteral{}, err
	}
	// If there is a closing bracket, stop here
	if self.currToken.Kind == RBracket {
		if err := self.advance(); err != nil {
			return AtomListLiteral{}, err
		}
		return AtomListLiteral{
			Values: make([]Expression, 0),
			Range: errors.Span{
				Start:    startLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}

	// Make initial expression
	isValidExpr := false
	for _, possible := range firstExpr {
		if possible == self.currToken.Kind {
			isValidExpr = true
			break
		}
	}
	if !isValidExpr {
		return AtomListLiteral{}, errors.NewError(errors.Span{
			Start:    self.currToken.StartLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
			fmt.Sprintf("expected expression, found %v", self.currToken.Kind),
			errors.SyntaxError,
		)
	}

	expressions := make([]Expression, 0)
	expression, err := self.expression()
	if err != nil {
		return AtomListLiteral{}, err
	}
	expressions = append(expressions, expression)

	// Add more values if possible
	for self.currToken.Kind == Comma {
		if err := self.advance(); err != nil {
			return AtomListLiteral{}, err
		}
		if self.currToken.Kind == RBracket {
			break
		}
		expression, err := self.expression()
		if err != nil {
			return AtomListLiteral{}, err
		}
		expressions = append(expressions, expression)
	}

	// Expect a closing bracket
	if self.currToken.Kind != RBracket {
		return AtomListLiteral{}, errors.NewError(
			errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
			fmt.Sprintf("unclosed list literal: expected %v, found %v", RBracket, self.currToken.Kind),
			errors.SyntaxError,
		)
	}
	if err := self.advance(); err != nil {
		return AtomListLiteral{}, err
	}
	return AtomListLiteral{
		Range: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
		Values: expressions,
	}, nil
}

func (self *parser) enumVariant() (Atom, *errors.Error) {
	startLocation := self.currToken.StartLocation

	enumName := self.currToken.Value

	if err := self.advance(); err != nil {
		return AtomEnumVariant{}, err
	}

	if self.currToken.Kind == Colon {
		if err := self.advance(); err != nil {
			return AtomEnumVariant{}, err
		}

		if self.currToken.Kind != Colon {
			return AtomEnumVariant{},
				errors.NewError(
					errors.Span{
						Start:    self.currToken.StartLocation,
						End:      self.currToken.EndLocation,
						Filename: self.filename,
					},
					fmt.Sprintf("Expected `%v`, found `%v`", Colon, self.currToken.Kind),
					errors.SyntaxError,
				)
		}

		if err := self.advance(); err != nil {
			return AtomEnumVariant{}, err
		}

		if self.currToken.Kind != Identifier {
			return AtomEnumVariant{},
				errors.NewError(
					errors.Span{
						Start:    self.currToken.StartLocation,
						End:      self.currToken.EndLocation,
						Filename: self.filename,
					},
					fmt.Sprintf("Expected `%v`, found `%v`", Identifier, self.currToken.Kind),
					errors.SyntaxError,
				)
		}

		variantName := self.currToken.Value

		if err := self.advance(); err != nil {
			return AtomEnumVariant{}, err
		}

		return AtomEnumVariant{
			RefersToEnum: enumName,
			Name:         variantName,
			Range: errors.Span{
				Start:    startLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}

	return AtomIdentifier{
		Identifier: self.prevToken.Value,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) ifExpr() (IfExpr, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return IfExpr{}, err
	}
	conditionExpr, err := self.expression()
	if err != nil {
		return IfExpr{}, err
	}

	block, err := self.curlyBlock()
	if err != nil {
		return IfExpr{}, err
	}

	// Handle else block
	if self.currToken.Kind == Else {
		if err := self.advance(); err != nil {
			return IfExpr{}, err
		}

		// Handle else if using recursion here
		if self.currToken.Kind == If {
			elseIfExpr, err := self.ifExpr()
			if err != nil {
				return IfExpr{}, err
			}

			return IfExpr{
				Condition:  conditionExpr,
				Block:      block,
				ElseBlock:  nil,
				ElseIfExpr: &elseIfExpr,
				Range: errors.Span{
					Start:    startLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				},
			}, nil
		}

		// Handle normal else block here
		elseBlock, err := self.curlyBlock()
		if err != nil {
			return IfExpr{}, err
		}
		return IfExpr{
			Condition:  conditionExpr,
			Block:      block,
			ElseBlock:  &elseBlock,
			ElseIfExpr: nil,
			Range: errors.Span{
				Start:    startLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}, nil
	}
	// If without else block
	return IfExpr{
		Condition:  conditionExpr,
		Block:      block,
		ElseBlock:  nil,
		ElseIfExpr: nil,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) forExpr() (AtomFor, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return AtomFor{}, err
	}
	if self.currToken.Kind != Identifier {
		return AtomFor{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	headIdentifier := struct {
		Identifier string
		Span       errors.Span
	}{
		Identifier: self.currToken.Value,
		Span: errors.Span{
			Start:    self.currToken.StartLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}

	if err := self.advance(); err != nil {
		return AtomFor{}, err
	}

	// Make in
	if self.currToken.Kind != In {
		return AtomFor{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected %v, found %v", In, self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return AtomFor{}, err
	}

	expr, err := self.expression()
	if err != nil {
		return AtomFor{}, err
	}

	iterationBlock, err := self.curlyBlock()
	if err != nil {
		return AtomFor{}, err
	}

	return AtomFor{
		HeadIdentifier: headIdentifier,
		IterExpr:       expr,
		IterationCode:  iterationBlock,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) rangeExpr(startTok Token) (AtomRange, *errors.Error) {
	start, _ := strconv.ParseFloat(startTok.Value, 64)
	if float64(int(start)) != start {
		return AtomRange{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: "Expected integer, found float",
			Span: errors.Span{
				Start:    startTok.StartLocation,
				End:      startTok.EndLocation,
				Filename: self.filename,
			},
		}
	}

	if self.currToken.Kind != Range {
		return AtomRange{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected %v, found %v", Range, self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}

	if err := self.advance(); err != nil {
		return AtomRange{}, err
	}

	if self.currToken.Kind != Number {
		return AtomRange{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected %v, found %v", Number, self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}

	end, _ := strconv.ParseFloat(self.currToken.Value, 64)
	if float64(int(end)) != end {
		return AtomRange{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: "Expected integer, found float",
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}

	if err := self.advance(); err != nil {
		return AtomRange{}, err
	}

	return AtomRange{
		Start: int(start),
		End:   int(end),
		Range: errors.Span{
			Start:    startTok.StartLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) whileExpr() (AtomWhile, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return AtomWhile{}, err
	}

	// Expect an expression or return an error
	isValidExpr := false
	for _, possible := range firstExpr {
		if possible == self.currToken.Kind {
			isValidExpr = true
			break
		}
	}
	if !isValidExpr {
		return AtomWhile{}, errors.NewError(errors.Span{
			Start:    self.currToken.StartLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
			fmt.Sprintf("Expected expression, found %v", self.currToken.Kind),
			errors.SyntaxError,
		)
	}

	conditionExpr, err := self.expression()
	if err != nil {
		return AtomWhile{}, err
	}

	iterationBlock, err := self.curlyBlock()
	if err != nil {
		return AtomWhile{}, err
	}

	return AtomWhile{
		HeadCondition: conditionExpr,
		IterationCode: iterationBlock,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) loopExpr() (AtomLoop, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return AtomLoop{}, err
	}
	iterationBlock, err := self.curlyBlock()
	if err != nil {
		return AtomLoop{}, err
	}
	return AtomLoop{
		IterationCode: iterationBlock,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) fnExpr() (AtomFunction, *errors.Error) {
	startLocation := self.currToken.StartLocation
	// Skip fn
	if err := self.advance(); err != nil {
		return AtomFunction{}, err
	}

	// Make optional function identifier
	var identifier *string
	if self.currToken.Kind == Identifier {
		// Assign to the identifier placeholder
		identifierTmp := self.currToken.Value
		identifier = &identifierTmp
		if err := self.advance(); err != nil {
			return AtomFunction{}, err
		}
	}

	// Expect Lparen (
	if self.currToken.Kind != LParen {
		return AtomFunction{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected (, found %v", self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return AtomFunction{}, err
	}

	// Make args
	args := make([]struct {
		Identifier string
		Span       errors.Span
	}, 0)

	// Only make args if there is no immediate closing bracket
	if self.currToken.Kind != RParen {
		// Add initial argument
		if self.currToken.Kind != Identifier {
			return AtomFunction{}, &errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
				Span: errors.Span{
					Start:    self.currToken.StartLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				},
			}
		}
		args = append(args, struct {
			Identifier string
			Span       errors.Span
		}{Identifier: self.currToken.Value, Span: errors.Span{
			Start:    self.currToken.StartLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		}})
		if err := self.advance(); err != nil {
			return AtomFunction{}, err
		}

		// Add additional arguments
		for self.currToken.Kind == Comma {
			if err := self.advance(); err != nil {
				return AtomFunction{}, err
			}
			// Allow trailing comma
			if self.currToken.Kind == RParen {
				break
			}
			if self.currToken.Kind != Identifier {
				return AtomFunction{}, &errors.Error{
					Kind:    errors.SyntaxError,
					Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
					Span: errors.Span{
						Start:    self.currToken.StartLocation,
						End:      self.currToken.EndLocation,
						Filename: self.filename,
					},
				}
			}
			args = append(args, struct {
				Identifier string
				Span       errors.Span
			}{
				Identifier: self.currToken.Value,
				Span: errors.Span{
					Start:    self.currToken.StartLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				},
			})
			if err := self.advance(); err != nil {
				return AtomFunction{}, err
			}
			// Stop here if the current token is a )
			if self.currToken.Kind == RParen {
				break
			}
		}

		if self.currToken.Kind != RParen {
			return AtomFunction{}, &errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Expected %v, found %v", RParen, self.currToken.Kind),
				Span: errors.Span{
					Start:    self.currToken.StartLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				},
			}
		}
	}
	// Skip closing paranthesis
	if err := self.advance(); err != nil {
		return AtomFunction{}, err
	}

	// Make function body
	functionBlock, err := self.curlyBlock()
	if err != nil {
		return AtomFunction{}, err
	}
	return AtomFunction{
		Ident:          identifier,
		ArgIdentifiers: args,
		Body:           functionBlock,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) tryExpr() (AtomTry, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return AtomTry{}, err
	}
	tryBlock, err := self.curlyBlock()
	if err != nil {
		return AtomTry{}, err
	}
	if self.currToken.Kind != Catch {
		return AtomTry{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected %v, found %v", Catch, self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return AtomTry{}, err
	}
	// Expect an identifier here
	if self.currToken.Kind != Identifier {
		return AtomTry{}, errors.NewError(
			errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
			fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
			errors.SyntaxError,
		)
	}
	catchIdentifier := self.currToken.Value
	if err := self.advance(); err != nil {
		return AtomTry{}, err
	}
	catchBlock, err := self.curlyBlock()
	if err != nil {
		return AtomTry{}, err
	}
	return AtomTry{
		TryBlock:        tryBlock,
		ErrorIdentifier: catchIdentifier,
		CatchBlock:      catchBlock,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) letStmt() (LetStmt, *errors.Error) {
	startLocation := self.currToken.StartLocation
	// Skip the `let`
	if err := self.advance(); err != nil {
		return LetStmt{}, err
	}

	if self.currToken.Kind != Identifier {
		return LetStmt{}, &errors.Error{
			Kind: errors.SyntaxError,

			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	// Copy the assignment identifier name
	assignIdentifier := self.currToken
	if err := self.advance(); err != nil {
		return LetStmt{}, err
	}

	if self.currToken.Kind != Assign {
		return LetStmt{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected assignment operator '=', found %v", self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return LetStmt{}, err
	}

	assignExpression, err := self.expression()
	if err != nil {
		return LetStmt{}, err
	}

	return LetStmt{
		Left: struct {
			Identifier string
			Span       errors.Span
		}{
			Identifier: assignIdentifier.Value,
			Span: errors.Span{
				Start:    assignIdentifier.StartLocation,
				End:      assignIdentifier.EndLocation,
				Filename: self.filename,
			},
		},
		Right: assignExpression,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) importStmt() (ImportStmt, *errors.Error) {
	startLocation := self.currToken.StartLocation
	// Skip the `import`
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}

	// Make the import function name
	functionName := self.currToken.Value
	if self.currToken.Kind != Identifier {
		return ImportStmt{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}

	// Make optional name rewrite
	var rewriteName *string = nil
	if self.currToken.Kind == As {
		//// TODO: implement alias imports correctly ////
		return ImportStmt{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: "Using the `as` keyword is currently unstable",
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}

		//// TODO: remove this /////

		if err := self.advance(); err != nil {
			return ImportStmt{}, err
		}

		if self.currToken.Kind != Identifier {
			return ImportStmt{}, &errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
				Span: errors.Span{
					Start:    self.currToken.StartLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				},
			}
		}
		rewriteNameTmp := self.currToken.Value
		rewriteName = &rewriteNameTmp
		if err := self.advance(); err != nil {
			return ImportStmt{}, err
		}
	}

	if self.currToken.Kind != From {
		return ImportStmt{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected 'from', found %v", self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}

	// Make target module name
	if self.currToken.Kind != Identifier {
		return ImportStmt{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}

	return ImportStmt{
		Function:   functionName,
		RewriteAs:  rewriteName,
		FromModule: self.prevToken.Value,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.prevToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) breakStmt() (BreakStmt, *errors.Error) {
	startLocation := self.currToken.StartLocation
	// Skip the break
	if err := self.advance(); err != nil {
		return BreakStmt{}, err
	}
	// Check for an additional expression
	shouldMakeAdditionExpr := false
	for _, possible := range firstExpr {
		if possible == self.currToken.Kind {
			shouldMakeAdditionExpr = true
			break
		}
	}
	var additionalExpr *Expression
	if shouldMakeAdditionExpr {
		additionalExprTemp, err := self.expression()
		if err != nil {
			return BreakStmt{}, err
		}
		additionalExpr = &additionalExprTemp
	}
	breakStmt := BreakStmt{
		Expression: additionalExpr,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}
	return breakStmt, nil
}

func (self *parser) continueStmt() (ContinueStmt, *errors.Error) {
	continueStmt := ContinueStmt{
		Range: errors.Span{
			Start:    self.currToken.StartLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}
	if err := self.advance(); err != nil {
		return ContinueStmt{}, err
	}
	return continueStmt, nil
}

func (self *parser) returnStmt() (ReturnStmt, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return ReturnStmt{}, err
	}
	// Check for an additional expression
	shouldMakeAdditionExpr := false
	for _, possible := range firstExpr {
		if possible == self.currToken.Kind {
			shouldMakeAdditionExpr = true
			break
		}
	}
	var additionalExpr *Expression = nil
	if shouldMakeAdditionExpr {
		additionalExprTemp, err := self.expression()
		if err != nil {
			return ReturnStmt{}, err
		}
		additionalExpr = &additionalExprTemp
	}
	return ReturnStmt{
		Expression: additionalExpr,
		Range: errors.Span{
			Start:    startLocation,
			End:      self.currToken.EndLocation,
			Filename: self.filename,
		},
	}, nil
}

func (self *parser) statement() (StatementOrExpr, *errors.Error) {
	var stmt Statement
	var expr Expression
	isExpr := false

	switch self.currToken.Kind {
	case Let:
		res, err := self.letStmt()
		if err != nil {
			return StatementOrExpr{}, err
		}
		stmt = res
	case Import:
		res, err := self.importStmt()
		if err != nil {
			return StatementOrExpr{}, err
		}
		stmt = res
	case Break:
		res, err := self.breakStmt()
		if err != nil {
			return StatementOrExpr{}, err
		}
		stmt = res
	case Continue:
		res, err := self.continueStmt()
		if err != nil {
			return StatementOrExpr{}, err
		}
		stmt = res
	case Return:
		res, err := self.returnStmt()
		if err != nil {
			return StatementOrExpr{}, err
		}
		stmt = res
	default:
		canBeExpr := false
		for _, possible := range firstExpr {
			if self.currToken.Kind == possible {
				canBeExpr = true
				break
			}
		}
		if !canBeExpr {
			return StatementOrExpr{}, &errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Invalid expression: expected one of %d tokens, found %v", len(firstExpr), self.currToken.Kind),
				Span: errors.Span{
					Start:    self.currToken.StartLocation,
					End:      self.currToken.EndLocation,
					Filename: self.filename,
				}}
		}

		res, err := self.expression()
		if err != nil {
			return StatementOrExpr{}, err
		}
		isExpr = true
		expr = res
	}

	if self.currToken.Kind == Semicolon {
		if err := self.advance(); err != nil {
			return StatementOrExpr{}, err
		}
		if isExpr {
			stmt = ExpressionStmt{Expression: expr}
		}
	} else if !isExpr {
		// Only report an error if the last node had no block

		end := self.prevToken.EndLocation
		// Increments the end column so that a missing semicolon is marked where it was expected
		end.Column++

		// Try to skip this error
		self.errors = append(self.errors,
			errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Missing ; after statement (Expected semicolon, found %s)", self.currToken.Kind),
				Span: errors.Span{
					Start:    end,
					End:      end,
					Filename: self.filename,
				},
			},
		)
	}

	if stmt != nil {
		return StatementOrExpr{Statement: stmt, Expression: nil}, nil
	} else {
		return StatementOrExpr{Statement: nil, Expression: &expr}, nil
	}
}

func (self *parser) statements(insideCurlyBlock bool) ([]StatementOrExpr, *errors.Error) {
	statements := make([]StatementOrExpr, 0)

	// If statements is invoked in an empty block or at the end of the file, quit immediately
	if self.currToken.Kind == EOF || self.currToken.Kind == RCurly {
		return statements, nil
	}

	// Make additional statements
	for {
		if (insideCurlyBlock && self.currToken.Kind == RCurly) || self.currToken.Kind == EOF {
			break
		}

		statement, err := self.statement()
		if err != nil {
			return nil, err
		}
		if statement.Expression != nil && self.prevToken.Kind != RCurly {
			self.errors = append(self.errors,
				errors.Error{
					Kind:    errors.SyntaxError,
					Message: fmt.Sprintf("A Missing ; after statement (Expected semicolon, found %s)", self.currToken.Kind),
					Span: errors.Span{
						Start:    self.prevToken.StartLocation,
						End:      self.prevToken.EndLocation,
						Filename: self.filename,
					},
				},
			)
		} else {
			statements = append(statements, statement)
		}
	}

	return statements, nil
}

func (self *parser) curlyBlock() (Block, *errors.Error) {
	startLoc := self.currToken.StartLocation

	if self.currToken.Kind != LCurly {
		self.errors = append(self.errors, errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Invalid block: expected %v, found %v", LCurly, self.currToken.Kind),
			Span: errors.Span{
				Start:    self.prevToken.EndLocation,
				End:      self.prevToken.EndLocation,
				Filename: self.filename,
			},
		},
		)
	} else {
		if err := self.advance(); err != nil {
			return Block{}, err
		}
	}

	statements := make([]Statement, 0)
	var expr *Expression

	// Make additional statements
	for {
		if self.currToken.Kind == RCurly || self.currToken.Kind == EOF {
			break
		}

		if expr != nil && self.prevToken.Kind != RCurly {
			self.errors = append(self.errors,
				errors.Error{
					Kind:    errors.SyntaxError,
					Message: fmt.Sprintf("Missing ; after statement (Expected semicolon, found %s)", self.currToken.Kind),
					Span: errors.Span{
						Start:    expr.Span.Start,
						End:      expr.Span.End,
						Filename: self.filename,
					},
				},
			)
		}

		statement, err := self.statement()
		if err != nil {
			return Block{}, err
		}

		if statement.Expression != nil {
			expr = statement.Expression
		} else {
			if expr != nil {
				statements = append(statements, ExpressionStmt{Expression: *expr})
				expr = nil
			}
			statements = append(statements, statement.Statement)
		}
	}

	if self.currToken.Kind != RCurly {
		return Block{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Invalid block: expected %v, found %v", RCurly, self.currToken.Kind),
			Span: errors.Span{
				Start:    self.currToken.StartLocation,
				End:      self.currToken.EndLocation,
				Filename: self.filename,
			},
		}
	}
	if err := self.advance(); err != nil {
		return Block{}, err
	}
	return Block{Statements: statements, Expr: expr, Span: errors.Span{
		Start:    startLoc,
		End:      self.prevToken.EndLocation,
		Filename: self.filename,
	}}, nil
}

// Returns the ast, any errors an an indication whether the error was critical
func (self *parser) parse() ([]StatementOrExpr, []errors.Error, bool) {
	if err := self.advance(); err != nil {
		self.errors = append(self.errors, *err)
		return nil, self.errors, true
	}
	// Do not accept a lone }
	statements, err := self.statements(false)
	if err != nil {
		self.errors = append(self.errors, *err)
		return nil, self.errors, true
	}
	return statements, self.errors, false
}
