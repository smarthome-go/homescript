package homescript

import (
	"fmt"
	"math"
	"time"

	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

var numberNames = []string{
	"first",
	"second",
	"third",
	"fourth",
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
			fmt.Sprintf("function '%s' takes %d argument%s but %d were given", name, len(types), s, len(args)),
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
		"first argument of function 'exit' has to be an integer",
		errors.TypeError,
	)
}

// Returns an intentional error
func Throw(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if len(args) != 1 {
		return nil, nil, errors.NewError(span, fmt.Sprintf("function 'throw' requires exactly 1 argument but %d were given", len(args)), errors.TypeError)
	}
	display, err := args[0].Display(executor, span)
	if err != nil {
		return nil, nil, err
	}
	return nil, nil, errors.NewError(
		span,
		display,
		errors.ThrowError,
	)
}

// Asserts that a statement is true, otherwise an error is returned
func Assert(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if len(args) != 1 {
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("function 'assert' takes 1 argument but %d were given", len(args)),
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
			fmt.Sprintf("assertion of %v value failed", args[0].Type()),
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

// Displays the given arguments as debug output
func Debug(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if len(args) == 0 {
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("function 'debug' requires at least 1 argument but %d were given", len(args)),
			errors.TypeError,
		)
	}
	for _, value := range args {
		debug, err := value.Debug(executor, span)
		if err != nil {
			return nil, nil, err
		}
		executor.Println(debug)
	}
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
	if err := executor.Print(msgs...); err != nil {
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("function 'print' failed: %s", err.Error()),
			errors.RuntimeError,
		)
	}
	return ValueNull{}, nil, nil
}

// Outputs a string with a newline at the end
func Println(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	msgs := make([]string, 0)
	for _, arg := range args {
		res, err := arg.Display(executor, span)
		if err != nil {
			return nil, nil, err
		}
		msgs = append(msgs, res)
	}
	if err := executor.Println(msgs...); err != nil {
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("function 'println' failed: %s", err.Error()),
			errors.RuntimeError,
		)
	}
	return ValueNull{}, nil, nil
}

// Retrieves data about a Smarthome switch
func GetSwitch(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("get_switch", span, args, TypeString); err != nil {
		return nil, nil, err
	}
	name := args[0].(ValueString).Value
	res, err := executor.GetSwitch(name)
	if err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}

	return ValueObject{
		ObjFields: map[string]*Value{
			"name":  valPtr(ValueString{Value: res.Name}),
			"power": valPtr(ValueBool{Value: res.Power}),
			"watts": valPtr(ValueNumber{Value: float64(res.Watts)}),
		},
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

// If a notification system is provided in the runtime environment, a notification is sent to the current user
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
			"third argument of function 'notify' has to be an integer",
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
			fmt.Sprintf("notification level has to be one of 1, 2, or 3, got %d", int(math.Round(rawLevel))),
			errors.ValueError,
		)
	}
	if err := executor.Notify(title, description, level); err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueNull{}, nil, nil
}

// Adds a new reminder to the current user's reminders.
func Remind(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("remind", span, args, TypeString, TypeString, TypeNumber, TypeObject); err != nil {
		return nil, nil, err
	}
	title := args[0].(ValueString).Value
	description := args[1].(ValueString).Value
	rawUrgency := args[2].(ValueNumber).Value

	var urgency ReminderUrgency
	switch rawUrgency {
	case 1:
		urgency = UrgencyLow
	case 2:
		urgency = UrgencyNormal
	case 3:
		urgency = UrgencyMedium
	case 4:
		urgency = UrgencyHigh
	case 5:
		urgency = UrgencyUrgent
	default:
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("reminder urgency has to be 0 < and < 6, got %d", int(math.Round(rawUrgency))),
			errors.ValueError,
		)
	}

	if rawUrgency != float64(int(math.Round(rawUrgency))) {
		return nil, nil, errors.NewError(
			span,
			"third argument of function 'remind' has to be an integer",
			errors.TypeError,
		)
	}

	date := args[3].(ValueObject)

	day, ok := date.ObjFields["day"]
	if !ok || day == nil || (*day).Type() != TypeNumber {
		return nil, nil, errors.NewError(
			date.Span(),
			"no field of type number named 'day' found on date object",
			errors.TypeError,
		)
	}

	month, ok := date.ObjFields["month"]
	if !ok || month == nil || (*month).Type() != TypeNumber {
		return nil, nil, errors.NewError(
			date.Span(),
			"no field of type number named 'month' found on date object",
			errors.TypeError,
		)
	}

	year, ok := date.ObjFields["year"]
	if !ok || year == nil || (*year).Type() != TypeNumber {
		return nil, nil, errors.NewError(
			date.Span(),
			"no field of type number named 'year' found on date object",
			errors.TypeError,
		)
	}

	goDate, valid := parseDate(int((*year).(ValueNumber).Value), int((*month).(ValueNumber).Value), int((*day).(ValueNumber).Value))
	if !valid {
		return nil, nil, errors.NewError(date.Span(), "date object contains invalid date", errors.RuntimeError)
	}
	id, err := executor.Remind(title, description, urgency, goDate)
	if err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}

	return ValueNumber{
		Value: float64(id),
		Range: span,
	}, nil, nil
}

