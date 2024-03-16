package analyzer

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//
// Statements
//

func (self *Analyzer) statement(node pAst.Statement) ast.AnalyzedStatement {
	switch node.Kind() {
	case pAst.TriggerStatementKind:
		src := node.(pAst.TriggerStatement)
		return self.triggerStatement(src)
	case pAst.TypeDefinitionStatementKind:
		src := node.(pAst.TypeDefinition)
		return self.typeDefStatement(src)
	case pAst.LetStatementKind:
		src := node.(pAst.LetStatement)
		return self.letStatement(src, false)
	case pAst.ReturnStatementKind:
		src := node.(pAst.ReturnStatement)
		return self.returnStatement(src)
	case pAst.BreakStatementKind:
		src := node.(pAst.BreakStatement)
		return self.breakStatement(src)
	case pAst.ContinueStatementKind:
		src := node.(pAst.ContinueStatement)
		return self.continueStatement(src)
	case pAst.LoopStatementKind:
		src := node.(pAst.LoopStatement)
		return self.loopStatement(src)
	case pAst.WhileStatementKind:
		src := node.(pAst.WhileStatement)
		return self.whileStatement(src)
	case pAst.ForStatementKind:
		src := node.(pAst.ForStatement)
		return self.forStatement(src)
	case pAst.ExpressionStatementKind:
		src := node.(pAst.ExpressionStatement)
		return ast.AnalyzedExpressionStatement{
			Expression: self.expression(src.Expression),
			Range:      src.Range,
		}
	default:
		panic("A new statement kind was introduced without updating this code")
	}
}

//
// Trigger statement
//

// Checks:
//
// - Trigger Event
//   - Exists (pay respect to the dispatcher keyword)
//
// - Callback function
//   - Exists
//   - Has the correct type
//   - Is an `event` function
func (self *Analyzer) triggerStatement(node pAst.TriggerStatement) ast.AnalyzedTriggerStatement {
	// Analyze event trigger
	trigger, triggerFound := self.currentModule.getTrigger(node.TriggerIdent.Ident())
	// triggerType, found, connectiveCorrect := self.host.GetTriggerEvent(node.EventIdent.Ident(), node.DispatchKeyword)
	if !triggerFound {
		self.error(
			fmt.Sprintf("Use of undefined trigger function '%s'", node.TriggerIdent.Ident()),
			[]string{
				fmt.Sprintf("Trigger functions can be imported like this: `import { trigger %s } from ... ;`", node.TriggerIdent.Ident()),
			},
			node.TriggerIdent.Span(),
		)
	}

	// Analyze callback function
	callbackFn, callbackFound := self.currentModule.getFunc(node.CallbackFnIdent.Ident())
	if !callbackFound {
		self.error(
			fmt.Sprintf("Use of undefined callback function '%s'", node.CallbackFnIdent.Ident()),
			[]string{
				fmt.Sprintf("Functions can be defined like this: `fn %s(...) { ... }`", node.CallbackFnIdent.Ident()),
			},
			node.CallbackFnIdent.Span(),
		)

		// Still analyze arguments
		for _, arg := range node.EventArguments.List {
			self.expression(arg)
		}

		return ast.AnalyzedTriggerStatement{
			CallbackIdent:     node.CallbackFnIdent,
			CallbackSignature: ast.FunctionType{},
			ConnectiveKeyword: node.DispatchKeyword,
			TriggerIdent:      node.TriggerIdent,
			TriggerSignature:  ast.FunctionType{},
			TriggerArguments:  ast.AnalyzedCallArgs{},
			Range:             errors.Span{},
		}
	}

	if self.currentModule.CurrentFunction.FnType.Kind() == normalFunctionKind {
		currFn := self.currentModule.CurrentFunction.FnType.(normalFunction)
		if callbackFn.FnType.Kind() == normalFunctionKind {
			toBeCalled := callbackFn.FnType.(normalFunction)
			if currFn.Ident.Ident() != toBeCalled.Ident.Ident() {
				callbackFn.Used = true
			} else {
				self.error(
					"Cannot trigger function from itself",
					[]string{
						"This could lead to unwanted recursive behavior",
						"Move this statement to another function",
					},
					node.Span(),
				)
			}
		}
	}

	// Analyze callback function compatibility with event trigger
	callbackFnType := callbackFn.Type(node.Span()).(ast.FunctionType)

	// if callbackFnType.Params.Kind() != trigger.Type.Params.Kind() {
	// 	panic("This is a bug in the host implementation: a trigger function provided by the host should always be of param kind `normal`")
	// }

	// callbackParams := callbackFnType.Params.(ast.NormalFunctionTypeParamKindIdentifier).Params
	// triggerParams := trigger.Type.Params.(ast.NormalFunctionTypeParamKindIdentifier).Params

	// if len(callbackParams) != len(triggerParams) {
	// 	// self.error(
	// 	// 	fmt.Sprintf("Expected function ")
	// 	// )
	// }

	args := ast.AnalyzedCallArgs{
		Span: node.EventArguments.Span,
		List: []ast.AnalyzedCallArgument{},
	}

	if triggerFound {
		if err := self.TypeCheck(callbackFnType, trigger.CallbackFnType, true); err != nil {
			self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
			if err.ExpectedDiagnostic != nil {
				self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
			}
		}
		args = self.callArgs(
			trigger.TriggerFnType,
			node.EventArguments,
			false,
		)
	}

	return ast.AnalyzedTriggerStatement{
		CallbackIdent:     node.CallbackFnIdent,
		CallbackSignature: callbackFnType,
		ConnectiveKeyword: node.DispatchKeyword,
		TriggerIdent:      node.TriggerIdent,
		TriggerSignature:  trigger.TriggerFnType,
		TriggerArguments:  args,
		Range:             node.Range,
	}
}

