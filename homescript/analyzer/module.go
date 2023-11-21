package analyzer

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

type Module struct {
	ImportsModules           []string
	Functions                []*function
	Scopes                   []scope
	CurrentFunction          *function
	LoopDepth                uint // continue and break are legal if > 0
	CurrentLoopIsTerminated  bool // specifies whether there is at least one `break` statement inside the current loop
	CreateErrorIfContainsAny bool // if enabled, every expression which contains `any` will be reported as an error
}

//
// Functions
//

type function struct {
	FnType         functionType
	Parameters     []ast.AnalyzedFnParam
	ParamsSpan     errors.Span
	ReturnType     ast.Type
	ReturnTypeSpan errors.Span
	Used           bool
	Modifier       pAst.FunctionModifier
}

func (self function) Type(span errors.Span) ast.Type {
	params := make([]ast.FunctionTypeParam, 0)
	for _, param := range self.Parameters {
		params = append(params, ast.NewFunctionTypeParam(
			param.Ident,
			param.Type,
		))
	}

	return ast.NewFunctionType(
		ast.NewNormalFunctionTypeParamKind(params),
		self.ParamsSpan,
		self.ReturnType,
		span,
	)
}

func newFunction(
	typ functionType,
	params []ast.AnalyzedFnParam,
	paramsSpan errors.Span,
	returnType ast.Type,
	returnSpan errors.Span,
	modifier pAst.FunctionModifier,
) function {
	return function{
		FnType:         typ,
		Parameters:     params,
		ReturnType:     returnType,
		ReturnTypeSpan: returnSpan,
		Used:           false,
		Modifier:       modifier,
	}
}

type functionTypeKind uint8

const (
	lambdaFunctionKind functionTypeKind = iota
	normalFunctionKind
)

type functionType interface {
	Kind() functionTypeKind
}

type lambdaFunction struct{}

func (self lambdaFunction) Kind() functionTypeKind { return lambdaFunctionKind }
func newLambdaFunction() functionType              { return functionType(lambdaFunction{}) }

type normalFunction struct {
	Ident pAst.SpannedIdent
}

func (self normalFunction) Kind() functionTypeKind { return normalFunctionKind }
func newNormalFunction(ident pAst.SpannedIdent) functionType {
	return functionType(normalFunction{Ident: ident})
}

//
// Type wrapper
//

type typeWrapper struct {
	Type     ast.Type
	IsPub    bool
	NameSpan errors.Span
	Used     bool
}

func newTypeWrapper(typ ast.Type, isPub bool, nameSpan errors.Span, used bool) typeWrapper {
	return typeWrapper{
		Type:     typ,
		IsPub:    isPub,
		NameSpan: nameSpan,
		Used:     used,
	}
}

//
// Types and scoping
//

type scope struct {
	Values map[string]*Variable    // stores variable and function types
	Types  map[string]*typeWrapper // like `Values`, but for types
}

func newScope() scope {
	return scope{
		Values: make(map[string]*Variable),
		Types:  make(map[string]*typeWrapper),
	}
}

type VariableOriginKind uint8

const (
	NormalVariableOriginKind VariableOriginKind = iota
	ImportedVariableOriginKind
	BuiltinVariableOriginKind
	ParameterVariableOriginKind
)

type Variable struct {
	Type   ast.Type
	Span   errors.Span
	Used   bool
	Origin VariableOriginKind
	IsPub  bool
}

func NewVar(typ ast.Type, span errors.Span, origin VariableOriginKind, isPub bool) Variable {
	return Variable{
		Type:   typ,
		Span:   span,
		Used:   false,
		Origin: origin,
		IsPub:  isPub,
	}
}

func NewBuiltinVar(typ ast.Type) Variable {
	return NewVar(typ, errors.Span{}, BuiltinVariableOriginKind, false)
}

//
// Utility methods for types
//

func (self Module) getType(ident string) (typ *typeWrapper, found bool) {
	// iterate through the types backwards (more current types dominate)
	for idx := len(self.Scopes) - 1; idx >= 0; idx-- {
		val, found := self.Scopes[idx].Types[ident]
		if found {
			return val, true
		}
	}
	return nil, false
}

func (self *Module) addType(ident string, typ typeWrapper) (previous *typeWrapper) {
	prev, exists := self.Scopes[len(self.Scopes)-1].Types[ident]
	if exists {
		return prev
	}
	self.Scopes[len(self.Scopes)-1].Types[ident] = &typ
	return nil
}

//
// Utility methods for scoping
//

func (self *Module) pushScope() {
	self.Scopes = append(self.Scopes, newScope())
}

// NOTE: this will fail if len == 0
func (self *Module) popScope() scope {
	last := self.Scopes[len(self.Scopes)-1]
	self.Scopes = self.Scopes[:len(self.Scopes)-1]
	return last
}

func (self Module) getVar(ident string) (val *Variable, scope uint, found bool) {
	// iterate through the scopes backwards
	for idx := len(self.Scopes) - 1; idx >= 0; idx-- {
		val, found := self.Scopes[idx].Values[ident]
		if found {
			return val, uint(idx), true
		}
	}
	return nil, 0, false
}

func (self *Module) addVar(ident string, val Variable, forceAdd bool) (previous *Variable) {
	prev, alreadyExists := self.Scopes[len(self.Scopes)-1].Values[ident]
	if alreadyExists {
		if forceAdd {
			self.Scopes[len(self.Scopes)-1].Values[ident] = &val
		}
		return prev
	}
	self.Scopes[len(self.Scopes)-1].Values[ident] = &val
	return nil
}

//
// Utility methods for functions
//

func (self Module) getFunc(ident string) (fn *function, found bool) {
	for _, fn := range self.Functions {
		if fn.FnType.(normalFunction).Ident.Ident() == ident {
			return fn, true
		}
	}
	return nil, false
}

func (self *Module) addFunc(function function) {
	self.Functions = append(self.Functions, &function)
}

func (self Module) getCurrentFunc() *function {
	return self.CurrentFunction
}

func (self *Module) setCurrentFunc(ident string) {
	for _, fn := range self.Functions {
		if fn.FnType.Kind() == normalFunctionKind {
			if fn.FnType.(normalFunction).Ident.Ident() == ident {
				self.CurrentFunction = fn
				return
			}
		}
	}
	panic(fmt.Sprintf("`setCurrentFunc` was called with a non-existing function as its identifier (%s)", ident))
}

//
// Utility for imports
//

func (self Module) Imports(test string) bool {
	for _, module := range self.ImportsModules {
		if module == test {
			return true
		}
	}
	return false
}
