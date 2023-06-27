package analyzer

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type HostDependencies interface {
	GetBuiltinImport(moduleName string, valueName string, span errors.Span) (valueType ast.Type, moduleFound bool, valueFound bool)
	ResolveCodeModule(moduleName string) (code string, moduleFound bool, err error)
}