//
// Type definition statement
//

func (self *Analyzer) typeDefStatement(node pAst.TypeDefinition) ast.AnalyzedTypeDefinition {
	// if the conversion fails, use unknown
	// NOTE: `SetSpan` is not used so that object fields can be shown as `expected`
	converted := self.ConvertType(node.RhsType, true)

	// also add the declaration to the current type scope
	if prev := (*self.currentModule).addType(node.LhsIdent.Ident(), newTypeWrapper(converted, node.IsPub, node.LhsIdent.Span(), node.IsPub)); prev != nil {
		self.error(
			fmt.Sprintf("Type '%s' is already declared as '%s' in this scope", node.LhsIdent.Ident(), prev.Type),
			[]string{"Consider altering this type's name"},
			node.LhsIdent.Span(),
		)
	}

	return ast.AnalyzedTypeDefinition{
		LhsIdent: node.LhsIdent.Ident(),
		RhsType:  converted,
		Range:    node.Range,
	}
}

//
// Singleton declaration statement
//

func (self *Analyzer) singletonDeclStatement(node pAst.SingletonTypeDefinition) ast.AnalyzedSingletonTypeDefinition {
	converted := self.ConvertType(node.Type, true)

	singleton, found := self.currentModule.Singletons[node.Ident.Ident()]
	if found {
		self.error(
			fmt.Sprintf("Singleton type '%s' is already declared as '%s' in this module", node.Ident.Ident(), singleton.Type),
			[]string{"Consider altering this type's name"},
			node.Ident.Span(),
		)
	} else {
		// Add this item to the map of singletons
		s := ast.NewSingleton(
			converted,
			make([]pAst.ImplBlockTemplate, 0),
			make([]ast.AnalyzedFunctionDefinition, 0),
		)

		self.currentModule.Singletons[node.Ident.Ident()] = &s
	}

	return ast.AnalyzedSingletonTypeDefinition{
		Ident:               node.Ident,
		SingletonType:       converted,
		Range:               node.Range,
		ImplementsTemplates: make([]pAst.ImplBlockTemplate, 0),
		Used:                false,
	}
}

//
// Let statement
//

