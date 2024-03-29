package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

const MainFunctionIdent = "main"
const EntryPointFunctionIdent = "@init"
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
}

func NewCompiler() Compiler {
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
	}
}

const (
	LIST_PUSH = "__internal_list_push"
)

func (self Compiler) CurrFn() *Function { return self.modules[self.currModule][self.currFn] }

func (self Compiler) currLoop() Loop { return self.loops[len(self.loops)-1] }
func (self *Compiler) pushLoop(l Loop) {
	self.loops = append(self.loops, l)
}
func (self *Compiler) popLoop() {
	self.loops = self.loops[:len(self.loops)-1]
}

func (self *Compiler) pushScope() {
	self.varScopes = append(self.varScopes, make(map[string]string))
	self.currScope = &self.varScopes[len(self.varScopes)-1]
}

func (self *Compiler) popScope() {
	self.varScopes = self.varScopes[:len(self.varScopes)-1]
	if len(self.varScopes) == 0 {
		self.currScope = nil
		return
	}
	self.currScope = &self.varScopes[len(self.varScopes)-1]
}

func (self *Compiler) mangleFn(input string) string {
	mangled := fmt.Sprintf("@%s_%s", self.currModule, input)
	return mangled
}

func (self *Compiler) addFn(srcIdent string, mangledName string) {
	fn := Function{
		MangledName:  mangledName,
		Instructions: make([]Instruction, 0),
		SourceMap:    make([]errors.Span, 0),
		CntVariables: 0,
		CleanupLabel: "",
	}
	self.modules[self.currModule][srcIdent] = &fn
}

func (self *Compiler) mangleVar(input string) string {
	self.CurrFn().CntVariables++
	cnt, exists := self.varNameMangle[input]
	if !exists {
		// The next time this variable is mangled, 0 MUST NOT be used as the counter.
		self.varNameMangle[input] = 1
		cnt = 0
	} else {
		self.varNameMangle[input]++
	}

	mangled := fmt.Sprintf("@%s_%s%d", self.currModule, input, cnt)
	(*self.currScope)[input] = mangled

	return mangled
}

func (self *Compiler) mangleLabel(input string) string {
	cnt, exists := self.labelNameMangle[input]
	if !exists {
		self.labelNameMangle[input]++
		cnt = 0
	} else {
		self.labelNameMangle[input]++
	}

	mangled := fmt.Sprintf("%s_%s%d", self.currModule, input, cnt)
	return mangled
}

func (self Compiler) getMangledFn(input string) (string, bool) {
	for key, fn := range self.modules[self.currModule] {
		if key == input {
			return fn.MangledName, true
		}
	}

	// TODO: i don't think that this is really reliable
	for _, module := range self.modules {
		for key, fn := range module {
			if key == input {
				return fn.MangledName, true
			}
		}
	}

	return "", false
}

func (self Compiler) getMangled(input string) (string, bool) {
	// Iterate over the scopes backwards: Most current scope should be considered first.
	for i := len(self.varScopes) - 1; i >= 0; i-- {
		scope := self.varScopes[i]
		name, found := scope[input]
		if found {
			return name, true
		}
	}

	return "", false
}

func (self *Compiler) relocateLabels() {
	for moduleName, module := range self.modules {
		for name, fn := range module {
			labels := make(map[string]int64)

			fnOut := make([]Instruction, 0)
			sourceMapOut := make([]errors.Span, 0)

			index := 0
			for idx, inst := range fn.Instructions {
				if inst.Opcode() == Opcode_Label {
					i := inst.(OneStringInstruction).Value
					labels[i] = int64(index)
				} else {
					fnOut = append(fnOut, inst)
					sourceMapOut = append(sourceMapOut, fn.SourceMap[idx])
					index++
				}
			}

			for idx, inst := range fnOut {
				switch inst.Opcode() {
				case Opcode_Jump, Opcode_JumpIfFalse:
					i := inst.(OneStringInstruction)

					ip, found := labels[i.Value]
					if !found {
						panic(fmt.Sprintf("Every label needs to appear in the code: %s", i.Value))
					}

					fnOut[idx] = newOneIntInstruction(inst.Opcode(), ip)
				case Opcode_SetTryLabel:
					i := inst.(TwoStringInstruction)

					ip, found := labels[i.Values[1]]
					if !found {
						panic("Every label needs to appear in the code")
					}

					fnOut[idx] = newOneIntOneStringInstruction(inst.Opcode(), i.Values[0], ip)
				case Opcode_Label:
					panic("This should not happen")
				}
			}

			self.modules[moduleName][name].Instructions = fnOut
			self.modules[moduleName][name].SourceMap = sourceMapOut
		}
	}
}

