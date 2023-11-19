package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

type Loop struct {
	labelStart    string
	labelBreak    string
	labelContinue string
}

type Compiler struct {
	functions     map[string][]Instruction
	currFn        string
	loops         []Loop
	varNameMangle map[string]uint64
	varScopes     []map[string]string
	currScope     *map[string]string
}

func NewCompiler() Compiler {
	return Compiler{
		functions:     make(map[string][]Instruction),
		loops:         make([]Loop, 0),
		varNameMangle: make(map[string]uint64),
		varScopes:     make([]map[string]string, 0),
		currScope:     nil,
	}
}

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

func (self *Compiler) mangle(input string) string {
	cnt, exists := self.varNameMangle[input]
	if !exists {
		self.varNameMangle[input]++
		cnt = 0
	}

	mangled := fmt.Sprintf("%s%d", input, cnt)
	(*self.currScope)[input] = mangled

	return mangled
}

func (self Compiler) getMangled(input string) (string, bool) {
	for _, scope := range self.varScopes {
		name, found := scope[input]
		if found {
			return name, true
		}
	}

	return "", false
}

func (self *Compiler) Compile(program ast.AnalyzedProgram) map[string][]Instruction {
	self.compileProgram(program)
	return self.functions
}

func (self *Compiler) insert(instruction Instruction) {
	self.functions[self.currFn] = append(self.functions[self.currFn], instruction)
}

func (self *Compiler) compileProgram(program ast.AnalyzedProgram) {
	// TODO: handle imports to also compile imported modules
	for _, item := range program.Imports {
		fmt.Println(item)
	}

	// compile all functions
	for _, fn := range program.Functions {
		self.compileFn(fn)
	}

	// compile all global functions
	for _, global := range program.Globals {
		fmt.Println(global)
	}
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

	after_label := self.mangle("if_after")
	else_label := self.mangle("else")

	if node.ElseBlock != nil {
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, else_label))
	} else {
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, after_label))
	}
	self.compileBlock(node.ThenBlock, true)
	self.insert(newOneStringInstruction(Opcode_Jump, after_label))

	if node.ElseBlock != nil {
		self.insert(newOneStringInstruction(Opcode_Label, else_label))
		self.compileBlock(*node.ElseBlock, true)
	}
	self.insert(newOneStringInstruction(Opcode_Label, after_label))
}

func (self *Compiler) compileFn(node ast.AnalyzedFunctionDefinition) {
	// set current function
	self.currFn = node.Ident.Ident()

	self.pushScope()
	defer self.popScope()

	for i := len(node.Parameters) - 1; i >= 0; i-- {
		name := self.mangle(node.Parameters[i].Ident.Ident())
		self.insert(newOneStringInstruction(Opcode_SetVatImm, name))
	}

	self.compileBlock(node.Body, false)
}

