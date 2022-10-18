package homescript

import (
	"fmt"
	"math"

	"github.com/smarthome-go/homescript/homescript/errors"
)

var numberNames = []string{
	"First",
	"Second",
	"Third",
	"Fourth",
}

// Helper function which checks the validity of args provided to builtin functions
func checkArgs(name string, span errors.Span, args []Value, types ...ValueType) *errors.Error {
	if len(args) != len(types) {
		s := ""
		if len(types) != 1 {
			s = "s"
		}
		return errors.NewError(
			span,
			fmt.Sprintf("Function '%s' takes %d argument%s but %d were given", name, len(types), s, len(args)),
			errors.TypeError,
		)
	}
	for i, typ := range types {
		if args[i].Type() != typ {
			return errors.NewError(
				span,
				fmt.Sprintf("%s argument of function '%s' has to be of type %v", numberNames[i], name, typ),
				errors.TypeError,
			)
		}
	}
	return nil
}

/// Builtins implemented by Homescript ///

// Terminates the execution of the current Homescript
// Exit code `0` indicates success, other values can be used for different purposes
func Exit(_ Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("exit", span, args, TypeNumber); err != nil {
		return nil, nil, err
	}
	code := args[0].(ValueNumber).Value
	if code == float64(int(math.Round(code))) {
		code := int(math.Round(code))
		return nil, &code, nil
	}
	return nil, nil, errors.NewError(
		span,
		"First argument of function 'exit' has to be an integer",
		errors.TypeError,
	)
}

// Returns an intentional error
func Throw(_ Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("throw", span, args, TypeString); err != nil {
		return nil, nil, err
	}
	return nil, nil, errors.NewError(
		span,
		args[0].(ValueString).Value,
		errors.ThrowError,
	)
}

// Asserts that a statement is true, otherwise an error is returned
func Assert(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if len(args) != 1 {
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("Function 'assert' takes 1 argument but %d were given", len(args)),
			errors.RuntimeError,
		)
	}
	isTrue, err := args[0].IsTrue(executor, span)
	if err != nil {
		return nil, nil, err
	}
	if !isTrue {
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("Assertion of %v value failed", args[0].Type()),
			errors.ValueError,
		)
	}
	return ValueNull{}, nil, nil
}

/// Builtins implemented by the executor ///

// Pauses the execution of the current script for a given amount of seconds
func Sleep(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("sleep", span, args, TypeNumber); err != nil {
		return nil, nil, err
	}
	seconds := args[0].(ValueNumber).Value
	// The sleep function has been migrated to the executor in order to allow better linting / dry run without delays
	executor.Sleep(seconds)
	return ValueNull{}, nil, nil
}

// Outputs a string
func Print(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	msgs := make([]string, 0)
	for _, arg := range args {
		res, err := arg.Display(executor, span)
		if err != nil {
			return nil, nil, err
		}
		msgs = append(msgs, res)
	}
	executor.Print(msgs...)
	return ValueNull{}, nil, nil
}

// Retrieves the current power state of the provided switch
func SwitchOn(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("switchOn", span, args, TypeString); err != nil {
		return nil, nil, err
	}
	name := args[0].(ValueString).Value
	value, err := executor.SwitchOn(name)
	if err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueBool{
		Value: value,
	}, nil, nil
}

// Used to interact with switches and change power states
func Switch(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("switch", span, args, TypeString, TypeBoolean); err != nil {
		return nil, nil, err
	}
	name := args[0].(ValueString).Value
	on := args[1].(ValueBool).Value
	if err := executor.Switch(name, on); err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueNull{}, nil, nil
}

// If a notification system is provided in the runtime environment a notification is sent to the current user
func Notify(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("notify", span, args, TypeString, TypeString, TypeNumber); err != nil {
		return nil, nil, err
	}
	title := args[0].(ValueString).Value
	description := args[1].(ValueString).Value
	rawLevel := args[2].(ValueNumber).Value
	if rawLevel != float64(int(math.Round(rawLevel))) {
		return nil, nil, errors.NewError(
			span,
			"Third argument of function 'notify' has to be an integer",
			errors.TypeError,
		)
	}
	var level NotificationLevel
	switch rawLevel {
	case 1:
		level = NotiInfo
	case 2:
		level = NotiWarning
	case 3:
		level = NotiCritical
	default:
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("Notification level has to be one of 1, 2, or 3, got %d", int(math.Round(rawLevel))),
			errors.ValueError,
		)
	}
	if err := executor.Notify(title, description, level); err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueNull{}, nil, nil
}

// Adds a event to the logging system
func Log(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("log", span, args, TypeString, TypeString, TypeNumber); err != nil {
		return nil, nil, err
	}
	title := args[0].(ValueString).Value
	description := args[1].(ValueString).Value
	rawLevel := args[2].(ValueNumber).Value
	if rawLevel != float64(int(math.Round(rawLevel))) {
		return nil, nil, errors.NewError(
			span,
			"Third argument of function 'log' has to be an integer",
			errors.TypeError,
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
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("Log level has to be one of 0, 1, 2, 3, 4, or 5 got %d", int(math.Round(rawLevel))),
			errors.ValueError,
		)
	}
	if err := executor.Log(title, description, level); err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueNull{}, nil, nil
}