func (self *Compiler) renameVariables() {
	slot := make(map[string]int64, 0)

	for _, module := range self.modules {
		for name, fn := range module {
			cnt := 0
			for idx, inst := range fn.Instructions {
				switch inst.Opcode() {
				case Opcode_GetVarImm:
					i := inst.(OneStringInstruction)
					if _, found := slot[i.Value]; !found {
						slot[i.Value] = int64(cnt)
						cnt++
					}
					module[name].Instructions[idx] = newOneIntInstruction(Opcode_GetVarImm, slot[i.Value])
				case Opcode_SetVarImm:
					i := inst.(OneStringInstruction)
					if _, found := slot[i.Value]; !found {
						slot[i.Value] = int64(cnt)
						cnt++
					}
					module[name].Instructions[idx] = newOneIntInstruction(Opcode_SetVarImm, slot[i.Value])
				default:
					continue
				}
			}
		}
	}
}

func (self *Compiler) Compile(program map[string]ast.AnalyzedProgram, entryPointModule string) Program {
	// BUG: cross-module calls do not work
	// BUG: furthermore, cross-module pub-let definitions also do not work.
	mappings := self.compileProgram(program, entryPointModule)

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

	return Program{
		Functions: functions,
		SourceMap: sourceMap,
		Mappings:  mappings,
	}
}

func (self *Compiler) insert(instruction Instruction, span errors.Span) int {
	self.CurrFn().Instructions = append(self.CurrFn().Instructions, instruction)
	self.CurrFn().SourceMap = append(self.CurrFn().SourceMap, span)
	return len(self.CurrFn().Instructions) - 1
}

func (self *Compiler) compileProgram(
	program map[string]ast.AnalyzedProgram,
	entryPointModule string,
) MangleMappings {
	initFns := make(map[string]string)

	mappings := MangleMappings{
		Functions:  make(map[string]string),
		Globals:    make(map[string]string),
		Singletons: make(map[string]string),
	}

	for moduleName, module := range program {
		self.currModule = moduleName
		self.modules[self.currModule] = make(map[string]*Function)

		initFn := self.mangleFn(EntryPointFunctionIdent)
		self.addFn(EntryPointFunctionIdent, initFn)
		initFns[moduleName] = initFn
		self.currFn = EntryPointFunctionIdent

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

	for moduleName, module := range program {
		self.currModule = moduleName

		// Compile all functions
		var mainFnSpan errors.Span
		for _, fn := range module.Functions {
			if fn.Ident.Ident() == MainFunctionIdent {
				mainFnSpan = fn.Range
			}
			self.compileFn(fn)
		}

		// Compile all impl block methods.
		for _, impl := range module.ImplBlocks {
			for _, fn := range impl.Methods {
				self.compileFn(fn)
			}
		}

		// Compile all events.
		// TODO: allow customizing this `@event` prefix
		for _, fn := range module.Events {
			oldIdent := fn.Ident.Ident()

			fn.Ident = pAst.NewSpannedIdent(fmt.Sprintf("@event_%s", fn.Ident.Ident()), fn.Ident.Span())
			mangled := self.mangleFn(fn.Ident.Ident())
			self.addFn(fn.Ident.Ident(), mangled)
			self.compileFn(fn)

			// Add event function to mappings.
			mappings.Functions[oldIdent] = mangled
		}

		if moduleName == entryPointModule {
			// If the current module is the entry module,
			// Go back to the init function and insert the main function call.
			self.currFn = EntryPointFunctionIdent
			self.currModule = entryPointModule

			for moduleName, otherInit := range initFns {
				if moduleName == entryPointModule {
					continue
				}

				self.insert(newOneStringInstruction(Opcode_Call_Imm, otherInit), mainFnSpan)
			}

			mangledMain, found := self.getMangledFn(MainFunctionIdent)
			if !found {
				panic(fmt.Sprintf("`%s` function not found in current module", MainFunctionIdent))
			}

			self.insert(newOneStringInstruction(Opcode_Call_Imm, mangledMain), mainFnSpan)
			self.insert(newPrimitiveInstruction(Opcode_Return), mainFnSpan)
		}
	}

	return mappings
}

func (self *Compiler) compileBlock(node ast.AnalyzedBlock, pushScope bool) {
	if pushScope {
		self.pushScope()
		defer self.popScope()
	}

	for _, stmt := range node.Statements {
		self.compileStmt(stmt)
	}

	if node.Expression != nil {
		self.compileExpr(node.Expression)
	}
}

func (self *Compiler) compileIfExpr(node ast.AnalyzedIfExpression) {
	self.compileExpr(node.Condition)

	after_label := self.mangleLabel("if_after")
	else_label := self.mangleLabel("else")

	if node.ElseBlock != nil {
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, else_label), node.Range)
	} else {
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, after_label), node.Range)
	}
	self.compileBlock(node.ThenBlock, true)
	self.insert(newOneStringInstruction(Opcode_Jump, after_label), node.Range)

	if node.ElseBlock != nil {
		self.insert(newOneStringInstruction(Opcode_Label, else_label), node.Range)
		self.compileBlock(*node.ElseBlock, true)
	}
	self.insert(newOneStringInstruction(Opcode_Label, after_label), node.Range)
}

