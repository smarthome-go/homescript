package analyzer

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Expression
//

func (self *Analyzer) expression(node pAst.Expression) ast.AnalyzedExpression {
	errOnAnyPrev := self.currentModule.CreateErrorIfContainsAny
	self.currentModule.CreateErrorIfContainsAny = true
	var res ast.AnalyzedExpression

	switch node.Kind() {
	case pAst.IntLiteralExpressionKind:
		src := node.(pAst.IntLiteralExpression)
		res = ast.AnalyzedIntLiteralExpression{Value: src.Value, Range: src.Range}
	case pAst.FloatLiteralExpressionKind:
		src := node.(pAst.FloatLiteralExpression)
		res = ast.AnalyzedFloatLiteralExpression{Value: src.Value, Range: src.Range}
	case pAst.BoolLiteralExpressionKind:
		src := node.(pAst.BoolLiteralExpression)
		res = ast.AnalyzedBoolLiteralExpression{Value: src.Value, Range: src.Range}
	case pAst.StringLiteralExpressionKind:
		src := node.(pAst.StringLiteralExpression)
		res = ast.AnalyzedStringLiteralExpression{Value: src.Value, Range: src.Range}
	case pAst.IdentExpressionKind:
		src := node.(pAst.IdentExpression)
		res = self.identExpression(src)
	case pAst.NullLiteralExpressionKind:
		res = ast.AnalyzedNullLiteralExpression{Range: node.Span()}
	case pAst.NoneLiteralExpressionKind:
		res = ast.AnalyzedNoneLiteralExpression{Range: node.Span()}
	case pAst.RangeLiteralExpressionKind:
		src := node.(pAst.RangeLiteralExpression)
		res = self.rangeLiteralExpression(src)
	case pAst.ListLiteralExpressionKind:
		src := node.(pAst.ListLiteralExpression)
		res = self.listLiteralExpression(src)
	case pAst.AnyObjectLiteralExpressionKind:
		src := node.(pAst.AnyObjectLiteralExpression)
		res = self.anyObjectLiteralExpression(src)
	case pAst.ObjectLiteralExpressionKind:
		src := node.(pAst.ObjectLiteralExpression)
		res = self.objectLiteralExpression(src)
	case pAst.FunctionLiteralExpressionKind:
		src := node.(pAst.FunctionLiteralExpression)
		res = self.functionLiteral(src)
	case pAst.GroupedExpressionKind:
		src := node.(pAst.GroupedExpression)
		analyzed := self.expression(src.Inner)
		res = ast.AnalyzedGroupedExpression{Inner: analyzed, Range: src.Range}
	case pAst.PrefixExpressionKind:
		src := node.(pAst.PrefixExpression)
		res = self.prefixExpression(src)
	case pAst.InfixExpressionKind:
		src := node.(pAst.InfixExpression)
		res = self.infixExpression(src)
	case pAst.AssignExpressionKind:
		src := node.(pAst.AssignExpression)
		res = self.assignExpression(src)
	case pAst.CallExpressionKind:
		src := node.(pAst.CallExpression)
		res = self.callExpression(src)
	case pAst.IndexExpressionKind:
		src := node.(pAst.IndexExpression)
		res = self.indexExpression(src)
	case pAst.MemberExpressionKind:
		src := node.(pAst.MemberExpression)
		res = self.memberExpression(src)
	case pAst.CastExpressionKind:
		src := node.(pAst.CastExpression)
		res = self.castExpression(src)
	case pAst.BlockExpressionKind:
		src := node.(pAst.BlockExpression)
		res = ast.AnalyzedBlockExpression{Block: self.block(src.Block, true)}
	case pAst.IfExpressionKind:
		src := node.(pAst.IfExpression)
		res = self.ifExpression(src)
	case pAst.MatchExpressionKind:
		src := node.(pAst.MatchExpression)
		res = self.matchExpression(src)
	case pAst.TryExpressionKind:
		src := node.(pAst.TryExpression)
		res = self.tryExpression(src)
	default:
		panic("A new expression kind was introduced without updating this code")
	}

	self.currentModule.CreateErrorIfContainsAny = errOnAnyPrev

	// if this is a `never` expression, count it like a loop termination
	if res.Type().Kind() == ast.NeverTypeKind {
		self.currentModule.CurrentLoopIsTerminated = true
	}

	// check for `any` parts in the type
	// types like `fn() -> any` are allowed, just not `(fn() -> any)()`, meaning `any`
	if self.currentModule.CreateErrorIfContainsAny && self.CheckAny(res.Type()) {
		switch res.Type().Kind() {
		case ast.FnTypeKind, ast.OptionTypeKind:
		default:
			self.error(
				"Implicit use of 'any' type: explicit type annotations required",
				[]string{"Consider casting this expression like this: `.. as type`"},
				res.Span(),
			)
			return ast.AnalyzedExpression(ast.UnknownExpression{})
		}
	}

	return res
}

//
// Ident expression
//