// Launches a Homescript based on the provided script Id
// If no valid script could be found or the user lacks permission to execute it, an error is returned
func Exec(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	// Validate that at least one argument was provided
	if len(args) == 0 {
		return nil, nil, errors.NewError(
			span,
			"Function 'exec' takes 1 or more arguments but 0 were given",
			errors.TypeError,
		)
	}
	// Validate that the first argument is of type string
	if args[0].Type() != TypeString {
		return nil, nil, errors.NewError(
			span,
			"First argument of function 'exec' has to be of type String",
			errors.TypeError,
		)
	}
	// Create call arguments from other args
	callArgsFinal := make(map[string]string, 0)
	for indexArg, arg := range args[1:] {
		if arg.Type() != TypePair {
			return nil, nil, errors.NewError(
				span,
				fmt.Sprintf("Argument %d of function 'exec' has to be of type Pair\nhint: you can create a value pair using `pair('key', 'value')`", indexArg),
				errors.TypeError,
			)
		}
		_, alreadyExists := callArgsFinal[arg.(ValuePair).Key]
		if alreadyExists {
			return nil, nil, errors.NewError(
				span,
				fmt.Sprintf("Call argument (value pair) %d of function 'exec' has duplicate key entry '%s'", indexArg+2, arg.(ValuePair).Key),
				errors.TypeError,
			)
		}
		// Add the argument to the argument map
		value, err := arg.(ValuePair).Value.Display(executor, span)
		if err != nil {
			return nil, nil, err
		}
		callArgsFinal[arg.(ValuePair).Key] = value
	}
	// Execute Homescript
	homescriptId := args[0].(ValueString).Value
	output, err := executor.Exec(homescriptId, callArgsFinal)
	if err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueObject{
		Fields: map[string]Value{
			"output": ValueString{
				Value: output.Output,
			},
			"elapsed": ValueNumber{
				Value: output.RuntimeSecs,
			},
		},
	}, nil, nil
}

// Makes a get-request to an arbitrary url and returns the result HTTP response
func Get(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("get", span, args, TypeString); err != nil {
		return nil, nil, err
	}
	res, err := executor.Get(args[0].(ValueString).Value)
	if err != nil {
		return ValueNumber{}, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueObject{
		Fields: map[string]Value{
			"status": ValueString{
				Value: res.Status,
			},
			"status_code": ValueNumber{
				Value: float64(res.StatusCode),
			},
			"body": ValueString{
				Value: res.Body,
			},
		},
	}, nil, nil
}

// Makes a network request using an arbitrary URL, method , body (as plaintext), (and optionally headers)
func Http(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	// Validate that at least three arguments are provided
	if len(args) < 3 {
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("Function 'http' takes three or more arguments but %d were given", len(args)),
			errors.TypeError,
		)
	}
	// Validate that the first three arguments are of type string
	for argIndex, arg := range args {
		if arg.Type() != TypeString {
			return nil, nil, errors.NewError(
				span,
				fmt.Sprintf("%s argument of function 'http' has to be of type String", numberNames[argIndex]),
				errors.TypeError,
			)
		}
		if argIndex == 2 {
			break
		}
	}
	// Create header values from remaining args
	headers := make(map[string]string, 0)
	for headerIndex, header := range args[3:] {
		if header.Type() != TypePair {
			return nil, nil, errors.NewError(
				span,
				fmt.Sprintf("Argument %d of function 'http' has to be of type Pair.\nhint: you can create a value pair using `pair('key', 'value')`", headerIndex+4),
				errors.TypeError,
			)
		}
		_, alreadyExists := headers[header.(ValuePair).Key]
		if alreadyExists {
			return nil, nil, errors.NewError(
				span,
				fmt.Sprintf("Header entry (value pair) %d of function 'http' has duplicate key entry '%s'", headerIndex+4, header.(ValuePair).Key),
				errors.ValueError,
			)
		}
		// Add the argument to the argument map
		value, err := header.(ValuePair).Value.Display(executor, span)
		if err != nil {
			return nil, nil, err
		}

		headers[header.(ValuePair).Key] = value
	}
	res, err := executor.Http(
		args[0].(ValueString).Value,
		args[1].(ValueString).Value,
		args[2].(ValueString).Value,
		headers,
	)
	if err != nil {
		return ValueNull{}, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueObject{
		Fields: map[string]Value{
			"status": ValueString{
				Value: res.Status,
			},
			"status_code": ValueNumber{
				Value: float64(res.StatusCode),
			},
			"body": ValueString{
				Value: res.Body,
			},
		},
	}, nil, nil
}

// //////////// Variables //////////////
func GetUser(executor Executor, _ errors.Span) (Value, *errors.Error) {
	return ValueString{Value: executor.GetUser()}, nil
}

func GetWeather(executor Executor, span errors.Span) (Value, *errors.Error) {
	data, err := executor.GetWeather()
	if err != nil {
		return nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueObject{
		Fields: map[string]Value{
			"title": ValueString{
				Value: data.WeatherTitle,
			},
			"description": ValueString{
				Value: data.WeatherDescription,
			},
			"temperature": ValueNumber{
				Value: data.Temperature,
			},
			"feels_like": ValueNumber{
				Value: data.FeelsLike,
			},
			"humidity": ValueNumber{
				Value: float64(data.Humidity),
			},
		},
	}, nil
}

func GetTime(executor Executor, _ errors.Span) (Value, *errors.Error) {
	time := executor.GetTime()
	return ValueObject{
		Fields: map[string]Value{
			"year": ValueNumber{
				Value: float64(time.Year),
			},
			"month": ValueNumber{
				Value: float64(time.Minute),
			},
			"week": ValueNumber{
				Value: float64(time.CalendarWeek),
			},
			"week_day_text": ValueString{
				Value: time.WeekDayText,
			},
			"week_day": ValueNumber{
				Value: float64(time.WeekDay),
			},
			"calendar_day": ValueNumber{
				Value: float64(time.CalendarDay),
			},
			"hour": ValueNumber{
				Value: float64(time.Hour),
			},
			"minue": ValueNumber{
				Value: float64(time.Minute),
			},
			"second": ValueNumber{
				Value: float64(time.Second),
			},
		},
	}, nil
}