func (self *Compiler) compileFn(node ast.AnalyzedFunctionDefinition) {
	self.currFn = node.Ident.Ident()
	self.pushScope()
	defer self.popScope()

	// Value / stack  depth is replaced later
	mpIdx := self.insert(newOneIntInstruction(Opcode_AddMempointer, 0), node.Range)

	singletonExtractors := make([]ast.AnalyzedFnParam, 0)

	// Parameters are pushed in reverse-order, so they can be popped in the correct order.
	for _, param := range node.Parameters.List {
		// TODO: If the current parameter is a singleton extraction, do extra work.
		// Add a name-alias for the singleton so that each time the extracted name is used, the singleton is accessed instead.

		if param.IsSingletonExtractor {
			singletonExtractors = append(singletonExtractors, param)
			continue
		}

		name := self.mangleVar(param.Ident.Ident())
		self.insert(newOneStringInstruction(Opcode_SetVarImm, name), node.Range)
	}

	// NOTE: this is required because the `vm.Spawn` method allows for inserting custom values onto the stack.
	// These values should then be used to construct function parameter variables.
	// However, if singletons are being extracted at the beginning, this might interfere with the normal stack layout,
	// causing runtime panics.
	for _, singletonParam := range singletonExtractors {
		// Load singleton first
		name, found := self.getMangled(singletonParam.SingletonIdent)
		if !found {
			panic("Existing singleton not found")
		}
		self.insert(newOneStringInstruction(Opcode_GetGlobImm, name), node.Range)
		self.insert(newOneStringInstruction(Opcode_GetGlobImm, name), node.Range)

		localName := self.mangleVar(singletonParam.Ident.Ident())
		self.insert(newOneStringInstruction(Opcode_SetVarImm, localName), node.Range)
	}

	// When `return` is encountered, the compiler inserts a jump to this label.
	// Needed for restoring the memory pointer.
	cleanupLabel := self.mangleLabel("cleanup")
	self.CurrFn().CleanupLabel = cleanupLabel

	self.compileBlock(node.Body, false)

	varCnt := int64(self.CurrFn().CntVariables)
	self.CurrFn().Instructions[mpIdx] = newOneIntInstruction(Opcode_AddMempointer, varCnt)

	self.insert(newOneStringInstruction(Opcode_Label, cleanupLabel), node.Range)
	self.insert(newOneIntInstruction(Opcode_AddMempointer, -varCnt), node.Range)
	self.insert(newPrimitiveInstruction(Opcode_Return), node.Span())
}

func (self *Compiler) compileLetStmt(node ast.AnalyzedLetStatement, isGlobal bool) (mangled string) {
	// Push value onto the stack
	self.compileExpr(node.Expression)

	// Handle deep casts if required
	if node.NeedsRuntimeTypeValidation {
		// TODO: test this: is this ok?
		self.insert(newCastInstruction(node.OptType, false), node.Type().Span())
	}

	opcode := Opcode_SetVarImm
	if isGlobal {
		opcode = Opcode_SetGlobImm
	}

	// Bind value to identifier
	mangledName := self.mangleVar(node.Ident.Ident())
	self.insert(newOneStringInstruction(opcode, mangledName), node.Range)
	self.CurrFn().CntVariables++ // FIXME: have reference

	return mangledName
}

