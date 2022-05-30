package homescript

import (
	"fmt"

	"github.com/smarthome-go/homescript/homescript/error"
	"github.com/smarthome-go/homescript/homescript/interpreter"
)

type Interpreter struct {
	StartNode Expressions
	Executor  interpreter.Executor
	Scope     map[string]interpreter.Value
}

func NewInterpreter(startNode Expressions, executor interpreter.Executor) Interpreter {
	scope := map[string]interpreter.Value{
		// special case `exit` implemented below
		"exit":          interpreter.ValueFunction{},
		"num":           interpreter.ValueFunction{Callback: interpreter.Num},
		"str":           interpreter.ValueFunction{Callback: interpreter.Str},
		"mkArg":         interpreter.ValueFunction{Callback: interpreter.MkArg},
		"checkArg":      interpreter.ValueFunction{Callback: interpreter.CheckArg},
		"getArg":        interpreter.ValueFunction{Callback: interpreter.GetArg},
		"sleep":         interpreter.ValueFunction{Callback: interpreter.Sleep},
		"print":         interpreter.ValueFunction{Callback: interpreter.Print},
		"switchOn":      interpreter.ValueFunction{Callback: interpreter.SwitchOn},
		"switch":        interpreter.ValueFunction{Callback: interpreter.Switch},
		"play":          interpreter.ValueFunction{Callback: interpreter.Play},
		"notify":        interpreter.ValueFunction{Callback: interpreter.Notify},
		"log":           interpreter.ValueFunction{Callback: interpreter.Log},
		"exec":          interpreter.ValueFunction{Callback: interpreter.Exec},
		"addUser":       interpreter.ValueFunction{Callback: interpreter.AddUser},
		"delUser":       interpreter.ValueFunction{Callback: interpreter.DelUser},
		"addPerm":       interpreter.ValueFunction{Callback: interpreter.AddPerm},
		"delPerm":       interpreter.ValueFunction{Callback: interpreter.DelPerm},
		"get":           interpreter.ValueFunction{Callback: interpreter.Get},
		"http":          interpreter.ValueFunction{Callback: interpreter.Http},
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
	_, err, code := self.visitExpressions(self.StartNode)
	if code == nil {
		return 0, err
	}
	return *code, err
}

func (self *Interpreter) visitExpressions(node Expressions) (interpreter.Value, *error.Error, *int) {
	var value interpreter.Value = interpreter.ValueVoid{}
	var err *error.Error
	var code *int
	for _, expr := range node {
		value, err, code = self.visitExpression(expr)
		if err != nil || code != nil {
			return nil, err, code
		}
	}
	return value, nil, nil
}

func (self *Interpreter) visitExpression(node Expression) (interpreter.Value, *error.Error, *int) {
	base, err, code := self.visitAndExpr(node.Base)
	if err != nil || code != nil {
		return nil, err, code
	}
	if len(node.Following) == 0 {
		return base, nil, nil
	}
	truth, err := base.IsTrue(self.Executor, node.Base.Base.Base.Base.Location)
	if err != nil {
		return nil, err, nil
	}
	if truth {
		return interpreter.ValueBoolean{Value: true}, nil, nil
	}
	for _, expression := range node.Following {
		other, err, code := self.visitAndExpr(expression)
		if err != nil || code != nil {
			return nil, err, code
		}
		truth, err := other.IsTrue(self.Executor, node.Base.Base.Base.Base.Location)
		if err != nil {
			return nil, err, nil
		}
		if truth {
			return interpreter.ValueBoolean{Value: true}, nil, nil
		}
	}
	return interpreter.ValueBoolean{Value: false}, nil, nil
}

func (self *Interpreter) visitAndExpr(node AndExpr) (interpreter.Value, *error.Error, *int) {
	base, err, code := self.visitEqExpr(node.Base)
	if err != nil || code != nil {
		return nil, err, code
	}
	if len(node.Following) == 0 {
		return base, nil, nil
	}
	truth, err := base.IsTrue(self.Executor, node.Base.Base.Base.Location)
	if err != nil {
		return nil, err, nil
	}
	if !truth {
		return interpreter.ValueBoolean{Value: false}, nil, nil
	}
	for _, expression := range node.Following {
		other, err, code := self.visitEqExpr(expression)
		if err != nil || code != nil {
			return nil, err, code
		}
		truth, err := other.IsTrue(self.Executor, node.Base.Base.Base.Location)
		if err != nil {
			return nil, err, nil
		}
		if !truth {
			return interpreter.ValueBoolean{Value: false}, nil, nil
		}
	}
	return interpreter.ValueBoolean{Value: true}, nil, nil
}

func (self *Interpreter) visitEqExpr(node EqExpr) (interpreter.Value, *error.Error, *int) {
	base, err, code := self.visitRelExpr(node.Base)
	if err != nil || code != nil {
		return nil, err, code
	}
	if node.Other == nil {
		return base, nil, nil
	}
	other, err, code := self.visitRelExpr(node.Other.RelExpr)
	if err != nil || code != nil {
		return nil, err, code
	}
	equal, err := base.IsEqual(self.Executor, node.Base.Base.Location, other)
	if err != nil {
		return nil, err, nil
	}
	switch node.Other.TokenType {
	case Equal:
		return interpreter.ValueBoolean{
			Value: equal,
		}, nil, nil
	case NotEqual:
		return interpreter.ValueBoolean{
			Value: !equal,
		}, nil, nil
	default:
		panic("unreachable")
	}
}

func (self *Interpreter) visitRelExpr(node RelExpr) (interpreter.Value, *error.Error, *int) {
	base, err, code := self.visitNotExpr(node.Base)
	if err != nil || code != nil {
		return nil, err, code
	}
	if node.Other == nil {
		return base, nil, nil
	}
	other, err, code := self.visitNotExpr(node.Other.NotExpr)
	if err != nil || code != nil {
		return nil, err, code
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
			fmt.Sprintf("Cannot compare %s type with %s type", base.Type().Name(), other.Type().Name()),
		), nil
	}
	var truth bool
	switch node.Other.TokenType {
	case GreaterThan:
		truth, err = leftSide.IsGreaterThan(self.Executor, other, node.Base.Location)
	case GreaterThanOrEqual:
		truth, err = leftSide.IsGreaterThanOrEqual(self.Executor, other, node.Base.Location)
	case LessThan:
		truth, err = leftSide.IsLessThan(self.Executor, other, node.Base.Location)
	case LessThanOrEqual:
		truth, err = leftSide.IsLessThanOrEqual(self.Executor, other, node.Base.Location)
	default:
		panic("unreachable")
	}
	if err != nil {
		return nil, err, nil
	}
	return interpreter.ValueBoolean{
		Value: truth,
	}, nil, nil
}

