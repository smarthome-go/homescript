package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

const MainFunctionIdent = "main"

const InitFunctionIdent = "@init"

// const EntryPointFunctionIdent = "@entrypoint"
const RegisterTriggerHostFn = "@trigger"

type Loop struct {
	labelStart    string
	labelBreak    string
	labelContinue string
}

type Function struct {
	MangledName  string
	Instructions []Instruction
	SourceMap    []errors.Span
	CntVariables uint
	// When `return` is encountered, a jump to this label is performed.
	// This label only restores the memory pointer to the value it was before the call.
	CleanupLabel string
}

type Compiler struct {
	modules         map[string]map[string]*Function
	currFn          string
	loops           []Loop
	fnNameMangle    map[string]uint64
	varNameMangle   map[string]uint64
	labelNameMangle map[string]uint64
	varScopes       []map[string]string
	currScope       *map[string]string
	currModule      string
	// Program source: required for invocations of the evaluator.
	analyzedSource   map[string]ast.AnalyzedProgram
	entryPointModule string
	// Used when the interpreter is invoked during compilation, for instance for annotations.
	executor value.Executor
}

func NewCompiler(program map[string]ast.AnalyzedProgram, entryPointModule string, executor value.Executor) Compiler {
	scopes := make([]map[string]string, 1)
	scopes[0] = make(map[string]string)
	currScope := &scopes[0]

	return Compiler{
		modules:         make(map[string]map[string]*Function),
		loops:           make([]Loop, 0),
		fnNameMangle:    make(map[string]uint64),
		varNameMangle:   make(map[string]uint64),
		labelNameMangle: make(map[string]uint64),
		varScopes:       scopes,
		currScope:       currScope,
		currModule:      "",
		currFn:          "",
		// Program source.
		analyzedSource:   program,
		entryPointModule: entryPointModule,
		// Executor.
		executor: executor,
	}
}

const (
	LIST_PUSH = "__internal_list_push"
)

func (self *Compiler) Compile() (CompileOutput, error) {
	// BUG: cross-module calls do not work
	// BUG: furthermore, cross-module pub-let definitions also do not work.
	mappings, annotations, i := self.compileProgram(self.analyzedSource, self.entryPointModule)

	if i != nil {
		return CompileOutput{}, i
	}

	self.relocateLabels()
	self.renameVariables()

	functions := make(map[string][]Instruction)
	sourceMap := make(map[string][]errors.Span)

	for _, module := range self.modules {
		for _, fn := range module {
			functions[fn.MangledName] = fn.Instructions
			sourceMap[fn.MangledName] = fn.SourceMap
		}
	}

	return CompileOutput{
		Functions:   functions,
		SourceMap:   sourceMap,
		Mappings:    mappings,
		Annotations: annotations,
	}, nil
}

// Maps an unmangled function identifier to its annotations.
type ModuleAnnotations = map[ModuleFunction]CompiledAnnotations
type ModuleFunction struct {
	Module            string
	UnmangledFunction string
}