func parseDate(year, month, day int) (time.Time, bool) {
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	y, m, d := t.Date()
	return t, y == year && int(m) == month && d == day
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
			"third argument of function 'log' has to be an integer",
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
			fmt.Sprintf("log level has to be one of 0, 1, 2, 3, 4, or 5 got %d", int(math.Round(rawLevel))),
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
			"function 'exec' takes 1 or more arguments but 0 were given",
			errors.TypeError,
		)
	}
	// Validate that the first argument is of type string
	if args[0].Type() != TypeString {
		return nil, nil, errors.NewError(
			span,
			"first argument of function 'exec' has to be of type String",
			errors.TypeError,
		)
	}
	// Create call arguments from other args
	callArgsFinal := make(map[string]string, 0)
	for indexArg, arg := range args[1:] {
		if arg.Type() != TypePair {
			return nil, nil, errors.NewError(
				span,
				fmt.Sprintf("argument %d of function 'exec' has to be of type Pair\nhint: you can create a value pair using `pair('key', 'value')`", indexArg),
				errors.TypeError,
			)
		}
		key, err := (*arg.(ValuePair).Key).Display(executor, span)
		if err != nil {
			return nil, nil, err
		}
		_, alreadyExists := callArgsFinal[key]
		if alreadyExists {
			key, err := (*arg.(ValuePair).Key).Display(executor, span)
			if err != nil {
				return nil, nil, err
			}
			return nil, nil, errors.NewError(
				span,
				fmt.Sprintf("call argument (value pair) %d of function 'exec' has duplicate key entry '%s'", indexArg+2, key),
				errors.TypeError,
			)
		}
		// Add the argument to the argument map
		value, err := (*arg.(ValuePair).Value).Display(executor, span)
		if err != nil {
			return nil, nil, err
		}
		callArgsFinal[key] = value
	}
	// Execute Homescript
	homescriptId := args[0].(ValueString).Value
	output, err := executor.Exec(homescriptId, callArgsFinal)
	if err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	if output.ReturnValue == nil {
		panic("return value is nil: please implement this correctly")
	}
	return ValueObject{
		ObjFields: map[string]*Value{
			"elapsed": valPtr(ValueNumber{
				Value: output.RuntimeSecs,
			}),
			"value": valPtr(output.ReturnValue),
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
		ObjFields: map[string]*Value{
			"status": valPtr(ValueString{
				Value: res.Status,
			}),
			"status_code": valPtr(ValueNumber{
				Value: float64(res.StatusCode),
			}),
			"body": valPtr(ValueString{
				Value: res.Body,
			}),
		},
	}, nil, nil
}

// Makes a network request using an arbitrary URL, method , body (as plaintext), (and optionally headers)
func Http(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	// Validate that at least three arguments are provided
	if len(args) < 3 {
		return nil, nil, errors.NewError(
			span,
			fmt.Sprintf("function 'http' takes three or more arguments but %d were given", len(args)),
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
				fmt.Sprintf("argument %d of function 'http' has to be of type pair", headerIndex+4),
				errors.TypeError,
			)
		}
		key, err := (*header.(ValuePair).Key).Display(executor, span)
		if err != nil {
			return nil, nil, err
		}
		_, alreadyExists := headers[key]
		if alreadyExists {
			key, err := (*header.(ValuePair).Key).Display(executor, span)
			if err != nil {
				return nil, nil, err
			}
			return nil, nil, errors.NewError(
				span,
				fmt.Sprintf("header entry (value pair) %d of function 'http' has duplicate key entry '%s'", headerIndex+4, key),
				errors.ValueError,
			)
		}
		// Add the argument to the argument map
		value, err := (*header.(ValuePair).Value).Display(executor, span)
		if err != nil {
			return nil, nil, err
		}

		headers[key] = value
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
		ObjFields: map[string]*Value{
			"status": valPtr(ValueString{
				Value: res.Status,
			}),
			"status_code": valPtr(ValueNumber{
				Value: float64(res.StatusCode),
			}),
			"body": valPtr(ValueString{
				Value: res.Body,
			}),
		},
	}, nil, nil
}

