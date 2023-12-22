package ast

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

//
// Spanned ident
//

func NewSpannedIdent(ident string, span errors.Span) SpannedIdent {
	return SpannedIdent{
		ident: ident,
		span:  span,
	}
}

type SpannedIdent struct {
	ident string
	span  errors.Span
}

func (self SpannedIdent) Ident() string     { return self.ident }
func (self SpannedIdent) Span() errors.Span { return self.span }
func (self SpannedIdent) String() string    { return self.ident }

//
// Block
//

type Block struct {
	Statements []Statement
	Expression Expression
	Range      errors.Span
}

func (self Block) String() string {
	contents := make([]string, 0)

	for _, stmt := range self.Statements {
		contents = append(contents, strings.ReplaceAll(stmt.String(), "\n", "\n    "))
	}

	if self.Expression != nil {
		contents = append(contents, strings.ReplaceAll(self.Expression.String(), "\n", "\n    "))
	}

	return fmt.Sprintf("{\n    %s\n}", strings.Join(contents, "\n    "))
}

//
// Program
//

type Program struct {
	Imports    []ImportStatement
	Types      []TypeDefinition
	Singletons []SingletonTypeDefinition
	Globals    []LetStatement
	Functions  []FunctionDefinition
	Filename   string
}

func (self Program) String() string {
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
		types += "\n"
	}

	singletonTypes := ""
	for _, singleton := range self.Singletons {
		singletonTypes += singleton.String() + "\n"
	}
	if singletonTypes != "" {
		singletonTypes += "\n"
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

	return fmt.Sprintf("%s%s%s%s%s", imports, types, singletonTypes, globals, strings.Join(functions, "\n\n"))
}
