package compiler

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

//
// If expression,
//

func (self *Compiler) compileIfExpr(node ast.AnalyzedIfExpression) {
	self.compileExpr(node.Condition)

	after_label := self.mangleLabel("if_after")
	else_label := self.mangleLabel("else")

	if node.ElseBlock != nil {
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, else_label), node.Range)
	} else {
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, after_label), node.Range)
	}
	self.compileBlock(node.ThenBlock, true)
	self.insert(newOneStringInstruction(Opcode_Jump, after_label), node.Range)

	if node.ElseBlock != nil {
		self.insert(newOneStringInstruction(Opcode_Label, else_label), node.Range)
		self.compileBlock(*node.ElseBlock, true)
	}
	self.insert(newOneStringInstruction(Opcode_Label, after_label), node.Range)
}

//
// Prefix expressions.
//

func (self *Compiler) compilePrefixOp(op ast.PrefixOperator, span errors.Span) {
	switch op {
	case ast.MinusPrefixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Neg), span)
	case ast.NegatePrefixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Not), span)
	case ast.IntoSomePrefixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Some), span)
	}
}

//
// Call expressions.
//

func (self *Compiler) compileCallExpr(node ast.AnalyzedCallExpression) {
	// Push each argument onto the stack
	// The order is reversed so that later popping can be done naturally
	for i := len(node.Arguments.List) - 1; i >= 0; i-- {
		self.compileExpr(node.Arguments.List[i].Expression)
	}

	if node.Base.Kind() == ast.IdentExpressionKind {
		base := node.Base.(ast.AnalyzedIdentExpression)

		// Special case: base is `throw`
		if base.Ident.Ident() == "throw" {
			self.insert(newPrimitiveInstruction(Opcode_Throw), node.Range)
			return
		}

		// Check whether the scope is local or global
		_, found := self.getMangled(base.Ident.Ident())
		if found {
			if node.IsSpawn {
				panic("This is an impossible state.")
			}

			self.compileExpr(node.Base)
			self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(int64(len(node.Arguments.List)))), node.Span())
			self.insert(newPrimitiveInstruction(Opcode_Call_Val), node.Span())
		} else {
			// TODO: the span mapping is broken here?
			name, found := self.getMangledFn(base.Ident.Ident())
			if found {
				opcode := Opcode_Call_Imm
				if node.IsSpawn {
					opcode = Opcode_Spawn
					self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(int64(len(node.Arguments.List)))), node.Span())
				}

				self.insert(newOneStringInstruction(opcode, name), node.Span())
			} else {
				// call a global value
				self.insert(newOneStringInstruction(Opcode_GetGlobImm, base.Ident.Ident()), node.Range)
				self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(int64(len(node.Arguments.List)))), node.Span())
				self.insert(newPrimitiveInstruction(Opcode_Call_Val), node.Range)
			}
		}
	} else {
		if node.IsSpawn {
			panic("This is an impossible state.")
		}

		self.compileExpr(node.Base)

		// insert number of args
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(int64(len(node.Arguments.List)))), node.Span())

		// perform the actual call
		self.insert(newPrimitiveInstruction(Opcode_Call_Val), node.Span())
	}
}

//
// Infix expressions.
//

func (self *Compiler) compileInfixExpr(node ast.AnalyzedInfixExpression) {
	switch node.Operator {
	case pAst.LogicalOrInfixOperator:
		returnTrue := self.mangleLabel("return_true")
		afterLabel := self.mangleLabel("after_infix")

		self.compileExpr(node.Lhs)
		self.insert(newPrimitiveInstruction(Opcode_Not), node.Range)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, returnTrue), node.Range)

		self.compileExpr(node.Rhs)
		self.insert(newOneStringInstruction(Opcode_Jump, afterLabel), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, returnTrue), node.Range)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueBool(true)), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, afterLabel), node.Range)
	case pAst.LogicalAndInfixOperator:
		returnFalse := self.mangleLabel("return_false")
		afterLabel := self.mangleLabel("after_infix")

		self.compileExpr(node.Lhs)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, returnFalse), node.Range)

		self.compileExpr(node.Rhs)
		self.insert(newOneStringInstruction(Opcode_Jump, afterLabel), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, returnFalse), node.Range)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueBool(false)), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, afterLabel), node.Range)
	default:
		self.compileExpr(node.Lhs)
		self.compileExpr(node.Rhs)
		self.arithmeticHelper(node.Operator, node.Range)
	}
}

