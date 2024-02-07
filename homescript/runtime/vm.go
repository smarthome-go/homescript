package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

const CATCH_PANIC = false

type Globals struct {
	Data  map[string]value.Value
	Mutex sync.RWMutex
}

func newGlobals(scopeAdditions map[string]value.Value) Globals {
	return Globals{
		Data:  scopeAdditions,
		Mutex: sync.RWMutex{},
	}
}

type Cores struct {
	Cores []Core
	Lock  sync.RWMutex
}

func newCores() Cores {
	return Cores{
		Cores: make([]Core, 0),
		Lock:  sync.RWMutex{},
	}
}

type VM struct {
	Program       compiler.Program
	globals       Globals
	Cores         Cores
	Executor      value.Executor
	Lock          sync.RWMutex
	coreCnt       uint
	CancelCtx     *context.Context
	CancelFunc    *context.CancelFunc
	Interrupts    map[uint]value.VmInterrupt
	LimitsPerCore CoreLimits
}

func NewVM(
	program compiler.Program,
	executor value.Executor,
	ctx *context.Context,
	cancelFunc *context.CancelFunc,
	globalScopeAdditions map[string]value.Value,
	limits CoreLimits,
) VM {
	return VM{
		Program:       program,
		globals:       newGlobals(globalScopeAdditions),
		Cores:         newCores(),
		Executor:      executor,
		Lock:          sync.RWMutex{},
		coreCnt:       0,
		CancelCtx:     ctx,
		CancelFunc:    cancelFunc,
		Interrupts:    make(map[uint]value.VmInterrupt),
		LimitsPerCore: limits,
	}
}

// TODO: why is this not a real method?
func hostcall(self *VM, function string, args []*value.Value) (*value.Value, *value.VmInterrupt) {
	// TODO: this is extremely bad!!!
	self.Lock.Lock()
	defer self.Lock.Unlock()

	if function == "__internal_list_push" {
		elem := args[0]
		list := (*args[1]).(value.ValueList)

		(*list.Values) = append((*list.Values), elem)
		return args[1], nil
	}

	panic("Invalid hostcall: " + function)
}

func (self *VM) GetGlobals() map[string]value.Value {
	// WARNING: this is unsafe before all cores have terminated.
	return self.globals.Data
}

func (self *VM) spawnCore() *Core {
	self.Lock.Lock()
	defer self.Lock.Unlock()

	ch := make(chan *value.VmInterrupt)
	core := NewCore(
		&self.Program.Functions,
		hostcall,
		self.Executor,
		self,
		self.coreCnt,
		ch,
		self.CancelCtx,
		self.LimitsPerCore,
	)

	self.Cores.Cores = append(self.Cores.Cores, core)
	self.coreCnt++
	return &core
}

type FunctionInvocationSignature struct {
	Params     map[string]ast.Type
	ReturnType ast.Type
}

func FunctionInvocationSignatureFromType(input ast.FunctionType) FunctionInvocationSignature {
	if input.Params.Kind() != ast.NormalFunctionTypeParamKindIdentifierKind {
		panic(fmt.Sprintf("Expected normal param kind, found: %d", input.Params.Kind()))
	}

	fromParams := input.Params.(ast.NormalFunctionTypeParamKindIdentifier).Params
	params := make(map[string]ast.Type)

	for _, param := range fromParams {
		params[param.Name.Ident()] = param.Type
	}

	return FunctionInvocationSignature{
		Params:     params,
		ReturnType: input.ReturnType,
	}
}

type FunctionInvocation struct {
	Function string
	Args     []value.Value
	// Required so that the VM can automatically pop the function's return value if it materializes into a value.
	// Furthermore, recursive type assertion is performed on the return value so that the caller does not have to
	// worry about safety.
	// Additionally, before the function is actually called, the passed parameter types are also validated.
	FunctionSignature FunctionInvocationSignature
}

type FunctionInvocationResult struct {
	Exception   *VMException
	ReturnValue value.Value
}

type VMException struct {
	CoreNum   uint
	Interrupt value.VmInterrupt
}

// Returns the core of the newly spawned process.
func (self *VM) SpawnAsync(invocation FunctionInvocation, debuggerOut *chan DebugOutput) *Core {
	return self.spawnCoreInternal(invocation.Function, invocation.Args, debuggerOut)
}