func (self *Analyzer) identExpression(node pAst.IdentExpression) ast.AnalyzedIdentExpression {
	// If this ident refers to a singleton, handle it specially
	if node.IsSingleton {
		resultType := ast.NewUnknownType()

		singleton, found := self.currentModule.Singletons[node.Ident.Ident()]
		if !found {
			self.error(
				fmt.Sprintf("Reference of undeclared singleton type '%s'", node.Ident.Ident()),
				[]string{
					fmt.Sprintf("Singleton types can be declared like this: `@%s\n type %sFoo = ...;`", node.Ident.Ident(), node.Ident.Ident()),
				},
				node.Ident.Span(),
			)
		} else {
			resultType = singleton.SetSpan(node.Span())
		}

		return ast.AnalyzedIdentExpression{
			Ident:       node.Ident,
			ResultType:  resultType,
			IsGlobal:    false,
			IsFunction:  false,
			IsSingleton: true,
		}
	}

	variable, scope, found := self.currentModule.getVar(node.Ident.Ident())

	// show an error and use `unknown` if the variable does not exist
	if !found {
		// check if there is a function with the required name
		fn, found := self.currentModule.getFunc(node.Ident.Ident())
		if !found {
			self.error(
				fmt.Sprintf("Use of undefined variable or function '%s'", node.Ident.Ident()),
				[]string{
					fmt.Sprintf("Variables can be defined like this: `let %s = ...;`", node.Ident.Ident()),
				},
				node.Ident.Span(),
			)

			return ast.AnalyzedIdentExpression{
				Ident:      node.Ident,
				ResultType: ast.NewUnknownType(),
				IsGlobal:   false,
				IsFunction: true,
			}

		}

		// only mark the function as `used` if the usage originates from another function
		if self.currentModule.CurrentFunction.FnType.Kind() == normalFunctionKind {
			currFn := self.currentModule.CurrentFunction.FnType.(normalFunction)
			if fn.FnType.Kind() == normalFunctionKind {
				toBeCalled := fn.FnType.(normalFunction)
				if currFn.Ident.Ident() != toBeCalled.Ident.Ident() {
					fn.Used = true
				}
			}

		}

		params := make([]ast.FunctionTypeParam, 0)
		for _, param := range fn.Parameters {
			params = append(params, ast.NewFunctionTypeParam(
				param.Ident,
				param.Type,
				param.IsSingletonExtractor,
			))
		}

		return ast.AnalyzedIdentExpression{
			Ident: node.Ident,
			ResultType: ast.NewFunctionType(
				ast.NewNormalFunctionTypeParamKind(params),
				fn.ParamsSpan,
				fn.ReturnType,
				fn.FnType.(normalFunction).Ident.Span(),
			),
			IsGlobal:   false,
			IsFunction: false,
		}
	}

	// otherwise, mark the variable as used
	variable.Used = true

	return ast.AnalyzedIdentExpression{
		Ident:      node.Ident,
		ResultType: variable.Type.SetSpan(node.Span()),
		IsGlobal:   scope == 0,
	}
}

//
// Range literal
//

func (self *Analyzer) rangeLiteralExpression(node pAst.RangeLiteralExpression) ast.AnalyzedRangeLiteralExpression {
	start := self.expression(node.Start)
	end := self.expression(node.End)

	// ensure that both values of the range are `int`
	if err := self.TypeCheck(start.Type(), ast.NewIntType(errors.Span{}), false); err != nil {
		self.error(
			fmt.Sprintf("Type mismatch: expected '%s', found '%s'", ast.NewIntType(errors.Span{}).Kind(), start.Type().Kind()),
			nil,
			start.Span(),
		)
	}

	if err := self.TypeCheck(end.Type(), ast.NewIntType(errors.Span{}), false); err != nil {
		self.error(
			fmt.Sprintf("Type mismatch: expected '%s', found '%s'", ast.NewIntType(errors.Span{}).Kind(), end.Type().Kind()),
			nil,
			end.Span(),
		)
	}

	return ast.AnalyzedRangeLiteralExpression{
		Start: start,
		End:   end,
		Range: node.Range,
	}
}

//
// List literal
//

func (self *Analyzer) listLiteralExpression(node pAst.ListLiteralExpression) ast.AnalyzedListLiteralExpression {
	// ensure type equality accross all values
	listType := ast.NewAnyType(node.Range) // any is the default: the analyzer will enforce type annotations
	newValues := make([]ast.AnalyzedExpression, 0)

	for _, val := range node.Values {
		valExpression := self.expression(val)
		newValues = append(newValues, valExpression)

		if err := self.TypeCheck(valExpression.Type(), listType, false); err != nil && listType.Kind() != ast.AnyTypeKind {
			self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
			if err.ExpectedDiagnostic != nil {
				self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
			}
			listType = ast.NewUnknownType()
		} else if listType.Kind() == ast.AnyTypeKind {
			listType = valExpression.Type()
		}
	}

	return ast.AnalyzedListLiteralExpression{
		Values:   newValues,
		Range:    node.Range,
		ListType: listType,
	}
}

//
// Any object literal
//

func (self *Analyzer) anyObjectLiteralExpression(node pAst.AnyObjectLiteralExpression) ast.AnalyzedAnyObjectExpression {
	return ast.AnalyzedAnyObjectExpression{Range: node.Range}
}

//
// Object literal
//

func (self *Analyzer) objectLiteralExpression(node pAst.ObjectLiteralExpression) ast.AnalyzedObjectLiteralExpression {
	newFields := make([]ast.AnalyzedObjectLiteralField, 0)
	fieldSet := make(map[string]struct{})

	for _, field := range node.Fields {
		// test if the current field conflicts with a builtin field
		if _, isBuiltin := ast.NewObjectType(make([]ast.ObjectTypeField, 0), node.Range).Fields(node.Range)[field.Key.Ident()]; isBuiltin {
			self.error(
				fmt.Sprintf("Cannot use '%s' as a field name: already used for builtin purposes", field.Key.Ident()),
				nil,
				field.Key.Span(),
			)
			continue
		}

		// test if the current field occurs multiple times
		if _, alreadyExists := fieldSet[field.Key.Ident()]; alreadyExists {
			self.error(
				fmt.Sprintf("Duplicate definition of field '%s'", field.Key.Ident()),
				nil,
				field.Key.Span(),
			)
			continue
		}
		fieldSet[field.Key.Ident()] = struct{}{}

		newFields = append(newFields, ast.AnalyzedObjectLiteralField{
			Key:        field.Key,
			Expression: self.expression(field.Expression),
			Range:      field.Range,
		})
	}

	return ast.AnalyzedObjectLiteralExpression{
		Fields: newFields,
		Range:  node.Range,
	}
}