func (self *Compiler) arithmeticHelper(op pAst.InfixOperator, span errors.Span) {
	switch op {
	case pAst.PlusInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Add), span)
	case pAst.MinusInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Sub), span)
	case pAst.MultiplyInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Mul), span)
	case pAst.DivideInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Div), span)
	case pAst.ModuloInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Rem), span)
	case pAst.PowerInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Pow), span)
	case pAst.ShiftLeftInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Shl), span)
	case pAst.ShiftRightInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Shr), span)
	case pAst.BitOrInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_BitOr), span)
	case pAst.BitAndInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_BitAnd), span)
	case pAst.BitXorInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_BitXor), span)
	case pAst.EqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Eq), span)
	case pAst.NotEqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Eq), span)
		self.insert(newPrimitiveInstruction(Opcode_Not), span)
	case pAst.LessThanInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Lt), span)
	case pAst.LessThanEqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Le), span)
	case pAst.GreaterThanInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Gt), span)
	case pAst.GreaterThanEqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Ge), span)
	default:
		panic("Unreachable")
	}
}

//
// Ident expressions.
//

func (self *Compiler) compileIdentExpression(node ast.AnalyzedIdentExpression) {
	name, varFound := self.getMangled(node.Ident.Ident())

	opCode := Opcode_GetVarImm
	if node.IsGlobal {
		opCode = Opcode_GetGlobImm
	}

	if varFound {
		self.insert(newOneStringInstruction(opCode, name), node.Span())
		return
	}

	name, fnFound := self.getMangledFn(node.Ident.Ident())
	if fnFound {
		// This value is a function, it should also be wrapped like one
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueVMFunction(
			name,
		)), node.Span())
	} else {
		// This value is not a function. Instead, it is a global variable.
		self.insert(newOneStringInstruction(opCode, node.Ident.Ident()), node.Span())
	}
}

//
// Generic expressions.
//

