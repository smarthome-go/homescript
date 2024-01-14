package ast

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

//
// Singleton (type definition)
//

type SingletonTypeDefinition struct {
	Ident SpannedIdent
	Type  HmsType
	Range errors.Span
}

func (self SingletonTypeDefinition) Span() errors.Span { return self.Range }
func (self SingletonTypeDefinition) String() string {
	return fmt.Sprintf("%s\n%s", self.Ident, self.Type)
}

// Impl block capabilities
type ImplBlockCapabilities struct {
	// Capabilities are optional: if this is `false`, there was no `with` token.
	Defined bool
	List    []SpannedIdent
	Span    errors.Span
}

// Impl block template
type ImplBlockTemplate struct {
	Template     SpannedIdent
	Capabilities ImplBlockCapabilities
}

// Impl block
type ImplBlock struct {
	SingletonIdent SpannedIdent
	// Template is optional
	UsingTemplate *ImplBlockTemplate
	Methods       []FunctionDefinition
	Span          errors.Span
}