// Spawns a new core but also calls `vm.Wait` internally.
func (self *VM) SpawnSync(invocation FunctionInvocation, debuggerOut *chan DebugOutput) FunctionInvocationResult {
	if invocation.FunctionSignature.ReturnType == nil {
		panic("Invocation called without return type specified.")
	}

	// Validate that the provided arguments match the function's signature.
	expectedParamLen := len(invocation.FunctionSignature.Params)
	gotArgLen := len(invocation.Args)

	if gotArgLen != expectedParamLen {
		panic(fmt.Sprintf("Illegal call: expected %d arguments due to function signature, got %d", expectedParamLen, gotArgLen))
	}

	index := 0
	for name, param := range invocation.FunctionSignature.Params {
		arg := invocation.Args[index]
		_, interrupt := value.DeepCast(arg, param, errors.Span{}, false)
		if interrupt != nil {
			panic(fmt.Sprintf("Argument %d for param `%s` type mismatch: %s", index, name, (*interrupt).Message()))
		}

		index++
	}

	coreHandle := self.spawnCoreInternal(invocation.Function, invocation.Args, debuggerOut)
	exceptionCore, interrupt := self.Wait()

	return self.HandleTermination(
		coreHandle,
		invocation,
		interrupt,
		exceptionCore,
	)
}

func (self *VM) HandleTermination(
	exitCore *Core,
	invocation FunctionInvocation,
	interrupt *value.VmInterrupt,
	exceptionCore uint,
) FunctionInvocationResult {
	if interrupt != nil {
		return FunctionInvocationResult{
			Exception: &VMException{
				CoreNum:   exceptionCore,
				Interrupt: *interrupt,
			},
			ReturnValue: nil,
		}
	}

	var returnValue value.Value

	switch invocation.FunctionSignature.ReturnType.Kind() {
	case ast.NullTypeKind, ast.NeverTypeKind, ast.UnknownTypeKind, ast.AnyObjectTypeKind:
		break
	default:
		// Get function return value.
		returnValueRaw := exitCore.Stack[len(exitCore.Stack)-1]

		// Perform type assertion.
		castValue, interrupt := value.DeepCast(
			*returnValueRaw,
			invocation.FunctionSignature.ReturnType,
			errors.Span{},
			false,
		)
		if interrupt != nil {
			panic(fmt.Sprintf("Foreign function invocation: return type assertion failed: %s", (*interrupt).Message()))
		}

		returnValue = *castValue
	}

	return FunctionInvocationResult{
		Exception:   nil,
		ReturnValue: returnValue,
	}
}

// Returns the corenum of the newly spawned process
func (self *VM) spawnCoreInternal(
	function string,
	addToStack []value.Value,
	debuggerOutput *chan DebugOutput,
) *Core {
	// Lookup the function to be invoked.
	toBeInvoked, found := self.Program.Mappings.Functions[function]
	if !found {
		panic(fmt.Sprintf("Requested function `%s` does not exist", function))
	}

	core := self.spawnCore()
	for _, elem := range addToStack {
		// TODO: However, the VM should not do this implicitly,
		// Smarter would be to insert clones manually?
		core.push(&elem) // Implement a deep copy? Or clone?
	}
	go (*core).Run(toBeInvoked, debuggerOutput)

	return core
}

func (self *VM) WaitNonConsuming() {
	for {
		self.Cores.Lock.RLock()
		defer self.Cores.Lock.RUnlock()

		if len(self.Cores.Cores) == 0 {
			// TODO: remove this once this message has been spotted.
			fmt.Printf("NOTE: Breakout..")
			break
		}
	}
}

func (self *VM) Wait() (coreNum uint, i *value.VmInterrupt) {
	for {
		self.Cores.Lock.RLock()

		for _, core := range self.Cores.Cores {
			// fmt.Printf("checking core: %d | %v\n", core.Corenum, time.Now())

			select {
			case i := <-core.SignalHandle:
				if i == nil {
					newCores := make([]Core, 0)

					for _, coreIter := range self.Cores.Cores {
						if coreIter.Corenum == core.Corenum {
							continue
						}

						newCores = append(newCores, coreIter)
					}

					self.Cores.Lock.RUnlock()

					self.Cores.Lock.Lock()
					self.Cores.Cores = newCores
					self.Cores.Lock.Unlock()

					self.Cores.Lock.RLock()
				} else {
					self.Cores.Lock.RUnlock()

					// TODO: is this OK?
					self.Cores.Lock.Lock()

					(*self.CancelFunc)()

					self.Cores.Cores = make([]Core, 0)
					self.Cores.Lock.Unlock()

					self.Cores.Lock.RLock()

					return core.Corenum, i
				}
			default:
			}
		}

		if len(self.Cores.Cores) == 0 {
			self.Cores.Lock.RUnlock()
			break
		}

		self.Cores.Lock.RUnlock()
	}

	return 0, nil
}
