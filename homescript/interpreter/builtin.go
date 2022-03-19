package interpreter

import (
	"fmt"
	"time"
)

func Sleep(_ Executor, args ...Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Function 'sleep' takes 1 argument but %d were given", len(args))
	}
	if args[0].Type() != Number {
		return nil, fmt.Errorf("First argument of function 'sleep' has to be of type Number")
	}
	seconds := args[0].(ValueNumber).value
	time.Sleep(time.Second * time.Duration(seconds))
	return ValueVoid{}, nil
}

func Print(executor Executor, args ...Value) (Value, error) {
	msgs := make([]string, 0)
	for _, arg := range args {
		if arg.Type() == Variable {
			val, err := arg.(ValueVariable).callback(executor)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, val.ToString())
		} else {
			msgs = append(msgs, arg.ToString())
		}
	}
	executor.Print(msgs...)
	return ValueVoid{}, nil
}

func SwitchOn(executor Executor, args ...Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Function 'switchOn' takes 1 argument but %d were given", len(args))
	}
	if args[0].Type() != String {
		return nil, fmt.Errorf("First argument of function 'switchOn' has to be of type String")
	}
	name := args[0].(ValueString).value
	value, err := executor.SwitchOn(name)
	if err != nil {
		return nil, err
	}
	return ValueBoolean{
		value: value,
	}, nil
}

func Switch(executor Executor, args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Function 'switch' takes 2 arguments but %d were given", len(args))
	}
	if args[0].Type() != String {
		return nil, fmt.Errorf("First argument of function 'switch' has to be of type String")
	}
	if args[1].Type() != Boolean {
		return nil, fmt.Errorf("Second argument of function 'switch' has to be of type Boolean")
	}
	name := args[0].(ValueString).value
	on := args[1].(ValueBoolean).value
	err := executor.Switch(name, on)
	if err != nil {
		return nil, err
	}
	return ValueVoid{}, nil
}

func Play(executor Executor, args ...Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Function 'play' takes 2 arguments but %d were given", len(args))
	}
	if args[0].Type() != String {
		return nil, fmt.Errorf("First argument of function 'play' has to be of type String")
	}
	if args[1].Type() != String {
		return nil, fmt.Errorf("Second argument of function 'play' has to be of type String")
	}
	server := args[0].(ValueString).value
	mode := args[1].(ValueString).value
	err := executor.Play(server, mode)
	if err != nil {
		return nil, err
	}
	return ValueVoid{}, nil
}

func Notify(executor Executor, args ...Value) (Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("Function 'notify' takes 3 arguments but %d were given", len(args))
	}
	if args[0].Type() != String {
		return nil, fmt.Errorf("First argument of function 'notify' has to be of type String")
	}
	if args[1].Type() != String {
		return nil, fmt.Errorf("Second argument of function 'notify' has to be of type String")
	}
	if args[2].Type() != Number {
		return nil, fmt.Errorf("Third argument of function 'notify' has to be of type Number")
	}
	title := args[0].(ValueString).value
	description := args[1].(ValueString).value
	rawLevel := args[2].(ValueNumber).value
	var level NotificationLevel
	switch rawLevel {
	case 1:
		level = LevelInfo
	case 2:
		level = LevelWarn
	case 3:
		level = LevelError
	default:
		return nil, fmt.Errorf("Notification level has to be one of 1, 2, or 3, got %d", rawLevel)
	}
	err := executor.Notify(title, description, level)
	if err != nil {
		return nil, err
	}
	return ValueVoid{}, nil
}

////////////// Variables //////////////
func GetUser(executor Executor) (Value, error) {
	return ValueString{value: executor.GetUser()}, nil
}

func GetWeather(executor Executor) (Value, error) {
	val, err := executor.GetWeather()
	if err != nil {
		return nil, err
	}
	return ValueString{value: val}, nil
}

func GetTemperature(executor Executor) (Value, error) {
	val, err := executor.GetTemperature()
	if err != nil {
		return nil, err
	}
	return ValueNumber{value: val}, nil
}

func GetCurrentYear(executor Executor) (Value, error) {
	year, _, _, _, _, _ := executor.GetDate()
	return ValueNumber{value: year}, nil
}

func GetCurrentMonth(executor Executor) (Value, error) {
	_, month, _, _, _, _ := executor.GetDate()
	return ValueNumber{value: month}, nil
}

func GetCurrentDay(executor Executor) (Value, error) {
	_, _, day, _, _, _ := executor.GetDate()
	return ValueNumber{value: day}, nil
}

func GetCurrentHour(executor Executor) (Value, error) {
	_, _, _, hour, _, _ := executor.GetDate()
	return ValueNumber{value: hour}, nil
}

func GetCurrentMinute(executor Executor) (Value, error) {
	_, _, _, _, minute, _ := executor.GetDate()
	return ValueNumber{value: minute}, nil
}

func GetCurrentSecond(executor Executor) (Value, error) {
	_, _, _, _, _, second := executor.GetDate()
	return ValueNumber{value: second}, nil
}
