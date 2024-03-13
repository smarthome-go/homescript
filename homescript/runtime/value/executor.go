package value

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
	// Is executed by the Host to release any ressources that the executed program allocated.
	Free() error
}