func (self *Compiler) compileSingletonInit(node ast.AnalyzedSingletonTypeDefinition) (mangledName string) {
	// Push default value onto the stack
	def := value.ZeroValue(node.SingletonType)
	self.insert(newValueInstruction(Opcode_Cloning_Push, *def), node.Range)

	// Load singleton, either pop default and use other or use default.
	self.insert(newTwoStringInstruction(Opcode_Load_Singleton, node.Ident.Ident(), node.Ident.Span().Filename), node.Range)

	// Bind value to identifier
	name := self.mangleVar(node.Ident.Ident())
	self.insert(newOneStringInstruction(Opcode_SetGlobImm, name), node.Range)
	self.CurrFn().CntVariables++ // FIXME: have reference

	return name
}

func (self *Compiler) compileStmt(node ast.AnalyzedStatement) {
	switch node.Kind() {
	case ast.TypeDefinitionStatementKind:
		// This kind of statement is just ignored.
		break
	case ast.TriggerStatementKind:
		node := node.(ast.AnalyzedTriggerStatement)
		const defaultArgC = 2
		hostCallArgc := defaultArgC + len(node.TriggerArguments.List)

		for idx := len(node.TriggerArguments.List) - 1; idx >= 0; idx-- {
			fmt.Printf("TRIGGER STATEMENT COMPILATION OF ARG: %s\n", node.TriggerArguments.List[idx].Expression)
			self.compileExpr(node.TriggerArguments.List[idx].Expression)
		}

		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueString(node.TriggerIdent.Ident())), node.Span())
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueString(node.CallbackIdent.Ident())), node.Span())
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(int64(hostCallArgc))), node.Span())
		self.insert(newOneStringInstruction(Opcode_HostCall, RegisterTriggerHostFn), node.Span())
	case ast.LetStatementKind:
		self.compileLetStmt(node.(ast.AnalyzedLetStatement), false)
	case ast.ReturnStatementKind:
		node := node.(ast.AnalyzedReturnStatement)
		// If there is a return-expression, insert it
		if node.ReturnValue != nil {
			self.compileExpr(node.ReturnValue)
		}

		self.insert(newOneStringInstruction(Opcode_Jump, self.CurrFn().CleanupLabel), node.Span())
	case ast.BreakStatementKind:
		self.insert(newOneStringInstruction(Opcode_Jump, self.currLoop().labelBreak), node.Span())
	case ast.ContinueStatementKind:
		self.insert(newOneStringInstruction(Opcode_Jump, self.currLoop().labelContinue), node.Span())
	case ast.LoopStatementKind:
		node := node.(ast.AnalyzedLoopStatement)

		head_label := self.mangleLabel("loop_head")
		after_label := self.mangleLabel("loop_end")
		self.insert(newOneStringInstruction(Opcode_Label, head_label), node.Span())

		self.pushLoop(Loop{
			labelStart:    head_label,
			labelBreak:    after_label,
			labelContinue: head_label,
		})
		defer self.popLoop()

		self.compileBlock(node.Body, true)
		self.insert(newOneStringInstruction(Opcode_Jump, head_label), node.Span())
		self.insert(newOneStringInstruction(Opcode_Label, after_label), node.Span())
	case ast.WhileStatementKind:
		node := node.(ast.AnalyzedWhileStatement)
		head_label := self.mangleLabel("loop_head")
		after_label := self.mangleLabel("loop_end")

		self.insert(newOneStringInstruction(Opcode_Label, head_label), node.Range)

		// check the condition
		self.compileExpr(node.Condition)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, after_label), node.Range)

		self.pushLoop(Loop{
			labelStart:    head_label,
			labelBreak:    after_label,
			labelContinue: head_label,
		})
		defer self.popLoop()

		self.compileBlock(node.Body, true)
		self.insert(newOneStringInstruction(Opcode_Jump, head_label), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, after_label), node.Range)
	case ast.ForStatementKind:
		// FIXME: account for variable count

		node := node.(ast.AnalyzedForStatement)

		head_label := self.mangleLabel("loop_head")
		update_label := self.mangleLabel("loop_update")
		after_label := self.mangleLabel("loop_end")

		// Create initial state of iterator
		self.pushScope()
		defer self.popScope()

		// Push iter expr onto the stack
		self.compileExpr(node.IterExpression)

		// Convert into iterator
		self.insert(newPrimitiveInstruction(Opcode_IntoIter), node.Range)
		iterName := self.mangleVar(fmt.Sprintf("$iter_%s", node.Identifier.Ident()))
		self.insert(newOneStringInstruction(Opcode_SetVarImm, iterName), node.Range)

		// Loop body
		headIdentName := self.mangleVar(node.Identifier.Ident())

		// Bind induction variable to name

		self.insert(newOneStringInstruction(Opcode_Label, head_label), node.Range)

		self.insert(newOneStringInstruction(Opcode_GetVarImm, iterName), node.Range)
		self.insert(newPrimitiveInstruction(Opcode_IteratorAdvance), node.Range)

		self.insert(newOneStringInstruction(Opcode_SetVarImm, headIdentName), node.Range)

		// On top of the stack, there will now be a bool describing whether or not to continue.
		// Check if there are still values left: if not, break.
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, after_label), node.Range)

		self.pushLoop(Loop{
			labelStart:    head_label,
			labelBreak:    after_label,
			labelContinue: update_label,
		})
		defer self.popLoop()

		self.compileBlock(node.Body, false)

		// Update iterator
		self.insert(newOneStringInstruction(Opcode_Label, update_label), node.Range)

		// Jump back to head
		self.insert(newOneStringInstruction(Opcode_Jump, head_label), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, after_label), node.Range)
	case ast.ExpressionStatementKind:
		node := node.(ast.AnalyzedExpressionStatement)
		self.compileExpr(node.Expression)
		if node.Expression.Type().Kind() != ast.NullTypeKind {
			// Drop every value that the expression might generate
			self.insert(newPrimitiveInstruction(Opcode_Drop), node.Range)
		}
	default:
		panic("Unreachable")
	}
}

