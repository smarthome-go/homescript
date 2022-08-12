package interpreter

import (
	"fmt"
	"math"
	"strconv"

	"github.com/smarthome-go/homescript/homescript/error"
)

var numberNames = []string{
	"First",
	"Second",
	"Third",
	"Fourth",
}

// Helper function which checks the validity of args provided to builtin functions
func checkArgs(name string, location error.Location, args []Value, types ...ValueType) *error.Error {
	if len(args) != len(types) {
		s := ""
		if len(types) != 1 {
			s = "s"
		}
		return error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Function '%s' takes %d argument%s but %d were given", name, len(types), s, len(args)),
		)
	}
	for i, typ := range types {
		if args[i].Type() != typ {
			return error.NewError(
				error.TypeError,
				location,
				fmt.Sprintf("%s argument of function '%s' has to be of type %s", numberNames[i], name, typ.Name()),
			)
		}
	}
	return nil
}

// Terminates the execution of the current Homescript
// Exit code `0` indicates success, other can be used for different purposes
func Exit(location error.Location, args ...Value) (*error.Error, *int) {
	if err := checkArgs("exit", location, args, Number); err != nil {
		return err, nil
	}
	code := args[0].(ValueNumber).Value
	if code == float64(int(math.Round(code))) {
		code := int(math.Round(code))
		return nil, &code
	}
	return error.NewError(
		error.TypeError,
		location,
		"First argument of function 'exit' has to be an integer",
	), nil
}

// Terminates the execution of the current Homescript using an error
func Panic(_ Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("panic", location, args, String); err != nil {
		return nil, err
	}
	return nil, error.NewError(
		error.Panic,
		location,
		args[0].(ValueString).Value,
	)
}

// Parses a given string to an integer
// If the string could not be parsed to an integer, an error is returned
func Num(_ Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("num", location, args, String); err != nil {
		return nil, err
	}
	valueInt, err := strconv.ParseFloat(args[0].(ValueString).Value, 32)
	if err != nil {
		return nil, error.NewError(
			error.ValueError,
			location, fmt.Sprintf("Argument must be parseable to a number: %s", err.Error()),
		)
	}
	return ValueNumber{
		Value: float64(valueInt),
	}, nil
}

// Converts an arbitrary value of any data type in Homescript to a textual representation
// The output will be returned as a string
func Str(self Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if len(args) != 1 {
		return nil, error.NewError(error.ValueError, location,
			fmt.Sprintf("Function 'str' requires exactly 1 argument but %d were given", len(args)))
	}
	res, err := args[0].ToString(self, location)
	if err != nil {
		return nil, err
	}
	return ValueString{
		Value: res,
	}, nil
}

// Concatenates the provided strings, forming a large string
func Concat(self Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if len(args) < 2 {
		return nil, error.NewError(error.ValueError, location,
			fmt.Sprintf("Function 'concat' requires at least 2 argument but %d were given", len(args)))
	}
	output := ""
	for argIndex, arg := range args {
		if arg.Type() != String {
			return nil, error.NewError(
				error.TypeError,
				location,
				fmt.Sprintf("Argument %d of function 'concat' has to be a string", argIndex+1),
			)
		}
		output += arg.(ValueString).Value
	}
	return ValueString{Value: output}, nil
}

// Checks if the provided number is even
func Even(self Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("even", location, args, Number); err != nil {
		return nil, err
	}
	return ValueBoolean{
		Value: int(args[0].(ValueNumber).Value)%2 == 0,
	}, nil
}

// Pauses the execution of the current script for a given amount of seconds
func Sleep(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("sleep", location, args, Number); err != nil {
		return nil, err
	}
	seconds := args[0].(ValueNumber).Value
	// The sleep function has been migrated to the executor in order to allow better linting / dry run without delays
	executor.Sleep(seconds)
	return ValueVoid{}, nil
}

// Indicates whether a certain argument has been passed to this Homescript
func CheckArg(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("checkArg", location, args, String); err != nil {
		return nil, err
	}
	toCheck := args[0].(ValueString).Value
	argProvided := executor.CheckArg(toCheck)
	return ValueBoolean{
		Value: argProvided,
	}, nil
}

// Returns the argument's value as a string
// If the argument was not passed, an error is returned
// It is recommended to validate the existence of an argument with the `checkArg` function
func GetArg(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("getArg", location, args, String); err != nil {
		return nil, err
	}
	toGet := args[0].(ValueString).Value
	argValue, err := executor.GetArg(toGet)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueString{
		Value: argValue,
	}, nil
}

// Outputs a string
func Print(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	msgs := make([]string, 0)
	for _, arg := range args {
		res, err := arg.ToString(executor, location)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, res)
	}
	executor.Print(msgs...)
	return ValueVoid{}, nil
}

