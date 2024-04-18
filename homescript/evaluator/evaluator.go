package evaluator

import (
	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

type RuntimeModule struct {
	singletons   map[string]*value.Value
	scopes       []map[string]*value.Value
	currentScope uint // Indexes the field above.
	filename     string
}

type Interpreter struct {
	modules    map[string]*RuntimeModule
	currModule *RuntimeModule
}

func NewInterpreter(program map[string]ast.AnalyzedProgram, entrypoint string, executor value.Executor) (Interpreter, *value.VmInterrupt) {
	modules := make(map[string]*RuntimeModule)

	for moduleName, module := range program {

		processedSingletons := make(map[string]value.Value)

		for _, singleton := range module.Singletons {
			loadedValue, loadFound, i := executor.LoadSingleton(singleton.Ident.Ident(), moduleName)
			if i != nil {
				return Interpreter{}, value.NewVMFatalException(
					i.Error(),
					value.Vm_HostErrorKind,
					singleton.Span(),
				)
			}

			if !loadFound {
				loadedValue = *value.ZeroValue(singleton.SingletonType)
			}

			processedSingletons[singleton.Ident.Ident()] = loadedValue
		}

		scopes := make([]map[string]*value.Value, 1)
		scopes[0] = make(map[string]*value.Value)

		for singletonName, val := range processedSingletons {
			scopes[0][singletonName] = &val
		}

		modules[moduleName] = &RuntimeModule{
			scopes:       scopes,
			currentScope: 0,
			filename:     moduleName,
		}
	}

	return Interpreter{
		modules:    modules,
		currModule: modules[entrypoint],
	}, nil
}
