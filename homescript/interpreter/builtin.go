package interpreter

import (
	"fmt"
	"math"
	"time"

	"github.com/MikMuellerDev/homescript/homescript/error"
)

var numberNames = [...]string{
	"First",
	"Second",
	"Third",
}

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

func Exit(location error.Location, args ...Value) (*error.Error, *int) {
	err := checkArgs("exit", location, args, Number)
	if err != nil {
		return err, nil
	}
	code := args[0].(ValueNumber).Value
	if code == float64(int(math.Round(code))) {
		code := int(math.Round(code))
		return nil, &code
	}
	return error.NewError(
		error.ValueError,
		location,
		"First argument of function 'exit' has to be an integer",
	), nil
}

func Sleep(_ Executor, location error.Location, args ...Value) (Value, *error.Error) {
	err := checkArgs("sleep", location, args, Number)
	if err != nil {
		return nil, err
	}
	seconds := args[0].(ValueNumber).Value
	time.Sleep(time.Second * time.Duration(seconds))
	return ValueVoid{}, nil
}

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

func SwitchOn(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	err := checkArgs("switchOn", location, args, String)
	if err != nil {
		return nil, err
	}
	name := args[0].(ValueString).Value
	value, execErr := executor.SwitchOn(name)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, execErr.Error())
	}
	return ValueBoolean{
		Value: value,
	}, nil
}

func Switch(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	err := checkArgs("switch", location, args, String, Boolean)
	if err != nil {
		return nil, err
	}
	name := args[0].(ValueString).Value
	on := args[1].(ValueBoolean).Value
	execErr := executor.Switch(name, on)
	if execErr != nil {
		return nil, error.NewError(error.RuntimeError, location, execErr.Error())
	}
	return ValueVoid{}, nil
}

func Play(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	err := checkArgs("play", location, args, String, String)
	if err != nil {
		return nil, err
	}
	server := args[0].(ValueString).Value
	mode := args[1].(ValueString).Value
	execErr := executor.Play(server, mode)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, execErr.Error())
	}
	return ValueVoid{}, nil
}

func Notify(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	err := checkArgs("notify", location, args, String, String, Number)
	if err != nil {
		return nil, err
	}
	title := args[0].(ValueString).Value
	description := args[1].(ValueString).Value
	rawLevel := args[2].(ValueNumber).Value
	if rawLevel != float64(int(math.Round(rawLevel))) {
		return nil, error.NewError(
			error.ValueError,
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
	execErr := executor.Notify(title, description, level)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, execErr.Error())
	}
	return ValueVoid{}, nil
}

func Log(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	err := checkArgs("log", location, args, String, String, Number)
	if err != nil {
		return nil, err
	}
	title := args[0].(ValueString).Value
	description := args[1].(ValueString).Value
	rawLevel := args[2].(ValueNumber).Value
	if rawLevel != float64(int(math.Round(rawLevel))) {
		return nil, error.NewError(
			error.ValueError,
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
	execErr := executor.Log(title, description, level)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, execErr.Error())
	}
	return ValueVoid{}, nil
}

////////////// Variables //////////////
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
	year, _, _, _, _, _ := executor.GetDate()
	return ValueNumber{Value: float64(year)}, nil
}

func GetCurrentMonth(executor Executor, _ error.Location) (Value, *error.Error) {
	_, month, _, _, _, _ := executor.GetDate()
	return ValueNumber{Value: float64(month)}, nil
}

func GetCurrentDay(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, day, _, _, _ := executor.GetDate()
	return ValueNumber{Value: float64(day)}, nil
}

func GetCurrentHour(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, hour, _, _ := executor.GetDate()
	return ValueNumber{Value: float64(hour)}, nil
}

func GetCurrentMinute(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, _, minute, _ := executor.GetDate()
	return ValueNumber{Value: float64(minute)}, nil
}

func GetCurrentSecond(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, _, _, second := executor.GetDate()
	return ValueNumber{Value: float64(second)}, nil
}

func GetDebugInfo(executor Executor, location error.Location) (Value, *error.Error) {
	value, err := executor.GetDebugInfo()
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueString{
		Value: value,
	}, nil
}