func (self *Analyzer) letStatement(node pAst.LetStatement, isGlobal bool) ast.AnalyzedLetStatement {
	// do not create an error if the expression contains `any`
	createAnyErrBefore := self.currentModule.CreateErrorIfContainsAny
	self.currentModule.CreateErrorIfContainsAny = false
	initExpr := self.expression(node.Expression)
	self.currentModule.CreateErrorIfContainsAny = createAnyErrBefore

	rhsType := initExpr.Type().SetSpan(node.Expression.Span())

	forceUnknownType := false

	if isGlobal {
		// ensure that the initializer is constant
		if !initExpr.Constant() {
			self.error(
				"Global initializer must be constant",
				[]string{fmt.Sprintf("Values of type `%s` are not allowed in global variables.", initExpr.Type()), "Consider using a value with a supported type."},
				node.Expression.Span(),
			)
			forceUnknownType = true
		}
	}

	rhsHasAny := self.CheckAny(rhsType)

	// determine which type to use for the variable
	varType := rhsType

	var optType ast.Type
	if node.OptType != nil {
		optType = self.ConvertType(node.OptType, true)
	}

	// check that the optional type annotation does not cause a conflict
	if node.OptType != nil {
		if err := self.TypeCheck(rhsType, optType, !rhsHasAny); err != nil {
			self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
			if err.ExpectedDiagnostic != nil {
				self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
			}
		} else {
			// if the optional type annotation fixed the issue, use the optional type
			varType = optType
		}
	} else if rhsHasAny {
		self.error(
			"Implicit use of 'any' type: explicit type annotations required",
			[]string{"An explicit type can be declared like this: `let foo: type = ...`"},
			node.Ident.Span(),
		)
		self.hint(
			fmt.Sprintf("This expression is of type '%s'", rhsType),
			[]string{"Or consider casting this expression: `... as type`"},
			initExpr.Span(),
		)
		forceUnknownType = true
	}

	if forceUnknownType {
		// required for preventing misleading errors for non-constant expressions
		varType = ast.NewUnknownType()
	}

	// `force-add` is desired here, the variable should be shadowed
	if prev := self.currentModule.addVar(node.Ident.Ident(), NewVar(varType, node.Ident.Span(), NormalVariableOriginKind, node.IsPub), true); prev != nil {
		if isGlobal {
			// prevent duplicate globals
			self.error(
				fmt.Sprintf("Duplicate definition of global '%s'", node.Ident.Ident()),
				make([]string, 0),
				node.Ident.Span(),
			)

			self.hint(
				fmt.Sprintf("Previous definition of global '%s'", node.Ident.Ident()),
				nil,
				prev.Span,
			)
		} else {
			if !strings.HasPrefix(node.Ident.Ident(), "_") && !prev.Used {
				label := ""
				switch prev.Origin {
				case NormalVariableOriginKind:
					label = "variable"
				case ImportedVariableOriginKind, BuiltinVariableOriginKind:
					// ignore these
				case ParameterVariableOriginKind:
					label = "parameter"
				}

				// variable is being shadowed, warn if the old variable was unused
				self.warn(
					fmt.Sprintf("Unused %s '%s'", label, node.Ident.Ident()),
					nil,
					prev.Span,
				)
				caser := cases.Title(language.AmericanEnglish)
				self.hint(
					fmt.Sprintf("%s '%s' shadowed here", caser.String(label), node.Ident.Ident()),
					nil,
					node.Ident.Span(),
				)
			}
		}
	}

	return ast.AnalyzedLetStatement{
		Ident:                      node.Ident,
		Expression:                 initExpr,
		VarType:                    varType,
		NeedsRuntimeTypeValidation: rhsHasAny,
		OptType:                    optType,
		Range:                      node.Range,
	}
}

//
// Return statement
//

func (self *Analyzer) returnStatement(node pAst.ReturnStatement) ast.AnalyzedReturnStatement {
	// analyze optional expression
	var returnExpression ast.AnalyzedExpression = nil
	gotReturnType := ast.NewNullType(node.Range)

	if node.Expression != nil {
		returnExpression = self.expression(node.Expression)
		gotReturnType = returnExpression.Type()
	}

	// check if the statement is inside a function or lambda literal
	if self.currentModule.CurrentFunction == nil {
		self.error(
			"Illegal use of return statement outside of function body",
			nil,
			node.Span(),
		)

		// NOTE: must return early in order to avoid nil pointer dereference
		return ast.AnalyzedReturnStatement{
			ReturnValue: returnExpression,
			Range:       node.Range,
		}
	}

	// check for possible type conflicts
	if err := self.TypeCheck(gotReturnType, self.currentModule.CurrentFunction.ReturnType, true); err != nil {
		self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
		if err.ExpectedDiagnostic != nil {
			self.diagnostics = append(self.diagnostics, *err.ExpectedDiagnostic)
		}
	}

	return ast.AnalyzedReturnStatement{
		ReturnValue: returnExpression,
		Range:       node.Range,
	}
}

//
// Break statement
//

func (self *Analyzer) breakStatement(node pAst.BreakStatement) ast.AnalyzedBreakStatement {
	// check that this statement is only called inside of a loop
	if self.currentModule.LoopDepth == 0 {
		self.error(
			"Illegal use of 'break' outside of a loop",
			[]string{"This statement can only be used in loop bodies"},
			node.Range,
		)
	}

	// signal that the current loop is terminated
	self.currentModule.CurrentLoopIsTerminated = true

	return ast.AnalyzedBreakStatement{
		Range: node.Range,
	}
}

//
// Continuue statement
//

func (self *Analyzer) continueStatement(node pAst.ContinueStatement) ast.AnalyzedContinueStatement {
	if self.currentModule.LoopDepth == 0 {
		self.error(
			"Illegal use of 'continue' statement outside of a loop",
			[]string{"This statement can only be used in loop bodies"},
			node.Range,
		)
	}

	return ast.AnalyzedContinueStatement{
		Range: node.Range,
	}
}

//
// Loop statement
//

