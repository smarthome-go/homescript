package value

type Executor interface {
	// if it exists, returns a value which is part of the host builtin modules
	GetBuiltinImport(moduleName string, toImport string) (val Value, found bool)
	// returns the Homescript code of the requested module
	ResolveModuleCode(moduleName string) (code string, found bool, err error)
	// Writes the given string (produced by a print function for instance) to any arbitrary source
	WriteStringTo(input string) error
	// Returns the username of the user who is executing the current script
	GetUser() string
}
