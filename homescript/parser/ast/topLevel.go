package ast

import (
	"fmt"
	"strings"

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
	Template                SpannedIdent
	UserDefinedCapabilities ImplBlockCapabilities
}

// Impl block
type ImplBlock struct {
	SingletonIdent SpannedIdent
	UsingTemplate  ImplBlockTemplate
	Methods        []FunctionDefinition
	Span           errors.Span
}

//
// Function annotation.
//

type FunctionAnnotation struct {
	Function FunctionDefinition
	Inner    FunctionAnnotationInner
	Span     errors.Span
}

type FunctionAnnotationInner struct {
	Span  errors.Span
	Items []AnnotationItem
}

func (self FunctionAnnotationInner) String() string {
	innerList := make([]string, 0)

	for _, annotation := range self.Items {
		innerList = append(innerList, annotation.String())
	}

	return fmt.Sprintf("#[%s]", strings.Join(innerList, ", "))
}

type AnnotationItem interface {
	Span() errors.Span
	String() string
}

//
// Ident, like `#[foo]`
//

type AnnotationItemIdent struct {
	Ident SpannedIdent
}

func (self AnnotationItemIdent) Span() errors.Span {
	return self.Ident.span
}

func (self AnnotationItemIdent) String() string {
	return self.Ident.ident
}

//
// Trigger, like #[trigger at noon()]
//

type AnnotationItemTrigger struct {
	TriggerConnective TriggerDispatchKeywordKind
	TriggerSource     SpannedIdent
	TriggerArgs       CallArgs
	Range             errors.Span
}

func (self AnnotationItemTrigger) Span() errors.Span {
	return self.Range
}

func (self AnnotationItemTrigger) String() string {
	return fmt.Sprintf("trigger %s %s(%s)", self.TriggerConnective, self.TriggerSource, self.TriggerArgs)
}