func (self *Analyzer) loopStatement(node pAst.LoopStatement) ast.AnalyzedLoopStatement {
	// validate that the block returns `null`
	oldLoopIsTerminated := self.currentModule.CurrentLoopIsTerminated
	self.currentModule.LoopDepth++

	body := self.block(node.Body, true)

	self.currentModule.LoopDepth--

	neverTerminates := !self.currentModule.CurrentLoopIsTerminated
	// restore old loop termination
	self.currentModule.CurrentLoopIsTerminated = oldLoopIsTerminated

	// ensure that the result value of the body is `null` or `never`
	self.expectLoopToReturnNull(body.ResultType, body.ResultSpan())

	return ast.AnalyzedLoopStatement{
		Body:            body,
		NeverTerminates: neverTerminates,
		Range:           node.Range,
	}
}

//
// While statement
//

func (self *Analyzer) whileStatement(node pAst.WhileStatement) ast.AnalyzedWhileStatement {
	condExpr := self.expression(node.Condition)

	// validate that the condition if of type `bool`
	if err := self.TypeCheck(condExpr.Type(), ast.NewBoolType(errors.Span{}), true); err != nil {
		self.diagnostics = append(self.diagnostics, err.GotDiagnostic)
	}

	// validate that the block returns `null`
	oldLoopIsTerminated := self.currentModule.CurrentLoopIsTerminated
	self.currentModule.LoopDepth++

	body := self.block(node.Body, true)

	self.currentModule.LoopDepth--

	neverTerminates := !self.currentModule.CurrentLoopIsTerminated
	// restore loop termination
	self.currentModule.CurrentLoopIsTerminated = oldLoopIsTerminated

	neverTerminates = false // TODO: implement this correctly

	// ensure that the result value of the body is `null` or `never`
	self.expectLoopToReturnNull(body.ResultType, body.ResultSpan())

	return ast.AnalyzedWhileStatement{
		Condition:       condExpr,
		Body:            body,
		NeverTerminates: neverTerminates,
		Range:           node.Range,
	}
}

//
// For statement
//

func (self *Analyzer) forStatement(node pAst.ForStatement) ast.AnalyzedForStatement {
	iterExpr := self.expression(node.IterExpression)

	iterType := ast.NewUnknownType()

	switch iterExpr.Type().Kind() {
	case ast.RangeTypeKind:
		// iterating over a range always produces a value of type `int`
		iterType = ast.NewIntType(node.Identifier.Span())
	case ast.StringTypeKind:
		// iterating over a string produces substrings
		iterType = ast.NewStringType(node.Identifier.Span())
	case ast.ListTypeKind:
		listType := iterExpr.Type().(ast.ListType)
		// iterating over a list produces the inner type of the list
		iterType = listType.Inner.SetSpan(node.Identifier.Span())
	case ast.UnknownTypeKind, ast.NeverTypeKind:
		// ignore these, caused by earlier errors / warnings
	default:
		self.error(
			fmt.Sprintf("A value of type '%s' cannot be used as an iterator", iterExpr.Type()),
			nil,
			iterExpr.Span(),
		)
	}

	oldLoopIsTerminated := self.currentModule.CurrentLoopIsTerminated
	self.currentModule.LoopDepth++
	self.pushScope()

	// add the iterator to the scope of the loop body
	self.currentModule.addVar(node.Identifier.Ident(), NewVar(
		iterType,
		node.Identifier.Span(),
		NormalVariableOriginKind,
		false,
	), false)

	body := self.block(node.Body, false)

	self.dropScope(true)
	self.currentModule.LoopDepth--

	neverTerminates := !self.currentModule.CurrentLoopIsTerminated
	// restore loop termination
	self.currentModule.CurrentLoopIsTerminated = oldLoopIsTerminated

	neverTerminates = false // TODO: implement this correctly

	// ensure that the result value of the body is `null` or `never`
	self.expectLoopToReturnNull(body.ResultType, body.ResultSpan())

	return ast.AnalyzedForStatement{
		Identifier:      node.Identifier,
		IterExpression:  iterExpr,
		IterVarType:     iterType,
		Body:            body,
		NeverTerminates: neverTerminates,
		Range:           node.Range,
	}
}

func (self *Analyzer) expectLoopToReturnNull(typ ast.Type, errSpan errors.Span) {
	switch typ.Kind() {
	case ast.UnknownTypeKind, ast.NeverTypeKind, ast.NullTypeKind:
		// ignore this, this is the desired state
	default:
		self.error(
			fmt.Sprintf(
				"Loop requires a block of type '%s' or '%s', found '%s'",
				ast.TypeKind(ast.NullTypeKind),
				ast.TypeKind(ast.NeverTypeKind),
				typ.Kind(),
			),
			nil,
			errSpan,
		)
	}
}
