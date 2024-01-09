package ast

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

//
// Singleton (Type definition)
//

type AnalyzedSingletonTypeDefinition struct {
	Ident   ast.SpannedIdent
	TypeDef AnalyzedTypeDefinition
	Range   errors.Span
}

func (self AnalyzedSingletonTypeDefinition) Kind() AnalyzedStatementKind {
	return SingletonTypeDefinitionStatementKind
}
func (self AnalyzedSingletonTypeDefinition) Span() errors.Span { return self.Range }
func (self AnalyzedSingletonTypeDefinition) String() string {
	return fmt.Sprintf("%s\n%s", self.Ident.Ident(), self.TypeDef)
}
func (self AnalyzedSingletonTypeDefinition) Type() Type { return NewNullType(self.Range) }

//
// Impl block
//

// Impl block
type AnalyzedImplBlock struct {
	SingletonIdent ast.SpannedIdent
	SingletonType  Type
	// Template implementation is optional
	UsingTemplate *ast.ImplBlockTemplate
	Methods       []AnalyzedFunctionDefinition
	Span          errors.Span
}
