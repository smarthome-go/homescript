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

	return fmt.Sprintf("%s%s%s%s", imports, types, globals, strings.Join(functions, "\n\n"))
}

//
// Import
//

type AnalyzedImport struct {
	ToImport   []AnalyzedImportValue
	FromModule ast.SpannedIdent
	Range      errors.Span
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
	IsPub      bool
	Range      errors.Span
}

func (self AnalyzedFunctionDefinition) Span() errors.Span { return self.Range }
func (self AnalyzedFunctionDefinition) String() string {
	params := make([]string, 0)
	for _, param := range self.Parameters {
		params = append(params, param.String())
	}

	pub := ""
	if self.IsPub {
		pub = "pub "
	}

	return fmt.Sprintf("%sfn %s(%s) -> %s %s", pub, self.Ident, strings.Join(params, ", "), self.ReturnType, self.Body)
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

	for _, stmt := range self.Statements {
		contents = append(contents, strings.ReplaceAll(stmt.String(), "\n", "\n    "))
	}

	if self.Expression != nil {
		contents = append(contents, strings.ReplaceAll(self.Expression.String(), "\n", "\n    "))
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