func (self *Interpreter) visitNotExpr(node NotExpr) (interpreter.Value, *error.Error, *int) {
	base, err, code := self.visitAtom(node.Base)
	if err != nil || code != nil {
		return nil, err, code
	}
	if node.Negated {
		truth, err := base.IsTrue(self.Executor, node.Location)
		if err != nil {
			return nil, err, nil
		}
		return interpreter.ValueBoolean{
			Value: !truth,
		}, nil, nil
	}
	return base, nil, nil
}

func (self *Interpreter) visitAtom(node Atom) (interpreter.Value, *error.Error, *int) {
	var result interpreter.Value
	var err *error.Error
	var code *int
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
			), nil
		}
		if value.Type() == interpreter.Variable {
			result, err = value.(interpreter.ValueVariable).Callback(self.Executor, node.(AtomIdentifier).Location)
		} else {
			result = value
		}
	case AtomIfKind:
		result, err, code = self.visitIfExpr(node.(AtomIf).IfExpr)
	case AtomCallKind:
		result, err, code = self.visitCallExpr(node.(AtomCall).CallExpr)
	case AtomExpressionKind:
		result, err, code = self.visitExpression(node.(AtomExpr).Expr)
	}
	return result, err, code
}

func (self *Interpreter) visitIfExpr(node IfExpr) (interpreter.Value, *error.Error, *int) {
	condition, err, code := self.visitExpression(node.Condition)
	if err != nil || code != nil {
		return nil, err, code
	}
	truth, err := condition.IsTrue(self.Executor, node.Location)
	if err != nil {
		return nil, err, nil
	}
	if truth {
		return self.visitExpressions(node.Body)
	}
	if node.ElseBody == nil {
		return interpreter.ValueVoid{}, nil, nil
	}
	return self.visitExpressions(node.ElseBody)
}

func (self *Interpreter) visitCallExpr(node CallExpr) (interpreter.Value, *error.Error, *int) {
	value, exists := self.Scope[node.Name]
	if !exists {
		return nil, error.NewError(
			error.ReferenceError,
			node.Location,
			fmt.Sprintf("Variable or function '%s' does not exists", node.Name),
		), nil
	}
	if value.Type() != interpreter.Function {
		return nil, error.NewError(
			error.TypeError,
			node.Location,
			fmt.Sprintf("Type %s is not callable", value.Type().Name()),
		), nil
	}
	arguments := make([]interpreter.Value, 0)
	for _, argument := range node.Arguments {
		val, err, code := self.visitExpression(argument)
		if err != nil || code != nil {
			return nil, err, code
		}
		arguments = append(arguments, val)
	}
	if node.Name == "exit" {
		err, code := interpreter.Exit(node.Location, arguments...)
		return interpreter.ValueVoid{}, err, code
	}
	val, err := value.(interpreter.ValueFunction).Callback(self.Executor, node.Location, arguments...)
	return val, err, nil
}