//
// Function literal
//

func (self *Analyzer) functionLiteral(node pAst.FunctionLiteralExpression) ast.AnalyzedFunctionLiteralExpression {
	// analyze parameters
	newParams := make([]ast.AnalyzedFnParam, 0)
	existentParams := make(map[string]struct{})

	// push a new scope for the function body
	self.pushScope()

	for _, param := range node.Parameters {
		if _, alreadyExists := existentParams[param.Ident.Ident()]; alreadyExists {
			self.error(
				fmt.Sprintf("Duplicate declaration of parameter '%s'", param.Ident),
				nil,
				param.Span,
			)
		}
		existentParams[param.Ident.Ident()] = struct{}{}

		converted := self.ConvertType(param.Type, true)
		newParams = append(newParams, ast.AnalyzedFnParam{
			Ident: param.Ident,
			Type:  converted,
			Span:  param.Span,
		})

		// add the parameter to the new scope
		self.currentModule.addVar(
			param.Ident.Ident(),
			NewVar(converted, param.Ident.Span(), ParameterVariableOriginKind, false),
			false,
		)
	}

	fnReturntype := self.ConvertType(node.ReturnType, true)

	// set current function
	moduleFn := newFunction(
		newLambdaFunction(),
		newParams,
		node.ParamSpan,
		fnReturntype,
		node.Span(),
		pAst.FN_MODIFIER_NONE,
	)
	self.currentModule.CurrentFunction = &moduleFn

	// analyze body
	analyzedBlock := self.block(node.Body, false)

	// analyze return type
	if err := self.TypeCheck(analyzedBlock.Type(), fnReturntype, true); err != nil {
		self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
		if err.ExpectedDiagnostic != nil {
			self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
		}
	}

	self.dropScope(true)

	return ast.AnalyzedFunctionLiteralExpression{
		Parameters: newParams,
		ParamSpan:  node.ParamSpan,
		ReturnType: fnReturntype,
		Body:       analyzedBlock,
		Range:      node.Range,
	}
}

//
// Prefix expression
//

func (self *Analyzer) prefixExpression(node pAst.PrefixExpression) ast.AnalyzedPrefixExpression {
	base := self.expression(node.Base)

	var operator ast.PrefixOperator
	resultType := base.Type().SetSpan(node.Range)

	switch node.Operator {
	case pAst.MinusPrefixOperator:
		operator = ast.MinusPrefixOperator
		switch base.Type().Kind() {
		case ast.IntTypeKind, ast.FloatTypeKind:
		case ast.NeverTypeKind, ast.UnknownTypeKind:
			resultType = ast.NewUnknownType()
		default:
			self.error(
				fmt.Sprintf("Prefix operator '%s' cannot be used on values of type '%s'", operator, base.Type().Kind()),
				nil,
				node.Range,
			)
			resultType = ast.NewUnknownType()
		}
	case pAst.NegatePrefixOperator:
		operator = ast.NegatePrefixOperator
		switch base.Type().Kind() {
		case ast.IntTypeKind, ast.BoolTypeKind:
		case ast.UnknownTypeKind, ast.NeverTypeKind:
			resultType = ast.NewUnknownType()
		default:
			self.error(
				fmt.Sprintf("Prefix operator '%s' cannot be used on values of type '%s'", operator, base.Type().Kind()),
				nil,
				node.Range,
			)
			resultType = ast.NewUnknownType()
		}
	case pAst.IntoSomePrefixOperator:
		operator = ast.IntoSomePrefixOperator
		resultType = ast.NewOptionType(base.Type().SetSpan(node.Range), node.Range)
	default:
		panic("A new prefix operator was added without updating this code")
	}

	return ast.AnalyzedPrefixExpression{
		Operator:   operator,
		Base:       base,
		ResultType: resultType,
		Range:      node.Range,
	}
}

//
// Infix expression
//

