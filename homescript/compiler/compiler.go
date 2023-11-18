package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
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

func (self *Compiler) compileFn(node ast.AnalyzedFunctionDefinition) {
	// TODO: implement name mangling correctly

	// set current function
	self.currFn = node.Ident.Ident()

	// TODO: handle args

	for _, stmt := range node.Body.Statements {
		self.compileStmt(stmt)
	}

	if node.Body.Expression != nil {
		self.compileExpr(node.Body.Expression)
	}
}

func (self *Compiler) compileStmt(node ast.AnalyzedStatement) {
	switch node.Kind() {
	case ast.TypeDefinitionStatementKind:
		panic("This should not be reachable!")
	case ast.LetStatementKind:
		// TODO
		panic("Let")
	case ast.ReturnStatementKind:
		self.insert(newPrimitiveInstruction(Opcode_Return))
	case ast.BreakStatementKind:
		panic("Loop compilation")
	case ast.ContinueStatementKind:
		panic("Loop compilation")
	case ast.LoopStatementKind:
		panic("Loop compilation")
	case ast.WhileStatementKind:
		panic("while Loop compilation")
	case ast.ForStatementKind:
		self.loops = append(self.loops, Loop{
			labelStart:    "loop-head0",
			labelBreak:    "loop-break0",
			labelContinue: "loop-continue0",
		})

		self.insert(newOneStringInstruction(Opcode_Label, self.loops[len(self.loops)-1].labelStart))
		// init

		self.insert(newOneStringInstruction(Opcode_Label, self.loops[len(self.loops)-1].labelContinue))
		// update

		self.insert(newOneStringInstruction(Opcode_Label, self.loops[len(self.loops)-1].labelBreak))

		panic("for Loop compilation")
	case ast.ExpressionStatementKind:
		node := node.(ast.AnalyzedExpressionStatement)
		self.compileExpr(node.Expression)
		// Drop every value that the expression might generate
		self.insert(newPrimitiveInstruction(Opcode_Drop))
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
		panic("TODO: handle operator")
		// TODO: operator
	case ast.InfixExpressionKind:
		node := node.(ast.AnalyzedInfixExpression)
		self.compileExpr(node.Lhs)
		self.compileExpr(node.Rhs)
		panic("TODO: handle operator")
		// TODO: operator
	case ast.AssignExpressionKind:
	case ast.CallExpressionKind:
	case ast.IndexExpressionKind:
	case ast.MemberExpressionKind:
	case ast.CastExpressionKind:
	case ast.BlockExpressionKind:
	case ast.IfExpressionKind:
	case ast.MatchExpressionKind:
	case ast.TryExpressionKind:
	}
}