// Retrieves the current power state of the provided switch
func SwitchOn(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("switchOn", location, args, String); err != nil {
		return nil, err
	}
	name := args[0].(ValueString).Value
	value, err := executor.SwitchOn(name)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueBoolean{
		Value: value,
	}, nil
}

// Used to interact with switches and change power states
func Switch(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("switch", location, args, String, Boolean); err != nil {
		return nil, err
	}
	name := args[0].(ValueString).Value
	on := args[1].(ValueBoolean).Value
	if err := executor.Switch(name, on); err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

// If a notification system is provided in the runtime environment a notification is sent to the current user
func Notify(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("notify", location, args, String, String, Number); err != nil {
		return nil, err
	}
	title := args[0].(ValueString).Value
	description := args[1].(ValueString).Value
	rawLevel := args[2].(ValueNumber).Value
	if rawLevel != float64(int(math.Round(rawLevel))) {
		return nil, error.NewError(
			error.TypeError,
			location,
			"Third argument of function 'notify' has to be an integer",
		)
	}
	var level LogLevel
	switch rawLevel {
	case 1:
		level = LevelInfo
	case 2:
		level = LevelWarn
	case 3:
		level = LevelError
	default:
		return nil, error.NewError(
			error.ValueError,
			location,
			fmt.Sprintf("Notification level has to be one of 1, 2, or 3, got %d", int(math.Round(rawLevel))),
		)
	}
	if err := executor.Notify(title, description, level); err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

// Adds a event to the logging system
func Log(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("log", location, args, String, String, Number); err != nil {
		return nil, err
	}
	title := args[0].(ValueString).Value
	description := args[1].(ValueString).Value
	rawLevel := args[2].(ValueNumber).Value
	if rawLevel != float64(int(math.Round(rawLevel))) {
		return nil, error.NewError(
			error.TypeError,
			location,
			"Third argument of function 'log' has to be an integer",
		)
	}
	var level LogLevel
	switch rawLevel {
	case 0:
		level = LevelTrace
	case 1:
		level = LevelDebug
	case 2:
		level = LevelInfo
	case 3:
		level = LevelWarn
	case 4:
		level = LevelError
	case 5:
		level = LevelFatal
	default:
		return nil, error.NewError(
			error.ValueError,
			location,
			fmt.Sprintf("Log level has to be one of 0, 1, 2, 3, 4, or 5 got %d", int(math.Round(rawLevel))),
		)
	}
	if err := executor.Log(title, description, level); err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

// Launches a Homescript based on the provided script Id
// If no valid script could be found or the user lacks permission to execute it, an error is returned
func Exec(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	var output string
	// Validate that at least one argument was provided
	if len(args) == 0 {
		return nil, error.NewError(
			error.TypeError,
			location,
			"Function 'exec' takes 1 or more arguments but 0 were given",
		)
	}
	// Validate that the first argument is of type string
	if args[0].Type() != String {
		return nil, error.NewError(
			error.TypeError,
			location,
			"First argument of function 'exec' has to be of type String",
		)
	}
	// Create call arguments from other args
	callArgsFinal := make(map[string]string, 0)
	for indexArg, arg := range args[1:] {
		if arg.Type() != PairType {
			return nil, error.NewError(
				error.TypeError,
				location,
				fmt.Sprintf("Argument %d of function 'exec' has to be of type Pair\nhint: you can create a value pair using `pair('key', 'value')`", indexArg),
			)
		}
		_, alreadyExists := callArgsFinal[arg.(ValuePair).Value.Key]
		if alreadyExists {
			return nil, error.NewError(
				error.TypeError,
				location,
				fmt.Sprintf("Call argument (value pair) %d of function 'exec' has duplicate key entry '%s'", indexArg+2, arg.(ValuePair).Value.Key),
			)
		}
		// Add the argument to the argument map
		callArgsFinal[arg.(ValuePair).Value.Key] = arg.(ValuePair).Value.Value
	}
	// Execute Homescript
	homescriptId := args[0].(ValueString).Value
	output, err := executor.Exec(homescriptId, callArgsFinal)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueString{
		Value: output,
	}, nil
}

// Creates a new user unless the username is already taken
func AddUser(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("addUser", location, args, String, String, String, String); err != nil {
		return nil, err
	}
	if err := executor.AddUser(
		args[0].(ValueString).Value,
		args[1].(ValueString).Value,
		args[2].(ValueString).Value,
		args[3].(ValueString).Value,
	); err != nil {
		return ValueVoid{}, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

// Deletes an arbitrary user
func DelUser(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("delUser", location, args, String); err != nil {
		return nil, err
	}
	if err := executor.DelUser(args[0].(ValueString).Value); err != nil {
		return ValueVoid{}, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

// Adds a permission to an arbitrary user
func AddPerm(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("addPerm", location, args, String, String); err != nil {
		return nil, err
	}
	if err := executor.AddPerm(args[0].(ValueString).Value, args[1].(ValueString).Value); err != nil {
		return ValueVoid{}, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

// Deletes an existing permission from arbitrary user
func DelPerm(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("delPerm", location, args, String, String); err != nil {
		return nil, err
	}
	if err := executor.DelPerm(args[0].(ValueString).Value, args[1].(ValueString).Value); err != nil {
		return ValueVoid{}, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

// Sends ICMP Echo Request to the target IP and checks for a response
// The first parameter specifies the host, the second the timeOut
func Ping(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("ping", location, args, String, Number); err != nil {
		return nil, err
	}
	hostAlive, err := executor.Ping(args[0].(ValueString).Value, args[1].(ValueNumber).Value)
	if err != nil {
		return ValueVoid{}, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueBoolean{
		Value: hostAlive,
	}, nil
}

// Makes a get-request to an arbitrary url and returns the result as plaintext
func Get(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("get", location, args, String); err != nil {
		return nil, err
	}
	body, err := executor.Get(args[0].(ValueString).Value)
	if err != nil {
		return ValueVoid{}, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueString{
		Value: body,
	}, nil
}

// Makes a network request using an arbitrary URL, method , body (as plaintext), (and optionally headers)
func Http(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	// Validate that at least three arguments are provided
	if len(args) < 3 {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Function 'http' takes three or more arguments but %d were given", len(args)),
		)
	}
	// Validate that the first three arguments are of type string
	for argIndex, arg := range args {
		if arg.Type() != String {
			return nil, error.NewError(
				error.TypeError,
				location,
				fmt.Sprintf("%s argument of function 'http' has to be of type String", numberNames[argIndex]),
			)
		}
		if argIndex == 2 {
			break
		}
	}
	// Create header values from remaining args
	headers := make(map[string]string, 0)
	for headerIndex, header := range args[3:] {
		if header.Type() != PairType {
			return nil, error.NewError(
				error.TypeError,
				location,
				fmt.Sprintf("Argument %d of function 'http' has to be of type Pair.\nhint: you can create a value pair using `pair('key', 'value')`", headerIndex+4),
			)
		}
		_, alreadyExists := headers[header.(ValuePair).Value.Key]
		if alreadyExists {
			return nil, error.NewError(
				error.ValueError,
				location,
				fmt.Sprintf("Header entry (value pair) %d of function 'http' has duplicate key entry '%s'", headerIndex+4, header.(ValuePair).Value.Key),
			)
		}
		// Add the argument to the argument map
		headers[header.(ValuePair).Value.Key] = header.(ValuePair).Value.Value
	}
	body, err := executor.Http(
		args[0].(ValueString).Value,
		args[1].(ValueString).Value,
		args[2].(ValueString).Value,
		headers,
	)
	if err != nil {
		return ValueVoid{}, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueString{
		Value: body,
	}, nil
}

// Acts like a factory for the `pair` type
// Requires two arguments: pair(key, value)
// The first argument is the key, the second one is the value
func Pair(_ Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if err := checkArgs("pair", location, args, String, String); err != nil {
		return nil, err
	}
	return ValuePair{
		Value: struct {
			Key   string
			Value string
		}{
			Key:   args[0].(ValueString).Value,
			Value: args[1].(ValueString).Value,
		},
	}, nil
}

// //////////// Variables //////////////
func GetUser(executor Executor, _ error.Location) (Value, *error.Error) {
	return ValueString{Value: executor.GetUser()}, nil
}

func GetWeather(executor Executor, location error.Location) (Value, *error.Error) {
	val, err := executor.GetWeather()
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueString{Value: val}, nil
}

func GetTemperature(executor Executor, location error.Location) (Value, *error.Error) {
	val, err := executor.GetTemperature()
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueNumber{Value: float64(val)}, nil
}

func GetCurrentYear(executor Executor, _ error.Location) (Value, *error.Error) {
	year, _, _, _, _, _, _ := executor.GetDate()
	return ValueNumber{Value: float64(year)}, nil
}

func GetCurrentMonth(executor Executor, _ error.Location) (Value, *error.Error) {
	_, month, _, _, _, _, _ := executor.GetDate()
	return ValueNumber{Value: float64(month)}, nil
}

func GetCurrentWeek(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, week, _, _, _, _ := executor.GetDate()
	return ValueNumber{Value: float64(week)}, nil
}

func GetCurrentDay(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, day, _, _, _ := executor.GetDate()
	return ValueNumber{Value: float64(day)}, nil
}

func GetCurrentHour(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, _, hour, _, _ := executor.GetDate()
	return ValueNumber{Value: float64(hour)}, nil
}

func GetCurrentMinute(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, _, _, minute, _ := executor.GetDate()
	return ValueNumber{Value: float64(minute)}, nil
}

func GetCurrentSecond(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, _, _, _, second := executor.GetDate()
	return ValueNumber{Value: float64(second)}, nil
}
