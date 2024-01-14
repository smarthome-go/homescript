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
	Ident         ast.SpannedIdent
	SingletonType Type
	Range         errors.Span
	// This is mainly used for the post-validation hook so that it can analyze these easily.
	ImplementsTemplates []ast.ImplBlockTemplate
	Used                bool
}

func (self AnalyzedSingletonTypeDefinition) Kind() AnalyzedStatementKind {
	return SingletonTypeDefinitionStatementKind
}
func (self AnalyzedSingletonTypeDefinition) Span() errors.Span { return self.Range }
func (self AnalyzedSingletonTypeDefinition) String() string {
	return fmt.Sprintf("%s\n%s", self.Ident.Ident(), self.SingletonType)
}

func (self AnalyzedSingletonTypeDefinition) Type() Type { return self.SingletonType }

//
// Impl block
//

// Impl block
type AnalyzedImplBlock struct {
	SingletonIdent ast.SpannedIdent
	SingletonType  Type
	UsingTemplate  ast.ImplBlockTemplate
	Methods        []AnalyzedFunctionDefinition
	Span           errors.Span
}
