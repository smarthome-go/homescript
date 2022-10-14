package homescript

import "github.com/smarthome-go/homescript/homescript/interpreter"

type Interpreter struct {
	program  []Statement
	executor interpreter.Executor
	scopes   []map[string]interpreter.Value

	// Can be used to terminate the script at any point in time
	sigTerm *chan int
}

func NewInterpreter(
	program []Statement,
	executor interpreter.Executor,
	sigTerm *chan int,
) Interpreter {
	scopes := make([]map[string]interpreter.Value, 0)
	scopes = append(scopes, map[string]interpreter.Value{
		// Builtin functions implemented by Homescript
		"exit":  interpreter.ValueBuiltinFunction{}, // Special function implemented below
		"throw": interpreter.ValueBuiltinFunction{Callback: interpreter.Throw},
		// Builtin functions implemented by the executor
		"sleep":     interpreter.ValueBuiltinFunction{Callback: interpreter.Sleep},
		"switch_on": interpreter.ValueBuiltinFunction{Callback: interpreter.SwitchOn},
		"switch":    interpreter.ValueBuiltinFunction{Callback: interpreter.Switch},
		"notify":    interpreter.ValueBuiltinFunction{Callback: interpreter.Notify},
		"log":       interpreter.ValueBuiltinFunction{Callback: interpreter.Log},
		"exec":      interpreter.ValueBuiltinFunction{Callback: interpreter.Exec},
		"get":       interpreter.ValueBuiltinFunction{Callback: interpreter.Get},
		"http":      interpreter.ValueBuiltinFunction{Callback: interpreter.Http},
		// Builtin variables
		"user":    interpreter.ValueBuiltinVariable{Callback: interpreter.GetUser},
		"weather": interpreter.ValueBuiltinVariable{Callback: interpreter.GetWeather},
		"time":    interpreter.ValueBuiltinVariable{Callback: interpreter.GetTime},
	})
	return Interpreter{
		program:  program,
		executor: executor,
		scopes:   scopes,
		sigTerm:  sigTerm,
	}
}

// TODO: interpreter code comes here