func (self *Compiler) compileStmt(node ast.AnalyzedStatement) {
	switch node.Kind() {
	case ast.TypeDefinitionStatementKind:
		panic("This should not be reachable!")
	case ast.LetStatementKind:
		node := node.(ast.AnalyzedLetStatement)

		// push value onto the stack
		self.compileExpr(node.Expression)

		// bind value to identifier
		name := self.mangle(node.Ident.Ident())
		self.insert(newOneStringInstruction(Opcode_SetVatImm, name)) // TODO: mangle
	case ast.ReturnStatementKind:
		self.insert(newPrimitiveInstruction(Opcode_Return))
	case ast.BreakStatementKind:
		self.insert(newOneStringInstruction(Opcode_Jump, self.currLoop().labelBreak))
	case ast.ContinueStatementKind:
		self.insert(newOneStringInstruction(Opcode_Jump, self.currLoop().labelContinue))
	case ast.LoopStatementKind:
		node := node.(ast.AnalyzedLoopStatement)

		head_label := self.mangle("loop_head")
		after_label := self.mangle("loop_end")
		self.insert(newOneStringInstruction(Opcode_Label, head_label))

		self.pushLoop(Loop{
			labelStart:    head_label,
			labelBreak:    after_label,
			labelContinue: head_label,
		})
		self.compileBlock(node.Body, true)
		self.insert(newOneStringInstruction(Opcode_Label, head_label))
		self.popLoop()
	case ast.WhileStatementKind:
		node := node.(ast.AnalyzedWhileStatement)
		head_label := self.mangle("loop_head")
		after_label := self.mangle("loop_end")
		self.pushLoop(Loop{
			labelStart:    head_label,
			labelBreak:    after_label,
			labelContinue: head_label,
		})

		self.insert(newOneStringInstruction(Opcode_Label, head_label))

		// check the condition
		self.compileExpr(node.Condition)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, after_label))

		self.compileBlock(node.Body, true)

		self.insert(newOneStringInstruction(Opcode_Label, after_label))
		self.popLoop()
	case ast.ForStatementKind:
		node := node.(ast.AnalyzedForStatement)

		head_label := self.mangle("loop_head")
		update_label := self.mangle("loop_update")
		after_label := self.mangle("loop_end")

		self.pushLoop(Loop{
			labelStart:    head_label,
			labelBreak:    after_label,
			labelContinue: update_label,
		})

		// push iter expr onto the stack
		self.compileExpr(node.IterExpression)

		// bind its initial value to the iter variable
		self.pushScope()
		defer self.popScope()

		name := self.mangle(node.Identifier.Ident())
		self.insert(newOneStringInstruction(Opcode_SetVatImm, name))

		// check the condition
		self.insert(newOneStringInstruction(Opcode_Label, head_label))
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, after_label))

		self.compileBlock(node.Body, false)

		self.insert(newOneStringInstruction(Opcode_Label, update_label))
		// TODO: update

		self.insert(newOneStringInstruction(Opcode_Label, after_label))
	case ast.ExpressionStatementKind:
		node := node.(ast.AnalyzedExpressionStatement)
		self.compileExpr(node.Expression)
		// Drop every value that the expression might generate
		self.insert(newPrimitiveInstruction(Opcode_Drop))
	default:
		panic("Unreachable")
	}
}

func (self *Compiler) compilePrefixOp(op ast.PrefixOperator) {
	switch op {
	case ast.MinusPrefixOperator:
		// This performs a `* (-1)` operation
		self.insert(newValueInstruction(Opcode_Push, IntValue{Value: -1}))
		self.insert(newPrimitiveInstruction(Opcode_Mul))
	case ast.NegatePrefixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Neg))
	case ast.IntoSomePrefixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Some))
	}
}

func (self *Compiler) compileCallExpr(node ast.AnalyzedCallExpression) {
	// push each arg onto the stack
	for _, arg := range node.Arguments {
		self.compileExpr(arg.Expression)
	}

	if node.Base.Kind() == ast.IdentExpressionKind {
		base := node.Base.(ast.AnalyzedIdentExpression)

		mangled, found := self.getMangled(base.Ident.Ident())
		if found {
			self.insert(newOneStringInstruction(Opcode_Call_Imm, mangled))
		} else {
			self.insert(newOneStringInstruction(Opcode_HostCall, base.Ident.Ident()))
		}
	} else {
		// TODO: wtf: compile the base
		self.compileExpr(node.Base)

		// perform the actual call
		self.insert(newPrimitiveInstruction(Opcode_Call_Val))
	}
}

func (self *Compiler) compileInfixExpr(node ast.AnalyzedInfixExpression) {
	if node.Operator != pAst.LogicalAndInfixOperator && node.Operator != pAst.LogicalOrInfixOperator {
		self.compileExpr(node.Lhs)
		self.compileExpr(node.Rhs)
	}

	switch node.Operator {
	case pAst.PlusInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Add))
	case pAst.MinusInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Sub))
	case pAst.MultiplyInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Mul))
	case pAst.DivideInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Div))
	case pAst.ModuloInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Rem))
	case pAst.PowerInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Pow))
	case pAst.ShiftLeftInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Shl))
	case pAst.ShiftRightInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Shr))
	case pAst.BitOrInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_BitOr))
	case pAst.BitAndInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_BitAnd))
	case pAst.BitXorInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_BitXor))
	case pAst.LogicalOrInfixOperator:
		// TODO: implement branching stuff
		fallthrough
	case pAst.LogicalAndInfixOperator:
		// TODO: implement branching stuff

		// TODO: implement this
		self.compileExpr(node.Lhs)
		self.compileExpr(node.Rhs)

		panic("TODO")
	case pAst.EqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Eq))
	case pAst.NotEqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Eq))
		self.insert(newPrimitiveInstruction(Opcode_Neg))
	case pAst.LessThanInfixOperator: // TODO: make this more RISC-y
		self.insert(newPrimitiveInstruction(Opcode_Lt))
	case pAst.LessThanEqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Le))
	case pAst.GreaterThanInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Gt))
	case pAst.GreaterThanEqualInfixOperator:
		self.insert(newPrimitiveInstruction(Opcode_Ge))
	default:
		panic("Unreachable")
	}
}

