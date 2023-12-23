package analyzer

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type BUILTIN_IMPORT_KIND uint8

const (
	BUILTIN_IMPORT_KIND_VALUE BUILTIN_IMPORT_KIND = iota
	BUILTIN_IMPORT_KIND_TYPE
	BUILTIN_IMPORT_KIND_TEMPLATE
)

// Is either a type (for types and values) or a template
type BuiltinImport struct {
	Type     ast.Type
	Template *ast.TemplateSpec
}

type HostProvider interface {
	GetBuiltinImport(
		moduleName string,
		valueName string,
		span errors.Span,
		kind BUILTIN_IMPORT_KIND,
	) (result BuiltinImport, moduleFound bool, valueFound bool)
	ResolveCodeModule(moduleName string) (code string, moduleFound bool, err error)
}