func (self *Compiler) compileProgram(
	program map[string]ast.AnalyzedProgram,
	entryPointModule string,
) (MangleMappings, ModuleAnnotations, error) {
	initFns := make(map[string]string)

	mappings := MangleMappings{
		Functions:  make(map[string]string),
		Globals:    make(map[string]string),
		Singletons: make(map[string]string),
	}

	for moduleName, module := range program {
		self.currModule = moduleName
		self.modules[self.currModule] = make(map[string]*Function)

		initFn := self.mangleFn(InitFunctionIdent)
		self.addFn(InitFunctionIdent, initFn)
		initFns[moduleName] = initFn
		self.currFn = InitFunctionIdent

		for _, singleton := range module.Singletons {
			// Save mangled name for external mapping.
			mappings.Singletons[singleton.Ident.Ident()] = self.compileSingletonInit(singleton)
		}

		for _, glob := range module.Globals {
			if moduleName == entryPointModule {
				// Save mangled name for external mapping.
				mappings.Globals[glob.Ident.Ident()] = self.compileLetStmt(glob, true)
			}
		}

		for _, item := range module.Imports {
			// No need to handle anything, the analyzer has already taken care of these cases.
			if item.TargetIsHMS {
				continue
			}

			for _, importItem := range item.ToImport {
				self.insert(newTwoStringInstruction(Opcode_Import, item.FromModule.Ident(), importItem.Ident.Ident()), item.Range)
			}
		}

		// Mangle all functions so that later stages know about them.
		for _, fn := range module.Functions {
			mangled := self.mangleFn(fn.Ident.Ident())
			self.addFn(fn.Ident.Ident(), mangled)
		}

		// Mangle all impl block functions due to the same reason.
		for _, impl := range module.ImplBlocks {
			for _, fn := range impl.Methods {
				mangled := self.mangleFn(fn.Ident.Ident())
				self.addFn(fn.Ident.Ident(), mangled)
			}
		}

		// If the current module is the entry module,
		// add all mangled functions to the `mangledEntryFunctions` map.
		if moduleName == entryPointModule {
			for srcIdent, fn := range self.modules[self.currModule] {
				mappings.Functions[srcIdent] = fn.MangledName
			}
		}
	}

	// self.currModule = entryPointModule
	// entryPointFN := self.mangleFn(EntryPointFunctionIdent)
	// self.addFn(EntryPointFunctionIdent, entryPointFN)

	// for moduleName, _ := range program {
	// 	self.currModule = moduleName
	//
	// 	// If the current module is the entry module,
	// 	// add all mangled functions to the `mangledEntryFunctions` map.
	// 	if moduleName == entryPointModule {
	// 		for srcIdent, fn := range self.modules[self.currModule] {
	// 			mappings.Functions[srcIdent] = fn.MangledName
	// 		}
	// 	}
	// }

	moduleAnnotations := make(ModuleAnnotations)

	for moduleName, module := range program {
		self.currModule = moduleName

		// Compile all functions
		var mainFnSpan errors.Span
		for _, fn := range module.Functions {
			if fn.Ident.Ident() == MainFunctionIdent {
				mainFnSpan = fn.Range
			}
			fnAnnotations, i := self.compileFn(fn)

			if i != nil {
				return MangleMappings{}, ModuleAnnotations{}, fmt.Errorf("Constant evaluation failed: %s", (*i).Message())
			}

			if fnAnnotations != nil {
				moduleAnnotations[ModuleFunction{
					Module:            moduleName,
					UnmangledFunction: fn.Ident.Ident(),
				}] = *fnAnnotations
			}
		}

		// Compile all impl block methods.
		for _, impl := range module.ImplBlocks {
			for _, fn := range impl.Methods {
				// TODO: annotations here
				self.compileFn(fn)
			}
		}

		// Compile all events.
		// TODO: allow customizing this `@event` prefix
		// for _, fn := range module.Events {
		// 	oldIdent := fn.Ident.Ident()
		//
		// 	fn.Ident = pAst.NewSpannedIdent(fmt.Sprintf("@event_%s", fn.Ident.Ident()), fn.Ident.Span())
		// 	mangled := self.mangleFn(fn.Ident.Ident())
		// 	self.addFn(fn.Ident.Ident(), mangled)
		// 	self.compileFn(fn)
		//
		// 	// Add event function to mappings.
		// 	mappings.Functions[oldIdent] = mangled
		// }

		if moduleName == entryPointModule {
			// If the current module is the entry module,
			// Go back to the entrypoint function and insert the main function call.
			self.currFn = InitFunctionIdent
			self.currModule = entryPointModule

			for moduleName, otherInit := range initFns {
				if moduleName == entryPointModule {
					continue
				}

				self.insert(newOneStringInstruction(Opcode_Call_Imm, otherInit), mainFnSpan)
			}

			self.insert(newPrimitiveInstruction(Opcode_Return), mainFnSpan)

			// mangledMain, found := self.getMangledFn(MainFunctionIdent)
			// if !found {
			// 	panic(fmt.Sprintf("`%s` function not found in current module", MainFunctionIdent))
			// }

			// // Also create the entrypoint function which performs calls the `main` function.
			// self.currFn = EntryPointFunctionIdent
			// self.insert(newOneStringInstruction(Opcode_Call_Imm, mangledMain), mainFnSpan)
			// self.insert(newPrimitiveInstruction(Opcode_Return), mainFnSpan)
		}
	}

	return mappings, moduleAnnotations, nil
}
