package interpreter

import (
	"fmt"
	"time"

	"github.com/MikMuellerDev/homescript/homescript/error"
)

func Exit(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if len(args) != 1 {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Function 'exit' takes 1 argument but %d were given", len(args)),
		)
	}
	if args[0].Type() != Number {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("First argument of function 'exit' has to be of type Number"),
		)
	}
	executor.Exit(args[0].(ValueNumber).Value)
	return ValueVoid{}, nil
}

func Sleep(_ Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if len(args) != 1 {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Function 'sleep' takes 1 argument but %d were given", len(args)),
		)
	}
	if args[0].Type() != Number {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("First argument of function 'sleep' has to be of type Number"),
		)
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
	if len(args) != 1 {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Function 'switchOn' takes 1 argument but %d were given", len(args)),
		)
	}
	if args[0].Type() != String {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("First argument of function 'switchOn' has to be of type String"),
		)
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

func Switch(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if len(args) != 2 {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Function 'switch' takes 2 arguments but %d were given", len(args)),
		)
	}
	if args[0].Type() != String {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("First argument of function 'switch' has to be of type String"),
		)
	}
	if args[1].Type() != Boolean {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Second argument of function 'switch' has to be of type Boolean"),
		)
	}
	name := args[0].(ValueString).Value
	on := args[1].(ValueBoolean).Value
	err := executor.Switch(name, on)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

func Play(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if len(args) != 2 {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Function 'play' takes 2 arguments but %d were given", len(args)),
		)
	}
	if args[0].Type() != String {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("First argument of function 'play' has to be of type String"),
		)
	}
	if args[1].Type() != String {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Second argument of function 'play' has to be of type String"),
		)
	}
	server := args[0].(ValueString).Value
	mode := args[1].(ValueString).Value
	err := executor.Play(server, mode)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

func Notify(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if len(args) != 3 {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Function 'notify' takes 3 arguments but %d were given", len(args)),
		)
	}
	if args[0].Type() != String {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("First argument of function 'notify' has to be of type String"),
		)
	}
	if args[1].Type() != String {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Second argument of function 'notify' has to be of type String"),
		)
	}
	if args[2].Type() != Number {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Third argument of function 'notify' has to be of type Number"),
		)
	}
	title := args[0].(ValueString).Value
	description := args[1].(ValueString).Value
	rawLevel := args[2].(ValueNumber).Value
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
			fmt.Sprintf("Notification level has to be one of 1, 2, or 3, got %d", rawLevel),
		)
	}
	err := executor.Notify(title, description, level)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
	}
	return ValueVoid{}, nil
}

func Log(executor Executor, location error.Location, args ...Value) (Value, *error.Error) {
	if len(args) != 3 {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Function 'log' takes 3 arguments but %d were given", len(args)),
		)
	}
	if args[0].Type() != String {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("First argument of function 'log' has to be of type String"),
		)
	}
	if args[1].Type() != String {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Second argument of function 'log' has to be of type String"),
		)
	}
	if args[2].Type() != Number {
		return nil, error.NewError(
			error.TypeError,
			location,
			fmt.Sprintf("Third argument of function 'log' has to be of type Number"),
		)
	}
	title := args[0].(ValueString).Value
	description := args[1].(ValueString).Value
	rawLevel := args[2].(ValueNumber).Value
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
			fmt.Sprintf("Log level has to be one of 0, 1, 2, 3, 4, or 5 got %d", rawLevel),
		)
	}
	err := executor.Log(title, description, level)
	if err != nil {
		return nil, error.NewError(error.RuntimeError, location, err.Error())
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
	return ValueNumber{Value: val}, nil
}

func GetCurrentYear(executor Executor, _ error.Location) (Value, *error.Error) {
	year, _, _, _, _, _ := executor.GetDate()
	return ValueNumber{Value: year}, nil
}

func GetCurrentMonth(executor Executor, _ error.Location) (Value, *error.Error) {
	_, month, _, _, _, _ := executor.GetDate()
	return ValueNumber{Value: month}, nil
}

func GetCurrentDay(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, day, _, _, _ := executor.GetDate()
	return ValueNumber{Value: day}, nil
}

func GetCurrentHour(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, hour, _, _ := executor.GetDate()
	return ValueNumber{Value: hour}, nil
}

func GetCurrentMinute(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, _, minute, _ := executor.GetDate()
	return ValueNumber{Value: minute}, nil
}

func GetCurrentSecond(executor Executor, _ error.Location) (Value, *error.Error) {
	_, _, _, _, _, second := executor.GetDate()
	return ValueNumber{Value: second}, nil
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
