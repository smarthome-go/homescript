package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

//
// Block (expression).
//

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

//
// All Statements.
//

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

//
// Let Statements.
//

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
