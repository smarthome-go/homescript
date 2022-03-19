package homescript

import (
	"github.com/MikMuellerDev/homescript-dev/homescript/interpreter"
)

type Interpreter struct {
	StartNode Expressions
	Executor  interpreter.Executor
	Scope     map[string]interpreter.Value
}

func NewInterpreter(startNode Expressions, executor interpreter.Executor) Interpreter {
	scope := map[string]interpreter.Value{
		"exit":          interpreter.ValueFunction{Callback: interpreter.Exit},
		"sleep":         interpreter.ValueFunction{Callback: interpreter.Sleep},
		"print":         interpreter.ValueFunction{Callback: interpreter.Print},
		"switchOn":      interpreter.ValueFunction{Callback: interpreter.SwitchOn},
		"switch":        interpreter.ValueFunction{Callback: interpreter.Switch},
		"play":          interpreter.ValueFunction{Callback: interpreter.Play},
		"notify":        interpreter.ValueFunction{Callback: interpreter.Notify},
		"user":          interpreter.ValueVariable{Callback: interpreter.GetUser},
		"weather":       interpreter.ValueVariable{Callback: interpreter.GetWeather},
		"temperature":   interpreter.ValueVariable{Callback: interpreter.GetTemperature},
		"currentYear":   interpreter.ValueVariable{Callback: interpreter.GetCurrentYear},
		"currentMonth":  interpreter.ValueVariable{Callback: interpreter.GetCurrentMonth},
		"currentDay":    interpreter.ValueVariable{Callback: interpreter.GetCurrentDay},
		"currentHour":   interpreter.ValueVariable{Callback: interpreter.GetCurrentHour},
		"currentMinute": interpreter.ValueVariable{Callback: interpreter.GetCurrentMinute},
		"currentSecond": interpreter.ValueVariable{Callback: interpreter.GetCurrentSecond},
	}
	return Interpreter{
		StartNode: startNode,
		Executor:  executor,
		Scope:     scope,
	}
}

func (self *Interpreter) Run() error {
	_, err := self.visitExpressions(self.StartNode)
	return err
}

func (self *Interpreter) visitExpressions(node Expressions) (interpreter.Value, error) {
	var value interpreter.Value = interpreter.ValueVoid{}
	var err error
	for _, expr := range node {
		value, err = self.visitExpression(expr)
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

func (self *Interpreter) visitExpression(node Expression) (interpreter.Value, error) {
	base, err := self.visitAndExpr(node.Base)
	if err != nil {
		return nil, err
	}
	if len(node.Following) == 0 {
		return base, nil
	}
	truth, err := base.IsTrue(self.Executor)
	if err != nil {
		return nil, err
	}
	if truth {
		return interpreter.ValueBoolean{Value: true}, nil
	}
	for _, expression := range node.Following {
		other, errOther := self.visitAndExpr(expression)
		if errOther != nil {
			return nil, errOther
		}
		truth, err := other.IsTrue(self.Executor)
		if err != nil {
			return nil, err
		}
		if truth {
			return interpreter.ValueBoolean{Value: true}, nil
		}
	}
	return interpreter.ValueBoolean{Value: false}, nil
}

func (self *Interpreter) visitAndExpr(node AndExpr) (interpreter.Value, error) {
	base, err := self.visitEqExpr(node.Base)
	if err != nil {
		return nil, err
	}
	if len(node.Following) == 0 {
		return base, nil
	}
	truth, err := base.IsTrue(self.Executor)
	if err != nil {
		return nil, err
	}
	if !truth {
		return interpreter.ValueBoolean{Value: false}, nil
	}
	for _, expression := range node.Following {
		other, errOther := self.visitEqExpr(expression)
		if errOther != nil {
			return nil, errOther
		}
		truth, err := other.IsTrue(self.Executor)
		if err != nil {
			return nil, err
		}
		if !truth {
			return interpreter.ValueBoolean{Value: false}, nil
		}
	}
	return interpreter.ValueBoolean{Value: true}, nil
}

func (self *Interpreter) visitEqExpr(node EqExpr) (interpreter.Value, error) {
	base, err := self.visitRelExpr(node.Base)
	if err != nil {
		return nil, err
	}
	if node.Other == nil {
		return base, nil
	}
	other, errOther := self.visitRelExpr(node.Other.RelExpr)
	if errOther != nil {
		return nil, err
	}
	equal := base == other
	switch node.Other.TokenType {
	case Equal:
		return interpreter.ValueBoolean{
			Value: equal,
		}, nil
	case NotEqual:
		return interpreter.ValueBoolean{
			Value: !equal,
		}, nil
	default:
		panic("this can not happen")
	}
}

func (self *Interpreter) visitRelExpr(node RelExpr) (interpreter.Value, error) {
	
	return nil, nil
}