// Performs a ping given the target-ip and a maximum timeout
func Ping(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("ping", span, args, TypeString, TypeNumber); err != nil {
		return nil, nil, err
	}
	hostAlive, err := executor.Ping(args[0].(ValueString).Value, args[1].(ValueNumber).Value)
	if err != nil {
		return nil, nil, errors.NewError(span, err.Error(), errors.RuntimeError)
	}
	return ValueBool{Value: hostAlive}, nil, nil
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
		ObjFields: map[string]*Value{
			"title": valPtr(ValueString{
				Value: data.WeatherTitle,
			}),
			"description": valPtr(ValueString{
				Value: data.WeatherDescription,
			}),
			"temperature": valPtr(ValueNumber{
				Value: data.Temperature,
			}),
			"feels_like": valPtr(ValueNumber{
				Value: data.FeelsLike,
			}),
			"humidity": valPtr(ValueNumber{
				Value: float64(data.Humidity),
			}),
		},
	}, nil
}

func Time(executor Executor, _ errors.Span) (Value, *errors.Error) {
	return ValueObject{
		DataType: "time_module",
		ObjFields: map[string]*Value{
			"now": valPtr(ValueBuiltinFunction{
				Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
					return CreateDateObj(time.Now()), nil, nil
				},
			}),
			"since": valPtr(ValueBuiltinFunction{Callback: timeSince}),
			// Time Methods
			"add_days": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("time.add_days", span, args, TypeObject, TypeNumber); err != nil {
					return nil, nil, err
				}

				then, err := requireTimeArgFirst(span, args...)
				if err != nil {
					return nil, nil, err
				}

				days := args[1].(ValueNumber).Value
				if float64(int(days)) != days {
					return nil, nil, errors.NewError(span, "cannot use float number as integer argument", errors.TypeError)
				}

				added := then.AddDate(0, 0, int(days))

				return CreateDateObj(added), nil, nil
			}}),
			"add_hours": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("time.add_hours", span, args, TypeObject, TypeNumber); err != nil {
					return nil, nil, err
				}

				then, err := requireTimeArgFirst(span, args...)
				if err != nil {
					return nil, nil, err
				}

				hours := args[1].(ValueNumber).Value
				if float64(int(hours)) != hours {
					return nil, nil, errors.NewError(span, "cannot use float number as integer argument", errors.TypeError)
				}

				added := then.Local().Add(time.Hour * time.Duration(hours))

				return CreateDateObj(added), nil, nil
			}}),
			"add_minutes": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("time.add_minutes", span, args, TypeObject, TypeNumber); err != nil {
					return nil, nil, err
				}

				then, err := requireTimeArgFirst(span, args...)
				if err != nil {
					return nil, nil, err
				}

				minutes := args[1].(ValueNumber).Value
				if float64(int(minutes)) != minutes {
					return nil, nil, errors.NewError(span, "cannot use float number as integer argument", errors.TypeError)
				}

				added := then.Local().Add(time.Minute * time.Duration(minutes))

				return CreateDateObj(added), nil, nil
			}}),
			"sleep": valPtr(ValueBuiltinFunction{Callback: Sleep}),
		},
	}, nil
}

func CreateDateObj(tm time.Time) Value {
	_, week := tm.ISOWeek()
	return ValueObject{
		DataType: "time",
		ObjFields: map[string]*Value{
			"year": valPtr(ValueNumber{
				Value: float64(tm.Year()),
			}),
			"month": valPtr(ValueNumber{
				Value: float64(tm.Month()),
			}),
			"week": valPtr(ValueNumber{
				Value: float64(week),
			}),
			"week_day_text": valPtr(ValueString{
				Value: tm.Weekday().String(),
			}),
			"week_day": valPtr(ValueNumber{
				Value: float64(tm.Weekday()),
			}),
			"calendar_day": valPtr(ValueNumber{
				Value: float64(tm.Day()),
			}),
			"hour": valPtr(ValueNumber{
				Value: float64(tm.Hour()),
			}),
			"minute": valPtr(ValueNumber{
				Value: float64(tm.Minute()),
			}),
			"second": valPtr(ValueNumber{
				Value: float64(tm.Second()),
			}),
			"unix": valPtr(ValueNumber{
				Value: float64(tm.UnixMilli()),
			}),
		},
	}
}