func (self *Compiler) compileExpr(node ast.AnalyzedExpression) {
	switch node.Kind() {
	case ast.UnknownExpressionKind:
		panic("Unreachable, this should not happen")
	case ast.IntLiteralExpressionKind:
		node := node.(ast.AnalyzedIntLiteralExpression)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(node.Value)), node.Range)
	case ast.FloatLiteralExpressionKind:
		node := node.(ast.AnalyzedFloatLiteralExpression)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueFloat(node.Value)), node.Range)
	case ast.BoolLiteralExpressionKind:
		node := node.(ast.AnalyzedBoolLiteralExpression)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueBool(node.Value)), node.Range)
	case ast.StringLiteralExpressionKind:
		node := node.(ast.AnalyzedStringLiteralExpression)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueString(node.Value)), node.Range)
	case ast.IdentExpressionKind:
		self.compileIdentExpression(node.(ast.AnalyzedIdentExpression))
	case ast.NullLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueNull()), node.Span())
	case ast.NoneLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewNoneOption()), node.Span())
	case ast.RangeLiteralExpressionKind:
		node := node.(ast.AnalyzedRangeLiteralExpression)
		self.compileExpr(node.Start)
		self.compileExpr(node.End)
		// Boolean instruction is used to mark that the end of this range can be inclusive.
		self.insert(newOneBoolInstruction(Opcode_Into_Range, node.EndIsInclusive), node.Range)
	case ast.ListLiteralExpressionKind:
		node := node.(ast.AnalyzedListLiteralExpression)
		// TODO: is this cloning required???
		self.insert(newValueInstruction(Opcode_Cloning_Push, *value.NewValueList(make([]*value.Value, 0))), node.Range)

		for _, element := range node.Values {
			self.compileExpr(element)
			self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(2)), node.Range)
			self.insert(newOneStringInstruction(Opcode_HostCall, LIST_PUSH), node.Range)
		}
	case ast.AnyObjectLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Cloning_Push, *value.NewValueAnyObject(make(map[string]*value.Value))), node.Span())
	case ast.ObjectLiteralExpressionKind:
		node := node.(ast.AnalyzedObjectLiteralExpression)

		fields := make(map[string]*value.Value)
		for _, field := range node.Fields {
			fields[field.Key.Ident()] = value.ZeroValue(field.Expression.Type())
		}

		object := *value.NewValueObject(fields)
		self.insert(newValueInstruction(Opcode_Cloning_Push, object), node.Range)

		for _, field := range node.Fields {
			self.insert(newPrimitiveInstruction(Opcode_Duplicate), node.Range)
			self.insert(newOneStringInstruction(Opcode_Member, field.Key.Ident()), node.Range)
			self.compileExpr(field.Expression)
			self.insert(newPrimitiveInstruction(Opcode_Assign), node.Range)
		}
	case ast.FunctionLiteralExpressionKind:
		node := node.(ast.AnalyzedFunctionLiteralExpression)

		fnName := self.mangleFn("$lambda")
		self.addFn(fnName, fnName)

		oldCurrFn := self.currFn

		self.compileFn(ast.AnalyzedFunctionDefinition{
			Ident: pAst.NewSpannedIdent(fnName, node.Range),
			Parameters: ast.AnalyzedFunctionParams{
				List: node.Parameters,
				Span: node.ParamSpan,
			},
			ReturnType: node.ReturnType,
			Body:       node.Body,
			Modifier:   pAst.FN_MODIFIER_NONE,
			Range:      node.Range,
		})
		self.currFn = oldCurrFn

		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueVMFunction(fnName)), node.Span())
	case ast.GroupedExpressionKind:
		node := node.(ast.AnalyzedGroupedExpression)
		self.compileExpr(node.Inner)
	case ast.PrefixExpressionKind:
		node := node.(ast.AnalyzedPrefixExpression)
		self.compileExpr(node.Base)
		self.compilePrefixOp(node.Operator, node.Range)
	case ast.InfixExpressionKind:
		node := node.(ast.AnalyzedInfixExpression)
		self.compileInfixExpr(node)
	case ast.AssignExpressionKind:
		node := node.(ast.AnalyzedAssignExpression)

		// TODO: implement all different assignment operators
		if node.Lhs.Kind() == ast.IdentExpressionKind {
			lhs := node.Lhs.(ast.AnalyzedIdentExpression)
			name, found := self.getMangled(lhs.Ident.Ident())
			if !found {
				name = lhs.Ident.Ident()
			}

			opCodeGet := Opcode_GetVarImm
			opCodeSet := Opcode_SetVarImm

			if lhs.IsGlobal {
				opCodeGet = Opcode_GetGlobImm
				opCodeSet = Opcode_SetGlobImm
			}

			if node.Operator != pAst.StdAssignOperatorKind {
				self.insert(newOneStringInstruction(opCodeGet, name), node.Range)
				self.compileExpr(node.Rhs)
				self.arithmeticHelper(node.Operator.IntoInfixOperator(), node.Range)
			} else {
				self.compileExpr(node.Rhs)
			}

			self.insert(newOneStringInstruction(opCodeSet, name), node.Range)
		} else {
			self.compileExpr(node.Lhs)

			if node.Operator != pAst.StdAssignOperatorKind {
				self.insert(newPrimitiveInstruction(Opcode_Duplicate), node.Range)
				self.compileExpr(node.Rhs)
				self.arithmeticHelper(node.Operator.IntoInfixOperator(), node.Range)
			} else {
				self.compileExpr(node.Rhs)
			}

			self.insert(newPrimitiveInstruction(Opcode_Assign), node.Range)
		}
	case ast.CallExpressionKind:
		node := node.(ast.AnalyzedCallExpression)
		self.compileCallExpr(node)
	case ast.IndexExpressionKind:
		node := node.(ast.AnalyzedIndexExpression)
		self.compileExpr(node.Base)
		self.compileExpr(node.Index)
		self.insert(newPrimitiveInstruction(Opcode_Index), node.Range)
	case ast.MemberExpressionKind:
		node := node.(ast.AnalyzedMemberExpression)
		self.compileExpr(node.Base)

		opcode := Opcode_Nop
		var additionalInst Instruction

		switch node.Operator {
		case pAst.DotMemberOperator:
			opcode = Opcode_Member
		case pAst.ArrowMemberOperator:
			opcode = Opcode_Member_Anyobj
		case pAst.TildeArrowMemberOperator:
			opcode = Opcode_Member_Anyobj
			additionalInst = newPrimitiveInstruction(Opcode_Member_Unwrap)
		}

		self.insert(newOneStringInstruction(opcode, node.Member.Ident()), node.Range)
		if additionalInst != nil {
			self.insert(additionalInst, node.Range)
		}

	case ast.CastExpressionKind:
		node := node.(ast.AnalyzedCastExpression)
		self.compileExpr(node.Base)
		self.insert(newCastInstruction(node.AsType, true), node.Range)
	case ast.BlockExpressionKind:
		self.compileBlock(node.(ast.AnalyzedBlockExpression).Block, true)
	case ast.IfExpressionKind:
		self.compileIfExpr(node.(ast.AnalyzedIfExpression))
	case ast.MatchExpressionKind:
		node := node.(ast.AnalyzedMatchExpression)

		// push the control value onto the stack
		self.compileExpr(node.ControlExpression)

		branches := make(map[int]string)
		after_branch := self.mangleLabel("match_after")

		for i, option := range node.Arms {
			name := self.mangleLabel("case")
			branches[i] = name

			// Insert value to compare with
			self.compileExpr(option.Literal)

			// Compare control and branch value
			// TODO: could DUP also work?
			self.insert(newPrimitiveInstruction(Opcode_Eq_PopOnce), node.Range)

			// if true, jump to the label of this branch
			self.insert(newPrimitiveInstruction(Opcode_Not), node.Range)
			self.insert(newOneStringInstruction(Opcode_JumpIfFalse, name), node.Range)
		}

		default_branch := self.mangleLabel("match_default")
		if node.DefaultArmAction != nil {
			self.insert(newOneStringInstruction(Opcode_Jump, default_branch), node.Range)
		} else {
			self.insert(newOneStringInstruction(Opcode_Jump, after_branch), node.Range)
		}

		// Each individual branch
		for i, option := range node.Arms {
			self.insert(newOneStringInstruction(Opcode_Label, branches[i]), node.Range)
			// Insert a `drop` since a eq_poponce was used
			self.insert(newPrimitiveInstruction(Opcode_Drop), node.Range)
			self.compileExpr(option.Action)
			self.insert(newOneStringInstruction(Opcode_Jump, after_branch), node.Range)
		}

		if node.DefaultArmAction != nil {
			self.insert(newOneStringInstruction(Opcode_Label, default_branch), node.Range)
			self.compileExpr(*node.DefaultArmAction)
			self.insert(newOneStringInstruction(Opcode_Jump, after_branch), node.Range)
		}

		self.insert(newOneStringInstruction(Opcode_Label, after_branch), node.Range)
	case ast.TryExpressionKind:
		mangledCurr, found := self.getMangledFn(self.currFn)
		if !found {
			panic("Impossible state: every current function should also be found")
		}

		node := node.(ast.AnalyzedTryExpression)
		exceptionLabel := self.mangleLabel("exception_label")
		afterCatchLabel := self.mangleLabel("after_catch_label")
		self.insert(newTwoStringInstruction(Opcode_SetTryLabel, mangledCurr, exceptionLabel), node.Range)
		self.compileBlock(node.TryBlock, true)
		self.insert(newPrimitiveInstruction(Opcode_PopTryLabel), node.Range)
		self.insert(newOneStringInstruction(Opcode_Jump, afterCatchLabel), node.Range)

		// exception case
		mangledExceptionName := self.mangleVar(node.CatchIdent.Ident())
		self.insert(newOneStringInstruction(Opcode_Label, exceptionLabel), node.Range)
		self.pushScope()
		defer self.popScope()
		self.insert(newOneStringInstruction(Opcode_SetVarImm, mangledExceptionName), node.Range)
		self.insert(newPrimitiveInstruction(Opcode_PopTryLabel), node.Range)
		self.compileBlock(node.CatchBlock, false)
		self.insert(newOneStringInstruction(Opcode_Label, afterCatchLabel), node.Range)
	default:
		panic("Unreachable")
	}
}
