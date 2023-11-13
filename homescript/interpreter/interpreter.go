package interpreter

import (
	"context"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
)

// after this period of time, KILL is sent regardless of whether the cleanup task is finished or not
const KILL_EVENT_TIMEOUT_SECS = 10

type Interpreter struct {
	callStackLimitSize uint
	scopeAdditions     map[string]value.Value
	sourceModules      map[string]ast.AnalyzedProgram
	Executor           value.Executor
	modules            map[string]*Module
	currentModule      *Module
	currentModuleName  string
	callStackSize      uint
	cancelCtx          *context.Context
}

type Module struct {
	scopes []map[string]*value.Value
	events map[string]value.ValueFunction
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
		Executor:           executor,
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
		// Run `kill` event if it exists
		if (*i).Kind() == value.TerminateInterruptKind {
			if fn, found := self.modules[entryModule].scopes[0]["@event_kill"]; found {
				cancelCtx, cancel := context.WithTimeout(context.Background(), time.Second*KILL_EVENT_TIMEOUT_SECS)
				self.cancelCtx = &cancelCtx
				_ = cancel

				_, i := self.callFunc(errors.Span{}, (*fn).(value.ValueFunction), make([]ast.AnalyzedCallArgument, 0))
				if i != nil {
					if (*i).Kind() == value.ThrowInterruptKind {
						throw := (*i).(value.ThrowInterrupt)
						return value.NewRuntimeErr(throw.Message(), value.UncaughtThrowKind, throw.Span)
					}
				}
			}
		}

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
