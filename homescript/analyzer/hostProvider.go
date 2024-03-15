package analyzer

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/diagnostic"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

// Is the return value if a trigger is imported successfully.
type TriggerFunction struct {
	TriggerFnType  ast.FunctionType
	CallbackFnType ast.FunctionType
	Connective     pAst.TriggerDispatchKeywordKind
	ImportedAt     errors.Span
}

// Is either a type (for types and values) or a template
type BuiltinImport struct {
	Type     ast.Type
	Template *ast.TemplateSpec
	Trigger  *TriggerFunction
}

type HostProvider interface {
	// Analyze imports, returns type of the imported value for type checks.
	// Example: `import { Foo } from bar;`
	// - `Foo` is the value name, if it was not found, `valueFound` is false.
	// - `Bar` is the module name, if it was not found, `moduleFound` is false.
	GetBuiltinImport(
		moduleName string,
		valueName string,
		span errors.Span,
		kind pAst.IMPORT_KIND,
	) (result BuiltinImport, moduleFound bool, valueFound bool)

	ResolveCodeModule(moduleName string) (code string, moduleFound bool, err error)

	// This method is invoked if the analyzer analyzes a module without errors
	// Then, this logical next stage is triggered to analyze the meta-semantics of the program.
	// This refers to any meaning of the program that is Homescript-agnostic and specific to that program.
	PostValidationHook(
		analyzedModules map[string]ast.AnalyzedProgram,
		mainModule string,
		// NOTE: this can be quite dangerous: the callee can mess up the analyzer and potentially cause crashes
		analyzer *Analyzer,
	) []diagnostic.Diagnostic
	// Returns a list of `known` object type annotations
	// The analyzer uses these in order to sanity-check every annotation
	// NOTE: these must not include the prefix token for annotations
	GetKnownObjectTypeFieldAnnotations() []string
}