func (self *Analyzer) infixExpression(node pAst.InfixExpression) ast.AnalyzedInfixExpression {
	lhs := self.expression(node.Lhs)
	rhs := self.expression(node.Rhs)

	resultType := ast.NewUnknownType()
	lhsTypeKind := lhs.Type().Kind()

	if err := self.TypeCheck(rhs.Type().SetSpan(node.Rhs.Span()), lhs.Type().SetSpan(node.Lhs.Span()), true); err != nil {
		self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
		if err.ExpectedDiagnostic != nil {
			self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
		}
		lhsTypeKind = ast.UnknownTypeKind
	}

	switch lhs.Type().Kind() {
	case ast.IntTypeKind:
		switch node.Operator {
		case pAst.PlusInfixOperator, pAst.MinusInfixOperator,
			pAst.MultiplyInfixOperator, pAst.DivideInfixOperator,
			pAst.ModuloInfixOperator, pAst.PowerInfixOperator,
			pAst.ShiftLeftInfixOperator, pAst.ShiftRightInfixOperator,
			pAst.BitOrInfixOperator, pAst.BitAndInfixOperator,
			pAst.BitXorInfixOperator:

			// this yields a value of type `int`
			resultType = ast.NewIntType(node.Range)
		case pAst.EqualInfixOperator, pAst.NotEqualInfixOperator,
			pAst.LessThanInfixOperator, pAst.GreaterThanInfixOperator,
			pAst.LessThanEqualInfixOperator, pAst.GreaterThanEqualInfixOperator:

			// this yields a value of type `bool`
			resultType = ast.NewBoolType(node.Range)
		default:
			self.error(
				fmt.Sprintf("Infix operator '%s' cannot be used on values of type '%s'", node.Operator, lhs.Type().Kind()),
				nil,
				node.Span(),
			)
		}
	case ast.FloatTypeKind:
		switch node.Operator {
		case pAst.PlusInfixOperator, pAst.MinusInfixOperator,
			pAst.MultiplyInfixOperator, pAst.DivideInfixOperator,
			pAst.PowerInfixOperator:

			// this yields a value of type `num`
			resultType = ast.NewFloatType(node.Range)
		case pAst.EqualInfixOperator, pAst.NotEqualInfixOperator,
			pAst.LessThanInfixOperator, pAst.GreaterThanInfixOperator,
			pAst.LessThanEqualInfixOperator, pAst.GreaterThanEqualInfixOperator:

			// this yields a value of type `bool`
			resultType = ast.NewBoolType(node.Range)
		default:
			self.error(
				fmt.Sprintf("Infix operator '%s' cannot be used on values of type '%s'", node.Operator, lhs.Type().Kind()),
				nil,
				node.Span(),
			)
		}
	case ast.BoolTypeKind:
		switch node.Operator {
		case pAst.BitOrInfixOperator, pAst.BitAndInfixOperator, // bitwise operators
			pAst.BitXorInfixOperator,
			pAst.LogicalOrInfixOperator, pAst.LogicalAndInfixOperator, // logical operatos
			pAst.EqualInfixOperator, pAst.NotEqualInfixOperator: // comparison operators

			// this yields a value of type `bool`
			resultType = ast.NewBoolType(node.Range)
		default:
			self.error(
				fmt.Sprintf("Infix operator '%s' cannot be used on values of type '%s'", node.Operator, lhs.Type().Kind()),
				nil,
				node.Span(),
			)
		}
	case ast.StringTypeKind:
		switch node.Operator {
		case pAst.PlusInfixOperator:
			// this yields a value of type `str`
			resultType = ast.NewStringType(node.Range)
		case pAst.EqualInfixOperator, pAst.NotEqualInfixOperator:
			// this yields a value of type `bool`
			resultType = ast.NewBoolType(node.Range)
		default:
			self.error(
				fmt.Sprintf("Infix operator '%s' cannot be used on values of type '%s'", node.Operator, lhs.Type().Kind()),
				nil,
				node.Span(),
			)
		}
	case ast.UnknownTypeKind, ast.NeverTypeKind:
		// ignore these
		resultType = lhs.Type()
	default:
		switch node.Operator {
		case pAst.EqualInfixOperator, pAst.NotEqualInfixOperator:
			// this always results in a value of type `bool`
			resultType = ast.NewBoolType(node.Range)
		default:
			self.error(
				fmt.Sprintf("Infix operator '%s' cannot be used on values of type '%s'", node.Operator, lhsTypeKind),
				nil,
				node.Span(),
			)
		}
	}

	return ast.AnalyzedInfixExpression{
		Lhs:        lhs,
		Rhs:        rhs,
		Operator:   node.Operator,
		ResultType: resultType,
		Range:      node.Range,
	}
}

//
// Assign expression
//

func (self *Analyzer) assignExpression(node pAst.AssignExpression) ast.AnalyzedAssignExpression {
	lhs := self.expression(node.Lhs)
	rhs := self.expression(node.Rhs)

	resultType := ast.NewNullType(node.Range)
	prevErr := false

	// if either the lhs or ths is `never`, use `never`
	if lhs.Type().Kind() == ast.NeverTypeKind || rhs.Type().Kind() == ast.NeverTypeKind {
		resultType = ast.NewNeverType()
	}

	if err := self.TypeCheck(rhs.Type(), lhs.Type(), true); err != nil {
		self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
		if err.ExpectedDiagnostic != nil {
			self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
		}
		prevErr = true
	}

	switch lhs.Type().Kind() {
	case ast.IntTypeKind:
		switch node.AssignOperator {
		case pAst.StdAssignOperatorKind, pAst.PlusAssignOperatorKind, pAst.MinusAssignOperatorKind,
			pAst.MultiplyAssignOperatorKind, pAst.DivideAssignOperatorKind, pAst.ModuloAssignOperatorKind, pAst.PowerAssignOperatorKind,
			pAst.ShiftLeftAssignOperatorKind, pAst.ShiftRightAssignOperatorKind,
			pAst.BitOrAssignOperatorKind, pAst.BitAndAssignOperatorKind, pAst.BitXorAssignOperatorKind:
		default:
			if prevErr {
				break
			}
			self.assignErr(node.AssignOperator, lhs.Type(), node.Span())
		}
	case ast.FloatTypeKind:
		switch node.AssignOperator {
		case pAst.StdAssignOperatorKind, pAst.PlusAssignOperatorKind,
			pAst.MinusAssignOperatorKind, pAst.MultiplyAssignOperatorKind,
			pAst.DivideAssignOperatorKind, pAst.ModuloAssignOperatorKind,
			pAst.PowerAssignOperatorKind:
		default:
			if prevErr {
				break
			}
			self.assignErr(node.AssignOperator, lhs.Type(), node.Span())
		}
	case ast.BoolTypeKind:
		switch node.AssignOperator {
		case pAst.StdAssignOperatorKind, pAst.BitOrAssignOperatorKind, pAst.BitAndAssignOperatorKind, // bitwise operators
			pAst.BitXorAssignOperatorKind:
		default:
			if prevErr {
				break
			}
			self.assignErr(node.AssignOperator, lhs.Type(), node.Span())
		}
	case ast.StringTypeKind:
		switch node.AssignOperator {
		case pAst.StdAssignOperatorKind, pAst.PlusAssignOperatorKind:
		default:
			if prevErr {
				break
			}
			self.assignErr(node.AssignOperator, lhs.Type(), node.Span())
		}
	case ast.AnyTypeKind:
		panic("Unreachable: the left-hand side of an assignment is never `any`")
	case ast.UnknownTypeKind, ast.NeverTypeKind:
		// ignore these
	default:
		switch node.AssignOperator {
		case pAst.StdAssignOperatorKind:
		default:
			if prevErr {
				break
			}
			self.assignErr(node.AssignOperator, lhs.Type(), node.Range)
		}
	}

	return ast.AnalyzedAssignExpression{
		Lhs:        lhs,
		Operator:   node.AssignOperator,
		Rhs:        rhs,
		ResultType: resultType,
		Range:      node.Range,
	}
}

