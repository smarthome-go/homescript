package homescript

import (
	"fmt"

	"github.com/MikMuellerDev/homescript/homescript/error"
	"github.com/MikMuellerDev/homescript/homescript/interpreter"
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
		"log":           interpreter.ValueFunction{Callback: interpreter.Log},
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

func (self *Interpreter) Run() (int, *error.Error) {
	_, err := self.visitExpressions(self.StartNode)
	return 0, err
}

func (self *Interpreter) visitExpressions(node Expressions) (interpreter.Value, *error.Error) {
	var value interpreter.Value = interpreter.ValueVoid{}
	var err *error.Error
	for _, expr := range node {
		value, err = self.visitExpression(expr)
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

func (self *Interpreter) visitExpression(node Expression) (interpreter.Value, *error.Error) {
	base, err := self.visitAndExpr(node.Base)
	if err != nil {
		return nil, err
	}
	if len(node.Following) == 0 {
		return base, nil
	}
	truth, err := base.IsTrue(self.Executor, node.Base.Base.Base.Base.Location)
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
		truth, err := other.IsTrue(self.Executor, node.Base.Base.Base.Base.Location)
		if err != nil {
			return nil, err
		}
		if truth {
			return interpreter.ValueBoolean{Value: true}, nil
		}
	}
	return interpreter.ValueBoolean{Value: false}, nil
}

func (self *Interpreter) visitAndExpr(node AndExpr) (interpreter.Value, *error.Error) {
	base, err := self.visitEqExpr(node.Base)
	if err != nil {
		return nil, err
	}
	if len(node.Following) == 0 {
		return base, nil
	}
	truth, err := base.IsTrue(self.Executor, node.Base.Base.Base.Location)
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
		truth, err := other.IsTrue(self.Executor, node.Base.Base.Base.Location)
		if err != nil {
			return nil, err
		}
		if !truth {
			return interpreter.ValueBoolean{Value: false}, nil
		}
	}
	return interpreter.ValueBoolean{Value: true}, nil
}

func (self *Interpreter) visitEqExpr(node EqExpr) (interpreter.Value, *error.Error) {
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
	equal, err := base.IsEqual(self.Executor, node.Base.Base.Location, other)
	if err != nil {
		return nil, err
	}
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
		panic("unreachable")
	}
}

func (self *Interpreter) visitRelExpr(node RelExpr) (interpreter.Value, *error.Error) {
	base, err := self.visitNotExpr(node.Base)
	if err != nil {
		return nil, err
	}
	if node.Other == nil {
		return base, nil
	}
	other, err := self.visitNotExpr(node.Other.NotExpr)
	if err != nil {
		return nil, err
	}
	var leftSide interpreter.ValueRelational
	if base.Type() == interpreter.Number {
		leftSide = base.(interpreter.ValueNumber)
	} else if base.Type() == interpreter.Variable {
		leftSide = base.(interpreter.ValueVariable)
	} else {
		return nil, error.NewError(
			error.TypeError,
			node.Base.Location,
			fmt.Sprintf("Cannot compare %s type with %s type", base.TypeName(), other.TypeName()),
		)
	}
	var truth bool
	var errTruth *error.Error
	switch node.Other.TokenType {
	case GreaterThan:
		truth, errTruth = leftSide.IsGreaterThan(self.Executor, other, node.Base.Location)
	case GreaterThanOrEqual:
		truth, errTruth = leftSide.IsGreaterThanOrEqual(self.Executor, other, node.Base.Location)
	case LessThan:
		truth, errTruth = leftSide.IsLessThan(self.Executor, other, node.Base.Location)
	case LessThanOrEqual:
		truth, errTruth = leftSide.IsLessThanOrEqual(self.Executor, other, node.Base.Location)
	default:
		panic("unreachable")
	}
	if errTruth != nil {
		return nil, errTruth
	}
	return interpreter.ValueBoolean{
		Value: truth,
	}, nil
}

func (self *Interpreter) visitNotExpr(node NotExpr) (interpreter.Value, *error.Error) {
	base, err := self.visitAtom(node.Base)
	if err != nil {
		return nil, err
	}
	if node.Negated {
		truth, err := base.IsTrue(self.Executor, node.Location)
		if err != nil {
			return nil, err
		}
		return interpreter.ValueBoolean{
			Value: !truth,
		}, nil
	}
	return base, nil
}

func (self *Interpreter) visitAtom(node Atom) (interpreter.Value, *error.Error) {
	var result interpreter.Value
	var err *error.Error
	switch node.Kind() {
	case AtomNumberKind:
		result = interpreter.ValueNumber{
			Value: node.(AtomNumber).Num,
		}
	case AtomStringKind:
		result = interpreter.ValueString{
			Value: node.(AtomString).Content,
		}
	case AtomBooleanKind:
		result = interpreter.ValueBoolean{
			Value: node.(AtomBoolean).Value,
		}
	case AtomIdentifierKind:
		name := node.(AtomIdentifier).Name
		value, exists := self.Scope[name]
		if !exists {
			return nil, error.NewError(
				error.ReferenceError,
				node.(AtomIdentifier).Location,
				fmt.Sprintf("Variable or function '%s' does not exists", name),
			)
		}
		if value.Type() == interpreter.Variable {
			result, err = value.(interpreter.ValueVariable).Callback(self.Executor, node.(AtomIdentifier).Location)
		} else {
			result = value
		}
	case AtomIfKind:
		result, err = self.visitIfExpr(node.(AtomIf).IfExpr)
	case AtomCallKind:
		result, err = self.visitCallExpr(node.(AtomCall).CallExpr)
	case AtomExpressionKind:
		result, err = self.visitExpression(node.(AtomExpr).Expr)
	}
	return result, err
}

func (self *Interpreter) visitIfExpr(node IfExpr) (interpreter.Value, *error.Error) {
	condition, err := self.visitExpression(node.Condition)
	if err != nil {
		return nil, err
	}
	truth, errTruth := condition.IsTrue(self.Executor, node.Location)
	if errTruth != nil {
		return nil, errTruth
	}
	if truth {
		return self.visitExpressions(node.Body)
	}
	if node.ElseBody == nil {
		return interpreter.ValueVoid{}, nil
	}
	return self.visitExpressions(node.ElseBody)
}

func (self *Interpreter) visitCallExpr(node CallExpr) (interpreter.Value, *error.Error) {
	value, exists := self.Scope[node.Name]
	if !exists {
		return nil, error.NewError(
			error.ReferenceError,
			node.Location,
			fmt.Sprintf("Variable or function '%s' does not exists", node.Name),
		)
	}
	if value.Type() != interpreter.Function {
		return nil, error.NewError(
			error.TypeError,
			node.Location,
			fmt.Sprintf("Type %s is not callable", value.TypeName()),
		)
	}
	arguments := make([]interpreter.Value, 0)
	for _, argument := range node.Arguments {
		val, err := self.visitExpression(argument)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, val)
	}
	return value.(interpreter.ValueFunction).Callback(self.Executor, node.Location, arguments...)
}