func (self *Compiler) compilePrefixOp(op ast.PrefixOperator, span errors.Span) {
	switch op {
	case ast.MinusPrefixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Neg), span)
	case ast.NegatePrefixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Not), span)
	case ast.IntoSomePrefixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Some), span)
	}
}

func (self *Compiler) compileCallExpr(node ast.AnalyzedCallExpression) {
	// Push each argument onto the stack
	// The order is reversed so that later popping can be done naturally
	for i := len(node.Arguments.List) - 1; i >= 0; i-- {
		self.compileExpr(node.Arguments.List[i].Expression)
	}

	if node.Base.Kind() == ast.IdentExpressionKind {
		base := node.Base.(ast.AnalyzedIdentExpression)

		// Special case: base is `throw`
		if base.Ident.Ident() == "throw" {
			self.insert(newPrimitiveInstruction(Opcode_Throw), node.Range)
			return
		}

		// Check whether the scope is local or global
		_, found := self.getMangled(base.Ident.Ident())
		if found {
			if node.IsSpawn {
				panic("This is an impossible state.")
			}

			self.compileExpr(node.Base)
			self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(int64(len(node.Arguments.List)))), node.Span())
			self.insert(newPrimitiveInstruction(Opcode_Call_Val), node.Span())
		} else {
			// TODO: the span mapping is broken here?
			name, found := self.getMangledFn(base.Ident.Ident())
			if found {
				opcode := Opcode_Call_Imm
				if node.IsSpawn {
					opcode = Opcode_Spawn
					self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(int64(len(node.Arguments.List)))), node.Span())
				}

				self.insert(newOneStringInstruction(opcode, name), node.Span())
			} else {
				// call a global value
				self.insert(newOneStringInstruction(Opcode_GetGlobImm, base.Ident.Ident()), node.Range)
				self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(int64(len(node.Arguments.List)))), node.Span())
				self.insert(newPrimitiveInstruction(Opcode_Call_Val), node.Range)
			}
		}
	} else {
		if node.IsSpawn {
			panic("This is an impossible state.")
		}

		self.compileExpr(node.Base)

		// insert number of args
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(int64(len(node.Arguments.List)))), node.Span())

		// perform the actual call
		self.insert(newPrimitiveInstruction(Opcode_Call_Val), node.Span())
	}
}

