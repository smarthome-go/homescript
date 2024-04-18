package evaluator

import (
	"fmt"
	"math"

	"github.com/davecgh/go-spew/spew"
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

func (i *Interpreter) Expression(node ast.AnalyzedExpression) (*value.Value, *value.VmInterrupt) {
	switch node.Kind() {
	case ast.IntLiteralExpressionKind:
		return value.NewValueInt(node.(ast.AnalyzedIntLiteralExpression).Value), nil
	case ast.FloatLiteralExpressionKind:
		return value.NewValueFloat(node.(ast.AnalyzedFloatLiteralExpression).Value), nil
	case ast.BoolLiteralExpressionKind:
		return value.NewValueBool(node.(ast.AnalyzedBoolLiteralExpression).Value), nil
	case ast.StringLiteralExpressionKind:
		return value.NewValueString(node.(ast.AnalyzedStringLiteralExpression).Value), nil
	case ast.IdentExpressionKind:
		const errMsg = "Value or function `%s` was not found"
		node := node.(ast.AnalyzedIdentExpression).Ident

		val, found := i.currModule.scopes[i.currModule.currentScope][node.Ident()]
		if !found {
			return nil, value.NewVMFatalException(
				fmt.Sprintf(errMsg, node),
				value.Vm_HostErrorKind,
				node.Span(),
			)
		} else {
			return val, nil
		}

	case ast.NullLiteralExpressionKind:
		return value.NewValueNull(), nil
	case ast.NoneLiteralExpressionKind:
		return value.NewNoneOption(), nil
	case ast.RangeLiteralExpressionKind:
		node := node.(ast.AnalyzedRangeLiteralExpression)

		start, err := i.Expression(node.Start)
		if err != nil {
			return nil, err
		}

		end, err := i.Expression(node.End)
		if err != nil {
			return nil, err
		}

		return value.NewValueRange(*start, *end, node.EndIsInclusive), nil
	case ast.ListLiteralExpressionKind:
		node := node.(ast.AnalyzedListLiteralExpression)

		values := make([]*value.Value, len(node.Values))

		for idx, expr := range node.Values {
			evalV, err := i.Expression(expr)
			if err != nil {
				return nil, err
			}

			values[idx] = evalV
		}

		return value.NewValueList(values), nil
	case ast.AnyObjectLiteralExpressionKind:
		return value.NewValueAnyObject(make(map[string]*value.Value)), nil
	case ast.ObjectLiteralExpressionKind:
		node := node.(ast.AnalyzedObjectLiteralExpression)

		fields := make(map[string]*value.Value)
		for _, field := range node.Fields {
			val, err := i.Expression(field.Expression)
			if err != nil {
				return nil, err
			}

			fields[field.Key.Ident()] = val
		}

		return value.NewValueObject(fields), nil
	case ast.FunctionLiteralExpressionKind:
		panic("Function literals are not supported")
	case ast.GroupedExpressionKind:
		return i.Expression(node.(ast.AnalyzedGroupedExpression).Inner)
	case ast.PrefixExpressionKind:
		return i.PrefixExpr(node.(ast.AnalyzedPrefixExpression))
	case ast.InfixExpressionKind:
		return i.InfixExpr(node.(ast.AnalyzedInfixExpression))
	case ast.AssignExpressionKind:
		return i.assignExpression(node.(ast.AnalyzedAssignExpression))
	case ast.CallExpressionKind:
		panic("Unreachable: this expression is not constant")
	case ast.IndexExpressionKind:
		return i.indexExpression(node.(ast.AnalyzedIndexExpression))
	case ast.MemberExpressionKind:
		return i.memberExpression(node.(ast.AnalyzedMemberExpression))
	case ast.CastExpressionKind:
		return i.castExpression(node.(ast.AnalyzedCastExpression))
	case ast.BlockExpressionKind:
		return i.block(node.(ast.AnalyzedBlockExpression).Block, true)
	case ast.IfExpressionKind:
		return i.ifExpression(node.(ast.AnalyzedIfExpression))
	case ast.MatchExpressionKind:
		return i.matchExpression(node.(ast.AnalyzedMatchExpression))
	case ast.TryExpressionKind:
		return i.tryExpression(node.(ast.AnalyzedTryExpression))
	default:
		panic("Unreachable: a new expression kind was added without updating this code")
	}
}

//
// Prefix expressions.
//

func (i *Interpreter) PrefixExpr(node ast.AnalyzedPrefixExpression) (*value.Value, *value.VmInterrupt) {
	base, err := i.Expression(node.Base)
	if err != nil {
		return nil, err
	}

	switch node.Operator {
	case ast.MinusPrefixOperator:
		switch node.Base.Type().Kind() {
		case ast.IntTypeKind:
			newBase := -(*base).(value.ValueInt).Inner
			return value.NewValueInt(newBase), nil
		case ast.FloatTypeKind:
			newBase := -(*base).(value.ValueFloat).Inner
			return value.NewValueFloat(newBase), nil
		default:
			panic("Unreachable: cannot apply this operator on this type")
		}
	case ast.NegatePrefixOperator:
		switch node.Base.Type().Kind() {
		case ast.BoolTypeKind:
			newBase := !(*base).(value.ValueBool).Inner
			return value.NewValueBool(newBase), nil
		default:
			panic("Unreachable: cannot apply this operator on this type")
		}
	case ast.IntoSomePrefixOperator:
		return value.NewValueOption(base), nil
	default:
		panic("A new kind of prefix operator was added without updating this code")
	}
}

//
// Infix expressions.
//

func (i *Interpreter) InfixExpr(node ast.AnalyzedInfixExpression) (*value.Value, *value.VmInterrupt) {
	res, _, err := i.infixHelper(node.Lhs, node.Rhs, node.Operator)
	return res, err
}

func (i *Interpreter) infixHelper(lhs ast.AnalyzedExpression, rhs ast.AnalyzedExpression, operator pAst.InfixOperator) (res *value.Value, lhsAddr *value.Value, err *value.VmInterrupt) {
	switch operator {
	case pAst.EqualInfixOperator:
		lhs, err := i.Expression(lhs)
		if err != nil {
			return nil, nil, err
		}
		rhs, err := i.Expression(rhs)
		if err != nil {
			return nil, nil, err
		}

		res, err := (*lhs).IsEqual(*rhs)
		if err != nil {
			return nil, nil, err
		}
		return value.NewValueBool(res), lhs, nil
	case pAst.NotEqualInfixOperator:
		lhs, err := i.Expression(lhs)
		if err != nil {
			return nil, nil, err
		}
		rhs, err := i.Expression(rhs)
		if err != nil {
			return nil, nil, err
		}

		res, err := (*lhs).IsEqual(*rhs)
		if err != nil {
			return nil, nil, err
		}
		return value.NewValueBool(!res), lhs, nil
	}

	switch lhs.Type().Kind() {
	case ast.IntTypeKind:
		var intRes int64

		lhsVal, err := i.Expression(lhs)
		if err != nil {
			return nil, nil, err
		}
		rhsVal, err := i.Expression(rhs)
		if i != nil {
			return nil, nil, err
		}

		lhsInt := (*lhsVal).(value.ValueInt)
		rhsInt := (*rhsVal).(value.ValueInt)

		// TODO: add checked operations + runtime crashes

		switch operator {
		case pAst.PlusInfixOperator:
			intRes = lhsInt.Inner + rhsInt.Inner
		case pAst.MinusInfixOperator:
			intRes = lhsInt.Inner - rhsInt.Inner
		case pAst.MultiplyInfixOperator:
			intRes = lhsInt.Inner * rhsInt.Inner
		case pAst.DivideInfixOperator:
			intRes = lhsInt.Inner / rhsInt.Inner
		case pAst.ModuloInfixOperator:
			intRes = lhsInt.Inner % rhsInt.Inner
		case pAst.PowerInfixOperator:
			intRes = int64(math.Pow(float64(lhsInt.Inner), float64(rhsInt.Inner)))
		case pAst.ShiftLeftInfixOperator:
			intRes = lhsInt.Inner << rhsInt.Inner
		case pAst.ShiftRightInfixOperator:
			intRes = lhsInt.Inner >> rhsInt.Inner
		case pAst.BitOrInfixOperator:
			intRes = lhsInt.Inner | rhsInt.Inner
		case pAst.BitAndInfixOperator:
			intRes = lhsInt.Inner & rhsInt.Inner
		case pAst.BitXorInfixOperator:
			intRes = lhsInt.Inner ^ rhsInt.Inner
		case pAst.LessThanInfixOperator:
			return value.NewValueBool(lhsInt.Inner < rhsInt.Inner), lhsVal, nil
		case pAst.LessThanEqualInfixOperator:
			return value.NewValueBool(lhsInt.Inner <= rhsInt.Inner), lhsVal, nil
		case pAst.GreaterThanInfixOperator:
			return value.NewValueBool(lhsInt.Inner > rhsInt.Inner), lhsVal, nil
		case pAst.GreaterThanEqualInfixOperator:
			return value.NewValueBool(lhsInt.Inner >= rhsInt.Inner), lhsVal, nil
		default:
			panic("A new operator kind was introduced without updating this code")
		}
		return value.NewValueInt(intRes), lhsVal, nil
	case ast.FloatTypeKind:
		var floatRes float64

		lhsVal, err := i.Expression(lhs)
		if err != nil {
			return nil, nil, err
		}
		rhsVal, err := i.Expression(rhs)
		if err != nil {
			return nil, nil, err
		}

		lhsFloat := (*lhsVal).(value.ValueFloat)
		rhsFloat := (*rhsVal).(value.ValueFloat)

		// TODO: add checked operations + runtime crashes

		switch operator {
		case pAst.PlusInfixOperator:
			floatRes = lhsFloat.Inner + rhsFloat.Inner
		case pAst.MinusInfixOperator:
			floatRes = lhsFloat.Inner - rhsFloat.Inner
		case pAst.MultiplyInfixOperator:
			floatRes = lhsFloat.Inner * rhsFloat.Inner
		case pAst.DivideInfixOperator:
			floatRes = lhsFloat.Inner / rhsFloat.Inner
		case pAst.PowerInfixOperator:
			floatRes = math.Pow(lhsFloat.Inner, rhsFloat.Inner)
		case pAst.LessThanInfixOperator:
			return value.NewValueBool(lhsFloat.Inner < rhsFloat.Inner), lhsVal, nil
		case pAst.LessThanEqualInfixOperator:
			return value.NewValueBool(lhsFloat.Inner <= rhsFloat.Inner), lhsVal, nil
		case pAst.GreaterThanInfixOperator:
			return value.NewValueBool(lhsFloat.Inner > rhsFloat.Inner), lhsVal, nil
		case pAst.GreaterThanEqualInfixOperator:
			return value.NewValueBool(lhsFloat.Inner >= rhsFloat.Inner), lhsVal, nil
		default:
			panic("A new operator kind was introduced without updating this code")
		}
		return value.NewValueFloat(floatRes), lhsVal, nil
	case ast.BoolTypeKind:
		var lhsVal, rhsVal *value.Value
		var lhsBool, rhsBool, boolRes bool

		if operator != pAst.LogicalOrInfixOperator && operator != pAst.LogicalAndInfixOperator {
			lhsTemp, err := i.Expression(lhs)
			if err != nil {
				return nil, nil, err
			}
			rhsTemp, i := i.Expression(rhs)
			if i != nil {
				return nil, nil, i
			}
			lhsVal = lhsTemp
			rhsVal = rhsTemp

			lhsBool = (*lhsVal).(value.ValueBool).Inner
			rhsBool = (*rhsVal).(value.ValueBool).Inner
		}

		switch operator {
		case pAst.BitOrInfixOperator:
			boolRes = lhsBool || rhsBool
		case pAst.BitAndInfixOperator:
			boolRes = lhsBool && rhsBool
		case pAst.BitXorInfixOperator:
			boolRes = lhsBool != rhsBool
		case pAst.LogicalOrInfixOperator:
			lhsTemp, err := i.Expression(lhs)
			if err != nil {
				return nil, nil, err
			}

			if (*lhsTemp).(value.ValueBool).Inner {
				return value.NewValueBool(true), lhsVal, nil
			}

			rhsTemp, err := i.Expression(rhs)
			if i != nil {
				return nil, nil, err
			}
			return value.NewValueBool((*rhsTemp).(value.ValueBool).Inner), lhsVal, nil
		case pAst.LogicalAndInfixOperator:
			lhsTemp, err := i.Expression(lhs)
			if err != nil {
				return nil, nil, err
			}

			if !(*lhsTemp).(value.ValueBool).Inner {
				return value.NewValueBool(false), lhsVal, nil
			}

			rhsTemp, err := i.Expression(rhs)
			if err != nil {
				return nil, nil, err
			}
			return value.NewValueBool((*rhsTemp).(value.ValueBool).Inner), lhsVal, nil
		default:
			panic("A new operator kind was introduced without updating this code")
		}
		return value.NewValueBool(boolRes), lhsVal, nil
	case ast.StringTypeKind:
		lhsTemp, err := i.Expression(lhs)
		if err != nil {
			return nil, nil, err
		}
		rhsTemp, err := i.Expression(rhs)
		if i != nil {
			return nil, nil, err
		}

		switch operator {
		case pAst.PlusInfixOperator:
			strRes := (*lhsTemp).(value.ValueString).Inner + (*rhsTemp).(value.ValueString).Inner
			return value.NewValueString(strRes), lhsTemp, nil
		default:
			panic("A new operator kind was introduced without updating this code")
		}
	}
	panic("Unreachable: a new type which is allowed in infix-expressions was added without updating this code")
}

//
// Assign expression
//

func (i *Interpreter) assignExpression(node ast.AnalyzedAssignExpression) (*value.Value, *value.VmInterrupt) {
	if node.Operator == pAst.StdAssignOperatorKind {
		lhs, err := i.Expression(node.Lhs)
		if err != nil {
			return nil, err
		}
		rhs, err := i.Expression(node.Rhs)
		if err != nil {
			return nil, err
		}

		*lhs = *rhs // perform the assignment
		return value.NewValueNull(), nil
	}

	res, lhsAddr, err := i.infixHelper(node.Lhs, node.Rhs, node.Operator.IntoInfixOperator())
	if err != nil {
		return nil, err
	}

	// perform the assignment
	*lhsAddr = *res

	return value.NewValueNull(), nil
}

//
// Index expression
//

func (i *Interpreter) indexExpression(node ast.AnalyzedIndexExpression) (*value.Value, *value.VmInterrupt) {
	base, err := i.Expression(node.Base)
	if err != nil {
		return nil, err
	}

	index, err := i.Expression(node.Index)
	if err != nil {
		return nil, err
	}

	span := func() errors.Span {
		return node.Span()
	}

	return value.IndexValue(base, index, span)
}

//
// Member expression
//

func (i *Interpreter) memberExpression(node ast.AnalyzedMemberExpression) (*value.Value, *value.VmInterrupt) {
	base, err := i.Expression(node.Base)
	if err != nil {
		return nil, err
	}

	fields, err := (*base).Fields()
	if err != nil {
		return nil, err
	}

	// BUG: Here, the issue is that the actual fields loaded from the singleton do not include all the fields of the type.
	val, found := fields[node.Member.Ident()]
	if !found {
		panic(fmt.Sprintf("Field '%s' not found on value of type '%s' | node: %s | fields: %s", node.Member.Ident(), node.Base.Type(), node, spew.Sdump(fields)))
	}

	return val, nil
}

//
// Cast expression
//

func (i *Interpreter) castExpression(node ast.AnalyzedCastExpression) (*value.Value, *value.VmInterrupt) {
	base, err := i.Expression(node.Base)
	if err != nil {
		return nil, err
	}

	// TODO: remove this
	// Equal types, or cast from `any` to other -> check internal compatibility
	// if node.Base.Type().Kind() == ast.AnyTypeKind || node.Base.Type().Kind() == node.AsType.Kind() {
	// 	if i := self.valueIsCompatibleToType(*base, node.AsType, node.Range); i != nil {
	// 		return nil, i
	// 	}
	//
	// 	if (*base).Kind() == value.ObjectValueKind && node.AsType.Kind() == ast.AnyObjectTypeKind {
	// 		return (*base).(value.ValueObject).IntoAnyObject(), nil
	// 	}
	// 	return base, nil
	// }

	// TODO: implement a `deepCast` method which can convert [ {} ] -> [ { ? } ]
	val, castErr := value.DeepCast(*base, node.AsType, node.Span(), true)
	if castErr != nil {
		return nil, value.NewVMFatalException(
			castErr.Message(),
			value.Vm_CastErrorKind,
			castErr.Span,
		)
	}

	return val, nil
}

//
// If expression
//

func (i *Interpreter) ifExpression(node ast.AnalyzedIfExpression) (*value.Value, *value.VmInterrupt) {
	condition, err := i.Expression(node.Condition)
	if err != nil {
		return nil, err
	}

	// cast condition to bool
	conditionIsTrue := (*condition).(value.ValueBool).Inner

	resultValue := value.NewValueNull()

	if conditionIsTrue {
		blockRes, err := i.block(node.ThenBlock, true)
		if err != nil {
			return nil, err
		}
		*resultValue = *blockRes
	} else if node.ElseBlock != nil {
		blockRes, err := i.block(*node.ElseBlock, true)
		if err != nil {
			return nil, err
		}
		*resultValue = *blockRes
	}

	return resultValue, nil
}

//
// Match expression
//

func (i *Interpreter) matchExpression(node ast.AnalyzedMatchExpression) (*value.Value, *value.VmInterrupt) {
	control, err := i.Expression(node.ControlExpression)
	if err != nil {
		return nil, err
	}

	for _, arm := range node.Arms {
		literal, err := i.Expression(arm.Literal)
		if err != nil {
			return nil, err
		}

		// check if the literal is equal to the value of the control expr
		isEqual, err := (*literal).IsEqual(*control)
		if err != nil {
			return nil, err
		}

		if isEqual {
			return i.Expression(arm.Action)
		}
	}

	if node.DefaultArmAction != nil {
		return i.Expression(*node.DefaultArmAction)
	}

	return value.NewValueNull(), nil
}

//
// Try expression
//

func (self *Interpreter) tryExpression(node ast.AnalyzedTryExpression) (*value.Value, *value.VmInterrupt) {
	tryRes, i := self.block(node.TryBlock, true)
	if i == nil {
		return tryRes, nil
	}

	if (*i).Kind() != value.Vm_NormalExceptionInterruptKind {
		// only non-fatal interrupts can be caught
		return nil, i
	}

	throwError := (*i).(value.Vm_NormalException)
	self.pushScope()
	defer self.popScope()

	errObj := *value.NewValueObject(map[string]*value.Value{
		"message":  value.NewValueString(throwError.Message()),
		"line":     value.NewValueInt(int64(throwError.Span.Start.Line)),
		"column":   value.NewValueInt(int64(throwError.Span.Start.Column)),
		"filename": value.NewValueString(throwError.Span.Filename),
	})

	self.addVar(node.CatchIdent.Ident(), errObj)
	return self.block(node.CatchBlock, false)
}

//
// Block
//

func (i *Interpreter) block(node ast.AnalyzedBlock, handleScoping bool) (*value.Value, *value.VmInterrupt) {
	if handleScoping {
		i.pushScope()
		defer i.popScope()
	}

	for _, statement := range node.Statements {
		// TODO: handle errors differently
		panic("Statements are not supported in this interpreter")
		panic(statement)
		// if interrupt := self.statement(statement); interrupt != nil {
		// 	return nil, interrupt
		// }
	}

	if node.Expression != nil {
		return i.Expression(node.Expression)
	}

	return value.NewValueNull(), nil
}
