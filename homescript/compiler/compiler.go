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
	functions map[string][]Instruction
	currFn    string
	loops     []Loop
}

func NewCompiler() Compiler {
	return Compiler{
		functions: make(map[string][]Instruction),
	}
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

func (self *Compiler) compileBlock(node ast.AnalyzedBlock) {

	for _, stmt := range node.Statements {
		self.compileStmt(stmt)
	}

	if node.Expression != nil {
		self.compileExpr(node.Expression)
	}
}

func (self *Compiler) compileFn(node ast.AnalyzedFunctionDefinition) {
	// TODO: implement name mangling correctly

	// set current function
	self.currFn = node.Ident.Ident()

	// TODO: handle args

	self.compileBlock(node.Body)
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
		self.insert(newOneStringInstruction(Opcode_SetVatImm, node.Ident.Ident())) // TODO: mangle
	case ast.ReturnStatementKind:
		self.insert(newPrimitiveInstruction(Opcode_Return))
	case ast.BreakStatementKind:
		panic("Loop compilation")
	case ast.ContinueStatementKind:
		panic("Loop compilation")
	case ast.LoopStatementKind:
		panic("Loop compilation")
	case ast.WhileStatementKind:
		node := node.(ast.AnalyzedWhileStatement)
		self.loops = append(self.loops, Loop{
			labelStart:    "loop-head0",
			labelBreak:    "loop-break0",
			labelContinue: "loop-head0",
		})

		self.insert(newOneStringInstruction(Opcode_Label, self.loops[len(self.loops)-1].labelStart))

		// check the condition
		self.compileExpr(node.Condition)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, self.loops[len(self.loops)-1].labelBreak)) // TODO: allow nested loops

		self.compileBlock(node.Body)

		self.insert(newOneStringInstruction(Opcode_Label, self.loops[len(self.loops)-1].labelBreak))

		// remove loop again
		self.loops = self.loops[:len(self.loops)-1]

	case ast.ForStatementKind:
		node := node.(ast.AnalyzedForStatement)
		self.loops = append(self.loops, Loop{
			labelStart:    "loop-head0",
			labelBreak:    "loop-break0",
			labelContinue: "loop-continue0",
		})

		// push iter expr onto the stack
		self.compileExpr(node.IterExpression)

		// bind its initial value to the iter variable
		self.insert(newOneStringInstruction(Opcode_SetVatImm, node.Identifier.Ident())) // TODO: name mangling

		self.insert(newOneStringInstruction(Opcode_Label, self.loops[len(self.loops)-1].labelStart))

		// check the condition
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, self.loops[len(self.loops)-1].labelBreak)) // TODO: allow nested loops

		self.insert(newOneStringInstruction(Opcode_Label, self.loops[len(self.loops)-1].labelContinue))

		// update

		self.insert(newOneStringInstruction(Opcode_Label, self.loops[len(self.loops)-1].labelBreak))
	case ast.ExpressionStatementKind:
		node := node.(ast.AnalyzedExpressionStatement)
		self.compileExpr(node.Expression)
		// Drop every value that the expression might generate
		self.insert(newPrimitiveInstruction(Opcode_Drop))
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

	// TODO: wtf: compile the base
	self.compileExpr(node.Base)

	// perform the actual call
	self.insert(newPrimitiveInstruction(Opcode_Call))
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
	case ast.MemberExpressionKind:
	case ast.CastExpressionKind:
	case ast.BlockExpressionKind:
	case ast.IfExpressionKind:
	case ast.MatchExpressionKind:
	case ast.TryExpressionKind:
	}
}
