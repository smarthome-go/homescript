package ast

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Program
//

type AnalyzedProgram struct {
	Imports    []AnalyzedImport
	Types      []AnalyzedTypeDefinition
	Singletons []AnalyzedSingletonTypeDefinition
	ImplBlocks []AnalyzedImplBlock
	Globals    []AnalyzedLetStatement
	Functions  []AnalyzedFunctionDefinition
}

func (self AnalyzedProgram) String() string {
	imports := ""
	for _, item := range self.Imports {
		imports += item.String() + "\n"
	}
	if imports != "" {
		imports += "\n"
	}

	types := ""
	for _, typ := range self.Types {
		types += typ.String() + "\n"
	}
	if types != "" {
		types += "\n\n"
	}

	singletons := ""
	for _, singleton := range self.Singletons {
		singletons += singleton.String() + "\n"
	}
	if singletons != "" {
		singletons += "\n\n"
	}

	globals := ""
	for _, glob := range self.Globals {
		globals += glob.String()
	}
	if globals != "" {
		globals += "\n\n"
	}

	functions := make([]string, 0)
	for _, fn := range self.Functions {
		functions = append(functions, fn.String())
	}

	return fmt.Sprintf("%s%s%s%s%s", imports, types, singletons, globals, strings.Join(functions, "\n\n"))
}

//
// Import
//

type AnalyzedImport struct {
	ToImport    []AnalyzedImportValue
	FromModule  ast.SpannedIdent
	Range       errors.Span
	TargetIsHMS bool
}

func (self AnalyzedImport) Span() errors.Span { return self.Range }
func (self AnalyzedImport) String() string {
	toImport := make([]string, 0)
	for _, imported := range self.ToImport {
		toImport = append(toImport, imported.String())
	}
	return fmt.Sprintf("import { %s } from %s;", strings.Join(toImport, ", "), self.FromModule)
}
func (self AnalyzedImport) Type() Type { return NewNullType(self.Range) }

type AnalyzedImportValue struct {
	Ident ast.SpannedIdent
	Kind  ast.IMPORT_KIND
	Type  Type
}

func (self AnalyzedImportValue) String() string {
	switch self.Kind {
	case ast.IMPORT_KIND_NORMAL:
		return self.Ident.Ident()
	case ast.IMPORT_KIND_TYPE:
		return fmt.Sprintf("type %s", self.Ident)
	case ast.IMPORT_KIND_TEMPLATE:
		return fmt.Sprintf("templ %s", self.Ident)
	case ast.IMPORT_KIND_TRIGGER:
		return fmt.Sprintf("trigger %s", self.Ident)
	default:
		panic("A new import kind was added without updating this code")
	}
}

//
// Function definition
//

type AnalyzedFunctionParams struct {
	List []AnalyzedFnParam
	Span errors.Span
}

func (self AnalyzedFunctionParams) Type() FunctionTypeParamKind {
	params := make([]FunctionTypeParam, 0)
	for _, param := range self.List {
		var singletonIdent *string = nil
		if param.IsSingletonExtractor {
			singletonIdent = &param.SingletonIdent
		}
		params = append(params, NewFunctionTypeParam(param.Ident, param.Type, singletonIdent))
	}
	return NewNormalFunctionTypeParamKind(params)
}

type AnalyzedFunctionDefinition struct {
	Ident      ast.SpannedIdent
	Parameters AnalyzedFunctionParams
	ReturnType Type
	Body       AnalyzedBlock
	Modifier   ast.FunctionModifier
	Annotation *AnalyzedFunctionAnnotation
	Range      errors.Span
}

func (self AnalyzedFunctionDefinition) Span() errors.Span { return self.Range }
func (self AnalyzedFunctionDefinition) String() string {
	annotation := ""
	if self.Annotation != nil {
		annotation = fmt.Sprintf("%s\n", *self.Annotation)
	}

	params := make([]string, 0)
	for _, param := range self.Parameters.List {
		params = append(params, param.String())
	}

	modifier := ""
	switch self.Modifier {
	case ast.FN_MODIFIER_PUB:
		modifier = "pub "
	case ast.FN_MODIFIER_EVENT:
		modifier = "event "
	case ast.FN_MODIFIER_NONE:
		break
	default:
		panic(fmt.Sprintf("This modifier is not implemented: %d.", self.Modifier))
	}

	return fmt.Sprintf("%s%sfn %s(%s) -> %s %s", annotation, modifier, self.Ident, strings.Join(params, ", "), self.ReturnType, self.Body)
}
func (self AnalyzedFunctionDefinition) Type() Type {
	return NewFunctionType(self.Parameters.Type(), self.Parameters.Span, self.ReturnType, self.Range)
}

type AnalyzedFnParam struct {
	Ident                ast.SpannedIdent
	Type                 Type
	Span                 errors.Span
	IsSingletonExtractor bool
	SingletonIdent       string
}

func (self AnalyzedFnParam) String() string {
	return fmt.Sprintf("%s: %s", self.Ident, self.Type)
}

//
// Function annotation
//

type AnalyzedFunctionAnnotation struct {
	Items []AnalyzedAnnotationItem
	Span  errors.Span
}

func (self AnalyzedFunctionAnnotation) String() string {
	items := make([]string, len(self.Items))

	for idx, item := range self.Items {
		items[idx] = item.String()
	}

	return fmt.Sprintf("#[%s]", strings.Join(items, ", "))
}

type AnalyzedAnnotationItem interface {
	Span() errors.Span
	String() string
}

type AnalyzedAnnotationItemIdent struct {
	Ident ast.SpannedIdent
}

func (self AnalyzedAnnotationItemIdent) Span() errors.Span {
	return self.Ident.Span()
}

func (self AnalyzedAnnotationItemIdent) String() string {
	return self.Ident.Ident()
}

type AnalyzedAnnotationItemTrigger struct {
	TriggerConnective ast.TriggerDispatchKeywordKind
	TriggerSource     ast.SpannedIdent
	TriggerArgs       AnalyzedCallArgs
	Range             errors.Span
}

func (self AnalyzedAnnotationItemTrigger) Span() errors.Span {
	return self.Range
}

func (self AnalyzedAnnotationItemTrigger) String() string {
	return fmt.Sprintf("trigger %s %s(%s)", self.TriggerConnective, self.TriggerSource, self.TriggerArgs)
}

//
// Block
//

type AnalyzedBlock struct {
	Statements []AnalyzedStatement
	Expression AnalyzedExpression
	Range      errors.Span
	ResultType Type
}

func (self AnalyzedBlock) String() string {
	contents := make([]string, 0)

	isEmpty := true

	for _, stmt := range self.Statements {
		contents = append(contents, strings.ReplaceAll(stmt.String(), "\n", "\n    "))
		isEmpty = false
	}

	if self.Expression != nil {
		contents = append(contents, strings.ReplaceAll(self.Expression.String(), "\n", "\n    "))
		isEmpty = false
	}

	if isEmpty {
		return "{}"
	}

	return fmt.Sprintf("{\n    %s\n}", strings.Join(contents, "\n    "))
}

func (self AnalyzedBlock) Type() Type { return self.ResultType }

// returns the span that is responsible for the block's result type
func (self AnalyzedBlock) ResultSpan() errors.Span {
	if self.Expression != nil {
		return self.Expression.Span()
	} else if len(self.Statements) > 0 {
		return self.Statements[len(self.Statements)-1].Span()
	} else {
		return self.Range
	}
}
