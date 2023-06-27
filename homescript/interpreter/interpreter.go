package interpreter

import (
	"context"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

type Interpreter struct {
	callStackLimitSize uint
	scopeAdditions     map[string]value.Value
	sourceModules      map[string]ast.AnalyzedProgram
	executor           value.Executor
	modules            map[string]*Module
	currentModule      *Module
	currentModuleName  string
	callStackSize      uint
	cancelCtx          *context.Context
}

type Module struct {
	scopes []map[string]*value.Value
}

func NewInterpreter(
	callStackLimitSize uint,
	executor value.Executor,
	sourceModules map[string]ast.AnalyzedProgram,
	scopeAdditions map[string]value.Value,
	cancelCtx *context.Context,
) Interpreter {
	scopeAdditions["throw"] = *value.NewValueBuiltinFunction(func(executor value.Executor, cancelCtx *context.Context, span errors.Span, args ...value.Value) (*value.Value, *value.Interrupt) {
		message, i := args[0].Display()
		if i != nil {
			return nil, i
		}
		return nil, value.NewThrowInterrupt(span, message)
	})

	return Interpreter{
		callStackLimitSize: callStackLimitSize,
		scopeAdditions:     scopeAdditions,
		sourceModules:      sourceModules,
		executor:           executor,
		modules:            make(map[string]*Module),
		currentModule:      nil,
		currentModuleName:  "",
		callStackSize:      0,
		cancelCtx:          cancelCtx,
	}
}

func (self *Interpreter) Execute(entryModule string) *value.Interrupt {
	if err := self.execModule(entryModule, false); err != nil {
		return err
	}

	_, i := self.callFunc(errors.Span{}, *self.currentModule.scopes[0]["main"], make([]ast.AnalyzedCallArgument, 0))
	if i != nil {
		if (*i).Kind() == value.ThrowInterruptKind {
			throw := (*i).(value.ThrowInterrupt)
			return value.NewRuntimeErr(throw.Message(), value.UncaughtThrowKind, throw.Span)
		}
	}
	return i
}

func (self *Interpreter) checkCancelation(span errors.Span) *value.Interrupt {
	select {
	case <-(*self.cancelCtx).Done():
		return value.NewTerminationInterrupt(context.Cause((*self.cancelCtx)).Error(), span)
	default:
		// do nothing, this should not block the entire interpreter
		return nil
	}
}