func (self *Analyzer) assignErr(operator pAst.AssignOperator, typ ast.Type, span errors.Span) {
	self.error(
		fmt.Sprintf("Assign operator '%s' cannot be used on values of type '%s'", operator, typ.Kind()),
		nil,
		span,
	)
}

//
// Call expression
//

// TODO: also forbid invoking a spawn fn which returns a closure.
// TODO: also completely rewrite this function, it is very obfuscated.
func (self *Analyzer) callExpression(node pAst.CallExpression) ast.AnalyzedCallExpression {
	// check that the base is a value that can be called
	base := self.expression(node.Base)

	returnType := base.Type()

	isNormalFunction := false
	if base.Kind() == ast.IdentExpressionKind {
		isNormalFunction = base.(ast.AnalyzedIdentExpression).IsFunction
	}

	// make arguments
	arguments := make([]ast.AnalyzedCallArgument, 0)

	switch base.Type().Kind() {
	case ast.NeverTypeKind, ast.UnknownTypeKind:
		// do nothing
	case ast.FnTypeKind:
		baseFn := base.Type().(ast.FunctionType)

		// validate arguments depending on the parameter type of the function
		switch baseFn.Params.Kind() {
		case ast.NormalFunctionTypeParamKindIdentifierKind:
			// validate that all arguments match the declared parameter
			baseParams := baseFn.Params.(ast.NormalFunctionTypeParamKindIdentifier)

			// Preprocess / skip singleton extractor params
			// NOTE: this is valid since the extractions occur at the start

			newParams := make([]ast.FunctionTypeParam, 0)
			for _, param := range baseParams.Params {
				if param.IsSingletonExtractor {
					continue
				}

				newParams = append(newParams, param)
			}

			if len(node.Arguments) != len(newParams) {
				pluralS := "s"
				if len(newParams) == 1 {
					pluralS = ""
				}

				verb := "were"
				if len(node.Arguments) == 1 {
					verb = "was"
				}

				paramList := make([]string, 0)
				for _, param := range newParams {
					paramList = append(paramList, param.Name.Ident())
				}

				self.error(
					fmt.Sprintf("Function requires %d argument%s (%s), however %d %s supplied", len(newParams), pluralS, strings.Join(paramList, ", "), len(node.Arguments), verb),
					nil,
					node.Range,
				)
			} else {
				for idx := 0; idx < len(newParams); idx++ {
					argExpr := self.expression(node.Arguments[idx])

					if argExpr.Type().Kind() == ast.NullTypeKind {
						self.error(
							fmt.Sprintf("Cannot use a value of result type `%s` in function call", argExpr.Type()),
							[]string{"This expression generates no value, therefore it can be omitted"},
							argExpr.Span(),
						)
						continue
					}

					// Sending closures accros threads is UB, prevent this.
					// When sending a closure which captures values to a new thread, the old captured values are not deepcopie'd.
					// Therefore, sending these closures accross threads will cause weird memory bugs which must not occur in a Smarthome system.
					if node.IsSpawn && argExpr.Type().Kind() == ast.FnTypeKind {
						self.error(
							"Sending closures across threads is undefined behaviour.",
							[]string{fmt.Sprintf("It is not possible to use a value of type `%s` as an argument to a `spawn` invocation.", argExpr.Type())},
							argExpr.Span(),
						)
						continue
					}

					if err := self.TypeCheck(argExpr.Type(), newParams[idx].Type, true); err != nil {
						self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
					} else {
						arguments = append(arguments, ast.AnalyzedCallArgument{
							Name:       newParams[idx].Name.Ident(),
							Expression: argExpr,
						})
					}
				}
			}
		case ast.VarArgsFunctionTypeParamKindIdentifierKind:
			// validate that every argument matches the type of the varargs
			varArgType := baseFn.Params.(ast.VarArgsFunctionTypeParamKindIdentifier)

			if len(varArgType.ParamTypes) != 0 && len(node.Arguments) < len(varArgType.ParamTypes) {
				pluralS := "s"
				if len(varArgType.ParamTypes) == 1 {
					pluralS = ""
				}

				verb := "were"
				if len(node.Arguments) == 1 {
					verb = "was"
				}

				self.error(
					fmt.Sprintf("Function requires at least %d argument%s, however %d %s supplied", len(varArgType.ParamTypes), pluralS, len(node.Arguments), verb),
					nil,
					node.Range,
				)
			} else {
				for idx, arg := range node.Arguments {
					argExpr := self.expression(arg)

					if argExpr.Type().Kind() == ast.NullTypeKind {
						self.error(
							fmt.Sprintf("Cannot use a value of result type `%s` in function call", argExpr.Type()),
							[]string{"This expression generates no value, therefore it can be omitted"},
							argExpr.Span(),
						)
						continue
					}

					// Sending closures accros threads is UB, prevent this.
					// When sending a closure which captures values to a new thread, the old captured values are not deepcopie'd.
					// Therefore, sending these closures accross threads will cause weird memory bugs which must not occur in a Smarthome system.
					if node.IsSpawn && argExpr.Type().Kind() == ast.FnTypeKind {
						self.error(
							"Sending closures across threads is undefined behaviour.",
							[]string{fmt.Sprintf("It is not possible to use a value of type `%s` as an argument to a `spawn` invocation.", argExpr.Type())},
							argExpr.Span(),
						)
						continue
					}

					toCheck := varArgType.RemainingType
					if idx < len(varArgType.ParamTypes) {
						toCheck = varArgType.ParamTypes[idx]
					}

					if err := self.TypeCheck(argExpr.Type(), toCheck, true); err != nil {
						self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
					} else {
						arguments = append(arguments, ast.AnalyzedCallArgument{
							Name:       "", // no name: is vararg
							Expression: argExpr,
						})
					}
				}
			}
		}

		// lookup the result type of the function
		returnType = baseFn.ReturnType
	default:
		notes := make([]string, 0)
		if base.Type().Kind() == ast.AnyTypeKind {
			notes = append(notes, "Consider casting this expression to a function type: `... as fn() -> type`")
		}

		self.error(
			fmt.Sprintf("Type '%s' cannot be called", base.Type().Kind()),
			notes,
			base.Span(),
		)
	}

	if node.IsSpawn {
		returnType = ast.NewObjectType([]ast.ObjectTypeField{
			ast.NewObjectTypeField(pAst.NewSpannedIdent("join", node.Span()), ast.NewFunctionType(
				ast.NewNormalFunctionTypeParamKind(make([]ast.FunctionTypeParam, 0)),
				node.Span(),
				returnType.SetSpan(node.Range),
				node.Span(),
			), node.Span()),
		}, node.Span())
	}

	return ast.AnalyzedCallExpression{
		Base:             base,
		Arguments:        arguments,
		ResultType:       returnType.SetSpan(node.Range),
		Range:            node.Range,
		IsSpawn:          node.IsSpawn,
		IsNormalFunction: isNormalFunction,
	}
}