func (self *Compiler) arithmeticHelper(op pAst.InfixOperator, span errors.Span) {
	switch op {
	case pAst.PlusInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Add), span)
	case pAst.MinusInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Sub), span)
	case pAst.MultiplyInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Mul), span)
	case pAst.DivideInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Div), span)
	case pAst.ModuloInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Rem), span)
	case pAst.PowerInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Pow), span)
	case pAst.ShiftLeftInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Shl), span)
	case pAst.ShiftRightInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Shr), span)
	case pAst.BitOrInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_BitOr), span)
	case pAst.BitAndInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_BitAnd), span)
	case pAst.BitXorInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_BitXor), span)
	case pAst.EqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Eq), span)
	case pAst.NotEqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Eq), span)
		self.insert(newPrimitiveInstruction(Opcode_Not), span)
	case pAst.LessThanInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Lt), span)
	case pAst.LessThanEqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Le), span)
	case pAst.GreaterThanInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Gt), span)
	case pAst.GreaterThanEqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Ge), span)
	default:
		panic("Unreachable")
	}
}

func (self *Compiler) compileInfixExpr(node ast.AnalyzedInfixExpression) {
	switch node.Operator {
	case pAst.LogicalOrInfixOperator:
		returnTrue := self.mangleLabel("return_true")
		afterLabel := self.mangleLabel("after_infix")

		self.compileExpr(node.Lhs)
		self.insert(newPrimitiveInstruction(Opcode_Not), node.Range)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, returnTrue), node.Range)

		self.compileExpr(node.Rhs)
		self.insert(newOneStringInstruction(Opcode_Jump, afterLabel), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, returnTrue), node.Range)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueBool(true)), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, afterLabel), node.Range)
	case pAst.LogicalAndInfixOperator:
		returnFalse := self.mangleLabel("return_false")
		afterLabel := self.mangleLabel("after_infix")

		self.compileExpr(node.Lhs)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, returnFalse), node.Range)

		self.compileExpr(node.Rhs)
		self.insert(newOneStringInstruction(Opcode_Jump, afterLabel), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, returnFalse), node.Range)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueBool(false)), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, afterLabel), node.Range)
	default:
		self.compileExpr(node.Lhs)
		self.compileExpr(node.Rhs)
		self.arithmeticHelper(node.Operator, node.Range)
	}
}

func (self *Compiler) compileIdentExpression(node ast.AnalyzedIdentExpression) {
	name, varFound := self.getMangled(node.Ident.Ident())

	opCode := Opcode_GetVarImm
	if node.IsGlobal {
		opCode = Opcode_GetGlobImm
	}

	if varFound {
		self.insert(newOneStringInstruction(opCode, name), node.Span())
		return
	}

	name, fnFound := self.getMangledFn(node.Ident.Ident())
	if fnFound {
		// This value is a function, it should also be wrapped like one
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueVMFunction(
			name,
		)), node.Span())
	} else {
		// This value is not a function. Instead, it is a global variable.
		self.insert(newOneStringInstruction(opCode, node.Ident.Ident()), node.Span())
	}
}

