package value

import "github.com/smarthome-go/homescript/v3/homescript/errors"

type Executor interface {
	// Tries to load a saved singleton instance from the host.
	// If the host cannot provide a saved instance, a default value is used.
	LoadSingleton(singletonIdent string, moduleName string) (val Value, isValid bool, err error)
	// If it exists, returns a value which is part of the host builtin modules.
	GetBuiltinImport(moduleName string, toImport string) (val Value, found bool)
	// Returns the Homescript code of the requested module.
	ResolveModuleCode(moduleName string) (code string, found bool, err error)
	// Writes the given string (produced by a print function for instance) to any arbitrary source.
	WriteStringTo(input string) error
	// Used for registering a trigger callback. (MQTT or time-based).
	RegisterTrigger(
		callbackFunctionIdent string,
		eventTriggerIdent string,
		span errors.Span,
		args []Value,
	) error
	// TODO: unregister function
	// Is executed by the host to release any resources that the executed program allocated.
	Free() error
}