//
// Index expression
//

func (self *Analyzer) indexExpression(node pAst.IndexExpression) ast.AnalyzedIndexExpression {
	base := self.expression(node.Base)
	index := self.expression(node.Index)

	resultType := ast.NewUnknownType()
	switch base.Type().Kind() {
	case ast.AnyObjectTypeKind:
		// ensure that the index expression is a `str`
		if index.Type().Kind() != ast.StringTypeKind {
			self.error(
				fmt.Sprintf("A value of type '%s' cannot be indexed by '%s'", base.Type().Kind(), index.Type().Kind()),
				nil,
				node.Index.Span(),
			)
			// this yields an `unknown` type
			resultType = ast.NewUnknownType()
		} else {
			// NOTE: this operation can fail during interpretation
			// this yields an `any` type (the user must validate this manually)
			resultType = ast.NewAnyType(node.Range)
		}
	case ast.ObjectTypeKind:
		// ensure that the index expression is a `str`
		if index.Type().Kind() != ast.StringTypeKind {
			self.error(
				fmt.Sprintf("A value of type '%s' cannot be indexed by '%s'", base.Type().Kind(), index.Type().Kind()),
				nil,
				node.Index.Span(),
			)
			// this yields an `unknown` type
			resultType = ast.NewUnknownType()
		} else {
			// NOTE: this operation can fail during interpretation

			if index.Kind() == ast.StringLiteralExpressionKind {
				objType := base.Type().(ast.ObjectType)
				indexStr := index.(ast.AnalyzedStringLiteralExpression).Value

				// check if the object contains this key

				var fieldRes ast.Type = nil
				for _, field := range objType.ObjFields {
					if field.FieldName.Ident() == indexStr {
						fieldRes = field.Type
						break
					}
				}

				if fieldRes == nil {
					self.error(
						fmt.Sprintf("Object does not contain a field with name '%s'", indexStr),
						nil,
						node.Range,
					)
					resultType = ast.NewUnknownType()
				} else {
					resultType = fieldRes.SetSpan(node.Range)
				}
			} else {
				// this yields an `any` type (the user must validate this manually)
				resultType = ast.NewAnyType(node.Range)
			}
		}
	case ast.ListTypeKind:
		// ensure that the index expression is an `int`
		if index.Type().Kind() != ast.IntTypeKind {
			self.error(
				fmt.Sprintf("A value of type '%s' cannot be indexed by '%s'", base.Type().Kind(), index.Type().Kind()),
				nil,
				node.Index.Span(),
			)
			resultType = ast.NewUnknownType()
		} else {
			// this yields a value of the inner list type
			list := base.Type().(ast.ListType)
			resultType = list.Inner.SetSpan(node.Range)
		}
	case ast.StringTypeKind:
		// ensure that the index expression is an `int`
		if index.Type().Kind() != ast.IntTypeKind {
			self.error(
				fmt.Sprintf("A value of type '%s' cannot be indexed by '%s'", base.Type().Kind(), index.Type().Kind()),
				nil,
				node.Index.Span(),
			)
			resultType = ast.NewUnknownType()
		} else {
			// this yields a value of another string
			resultType = ast.NewStringType(node.Range)
		}
	case ast.UnknownTypeKind, ast.NeverTypeKind:
		// this also yields the same type
		resultType = base.Type()
	default:
		// show an error
		self.error(
			fmt.Sprintf("Cannot index a value of type '%s'", base.Type().Kind()),
			nil,
			node.Span(),
		)
	}

	return ast.AnalyzedIndexExpression{
		Base:       base,
		Index:      index,
		ResultType: resultType,
		Range:      node.Range,
	}
}

//
// Member expression
//

