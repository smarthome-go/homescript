package interpreter

import (
	"fmt"
	"math"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

func (self *Interpreter) expression(node ast.AnalyzedExpression) (*value.Value, *value.Interrupt) {
	// Check for the cancelation signal
	if i := self.checkCancelation(node.Span()); i != nil {
		return nil, i
	}

	switch node.Kind() {
	case ast.IntLiteralExpressionKind:
		node := node.(ast.AnalyzedIntLiteralExpression)
		return value.NewValueInt(node.Value), nil
	case ast.FloatLiteralExpressionKind:
		node := node.(ast.AnalyzedFloatLiteralExpression)
		return value.NewValueFloat(node.Value), nil
	case ast.BoolLiteralExpressionKind:
		node := node.(ast.AnalyzedBoolLiteralExpression)
		return value.NewValueBool(node.Value), nil
	case ast.StringLiteralExpressionKind:
		node := node.(ast.AnalyzedStringLiteralExpression)
		return value.NewValueString(node.Value), nil
	case ast.IdentExpressionKind:
		node := node.(ast.AnalyzedIdentExpression)
		return self.getVar(node.Ident.Ident()), nil
	case ast.NullLiteralExpressionKind:
		return value.NewValueNull(), nil
	case ast.NoneLiteralExpressionKind:
		return value.NewNoneOption(), nil
	case ast.RangeLiteralExpressionKind:
		node := node.(ast.AnalyzedRangeLiteralExpression)
		return self.rangeLiteral(node)
	case ast.ListLiteralExpressionKind:
		node := node.(ast.AnalyzedListLiteralExpression)
		return self.listLiteral(node)
	case ast.AnyObjectLiteralExpressionKind:
		node := node.(ast.AnalyzedAnyObjectExpression)
		return self.anyObjectLiteral(node)
	case ast.ObjectLiteralExpressionKind:
		node := node.(ast.AnalyzedObjectLiteralExpression)
		return self.objectLiteral(node)
	case ast.FunctionLiteralExpressionKind:
		node := node.(ast.AnalyzedFunctionLiteralExpression)
		return self.functionLiteral(node)
	case ast.GroupedExpressionKind:
		node := node.(ast.AnalyzedGroupedExpression)
		return self.expression(node.Inner)
	case ast.PrefixExpressionKind:
		node := node.(ast.AnalyzedPrefixExpression)
		return self.prefixExpression(node)
	case ast.InfixExpressionKind:
		node := node.(ast.AnalyzedInfixExpression)
		return self.infixExpression(node)
	case ast.AssignExpressionKind:
		node := node.(ast.AnalyzedAssignExpression)
		return self.assignExpression(node)
	case ast.CallExpressionKind:
		node := node.(ast.AnalyzedCallExpression)
		base, i := self.expression(node.Base)
		if i != nil {
			return nil, i
		}
		// call the function and return the result
		return self.callFunc(node.Range, *base, node.Arguments)
	case ast.IndexExpressionKind:
		node := node.(ast.AnalyzedIndexExpression)
		return self.indexExpression(node)
	case ast.MemberExpressionKind:
		node := node.(ast.AnalyzedMemberExpression)
		return self.memberExpression(node)
	case ast.CastExpressionKind:
		node := node.(ast.AnalyzedCastExpression)
		return self.castExpression(node)
	case ast.BlockExpressionKind:
		node := node.(ast.AnalyzedBlockExpression).Block
		return self.block(node, true)
	case ast.IfExpressionKind:
		node := node.(ast.AnalyzedIfExpression)
		return self.ifExpression(node)
	case ast.MatchExpressionKind:
		node := node.(ast.AnalyzedMatchExpression)
		return self.matchExpression(node)
	case ast.TryExpressionKind:
		node := node.(ast.AnalyzedTryExpression)
		return self.tryExpression(node)
	default:
		panic(fmt.Sprintf("A new expression kind (%v) was introduced without updating this code ", node.Kind()))
	}
}

//
// Range literal
//

func (self *Interpreter) rangeLiteral(node ast.AnalyzedRangeLiteralExpression) (*value.Value, *value.Interrupt) {
	rangeStart, i := self.expression(node.Start)
	if i != nil {
		return nil, i
	}
	rangeEnd, i := self.expression(node.End)
	if i != nil {
		return nil, i
	}

	return value.NewValueRange(*rangeStart, *rangeEnd), nil
}

//
// List literal
//

func (self *Interpreter) listLiteral(node ast.AnalyzedListLiteralExpression) (*value.Value, *value.Interrupt) {
	values := make([]*value.Value, 0)

	for _, expr := range node.Values {
		val, i := self.expression(expr)
		if i != nil {
			return nil, i
		}
		values = append(values, val)
	}

	return value.NewValueList(values), nil
}

//
// Any object literal
//

func (self *Interpreter) anyObjectLiteral(node ast.AnalyzedAnyObjectExpression) (*value.Value, *value.Interrupt) {
	return value.NewValueAnyObject(make(map[string]*value.Value)), nil
}

//
// Object literal
//

func (self *Interpreter) objectLiteral(node ast.AnalyzedObjectLiteralExpression) (*value.Value, *value.Interrupt) {
	fields := make(map[string]*value.Value)
	for _, field := range node.Fields {
		fieldValue, i := self.expression(field.Expression)
		if i != nil {
			return nil, i
		}
		fields[field.Key.Ident()] = fieldValue
	}
	return value.NewValueObject(fields), nil
}

//
// Function literal
//

func (self *Interpreter) functionLiteral(node ast.AnalyzedFunctionLiteralExpression) (*value.Value, *value.Interrupt) {
	// TODO: test if closures work correctly
	// TODO: evaluate whether a deep copy is required

	return value.NewValueClosure(
		node.Body,
		self.currentModule.scopes,
	), nil
}

//
// Prefix expression
//

func (self *Interpreter) prefixExpression(node ast.AnalyzedPrefixExpression) (*value.Value, *value.Interrupt) {
	base, i := self.expression(node.Base)
	if i != nil {
		return nil, i
	}

	switch node.Operator {
	case ast.IntoSomePrefixOperator:
		baseDeref := *base // copy this value
		return value.NewValueOption(&baseDeref), nil
	case ast.MinusPrefixOperator:
		switch (*base).Kind() {
		case value.IntValueKind:
			baseInt := (*base).(value.ValueInt).Inner
			return value.NewValueInt(-baseInt), nil
		case value.FloatValueKind:
			baseFloat := (*base).(value.ValueFloat).Inner
			return value.NewValueFloat(-baseFloat), nil
		}
	case ast.NegatePrefixOperator:
		switch (*base).Kind() {
		case value.BoolValueKind:
			baseBool := (*base).(value.ValueBool).Inner
			return value.NewValueBool(!baseBool), nil
		case value.IntValueKind:
			baseInt := (*base).(value.ValueInt).Inner
			return value.NewValueInt(^baseInt), nil
		}
	}

	panic("Unreachable, either a new operator or base type was added")
}

//
// Infix expression
//

func (self *Interpreter) infixExpression(node ast.AnalyzedInfixExpression) (*value.Value, *value.Interrupt) {
	res, _, i := self.infixHelper(node.Lhs, node.Rhs, node.Operator)
	return res, i
}

func (self *Interpreter) infixHelper(lhs ast.AnalyzedExpression, rhs ast.AnalyzedExpression, operator pAst.InfixOperator) (res *value.Value, lhsAddr *value.Value, i *value.Interrupt) {
	switch operator {
	case pAst.EqualInfixOperator:
		lhs, i := self.expression(lhs)
		if i != nil {
			return nil, nil, i
		}
		rhs, i := self.expression(rhs)
		if i != nil {
			return nil, nil, i
		}

		res, i := (*lhs).IsEqual(*rhs)
		if i != nil {
			return nil, nil, i
		}
		return value.NewValueBool(res), lhs, nil
	case pAst.NotEqualInfixOperator:
		lhs, i := self.expression(lhs)
		if i != nil {
			return nil, nil, i
		}
		rhs, i := self.expression(rhs)
		if i != nil {
			return nil, nil, i
		}

		res, i := (*lhs).IsEqual(*rhs)
		if i != nil {
			return nil, nil, i
		}
		return value.NewValueBool(!res), lhs, nil
	}

	switch lhs.Type().Kind() {
	case ast.IntTypeKind:
		var intRes int64

		lhsVal, i := self.expression(lhs)
		if i != nil {
			return nil, nil, i
		}
		rhsVal, i := self.expression(rhs)
		if i != nil {
			return nil, nil, i
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

		lhsVal, i := self.expression(lhs)
		if i != nil {
			return nil, nil, i
		}
		rhsVal, i := self.expression(rhs)
		if i != nil {
			return nil, nil, i
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
			lhsTemp, i := self.expression(lhs)
			if i != nil {
				return nil, nil, i
			}
			rhsTemp, i := self.expression(rhs)
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
			lhsTemp, i := self.expression(lhs)
			if i != nil {
				return nil, nil, i
			}

			if (*lhsTemp).(value.ValueBool).Inner {
				return value.NewValueBool(true), lhsVal, nil
			}

			rhsTemp, i := self.expression(rhs)
			if i != nil {
				return nil, nil, i
			}
			return value.NewValueBool((*rhsTemp).(value.ValueBool).Inner), lhsVal, nil
		case pAst.LogicalAndInfixOperator:
			lhsTemp, i := self.expression(lhs)
			if i != nil {
				return nil, nil, i
			}

			if !(*lhsTemp).(value.ValueBool).Inner {
				return value.NewValueBool(false), lhsVal, nil
			}

			rhsTemp, i := self.expression(rhs)
			if i != nil {
				return nil, nil, i
			}
			return value.NewValueBool((*rhsTemp).(value.ValueBool).Inner), lhsVal, nil
		default:
			panic("A new operator kind was introduced without updating this code")
		}
		return value.NewValueBool(boolRes), lhsVal, nil
	case ast.StringTypeKind:
		lhsTemp, i := self.expression(lhs)
		if i != nil {
			return nil, nil, i
		}
		rhsTemp, i := self.expression(rhs)
		if i != nil {
			return nil, nil, i
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

func (self *Interpreter) assignExpression(node ast.AnalyzedAssignExpression) (*value.Value, *value.Interrupt) {
	if node.Operator == pAst.StdAssignOperatorKind {
		lhs, i := self.expression(node.Lhs)
		if i != nil {
			return nil, i
		}
		rhs, i := self.expression(node.Rhs)
		if i != nil {
			return nil, i
		}

		*lhs = *rhs // perform the assignment
		return value.NewValueNull(), nil
	}

	res, lhsAddr, i := self.infixHelper(node.Lhs, node.Rhs, node.Operator.IntoInfixOperator())
	if i != nil {
		return nil, i
	}

	// perform the assignment
	*lhsAddr = *res

	return value.NewValueNull(), nil
}

//
// Index expression
//

func (self *Interpreter) indexExpression(node ast.AnalyzedIndexExpression) (*value.Value, *value.Interrupt) {
	base, i := self.expression(node.Base)
	if i != nil {
		return nil, i
	}

	index, i := self.expression(node.Index)
	if i != nil {
		return nil, i
	}

	switch (*base).Kind() {
	case value.ObjectValueKind, value.AnyObjectValueKind:
		idx := (*index).(value.ValueString)
		fields, i := (*base).Fields()
		if i != nil {
			return nil, i
		}
		val, found := fields[idx.Inner]
		if !found {
			return nil, value.NewRuntimeErr(
				fmt.Sprintf("Value of type '%s' has no field named '%s'", (*base).Kind(), idx.Inner),
				value.IndexOutOfBoundsErrorKind,
				node.Range,
			)
		}
		return val, nil
	case value.ListValueKind:
		list := (*base).(value.ValueList)
		index := (*index).(value.ValueInt).Inner

		// handle index wrapping (-1 = len - 1)
		length := int64(len(*list.Values))
		if index < 0 {
			index = index + length
		}

		if index < 0 || index >= length {
			return nil, value.NewRuntimeErr(
				fmt.Sprintf("Index out of bounds: cannot index a list of length %d with %d", len(*list.Values), index),
				value.IndexOutOfBoundsErrorKind,
				node.Index.Span(),
			)
		}
		return (*list.Values)[int(index)], nil
	case value.StringValueKind:
		str := (*base).(value.ValueString).Inner
		index := (*index).(value.ValueInt).Inner

		// handle index wrapping (-1 = len - 1)
		length := int64(len(str))
		if index < 0 {
			index = index + length
		}

		if index < 0 || index >= length {
			return nil, value.NewRuntimeErr(
				fmt.Sprintf("Index out of bounds: cannot index a string of length %d with %d", len(str), index),
				value.IndexOutOfBoundsErrorKind,
				node.Index.Span(),
			)
		}
		return value.NewValueString(string((str)[int(index)])), nil
	}

	panic(fmt.Sprintf("A new type which can be indexed was added without updating this code: %s", (*base).Kind()))
}

//
// Member expression
//

func (self *Interpreter) memberExpression(node ast.AnalyzedMemberExpression) (*value.Value, *value.Interrupt) {
	base, i := self.expression(node.Base)
	if i != nil {
		return nil, i
	}

	fields, i := (*base).Fields()
	if i != nil {
		return nil, i
	}

	val, found := fields[node.Member.Ident()]
	if !found {
		panic(fmt.Sprintf("Field '%s' not found on value of type '%s' | node: %s", node.Member.Ident(), node.Base.Type(), node))
	}

	return val, nil
}

func (self *Interpreter) deepCast(val value.Value, typ ast.Type, span errors.Span) (*value.Value, *value.Interrupt) {
	if typ.Kind() == ast.OptionTypeKind {
		return value.NewValueOption(&val), nil
	}

	switch val.Kind() {
	case value.BoolValueKind:
		baseBool := val.(value.ValueBool).Inner
		switch typ.Kind() {
		case ast.IntTypeKind:
			var intRes int64
			if baseBool {
				intRes = 1
			} else {
				intRes = 0
			}
			return value.NewValueInt(intRes), nil
		case ast.FloatTypeKind:
			var floatRes float64
			if baseBool {
				floatRes = 1.0
			} else {
				floatRes = 0.0
			}
			return value.NewValueFloat(floatRes), nil
		case ast.BoolTypeKind:
			return &val, nil
		}
	case value.IntValueKind:
		baseInt := val.(value.ValueInt).Inner
		switch typ.Kind() {
		case ast.BoolTypeKind:
			var boolRes bool
			if baseInt == 0 {
				boolRes = false
			} else {
				boolRes = true
			}
			return value.NewValueBool(boolRes), nil
		case ast.FloatTypeKind:
			baseInt := val.(value.ValueInt).Inner
			return value.NewValueFloat(float64(baseInt)), nil
		case ast.IntTypeKind:
			return &val, nil
		}
	case value.FloatValueKind:
		baseFloat := val.(value.ValueFloat).Inner
		switch typ.Kind() {
		case ast.BoolTypeKind:
			var boolRes bool
			if baseFloat == 0 {
				boolRes = false
			} else {
				boolRes = true
			}
			return value.NewValueBool(boolRes), nil
		case ast.IntTypeKind:
			baseFloat := val.(value.ValueFloat).Inner
			return value.NewValueInt(int64(baseFloat)), nil
		case ast.FloatTypeKind:
			return &val, nil
		}
	case value.ObjectValueKind:
		objVal := val.(value.ValueObject)
		switch typ.Kind() {
		case ast.AnyObjectTypeKind:
			return objVal.IntoAnyObject(), nil
		case ast.ObjectTypeKind:
			objVal := val.(value.ValueObject)
			objType := typ.(ast.ObjectType)

			outputFields := make(map[string]*value.Value)

			for key, field := range objVal.FieldsInternal {
				found := false
				for _, otherField := range objType.ObjFields {
					if key == otherField.FieldName.Ident() {
						newField, i := self.deepCast(*field, otherField.Type, span)
						if i != nil {
							return nil, i
						}
						fmt.Printf("PREV %s (%v) | CURR: %s (%v)\n", (*field).Kind(), field, (*newField).Kind(), newField)
						outputFields[key] = newField
						found = true
						break
					}
				}
				if !found {
					return nil, value.NewRuntimeErr(
						fmt.Sprintf("Incompatible values: found unexpected field '%s'", key),
						value.CastErrorKind,
						span,
					)
				}
			}

			for _, field := range objType.ObjFields {
				_, found := objVal.FieldsInternal[field.FieldName.Ident()]
				if !found {
					return nil, value.NewRuntimeErr(
						fmt.Sprintf("Incompatible values: field '%s' was expected but not found", field.FieldName.Ident()),
						value.CastErrorKind,
						span,
					)
				}
			}

			return value.NewValueObject(outputFields), nil
		default:
			return nil, value.NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				value.CastErrorKind,
				span,
			)
		}
	case value.ListValueKind:
		listVal := val.(value.ValueList)
		switch typ.Kind() {
		case ast.ListTypeKind:
			asType := typ.(ast.ListType)

			outputList := make([]*value.Value, 0)
			for _, item := range *listVal.Values {
				newVal, i := self.deepCast(*item, asType.Inner, span)
				if i != nil {
					return nil, i
				}
				outputList = append(outputList, newVal)
			}

			return value.NewValueList(outputList), nil
		}
	case value.AnyObjectValueKind:
		if typ.Kind() != ast.AnyObjectTypeKind {
			return nil, value.NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				value.CastErrorKind,
				span,
			)
		}
	case value.OptionValueKind:
		if typ.Kind() != ast.OptionTypeKind {
			return nil, value.NewRuntimeErr(
				fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
				value.CastErrorKind,
				span,
			)
		}

		panic("TODO: handle these casts")

		return nil, nil
	case value.ClosureValueKind, value.FunctionValueKind, value.BuiltinFunctionValueKind:
		panic("Unreachable, the analyzer prevents this")
	case value.NullValueKind:
		switch typ.Kind() {
		case ast.NullTypeKind:
			return &val, nil
		}
	case value.StringValueKind:
		switch typ.Kind() {
		case ast.StringTypeKind:
			return &val, nil
		}
	case value.RangeValueKind:
		switch typ.Kind() {
		case ast.RangeTypeKind:
			return &val, nil
		}
	}
	return nil, value.NewRuntimeErr(
		fmt.Sprintf("Incompatible values: a value of type '%s' is not compatible with a value of type '%s'", val.Kind(), typ),
		value.CastErrorKind,
		span,
	)
}

//
// Cast expression
//

func (self *Interpreter) castExpression(node ast.AnalyzedCastExpression) (*value.Value, *value.Interrupt) {
	base, i := self.expression(node.Base)
	if i != nil {
		return nil, i
	}

	// TODO: remove this
	// Equal types, or cast from `any` to other -> check internal compatibilty
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
	return self.deepCast(*base, node.AsType, node.Span())

	panic(fmt.Sprintf("Unsupported runtime cast from %v to %s", (*base).Kind(), node.AsType.Kind()))
}

//
// If expression
//

func (self *Interpreter) ifExpression(node ast.AnalyzedIfExpression) (*value.Value, *value.Interrupt) {
	condition, i := self.expression(node.Condition)
	if i != nil {
		return nil, i
	}

	// cast condition to bool
	conditionIsTrue := (*condition).(value.ValueBool).Inner

	resultValue := value.NewValueNull()

	if conditionIsTrue {
		blockRes, i := self.block(node.ThenBlock, true)
		if i != nil {
			return nil, i
		}
		*resultValue = *blockRes
	} else if node.ElseBlock != nil {
		blockRes, i := self.block(*node.ElseBlock, true)
		if i != nil {
			return nil, i
		}
		*resultValue = *blockRes
	}

	return resultValue, nil
}

//
// Match expression
//

func (self *Interpreter) matchExpression(node ast.AnalyzedMatchExpression) (*value.Value, *value.Interrupt) {
	control, i := self.expression(node.ControlExpression)
	if i != nil {
		return nil, i
	}

	for _, arm := range node.Arms {
		literal, i := self.expression(arm.Literal)
		if i != nil {
			return nil, i
		}

		// check if the literal is equal to the value of the control expr
		isEqual, i := (*literal).IsEqual(*control)
		if i != nil {
			return nil, i
		}

		if isEqual {
			return self.expression(arm.Action)
		}
	}

	if node.DefaultArmAction != nil {
		return self.expression(*node.DefaultArmAction)
	}

	return value.NewValueNull(), nil
}

//
// Try expression
//

func (self *Interpreter) tryExpression(node ast.AnalyzedTryExpression) (*value.Value, *value.Interrupt) {
	tryRes, i := self.block(node.TryBlock, true)
	if i == nil {
		return tryRes, nil
	}

	if (*i).Kind() != value.ThrowInterruptKind {
		// only throw-errors can be caught
		return nil, i
	}

	throwError := (*i).(value.ThrowInterrupt)
	self.pushScope()
	defer self.popScope()

	self.addVar(node.CatchIdent.Ident(), *value.NewValueString(throwError.MessageInternal))
	return self.block(node.CatchBlock, false)
}
