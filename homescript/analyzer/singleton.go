package analyzer

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

func (self *Analyzer) templateCapabilitiesWithDefault(
	templateSpecName string,
	templateSpec ast.TemplateSpec,
	userImplementsCapabilities pAst.ImplBlockCapabilities,
) map[string]ast.TemplateCapabilityWithSpan {
	capabilities := make(map[string]ast.TemplateCapabilityWithSpan)

	// Use the default capabilities
	for _, defaultCapability := range templateSpec.DefaultCapabilities {
		cap, found := templateSpec.Capabilities[defaultCapability]
		if !found {
			panic(fmt.Sprintf("Encountered default capability `%s` that is not in capabilities", defaultCapability))
		}

		capabilities[defaultCapability] = ast.TemplateCapabilityWithSpan{
			Capability: cap,
			Span:       errors.Span{}, // TODO: does this break things? -> It should not
		}
	}

	// Furthermore, use the implemented capabilities
	for _, implementedCap := range userImplementsCapabilities.List {
		capability, exists := templateSpec.Capabilities[implementedCap.Ident()]
		if !exists {
			self.error(
				fmt.Sprintf("Capability `%s` not found on template `%s`", implementedCap.Ident(), templateSpecName),
				[]string{"Remove this capability from the `impl` block"},
				implementedCap.Span(),
			)

			// Ignore this erronous capability
			continue
		}

		// If everything went well, use this capability
		capabilities[implementedCap.Ident()] = ast.TemplateCapabilityWithSpan{
			Capability: capability,
			Span:       implementedCap.Span(),
		}
	}

	return capabilities
}

// Maps a selected capability to its span in the source code (`impl Foo with { CAP } for`),
// Here, the span corresponds to the position to CAP
func (self *Analyzer) WithCapabilities(
	templateSpec ast.TemplateSpec,
	capabilitiesOfImplementation map[string]ast.TemplateCapabilityWithSpan,
) (methods map[string]ast.TemplateMethod, err bool) {
	// Compute the set of required methods based on the capabilities
	// Also check that this capability does not conflict with other capabilities.
	methods = make(map[string]ast.TemplateMethod)

	err = false

	// Reverse tupel of the conflicts found so far
	// If `foo` finds `bar` as a conflict, (`bar`, `foo`) is added to the map for later lookup.
	// This way, redundant compatability errors are not shown
	conflictsReverse := make(map[string]string)

	for capName, capability := range capabilitiesOfImplementation {
		// Check that there are no capability conflicts
		containsErr, conflictFound, diagnotsics := ast.DetermineCapabilityConflicts(
			templateSpec,
			capName,
			capability.Capability,
			capabilitiesOfImplementation,
			capability.Span,
		)
		if containsErr {
			// Prevent redundant compatability errors
			if _, found := conflictsReverse[capName]; found {
				continue
			}

			self.diagnostics = append(self.diagnostics, diagnotsics)
			err = true
			conflictsReverse[conflictFound] = capName
			continue
		}

		// For this capability, add all required methods
		for _, method := range capability.Capability.RequiresMethods {
			meth, found := templateSpec.BaseMethods[method]
			if !found {
				panic(fmt.Sprintf("This is a bug warning: capability `%s` references template method `%s` which does not exist on the template", capName, method))
			}
			methods[method] = meth
		}
	}

	// BUG: this returns nothing currently
	return methods, err
}