func (self *Compiler) compileExpr(node ast.AnalyzedExpression) {
	switch node.Kind() {
	case ast.UnknownExpressionKind:
		panic("WTF")
	case ast.IntLiteralExpressionKind:
		node := node.(ast.AnalyzedIntLiteralExpression)
		self.insert(newValueInstruction(Opcode_Push, IntValue{Value: node.Value}))
	case ast.FloatLiteralExpressionKind:
		node := node.(ast.AnalyzedFloatLiteralExpression)
		self.insert(newValueInstruction(Opcode_Push, FloatValue{Value: node.Value}))
	case ast.BoolLiteralExpressionKind:
		node := node.(ast.AnalyzedBoolLiteralExpression)
		self.insert(newValueInstruction(Opcode_Push, BoolValue{Value: node.Value}))
	case ast.StringLiteralExpressionKind:
		node := node.(ast.AnalyzedStringLiteralExpression)
		self.insert(newValueInstruction(Opcode_Push, StringValue{Value: node.Value}))
	case ast.IdentExpressionKind:
		// TODO: must find out if global.

		node := node.(ast.AnalyzedIdentExpression)
		self.insert(newOneStringInstruction(Opcode_GetVarImm, node.Ident.Ident()))
	case ast.NullLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Push, NullValue{}))
	case ast.NoneLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Push, OptionValue{inner: nil}))
	case ast.RangeLiteralExpressionKind:
		panic("TODO: value")
	case ast.ListLiteralExpressionKind:
		panic("TODO: value")
	case ast.AnyObjectLiteralExpressionKind:
		panic("TODO: value")
	case ast.ObjectLiteralExpressionKind:
		panic("TODO: value")
	case ast.FunctionLiteralExpressionKind:
		panic("TODO: value")
	case ast.GroupedExpressionKind:
		node := node.(ast.AnalyzedGroupedExpression)
		self.compileExpr(node.Inner)
	case ast.PrefixExpressionKind:
		node := node.(ast.AnalyzedPrefixExpression)
		self.compileExpr(node.Base)
		self.compilePrefixOp(node.Operator)
	case ast.InfixExpressionKind:
		node := node.(ast.AnalyzedInfixExpression)
		self.compileInfixExpr(node)
	case ast.AssignExpressionKind:
	case ast.CallExpressionKind:
		node := node.(ast.AnalyzedCallExpression)
		self.compileCallExpr(node)
	case ast.IndexExpressionKind:
		node := node.(ast.AnalyzedIndexExpression)
		self.compileExpr(node.Base)
		self.compileExpr(node.Index)
		self.insert(newPrimitiveInstruction(Opcode_Index))
	case ast.MemberExpressionKind:
		node := node.(ast.AnalyzedMemberExpression)
		self.compileExpr(node.Base)
		self.insert(newOneStringInstruction(Opcode_Member, node.Member.Ident())) // TODO: add monomorphization
	case ast.CastExpressionKind:
		node := node.(ast.AnalyzedCastExpression)
		self.compileExpr(node.Base)
		self.insert(newCastInstruction(node.AsType)) // TODO: add monomorphization
	case ast.BlockExpressionKind:
		self.compileBlock(node.(ast.AnalyzedBlockExpression).Block, true)
	case ast.IfExpressionKind:
		self.compileIfExpr(node.(ast.AnalyzedIfExpression))
	case ast.MatchExpressionKind:
		// TODO: match
	case ast.TryExpressionKind:
		node := node.(ast.AnalyzedTryExpression)
		// TODO: name mangling
		exceptionLabel := "exception_label"
		afterTryLabel := "after_try_label"
		self.insert(newOneStringInstruction(Opcode_SetTryLabel, exceptionLabel))
		self.compileBlock(node.TryBlock, true)
		self.insert(newOneStringInstruction(Opcode_Jump, afterTryLabel))
		self.insert(newOneStringInstruction(Opcode_Label, exceptionLabel))

		self.pushScope()
		defer self.popScope()

		self.insert(newOneStringInstruction(Opcode_SetVatImm, node.CatchIdent.Ident())) // TODO: mangle names
		self.compileBlock(node.CatchBlock, false)
	default:
		panic("Unreachable")
	}
}