func (self *Compiler) compileExpr(node ast.AnalyzedExpression) {
	switch node.Kind() {
	case ast.UnknownExpressionKind:
		panic("Unreachable, this should not happen")
	case ast.IntLiteralExpressionKind:
		node := node.(ast.AnalyzedIntLiteralExpression)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(node.Value)), node.Range)
	case ast.FloatLiteralExpressionKind:
		node := node.(ast.AnalyzedFloatLiteralExpression)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueFloat(node.Value)), node.Range)
	case ast.BoolLiteralExpressionKind:
		node := node.(ast.AnalyzedBoolLiteralExpression)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueBool(node.Value)), node.Range)
	case ast.StringLiteralExpressionKind:
		node := node.(ast.AnalyzedStringLiteralExpression)
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueString(node.Value)), node.Range)
	case ast.IdentExpressionKind:
		self.compileIdentExpression(node.(ast.AnalyzedIdentExpression))
	case ast.NullLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueNull()), node.Span())
	case ast.NoneLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewNoneOption()), node.Span())
	case ast.RangeLiteralExpressionKind:
		node := node.(ast.AnalyzedRangeLiteralExpression)
		self.compileExpr(node.Start)
		self.compileExpr(node.End)
		// Boolean instruction is used to mark that the end of this range can be inclusive.
		self.insert(newOneBoolInstruction(Opcode_Into_Range, node.EndIsInclusive), node.Range)
	case ast.ListLiteralExpressionKind:
		node := node.(ast.AnalyzedListLiteralExpression)
		// TODO: is this cloning required???
		self.insert(newValueInstruction(Opcode_Cloning_Push, *value.NewValueList(make([]*value.Value, 0))), node.Range)

		for _, element := range node.Values {
			self.compileExpr(element)
			self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueInt(2)), node.Range)
			self.insert(newOneStringInstruction(Opcode_HostCall, LIST_PUSH), node.Range)
		}
	case ast.AnyObjectLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Cloning_Push, *value.NewValueAnyObject(make(map[string]*value.Value))), node.Span())
	case ast.ObjectLiteralExpressionKind:
		node := node.(ast.AnalyzedObjectLiteralExpression)

		fields := make(map[string]*value.Value)
		for _, field := range node.Fields {
			fields[field.Key.Ident()] = value.ZeroValue(field.Expression.Type())
		}

		object := *value.NewValueObject(fields)
		self.insert(newValueInstruction(Opcode_Cloning_Push, object), node.Range)

		for _, field := range node.Fields {
			self.insert(newPrimitiveInstruction(Opcode_Duplicate), node.Range)
			self.insert(newOneStringInstruction(Opcode_Member, field.Key.Ident()), node.Range)
			self.compileExpr(field.Expression)
			self.insert(newPrimitiveInstruction(Opcode_Assign), node.Range)
		}
	case ast.FunctionLiteralExpressionKind:
		node := node.(ast.AnalyzedFunctionLiteralExpression)

		fnName := self.mangleFn("$lambda")
		self.addFn(fnName, fnName)

		oldCurrFn := self.currFn

		self.compileFn(ast.AnalyzedFunctionDefinition{
			Ident: pAst.NewSpannedIdent(fnName, node.Range),
			Parameters: ast.AnalyzedFunctionParams{
				List: node.Parameters,
				Span: node.ParamSpan,
			},
			ReturnType: node.ReturnType,
			Body:       node.Body,
			Modifier:   pAst.FN_MODIFIER_NONE,
			Range:      node.Range,
		})
		self.currFn = oldCurrFn

		self.insert(newValueInstruction(Opcode_Copy_Push, *value.NewValueVMFunction(fnName)), node.Span())
	case ast.GroupedExpressionKind:
		node := node.(ast.AnalyzedGroupedExpression)
		self.compileExpr(node.Inner)
	case ast.PrefixExpressionKind:
		node := node.(ast.AnalyzedPrefixExpression)
		self.compileExpr(node.Base)
		self.compilePrefixOp(node.Operator, node.Range)
	case ast.InfixExpressionKind:
		node := node.(ast.AnalyzedInfixExpression)
		self.compileInfixExpr(node)
	case ast.AssignExpressionKind:
		node := node.(ast.AnalyzedAssignExpression)

		// TODO: implement all different assignment operators
		if node.Lhs.Kind() == ast.IdentExpressionKind {
			lhs := node.Lhs.(ast.AnalyzedIdentExpression)
			name, found := self.getMangled(lhs.Ident.Ident())
			if !found {
				name = lhs.Ident.Ident()
			}

			opCodeGet := Opcode_GetVarImm
			opCodeSet := Opcode_SetVarImm

			if lhs.IsGlobal {
				opCodeGet = Opcode_GetGlobImm
				opCodeSet = Opcode_SetGlobImm
			}

			if node.Operator != pAst.StdAssignOperatorKind {
				self.insert(newOneStringInstruction(opCodeGet, name), node.Range)
				self.compileExpr(node.Rhs)
				self.arithmeticHelper(node.Operator.IntoInfixOperator(), node.Range)
			} else {
				self.compileExpr(node.Rhs)
			}

			self.insert(newOneStringInstruction(opCodeSet, name), node.Range)
		} else {
			self.compileExpr(node.Lhs)

			if node.Operator != pAst.StdAssignOperatorKind {
				self.insert(newPrimitiveInstruction(Opcode_Duplicate), node.Range)
				self.compileExpr(node.Rhs)
				self.arithmeticHelper(node.Operator.IntoInfixOperator(), node.Range)
			} else {
				self.compileExpr(node.Rhs)
			}

			self.insert(newPrimitiveInstruction(Opcode_Assign), node.Range)
		}
	case ast.CallExpressionKind:
		node := node.(ast.AnalyzedCallExpression)
		self.compileCallExpr(node)
	case ast.IndexExpressionKind:
		node := node.(ast.AnalyzedIndexExpression)
		self.compileExpr(node.Base)
		self.compileExpr(node.Index)
		self.insert(newPrimitiveInstruction(Opcode_Index), node.Range)
	case ast.MemberExpressionKind:
		node := node.(ast.AnalyzedMemberExpression)
		self.compileExpr(node.Base)
		self.insert(newOneStringInstruction(Opcode_Member, node.Member.Ident()), node.Range)
	case ast.CastExpressionKind:
		node := node.(ast.AnalyzedCastExpression)
		self.compileExpr(node.Base)
		self.insert(newCastInstruction(node.AsType, true), node.Range)
	case ast.BlockExpressionKind:
		self.compileBlock(node.(ast.AnalyzedBlockExpression).Block, true)
	case ast.IfExpressionKind:
		self.compileIfExpr(node.(ast.AnalyzedIfExpression))
	case ast.MatchExpressionKind:
		node := node.(ast.AnalyzedMatchExpression)

		// push the control value onto the stack
		self.compileExpr(node.ControlExpression)

		branches := make(map[int]string)
		after_branch := self.mangleLabel("match_after")

		for i, option := range node.Arms {
			name := self.mangleLabel("case")
			branches[i] = name

			// Insert value to compare with
			self.compileExpr(option.Literal)

			// Compare control and branch value
			// TODO: could DUP also work?
			self.insert(newPrimitiveInstruction(Opcode_Eq_PopOnce), node.Range)

			// if true, jump to the label of this branch
			self.insert(newPrimitiveInstruction(Opcode_Not), node.Range)
			self.insert(newOneStringInstruction(Opcode_JumpIfFalse, name), node.Range)
		}

		default_branch := self.mangleLabel("match_default")
		if node.DefaultArmAction != nil {
			self.insert(newOneStringInstruction(Opcode_Jump, default_branch), node.Range)
		} else {
			self.insert(newOneStringInstruction(Opcode_Jump, after_branch), node.Range)
		}

		// Each individual branch
		for i, option := range node.Arms {
			self.insert(newOneStringInstruction(Opcode_Label, branches[i]), node.Range)
			// Insert a `drop` since a eq_poponce was used
			self.insert(newPrimitiveInstruction(Opcode_Drop), node.Range)
			self.compileExpr(option.Action)
			self.insert(newOneStringInstruction(Opcode_Jump, after_branch), node.Range)
		}

		if node.DefaultArmAction != nil {
			self.insert(newOneStringInstruction(Opcode_Label, default_branch), node.Range)
			self.compileExpr(*node.DefaultArmAction)
			self.insert(newOneStringInstruction(Opcode_Jump, after_branch), node.Range)
		}

		self.insert(newOneStringInstruction(Opcode_Label, after_branch), node.Range)
	case ast.TryExpressionKind:
		mangledCurr, found := self.getMangledFn(self.currFn)
		if !found {
			panic("Impossible state: every current function should also be found")
		}

		node := node.(ast.AnalyzedTryExpression)
		exceptionLabel := self.mangleLabel("exception_label")
		afterCatchLabel := self.mangleLabel("after_catch_label")
		self.insert(newTwoStringInstruction(Opcode_SetTryLabel, mangledCurr, exceptionLabel), node.Range)
		self.compileBlock(node.TryBlock, true)
		self.insert(newPrimitiveInstruction(Opcode_PopTryLabel), node.Range)
		self.insert(newOneStringInstruction(Opcode_Jump, afterCatchLabel), node.Range)

		// exception case
		mangledExceptionName := self.mangleVar(node.CatchIdent.Ident())
		self.insert(newOneStringInstruction(Opcode_Label, exceptionLabel), node.Range)
		self.pushScope()
		defer self.popScope()
		self.insert(newOneStringInstruction(Opcode_SetVarImm, mangledExceptionName), node.Range)
		self.insert(newPrimitiveInstruction(Opcode_PopTryLabel), node.Range)
		self.compileBlock(node.CatchBlock, false)
		self.insert(newOneStringInstruction(Opcode_Label, afterCatchLabel), node.Range)
	default:
		panic("Unreachable")
	}
}
