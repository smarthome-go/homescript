package homescript

import (
	"fmt"
	"strconv"

	"github.com/smarthome-go/homescript/homescript/errors"
)

// A list of possible tokens which can start an expression
var firstExpr = []TokenKind{
	LParen,
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
	Identifier,
}

type parser struct {
	lexer     lexer
	prevToken Token
	currToken Token
}

func newParser(filename string, program string) parser {
	return parser{
		lexer:     newLexer(filename, program),
		prevToken: unknownToken(errors.Location{}),
		currToken: unknownToken(errors.Location{}),
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
			Start: startLocation,
			End:   self.prevToken.EndLocation,
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
			Start: startLocation,
			End:   self.prevToken.EndLocation,
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
				Start: startLocation,
				End:   self.prevToken.EndLocation,
			},
		}, nil
	}
	return EqExpression{
		Base:  base,
		Other: nil,
		Span: errors.Span{
			Start: startLocation,
			End:   self.prevToken.EndLocation,
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
				Start: startLocation,
				End:   self.currToken.EndLocation,
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
			Start: startLocation,
			End:   self.prevToken.EndLocation,
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
	}, 0)
	for self.currToken.Kind == Minus || self.currToken.Kind == Plus {
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
		}{
			AddOperator: addOp,
			Other:       other,
		})
	}

	return AddExpression{
		Base:      base,
		Following: following,
		Span: errors.Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
	}, 0)

	for self.currToken.Kind == Multiply || self.currToken.Kind == Divide || self.currToken.Kind == Reminder {
		var mulOp MulOperator
		switch self.currToken.Kind {
		case Multiply:
			mulOp = MulOpMul
		case Divide:
			mulOp = MulOpDiv
		case Reminder:
			mulOp = MullOpReminder
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
		}{
			MulOperator: mulOp,
			Other:       other,
		})
	}

	return MulExpression{
		Base:      base,
		Following: following,
		Span: errors.Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
				Start: startLocation,
				End:   self.currToken.EndLocation,
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
			Message: fmt.Sprintf("Typecast requires a valid type: expected type name, found %v", self.currToken.Kind),
			Span: errors.Span{
				Start: self.prevToken.StartLocation,
				End:   self.currToken.EndLocation,
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
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
			panic("err")
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
				Start: startLocation,
				End:   self.prevToken.EndLocation,
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
			Start: startLocation,
			End:   self.prevToken.EndLocation,
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
				Start: startLocation,
				End:   self.currToken.EndLocation,
			},
		}, nil
	}
	return ExpExpression{
		Base:  base,
		Other: nil,
		Span: errors.Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
				Start: startLocation,
				End:   self.currToken.EndLocation,
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
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}, nil
}