func (self *Analyzer) memberExpression(node pAst.MemberExpression) ast.AnalyzedExpression {
	errOnAnyPrev := self.currentModule.CreateErrorIfContainsAny
	self.currentModule.CreateErrorIfContainsAny = false

	// make base
	base := self.expression(node.Base)

	self.currentModule.CreateErrorIfContainsAny = errOnAnyPrev

	var resultType ast.Type

	switch base.Type().Kind() {
	case ast.UnknownTypeKind, ast.NeverTypeKind:
		// ignore these, only use them as the result type
		resultType = base.Type()
	case ast.AnyTypeKind:
		self.error(
			"Implicit use of 'any' type: explicit type annotations required",
			[]string{"Consider casting this expression like this: `.. as type`"},
			base.Span(),
		)
		return ast.AnalyzedMemberExpression{
			Base:       ast.UnknownExpression{},
			Member:     node.Member,
			ResultType: ast.NewUnknownType(),
			Range:      node.Range,
		}
	default:
		// ensure that the field exists on the type of base
		fields := base.Type().Fields(node.Member.Span())
		res, found := fields[node.Member.Ident()]
		resultType = res

		if !found {
			self.error(
				fmt.Sprintf("Type '%s' has no member named '%s'", base.Type(), node.Member.Ident()),
				nil,
				node.Member.Span(),
			)

			// use `unknown` as the result type
			resultType = ast.NewUnknownType()
		}
	}

	return ast.AnalyzedMemberExpression{
		Base:       base,
		Member:     node.Member,
		ResultType: resultType.SetSpan(node.Range),
		Range:      node.Range,
	}
}

//
// Cast expression
//

func (self *Analyzer) castExpression(node pAst.CastExpression) ast.AnalyzedCastExpression {
	// TODO: detect if the types are equal: create warning
	self.currentModule.CreateErrorIfContainsAny = false
	base := self.expression(node.Base)
	self.currentModule.CreateErrorIfContainsAny = true

	asType := self.ConvertType(node.AsType, true)

	switch base.Type().Kind() {
	case ast.BoolTypeKind:
		switch asType.Kind() {
		case ast.IntTypeKind, ast.FloatTypeKind:
			return ast.AnalyzedCastExpression{
				Base:   base,
				AsType: asType,
				Range:  node.Range,
			}
		default:
			// this will probably result in an error
		}
	case ast.IntTypeKind:
		switch asType.Kind() {
		case ast.BoolTypeKind, ast.FloatTypeKind:
			return ast.AnalyzedCastExpression{
				Base:   base,
				AsType: asType,
				Range:  node.Range,
			}
		}
	case ast.FloatTypeKind:
		switch asType.Kind() {
		case ast.BoolTypeKind, ast.IntTypeKind:
			return ast.AnalyzedCastExpression{
				Base:   base,
				AsType: asType,
				Range:  node.Range,
			}
		default:
			// this will probably result in an error
		}
	case ast.ObjectTypeKind:
		switch asType.Kind() {
		case ast.AnyObjectTypeKind:
			return ast.AnalyzedCastExpression{
				Base:   base,
				AsType: asType,
				Range:  node.Range,
			}
		}
	}

	if err := self.TypeCheck(base.Type(), asType, false); err != nil {
		// check if the cast is legal
		self.error(
			fmt.Sprintf("Impossible cast: cannot cast value of type '%s' to '%s'", base.Type(), asType),
			[]string{err.GotDiagnostic.Message},
			node.Span(),
		)
	} else if asType.Kind() == ast.FnTypeKind {
		// check if the rhs contains a fn
		self.error(
			"Impossible cast: cannot cast a function value at runtime",
			[]string{"If the function is called later, consider casting its result value: `func() as type`"},
			node.Span(),
		)
	}

	return ast.AnalyzedCastExpression{
		Base:   base,
		AsType: asType.SetSpan(node.Range),
		Range:  node.Range,
	}
}

//
// If expression
//

func (self *Analyzer) ifExpression(node pAst.IfExpression) ast.AnalyzedIfExpression {
	// TODO: implement constant folding
	// analyze condition (must be bool)
	cond := self.expression(node.Condition)
	if err := self.TypeCheck(cond.Type(), ast.NewBoolType(errors.Span{}), true); err != nil {
		err.GotDiagnostic.Notes = append(
			err.GotDiagnostic.Notes,
			fmt.Sprintf("A condition must be of type '%s'", ast.TypeKind(ast.BoolTypeKind)),
		)
		self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
	}

	var resultType ast.Type

	// analyze then block
	thenBlock := self.block(node.ThenBlock, true)

	// if an else block exists, analyze it
	var elseBlock *ast.AnalyzedBlock = nil
	if node.ElseBlock != nil {
		elseBlockTemp := self.block(*node.ElseBlock, true)
		elseBlock = &elseBlockTemp

		// the two blocks must have the identical type
		if err := self.TypeCheck(elseBlock.ResultType, thenBlock.ResultType, true); err != nil {
			err.GotDiagnostic.Notes = append(err.GotDiagnostic.Notes, "The `if` and `else` branches must result in the identical type")
			self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
			if err.ExpectedDiagnostic != nil {
				self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
			}
			resultType = ast.NewUnknownType()
		} else {
			resultType = elseBlock.ResultType

			// only if both branches return `never`, use `never`
			if thenBlock.ResultType.Kind() == ast.NeverTypeKind {
				resultType = elseBlock.ResultType.SetSpan(node.Range)
				if elseBlockTemp.ResultType.Kind() == ast.NeverTypeKind {
					resultType = ast.NewNeverType()
				}
			} else if elseBlockTemp.ResultType.Kind() == ast.NeverTypeKind {
				resultType = thenBlock.ResultType.SetSpan(node.Range)
			}
		}
	} else {
		// check that the `then` branch results in null
		if err := self.TypeCheck(thenBlock.ResultType, ast.NewNullType(errors.Span{}), true); err != nil {
			self.diagnostics = append(self.diagnostics, diagnostic.Diagnostic{
				Level:   diagnostic.DiagnosticLevelError,
				Message: fmt.Sprintf("%s: missing `else` branch with result type '%s'", err.GotDiagnostic.Message, thenBlock.ResultType.Kind()),
				Notes:   []string{fmt.Sprintf("The `then` branch results in a value of type '%s', therefore an else branch is expected", thenBlock.ResultType.Kind())},
				Span:    err.GotDiagnostic.Span,
			})
			resultType = ast.NewUnknownType()
		} else {
			resultType = ast.NewNullType(thenBlock.ResultSpan())
		}
	}

	return ast.AnalyzedIfExpression{
		Condition:  cond,
		ThenBlock:  thenBlock,
		ElseBlock:  elseBlock,
		ResultType: resultType,
		Range:      node.Range,
	}
}

