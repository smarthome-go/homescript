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
	Imports   []AnalyzedImport
	Types     []AnalyzedTypeDefinition
	Globals   []AnalyzedLetStatement
	Functions []AnalyzedFunctionDefinition
	Events    []AnalyzedFunctionDefinition
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
		types = "\n"
	}

	globals := ""
	for _, glob := range self.Globals {
		globals += glob.String()
	}
	if globals != "" {
		globals += "\n"
	}

	functions := make([]string, 0)
	for _, fn := range self.Functions {
		functions = append(functions, fn.String())
	}

	events := make([]string, 0)
	for _, fn := range self.Events {
		events = append(events, fn.String())
	}

	eventsStr := ""
	if len(events) > 0 {
		eventsStr = "\n\n" + strings.Join(events, "\n\n")
	}

	return fmt.Sprintf("%s%s%s%s%s", imports, types, globals, strings.Join(functions, "\n\n"), eventsStr)
}

func (self AnalyzedProgram) SupportsEvent(ident string) bool {
	for _, event := range self.Events {
		if event.Ident.Ident() == ident {
			return true
		}
	}

	return false
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
	return fmt.Sprintf("import {%s} from %s;", strings.Join(toImport, ", "), self.FromModule)
}
func (self AnalyzedImport) Type() Type { return NewNullType(self.Range) }

type AnalyzedImportValue struct {
	Ident ast.SpannedIdent
	Type  Type
}

func (self AnalyzedImportValue) String() string { return self.Ident.Ident() }

//
// Function definition
//

type AnalyzedFunctionDefinition struct {
	Ident      ast.SpannedIdent
	Parameters []AnalyzedFnParam
	ReturnType Type
	Body       AnalyzedBlock
	Modifier   ast.FunctionModifier
	Range      errors.Span
}

func (self AnalyzedFunctionDefinition) Span() errors.Span { return self.Range }
func (self AnalyzedFunctionDefinition) String() string {
	params := make([]string, 0)
	for _, param := range self.Parameters {
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

	return fmt.Sprintf("%sfn %s(%s) -> %s %s", modifier, self.Ident, strings.Join(params, ", "), self.ReturnType, self.Body)
}
func (self AnalyzedFunctionDefinition) Type() Type { return NewNullType(self.Range) }

type AnalyzedFnParam struct {
	Ident ast.SpannedIdent
	Type  Type
	Span  errors.Span
}

func (self AnalyzedFnParam) String() string {
	return fmt.Sprintf("%s: %s", self.Ident, self.Type)
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