func requireTimeArgFirst(span errors.Span, args ...Value) (time.Time, *errors.Error) {
	arg := args[0].(ValueObject)
	if arg.DataType != "time" {
		return time.Time{}, errors.NewError(
			span,
			fmt.Sprintf("function 'since' requires an object of type 'time', got '%s'", arg.DataType),
			errors.TypeError,
		)
	}
	millis, ok := arg.ObjFields["unix"]
	if !ok || millis == nil || (*millis).Type() != TypeNumber {
		return time.Time{}, errors.NewError(
			span,
			"no field of type number named 'unix' found on time object",
			errors.RuntimeError,
		)
	}
	return time.UnixMilli(int64((*millis).(ValueNumber).Value)), nil
}

func timeSince(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if err := checkArgs("time.since", span, args, TypeObject); err != nil {
		return nil, nil, err
	}
	then, err := requireTimeArgFirst(span, args...)
	if err != nil {
		return nil, nil, err
	}
	since := time.Since(then)
	return ValueObject{
		DataType: "duration",
		ObjFields: map[string]*Value{
			"millis":  valPtr(ValueNumber{Value: float64(since.Milliseconds())}),
			"seconds": valPtr(ValueNumber{Value: float64(since.Seconds())}),
			"minutes": valPtr(ValueNumber{Value: float64(since.Minutes())}),
			"hours":   valPtr(ValueNumber{Value: float64(since.Hours())}),
			"display": valPtr(ValueString{Value: fmt.Sprintf("%v", since)}),
		},
	}, nil, nil
}

func Storage(executor Executor, _ errors.Span) (Value, *errors.Error) {
	return ValueObject{
		DataType: "storage",
		ObjFields: map[string]*Value{
			"get": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if err := checkArgs("STORAGE.get", span, args, TypeString); err != nil {
					return nil, nil, err
				}
				key := args[0].(ValueString).Value
				value, err := executor.GetStorage(key)

				if err != nil {
					return nil, nil, errors.NewError(span, fmt.Sprintf("could not get entry from storage: %s", err.Error()), errors.RuntimeError)
				}

				if value != nil {
					return ValueString{Value: *value, Range: span}, nil, nil
				} else {
					return ValueNull{Range: span}, nil, nil
				}
			}}),
			"set": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				if len(args) != 2 {
					return nil, nil, errors.NewError(
						span,
						fmt.Sprintf("function 'STORAGE.set' takes 2 arguments but %d were given", len(args)),
						errors.TypeError,
					)
				}

				if args[0].Type() != TypeString {
					return nil, nil, errors.NewError(
						span,
						"First argument of function 'STORAGE.set' has to be of type string",
						errors.TypeError,
					)
				}

				key := args[0].(ValueString).Value
				value, err := args[1].Display(executor, span)
				if err != nil {
					return nil, nil, err
				}

				if err := executor.SetStorage(key, value); err != nil {
					return nil, nil, errors.NewError(span, fmt.Sprintf("could not set entry in storage: %s", err.Error()), errors.RuntimeError)
				}

				return ValueNull{Range: span}, nil, nil
			}}),
			"fields": valPtr(ValueBuiltinFunction{Callback: func(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
				return nil, nil, errors.NewError(span, "NOT IMPLEMENTED", errors.RuntimeError)
			}}),
		},
	}, nil
}

func Fmt(executor Executor, span errors.Span, args ...Value) (Value, *int, *errors.Error) {
	if len(args) < 2 {
		return nil, nil, errors.NewError(span, "Function `fmt` requires at least two arguments", errors.TypeError)
	}
	if args[0].Type() != TypeString {
		return nil, nil, errors.NewError(span, "First argument of function `fmt` must be of type string", errors.TypeError)
	}
	displays := make([]any, 0)

	for idx, arg := range args {
		if idx == 0 {
			continue
		}

		var out any

		switch arg.Type() {
		case TypeNull:
			out = "null"
		case TypeNumber:
			num := arg.(ValueNumber).Value
			if float64(int(num)) == num {
				out = int(num)
			} else {
				out = num
			}
		case TypeBoolean:
			out = arg.(ValueBool).Value
		case TypeString:
			out = arg.(ValueString).Value
		default:
			display, err := arg.Display(executor, span)
			if err != nil {
				return nil, nil, err
			}
			out = display
		}

		displays = append(displays, out)
	}

	out := fmt.Sprintf(args[0].(ValueString).Value, displays...)

	return ValueString{Value: out, Range: span}, nil, nil
}