//
// Match expression
//

func (self *Analyzer) matchExpression(node pAst.MatchExpression) ast.AnalyzedMatchExpression {
	controlExpr := self.expression(node.ControlExpression)

	resultType := ast.NewUnknownType()
	hadTypeErr := false
	arms := make([]ast.AnalyzedMatchArm, 0)

	var defaultArmSpan *errors.Span

	var defaultArm *ast.AnalyzedExpression
	warnedUnreachable := false

	for _, arm := range node.Arms {
		if !arm.Literal.IsLiteral() {
			defaultArmSpan = &arm.Range
			action := self.expression(arm.Action)
			defaultArm = &action
			continue
		}

		if defaultArmSpan != nil && !warnedUnreachable {
			self.warn(
				"This match-arm is unreachable",
				nil,
				arm.Range,
			)

			self.hint(
				"Any branches following this arm are unreachable",
				nil,
				*defaultArmSpan,
			)

			warnedUnreachable = true
		}

		condition := self.expression(arm.Literal.Literal)

		if err := self.TypeCheck(condition.Type(), controlExpr.Type(), true); err != nil {
			self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
			if err.ExpectedDiagnostic != nil {
				self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
			}
		}

		action := self.expression(arm.Action)

		if !hadTypeErr && (resultType.Kind() == ast.UnknownTypeKind || resultType.Kind() == ast.NeverTypeKind) {
			resultType = action.Type()
		} else if err := self.TypeCheck(action.Type(), resultType, true); err != nil {
			hadTypeErr = true
			self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
			if err.ExpectedDiagnostic != nil {
				self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
			}
		}

		arms = append(arms, ast.AnalyzedMatchArm{
			Literal: condition,
			Action:  action,
		})
	}

	// create an error if the result type is != unknown and there is no default branch
	lastSpan := node.Span()
	if len(node.Arms) > 0 {
		lastSpan = node.Arms[len(node.Arms)-1].Range
	}
	if err := self.TypeCheck(ast.NewNullType(lastSpan), resultType, true); defaultArm == nil && err != nil {
		self.error(
			"Missing default branch",
			[]string{
				fmt.Sprintf("A value of type '%s' is expected, therefore cannot result in 'null'", resultType),
				"A default branch can be created like this: `_ => { ... },`",
			},
			lastSpan,
		)
		if err.ExpectedDiagnostic != nil {
			self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
		}
	}

	return ast.AnalyzedMatchExpression{
		ControlExpression: controlExpr,
		Arms:              arms,
		DefaultArmAction:  defaultArm,
		Range:             node.Range,
		ResultType:        resultType.SetSpan(node.Range),
	}
}

//
// Try expression
//

func (self *Analyzer) tryExpression(node pAst.TryExpression) ast.AnalyzedTryExpression {
	tryBlock := self.block(node.TryBlock, true)

	// add the error identifier to the new scope
	self.pushScope()
	self.currentModule.addVar(
		node.CatchIdent.Ident(),
		NewVar(
			errorType(node.CatchIdent.Span()),
			node.CatchIdent.Span(),
			NormalVariableOriginKind,
			false,
		),
		false,
	)

	catchBlock := self.block(node.CatchBlock, false)
	self.dropScope(true)

	var resultType ast.Type

	// only if both branches return `never`, use `never`
	if tryBlock.ResultType.Kind() == ast.NeverTypeKind {
		resultType = catchBlock.ResultType.SetSpan(node.Range)
		if catchBlock.ResultType.Kind() == ast.NeverTypeKind {
			resultType = ast.NewNeverType()
		}
	} else {
		resultType = tryBlock.ResultType.SetSpan(node.Range)
	}

	if err := self.TypeCheck(catchBlock.ResultType, tryBlock.ResultType, true); err != nil {
		err.GotDiagnostic.Notes = append(err.GotDiagnostic.Notes, "The `try` and `catch` branches must result in the identical type")
		self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
		if err.ExpectedDiagnostic != nil {
			self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
		}
		resultType = ast.NewUnknownType()
	}

	return ast.AnalyzedTryExpression{
		TryBlock:   tryBlock,
		CatchIdent: node.CatchIdent,
		CatchBlock: catchBlock,
		ResultType: resultType,
		Range:      node.Range,
	}
}

func errorType(span errors.Span) ast.Type {
	return ast.NewObjectType(
		[]ast.ObjectTypeField{
			ast.NewObjectTypeField(
				pAst.NewSpannedIdent(
					"message", span,
				),
				ast.NewStringType(span),
				span,
			),
			ast.NewObjectTypeField(
				pAst.NewSpannedIdent(
					"line", span,
				),
				ast.NewIntType(span),
				span,
			),
			ast.NewObjectTypeField(
				pAst.NewSpannedIdent(
					"column", span,
				),
				ast.NewIntType(span),
				span,
			),
			ast.NewObjectTypeField(
				pAst.NewSpannedIdent(
					"filename", span,
				),
				ast.NewStringType(span),
				span,
			),
		},
		span,
	)
}
