package ast

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

type AnalyzedSingleton struct {
	Type                Type
	ImplementsTemplates []ast.ImplBlockTemplate
	// Methods are unlike methods in languages like Rust or Java.
	// Here, a method is just a function that is implemented in a template block.
	Methods []AnalyzedFunctionDefinition
	Used    bool
}

func NewSingleton(typ Type, implementsTemplates []ast.ImplBlockTemplate, methods []AnalyzedFunctionDefinition) AnalyzedSingleton {
	return AnalyzedSingleton{
		Type:                typ,
		ImplementsTemplates: implementsTemplates,
		Methods:             methods,
	}
}

// A template is comparable to a trait in Rust.
// It describes which methods (and their signatures) need to be implemented on a singleton.
type TemplateCapability struct {
	RequiresMethods           []string
	ConflictsWithCapabilities []TemplateConflict
}

type TemplateConflict struct {
	ConflictingCapability string
	ConflictReason        string
}

type TemplateCapabilityWithSpan struct {
	Capability TemplateCapability
	Span       errors.Span
}

type TemplateMethod struct {
	Signature FunctionType
	Modifier  ast.FunctionModifier
}

type TemplateSpec struct {
	// These methods are not always required.
	// Depending on the capabilities, some of them will be required.
	BaseMethods  map[string]TemplateMethod
	Capabilities map[string]TemplateCapability
	// These capabilities are automaticallty added if the user does not add any explicitly
	DefaultCapabilities []string
	Span                errors.Span
}

func DetermineCapabilityConflicts(
	capabilityName string,
	capability TemplateCapability,
	selectedCapabilities map[string]TemplateCapabilityWithSpan,
	span errors.Span,
) (containsErr bool, conflictFound string, err diagnostic.Diagnostic) {
	// BUG: this method currently does not work

	for _, conflict := range capability.ConflictsWithCapabilities {
		_, containsConflict := selectedCapabilities[conflict.ConflictingCapability]

		remainingText := ""
		remainingConflicts := len(capability.ConflictsWithCapabilities) - 1
		if remainingConflicts > 0 {
			remainingText = fmt.Sprintf(" and %d others", remainingConflicts)
		}

		if containsConflict {
			notes := []string{
				fmt.Sprintf("The capability `%s` cannot be implemented alongside `%s`%s", capabilityName, conflict.ConflictingCapability, remainingText),
			}

			if conflict.ConflictReason != "" {
				notes = append(notes, conflict.ConflictReason)
			}

			return true, conflict.ConflictingCapability, diagnostic.Diagnostic{
				Level:   diagnostic.DiagnosticLevelError,
				Message: fmt.Sprintf("Template capability `%s` conflicts with other capability `%s`", capabilityName, conflict.ConflictingCapability),
				Notes:   notes,
				Span:    span,
			}
		}
	}

	return false, "", diagnostic.Diagnostic{}
}
