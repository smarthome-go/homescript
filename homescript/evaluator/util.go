package evaluator

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
	"github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

func (i *Interpreter) ident(ident ast.SpannedIdent) (*value.Value, *value.Interrupt) {
	const errMsg = "Value or function `%s` was not found"

	val, found := i.currModule.scopes[i.currModule.currentScope][ident.Ident()]
	if !found {
		return nil, value.NewRuntimeErr(
			fmt.Sprintf(errMsg, ident),
			value.UncaughtThrowKind,
			ident.Span(),
		)
	} else {
		return val, nil
	}

	fn, found := i.currModule.functions[ident.Ident()]
	if !found {
		return nil, value.NewRuntimeErr(
			fmt.Sprintf(errMsg, ident),
			value.UncaughtThrowKind,
			ident.Span(),
		)
	}

	return value.NewValueFunction(
		i.currModule.filename,
		fn.Body,
		nil, // TODO: this might be broken
	), nil

}

func (i *Interpreter) pushScope() {
	i.currModule.currentScope++
	if int(i.currModule.currentScope) >= len(i.currModule.scopes) {
		i.currModule.scopes = append(i.currModule.scopes, make(map[string]*value.Value))
	}
}

func (i *Interpreter) popScope() {
	if i.currModule.currentScope == 0 {
		panic("Tried popping last scope")
	}
	i.currModule.currentScope--
}

func (i *Interpreter) addVar(ident string, val value.Value) {
	i.currModule.scopes[i.currModule.currentScope][ident] = &val
}
