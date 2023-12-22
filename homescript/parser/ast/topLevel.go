package ast

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

//
// Singleton (type definition)
//

type SingletonTypeDefinition struct {
	Ident   SpannedIdent
	TypeDef TypeDefinition
	Range   errors.Span
}

func (self SingletonTypeDefinition) Span() errors.Span { return self.Range }
func (self SingletonTypeDefinition) String() string {
	return fmt.Sprintf("%s\n%s", self.Ident, self.TypeDef)
}

// Impl block
type ImplBlock struct {
	SingletonIdent SpannedIdent
	// Template is optional
	TemplateIdent *SpannedIdent
	Methods       []FunctionDefinition

	Span errors.Span
}