func (self *parser) argsOrCallExprPart() ([]CallExprPart, *errors.Error) {
	var parts []CallExprPart = nil
	for self.currToken.Kind != EOF {
		startLocation := self.currToken.StartLocation
		switch self.currToken.Kind {
		case LParen:
			// Make args
			argsItem, err := self.args()
			if err != nil {
				return nil, err
			}
			parts = append(parts, CallExprPart{
				MemberExpressionPart: nil,
				Args:                 &argsItem,
				Span: errors.Span{
					Start: startLocation,
					End:   self.currToken.EndLocation,
				},
			})
		case Dot:
			// Make call expr part
			if err := self.advance(); err != nil {
				return nil, err
			}
			if self.currToken.Kind != Identifier {
				return nil, &errors.Error{
					Kind:    errors.SyntaxError,
					Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
					Span: errors.Span{
						Start: self.currToken.StartLocation,
						End:   self.currToken.EndLocation,
					},
				}
			}
			parts = append(parts, CallExprPart{
				MemberExpressionPart: &self.currToken.Value,
				Args:                 nil,
				Span: errors.Span{
					Start: startLocation,
					End:   self.currToken.EndLocation,
				},
			})
			if err := self.advance(); err != nil {
				return nil, err
			}
		default:
			return parts, nil
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
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
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
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
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
	members := make([]string, 0)
	for self.currToken.Kind == Dot {
		if err := self.advance(); err != nil {
			return MemberExpression{}, err
		}
		if self.currToken.Kind != Identifier {
			return MemberExpression{}, &errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
				Span: errors.Span{
					Start: self.currToken.StartLocation,
					End:   self.currToken.EndLocation,
				},
			}
		}
		members = append(members, self.currToken.Value)
		if err := self.advance(); err != nil {
			return MemberExpression{}, err
		}
	}
	return MemberExpression{
		Base:    base,
		Members: members,
		Span: errors.Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}, nil
}

func (self *parser) atom() (Atom, *errors.Error) {
	startLocation := self.currToken.StartLocation
	switch self.currToken.Kind {
	case Number:
		number, err := strconv.ParseFloat(self.currToken.Value, 64)
		if err != nil {
			panic(fmt.Sprintf("Number must be parseable: %s", err.Error()))
		}
		if err := self.advance(); err != nil {
			return nil, err
		}
		return AtomNumber{
			Num: float64(number),
			Range: errors.Span{
				Start: startLocation,
				End:   self.prevToken.EndLocation,
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
				Start: startLocation,
				End:   self.prevToken.EndLocation,
			},
		}, nil
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
					Start: startLocation,
					End:   self.currToken.EndLocation,
				},
			}, nil
		}
		// Atom is not a pair but a string
		return AtomString{
			Content: self.prevToken.Value,
			Range: errors.Span{
				Start: startLocation,
				End:   self.prevToken.EndLocation,
			},
		}, nil
	case Identifier:
		if err := self.advance(); err != nil {
			return nil, err
		}
		return AtomIdentifier{
			Identifier: self.prevToken.Value,
			Range: errors.Span{
				Start: startLocation,
				End:   self.prevToken.EndLocation,
			},
		}, nil
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
					Start: self.currToken.StartLocation,
					End:   self.currToken.EndLocation,
				},
			}
		}
		if err := self.advance(); err != nil {
			return nil, err
		}
		return AtomExpression{
			Expression: nestedExpr,
			Range: errors.Span{
				Start: startLocation,
				End:   self.prevToken.EndLocation,
			},
		}, nil
	}
	return nil, nil
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
					Start: startLocation,
					End:   self.currToken.EndLocation,
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
				Start: startLocation,
				End:   self.currToken.EndLocation,
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
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	headIdentifier := self.currToken.Value
	if err := self.advance(); err != nil {
		return AtomFor{}, err
	}

	// Make in
	if self.currToken.Kind != In {
		return AtomFor{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected %v, found %v", In, self.currToken.Kind),
			Span: errors.Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	if err := self.advance(); err != nil {
		return AtomFor{}, err
	}

	// Make range
	rangeLowerExpr, err := self.expression()
	if err != nil {
		return AtomFor{}, err
	}
	if self.currToken.Kind != Range {
		return AtomFor{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected range (%v), found %v", Range, self.currToken.Kind),
			Span: errors.Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	if err := self.advance(); err != nil {
		return AtomFor{}, err
	}
	rangeUpperExpr, err := self.expression()
	if err != nil {
		return AtomFor{}, err
	}

	iterationBlock, err := self.curlyBlock()
	if err != nil {
		return AtomFor{}, err
	}

	return AtomFor{
		HeadIdentifier: headIdentifier,
		RangeLowerExpr: rangeLowerExpr,
		RangeUpperExpr: rangeUpperExpr,
		IterationCode:  iterationBlock,
		Range: errors.Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}, nil
}

func (self *parser) whileExpr() (AtomWhile, *errors.Error) {
	startLocation := self.currToken.StartLocation
	if err := self.advance(); err != nil {
		return AtomWhile{}, err
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
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}, nil
}

func (self *parser) fnExpr() (AtomFunction, *errors.Error) {
	startLocation := self.currToken.StartLocation
	// Skip fn
	if err := self.advance(); err != nil {
		return AtomFunction{}, err
	}

	// Make function identifier
	if self.currToken.Kind != Identifier {
		return AtomFunction{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
			Span: errors.Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	functionName := self.currToken.Value
	if err := self.advance(); err != nil {
		return AtomFunction{}, err
	}

	// Expect Lparen (
	if self.currToken.Kind != LParen {
		return AtomFunction{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected (, found %v", self.currToken.Kind),
			Span: errors.Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	if err := self.advance(); err != nil {
		return AtomFunction{}, err
	}

	// Make args
	args := make([]string, 0)

	// Only make args if there is no immediate closing bracket
	if self.currToken.Kind != RParen {
		// Add initial argument
		if self.currToken.Kind != Identifier {
			return AtomFunction{}, &errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
				Span: errors.Span{
					Start: self.currToken.StartLocation,
					End:   self.currToken.EndLocation,
				},
			}
		}
		args = append(args, self.currToken.Value)
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
						Start: self.currToken.StartLocation,
						End:   self.currToken.EndLocation,
					},
				}
			}
			args = append(args, self.currToken.Value)
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
					Start: self.currToken.StartLocation,
					End:   self.currToken.EndLocation,
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
		Name:           functionName,
		ArgIdentifiers: args,
		Body:           functionBlock,
		Range: errors.Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	if err := self.advance(); err != nil {
		return AtomTry{}, err
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
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	// Copy the assignment identifier name
	assignIdentifier := self.currToken.Value
	if err := self.advance(); err != nil {
		return LetStmt{}, err
	}

	if self.currToken.Kind != Assign {
		return LetStmt{}, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Expected assignment operator '=', found %v", self.currToken.Kind),
			Span: errors.Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
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
		Left:  assignIdentifier,
		Right: assignExpression,
		Range: errors.Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
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
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	if err := self.advance(); err != nil {
		return ImportStmt{}, err
	}

	// Make optional name rewrite
	var rewriteName *string = nil
	if self.currToken.Kind == As {
		if err := self.advance(); err != nil {
			return ImportStmt{}, err
		}
		if self.currToken.Kind != Identifier {
			return ImportStmt{}, &errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Expected identifier, found %v", self.currToken.Kind),
				Span: errors.Span{
					Start: self.currToken.StartLocation,
					End:   self.currToken.EndLocation,
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
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
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
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
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
			Start: startLocation,
			End:   self.prevToken.EndLocation,
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
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	return breakStmt, nil
}

func (self *parser) continueStmt() (ContinueStmt, *errors.Error) {
	continueStmt := ContinueStmt{
		Range: errors.Span{
			Start: self.currToken.StartLocation,
			End:   self.currToken.EndLocation,
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
	returnStmt := ReturnStmt{
		Expression: additionalExpr,
		Range: errors.Span{
			Start: startLocation,
			End:   self.currToken.EndLocation,
		},
	}
	return returnStmt, nil
}

func (self *parser) statement() (Statement, *errors.Error) {
	switch self.currToken.Kind {
	case Let:
		return self.letStmt()
	case Import:
		return self.importStmt()
	case Break:
		return self.breakStmt()
	case Continue:
		return self.continueStmt()
	case Return:
		return self.returnStmt()
	default:
		canBeExpr := false
		for _, possible := range firstExpr {
			if self.currToken.Kind == possible {
				canBeExpr = true
				break
			}
		}
		if canBeExpr {
			expr, err := self.expression()
			if err != nil {
				return nil, err
			}
			return ExpressionStmt{
				Expression: expr,
			}, nil
		}
		return nil, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Invalid expression: expected one of the tokens 'let', 'import', 'break', 'continue', 'return', found %v", self.currToken.Kind),
			Span: errors.Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
}

func (self *parser) statements() ([]Statement, *errors.Error) {
	statements := make([]Statement, 0)

	// If statements is invoked in an empty block or at the end of the file, quit immediately
	if self.currToken.Kind == EOF || self.currToken.Kind == RCurly {
		return statements, nil
	}

	// Make additional statements
	for {
		if self.currToken.Kind == RCurly || self.currToken.Kind == EOF {
			break
		}

		statement, err := self.statement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, statement)

		if self.currToken.Kind != Semicolon {
			return nil, &errors.Error{
				Kind:    errors.SyntaxError,
				Message: fmt.Sprintf("Expected semicolon, found %v (Hint: add ';' after previous statement)", self.currToken.Kind),
				Span: errors.Span{
					Start: self.currToken.StartLocation,
					End:   self.currToken.EndLocation,
				},
			}
		}

		if self.advance(); err != nil {
			return nil, err
		}
	}

	return statements, nil
}

func (self *parser) curlyBlock() (Block, *errors.Error) {
	if self.currToken.Kind != LCurly {
		return nil, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Invalid block: expected %v, found %v", LCurly, self.currToken.Kind),
			Span: errors.Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	if err := self.advance(); err != nil {
		return nil, err
	}
	statements, err := self.statements()
	if err != nil {
		return nil, err
	}
	if self.currToken.Kind != RCurly {
		return nil, &errors.Error{
			Kind:    errors.SyntaxError,
			Message: fmt.Sprintf("Invalid block: expected %v, found %v", RCurly, self.currToken.Kind),
			Span: errors.Span{
				Start: self.currToken.StartLocation,
				End:   self.currToken.EndLocation,
			},
		}
	}
	if err := self.advance(); err != nil {
		return nil, err
	}
	return statements, nil
}

func (self *parser) parse() ([]Statement, *errors.Error) {
	if err := self.advance(); err != nil {
		return nil, err
	}
	statements, err := self.statements()
	if err != nil {
		return nil, err
	}
	return statements, nil
}
