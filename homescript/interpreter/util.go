package interpreter

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

func (self *Interpreter) callFunc(span errors.Span, val value.Value, args []ast.AnalyzedCallArgument) (*value.Value, *value.Interrupt) {
	if self.callStackSize > self.callStackLimitSize {
		return nil, value.NewRuntimeErr(
			fmt.Sprintf("Maximum callstack size of %d was exceeded", self.callStackLimitSize),
			value.StackOverFlowErrorKind,
			span,
		)
	}

	// cast to correct function type
	switch val.Kind() {
	case value.FunctionValueKind:
		fn := val.(value.ValueFunction)

		argsOut := make(map[string]value.Value)
		for _, arg := range args {
			argVal, i := self.expression(arg.Expression)
			if i != nil {
				return nil, i
			}
			argsOut[arg.Name] = *argVal
		}

		// determine if the module must be switched
		var previousModule *string
		if fn.Module != self.currentModuleName {
			currModulePrev := self.currentModuleName
			previousModule = &currModulePrev
			self.switchModule(fn.Module)
		}

		self.callStackSize++
		self.pushScope()
		defer func() {
			self.popScope()
			self.callStackSize--
			if previousModule != nil {
				self.switchModule(*previousModule)
			}
		}()

		// Add singleton extractors
		for _, singleton := range fn.ExtractedSingletonParams {
			self.addVar(singleton.ParameterIdent, *self.getVar(singleton.SingletonIdent))
		}

		// Add arguments
		for key, val := range argsOut {
			self.addVar(key, val)
		}

		val, i := self.block(fn.Block, false)
		if i != nil {
			switch (*i).Kind() {
			case value.ReturnInterruptKind:
				ret := (*i).(value.ReturnInterrupt).ReturnValue
				return &ret, nil
			default:
				return nil, i
			}
		}
		return val, nil
	case value.ClosureValueKind:
		closure := val.(value.ValueClosure)

		// push a scope into the closure
		closure.Scopes = append(closure.Scopes, make(map[string]*value.Value))
		self.callStackSize++

		// use the closure's scopes as the scopes of the current module
		scopesPrev := self.currentModule.scopes
		// use the closure's scope here
		self.currentModule.scopes = closure.Scopes

		// TODO: copy all used variables BY VALUE
		// TODO: implement an analyzer step which detects all variables which the closure captures
		// TODO: verify that this really works
		// TODO: this can be done more efficiently

		defer func() {
			self.callStackSize--
			// pop the closure scope again
			closure.Scopes = closure.Scopes[:len(closure.Scopes)-1]
			// restore scopes
			self.currentModule.scopes = scopesPrev
		}()

		for _, arg := range args {
			argVal, i := self.expression(arg.Expression)
			if i != nil {
				return nil, i
			}

			closure.Scopes[len(closure.Scopes)-1][arg.Name] = argVal
		}

		val, i := self.block(closure.Block, false)
		if i != nil {
			if (*i).Kind() != value.ReturnInterruptKind {
				// this is an error or a terminating interrupt
				return nil, i
			}
			ret := (*i).(value.ReturnInterrupt).ReturnValue
			return &ret, nil
		}
		return val, nil
	case value.BuiltinFunctionValueKind:
		self.callStackSize++
		defer func() {
			self.callStackSize--
		}()

		fn := val.(value.ValueBuiltinFunction)
		newArgs := make([]value.Value, 0)
		for _, arg := range args {
			argVal, i := self.expression(arg.Expression)
			if i != nil {
				return nil, i
			}

			newArgs = append(newArgs, *argVal)
		}
		// the cancel context of the interpreter is temporarely handed over
		// so that expensive / long-running builtin functions (like sleep) can terminate themselves
		return fn.Callback(self.Executor, self.cancelCtx, span, newArgs...)
	default:
		panic(fmt.Sprintf("A new callable value (%v) was introduced without updating this code", val.Kind()))
	}
}

func (self *Interpreter) block(node ast.AnalyzedBlock, handleScoping bool) (*value.Value, *value.Interrupt) {
	if handleScoping {
		self.pushScope()
		defer self.popScope()
	}

	for _, statement := range node.Statements {
		if interrupt := self.statement(statement); interrupt != nil {
			return nil, interrupt
		}
	}

	if node.Expression != nil {
		return self.expression(node.Expression)
	}

	return value.NewValueNull(), nil
}
