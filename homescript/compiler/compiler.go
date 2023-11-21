package compiler

import (
	"fmt"

	"github.com/smarthome-go/homescript/v3/homescript/analyzer/ast"
	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/interpreter/value"
	pAst "github.com/smarthome-go/homescript/v3/homescript/parser/ast"
)

type Loop struct {
	labelStart    string
	labelBreak    string
	labelContinue string
}

type Function struct {
	MangledName  string
	Instructions []Instruction
	// TODO: disable these in an elegant manner
	SourceMap []errors.Span
}

type Compiler struct {
	functions     map[string]*Function
	currFn        string
	loops         []Loop
	fnNameMangle  map[string]uint64
	varNameMangle map[string]uint64
	varScopes     []map[string]string
	currScope     *map[string]string
}

func NewCompiler() Compiler {
	scopes := make([]map[string]string, 1)
	scopes[0] = make(map[string]string)
	currScope := &scopes[0]

	return Compiler{
		functions:     make(map[string]*Function),
		loops:         make([]Loop, 0),
		fnNameMangle:  make(map[string]uint64),
		varNameMangle: make(map[string]uint64),
		varScopes:     scopes,
		currScope:     currScope,
	}
}

const (
	LIST_PUSH = "__internal_list_push"
)

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
	cnt, exists := self.varNameMangle[input]
	if !exists {
		self.varNameMangle[input]++
		cnt = 0
	}

	mangled := fmt.Sprintf("%s%d", input, cnt)

	fn := Function{
		MangledName:  mangled,
		Instructions: make([]Instruction, 0),
		SourceMap:    make([]errors.Span, 0),
	}
	self.functions[input] = &fn
	return mangled
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

func (self Compiler) getMangledFn(input string) (string, bool) {
	for key, fn := range self.functions {
		if key == input {
			return fn.MangledName, true
		}
	}

	return "", false
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

func (self Compiler) relocateLabels() {
	for name, fn := range self.functions {
		labels := make(map[string]int64)

		fnOut := make([]Instruction, 0)
		sourceMapOut := make([]errors.Span, 0)

		index := 0
		for idx, inst := range fn.Instructions {
			if inst.Opcode() == Opcode_Label {
				i := inst.(OneStringInstruction).Value
				labels[i] = int64(index)
				fmt.Printf("I: %v | IP: %d\n", inst, index)
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
				fnOut[idx] = newOneIntInstruction(inst.Opcode(), labels[i.Value])
				fmt.Printf(":: :: Patched %v -> %v\n", inst, fnOut[idx])
			case Opcode_Label:
				panic("This should not happen")
			}
		}

		self.functions[name].Instructions = fnOut
		self.functions[name].SourceMap = sourceMapOut
	}
}

func (self *Compiler) Compile(program ast.AnalyzedProgram) Program {
	self.compileProgram(program)

	self.relocateLabels()

	functions := make(map[string][]Instruction)
	sourceMap := make(map[string][]errors.Span)

	for _, fn := range self.functions {
		functions[fn.MangledName] = fn.Instructions
		sourceMap[fn.MangledName] = fn.SourceMap
	}

	return Program{
		Functions: functions,
		SourceMap: sourceMap,
	}
}

func (self *Compiler) insert(instruction Instruction, span errors.Span) {
	// fmt.Printf("fn: `%s` %v\n", self.currFn, self.functions[self.currFn])
	self.functions[self.currFn].Instructions = append(self.functions[self.currFn].Instructions, instruction)
	self.functions[self.currFn].SourceMap = append(self.functions[self.currFn].SourceMap, span)
}

func (self *Compiler) compileProgram(program ast.AnalyzedProgram) {
	// TODO: handle imports to also compile imported modules

	for _, item := range program.Imports {
		for _, importItem := range item.ToImport {
			self.insert(newTwoStringInstruction(Opcode_Import, item.FromModule.Ident(), importItem.Ident.Ident()), item.Range)
		}
	}

	// compile all globals
	initFn := ast.AnalyzedFunctionDefinition{
		Ident:      pAst.SpannedIdent{},
		Parameters: make([]ast.AnalyzedFnParam, 0),
		ReturnType: ast.NewNullType(),
		Body:       ast.AnalyzedBlock{},
		Modifier:   0,
		Range:      errors.Span{},
	}

	for _, glob := range program.Globals {
		initFn
	}

	// compile all function declarations
	for _, fn := range program.Functions {
		self.mangleFn(fn.Ident.Ident())
	}

	// compile all functions
	for _, fn := range program.Functions {
		self.compileFn(fn)
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
	// set current function
	self.currFn = node.Ident.Ident()

	self.pushScope()
	defer self.popScope()

	for i := len(node.Parameters) - 1; i >= 0; i-- {
		name := self.mangle(node.Parameters[i].Ident.Ident())
		self.insert(newOneStringInstruction(Opcode_SetVatImm, name), node.Range)
	}

	self.compileBlock(node.Body, false)
}

func (self *Compiler) compileLetStmt(node ast.AnalyzedLetStatement, isGlobal bool) {
	// TODO: handle deep casts

	// push value onto the stack
	self.compileExpr(node.Expression)

	// TODO: handle global
	opcode := Opcode_SetVatImm
	if isGlobal {
		opcode = Opcode_SetGlobImm
	}

	// bind value to identifier
	name := self.mangle(node.Ident.Ident())
	self.insert(newOneStringInstruction(opcode, name), node.Range) // TODO: mangle
}

func (self *Compiler) compileStmt(node ast.AnalyzedStatement) {
	switch node.Kind() {
	case ast.TypeDefinitionStatementKind:
		panic("This should not be reachable!")
	case ast.LetStatementKind:
		self.compileLetStmt(node.(ast.AnalyzedLetStatement), false)
	case ast.ReturnStatementKind:
		self.insert(newPrimitiveInstruction(Opcode_Return), node.Span())
	case ast.BreakStatementKind:
		self.insert(newOneStringInstruction(Opcode_Jump, self.currLoop().labelBreak), node.Span())
	case ast.ContinueStatementKind:
		self.insert(newOneStringInstruction(Opcode_Jump, self.currLoop().labelContinue), node.Span())
	case ast.LoopStatementKind:
		node := node.(ast.AnalyzedLoopStatement)

		head_label := self.mangle("loop_head")
		after_label := self.mangle("loop_end")
		self.insert(newOneStringInstruction(Opcode_Label, head_label), node.Span())

		self.pushLoop(Loop{
			labelStart:    head_label,
			labelBreak:    after_label,
			labelContinue: head_label,
		})
		self.compileBlock(node.Body, true)
		self.insert(newOneStringInstruction(Opcode_Jump, head_label), node.Span())
		self.insert(newOneStringInstruction(Opcode_Label, after_label), node.Span())
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

		self.insert(newOneStringInstruction(Opcode_Label, head_label), node.Range)

		// check the condition
		self.compileExpr(node.Condition)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, after_label), node.Range)

		self.compileBlock(node.Body, true)
		self.insert(newOneStringInstruction(Opcode_Jump, head_label), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, after_label), node.Range)
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
		self.insert(newOneStringInstruction(Opcode_SetVatImm, name), node.Range)

		// check the condition
		self.insert(newOneStringInstruction(Opcode_Label, head_label), node.Range)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, after_label), node.Range)

		self.compileBlock(node.Body, false)
		self.insert(newOneStringInstruction(Opcode_Jump, head_label), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, update_label), node.Range)
		// TODO: update

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
	// push each arg onto the stack
	for _, arg := range node.Arguments {
		self.compileExpr(arg.Expression)
	}

	if node.Base.Kind() == ast.IdentExpressionKind {
		base := node.Base.(ast.AnalyzedIdentExpression)

		mangled, found := self.getMangled(base.Ident.Ident())
		if found {
			if node.IsSpawn {
				panic("This is an impossible state.")
			}

			self.insert(newOneStringInstruction(Opcode_Call_Imm, mangled), node.Span())
		} else {
			name, found := self.getMangledFn(base.Ident.Ident())
			if found {
				opcode := Opcode_Call_Imm
				if node.IsSpawn {
					opcode = Opcode_Spawn
				}

				self.insert(newOneStringInstruction(opcode, name), node.Span())
			} else {
				if node.IsSpawn {
					panic("This is an impossible state.")
				}

				// insert number of args
				self.insert(newValueInstruction(Opcode_Push, *value.NewValueInt(int64(len(node.Arguments)))), node.Span())
				// perform actual call
				self.insert(newOneStringInstruction(Opcode_HostCall, base.Ident.Ident()), node.Span())
			}
		}
	} else {
		if node.IsSpawn {
			panic("This is an impossible state.")
		}

		// TODO: wtf: compile the base
		self.compileExpr(node.Base)

		// insert number of args
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueInt(int64(len(node.Arguments)))), node.Span())

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
	case pAst.LessThanInfixOperator: // TODO: make this more RISC-y
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
		returnTrue := self.mangle("return_true")
		afterLabel := self.mangle("after_infix")

		self.compileExpr(node.Lhs)
		self.insert(newPrimitiveInstruction(Opcode_Not), node.Range)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, returnTrue), node.Range)

		self.compileExpr(node.Rhs)
		self.insert(newOneStringInstruction(Opcode_Jump, afterLabel), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, returnTrue), node.Range)
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueBool(true)), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, afterLabel), node.Range)
	case pAst.LogicalAndInfixOperator:
		returnFalse := self.mangle("return_false")
		afterLabel := self.mangle("after_infix")

		self.compileExpr(node.Lhs)
		self.insert(newOneStringInstruction(Opcode_JumpIfFalse, returnFalse), node.Range)

		self.compileExpr(node.Rhs)
		self.insert(newOneStringInstruction(Opcode_Jump, afterLabel), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, returnFalse), node.Range)
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueBool(false)), node.Range)

		self.insert(newOneStringInstruction(Opcode_Label, afterLabel), node.Range)
	default:
		self.compileExpr(node.Lhs)
		self.compileExpr(node.Rhs)
		self.arithmeticHelper(node.Operator, node.Range)
	}
}

func (self *Compiler) compileExpr(node ast.AnalyzedExpression) {
	switch node.Kind() {
	case ast.UnknownExpressionKind:
		panic("Unreachable, this should not happen")
	case ast.IntLiteralExpressionKind:
		node := node.(ast.AnalyzedIntLiteralExpression)
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueInt(node.Value)), node.Range)
	case ast.FloatLiteralExpressionKind:
		node := node.(ast.AnalyzedFloatLiteralExpression)
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueFloat(node.Value)), node.Range)
	case ast.BoolLiteralExpressionKind:
		node := node.(ast.AnalyzedBoolLiteralExpression)
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueBool(node.Value)), node.Range)
	case ast.StringLiteralExpressionKind:
		node := node.(ast.AnalyzedStringLiteralExpression)
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueString(node.Value)), node.Range)
	case ast.IdentExpressionKind:
		// TODO: must find out if global.
		node := node.(ast.AnalyzedIdentExpression)
		name, found := self.getMangled(node.Ident.Ident())

		opCode := Opcode_GetVarImm
		// TODO: i hope this does not break
		if node.IsGlobal {
			opCode = Opcode_GetGlobImm
		}

		if found {
			self.insert(newOneStringInstruction(opCode, name), node.Span())
		} else {
			name, found := self.getMangledFn(node.Ident.Ident())
			if found {
				self.insert(newOneStringInstruction(opCode, name), node.Span())
			} else {
				self.insert(newOneStringInstruction(opCode, node.Ident.Ident()), node.Span())
			}
		}
	case ast.NullLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueNull()), node.Span())
	case ast.NoneLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Push, *value.NewNoneOption()), node.Span())
	case ast.RangeLiteralExpressionKind:
		// TODO: eliminate ranges at compile time
		node := node.(ast.AnalyzedRangeLiteralExpression)
		self.compileExpr(node.Start)
		self.compileExpr(node.End)
		self.insert(newPrimitiveInstruction(Opcode_Into_Range), node.Range)
	case ast.ListLiteralExpressionKind:
		node := node.(ast.AnalyzedListLiteralExpression)
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueList(make([]*value.Value, 0))), node.Range)

		for _, element := range node.Values {
			self.compileExpr(element)
			self.insert(newValueInstruction(Opcode_Push, *value.NewValueInt(1)), node.Range)
			self.insert(newOneStringInstruction(Opcode_HostCall, LIST_PUSH), node.Range)
		}
	case ast.AnyObjectLiteralExpressionKind:
		self.insert(newValueInstruction(Opcode_Push, *value.NewValueAnyObject(make(map[string]*value.Value))), node.Span())
	case ast.ObjectLiteralExpressionKind:
		node := node.(ast.AnalyzedObjectLiteralExpression)

		fields := make(map[string]*value.Value)
		for _, field := range node.Fields {
			fields[field.Key.Ident()] = value.ZeroValue(field.Expression.Type())
		}

		object := *value.NewValueObject(fields)
		self.insert(newValueInstruction(Opcode_Push, object), node.Range)

		for _, field := range node.Fields {
			self.insert(newPrimitiveInstruction(Opcode_Duplicate), node.Range)
			self.insert(newOneStringInstruction(Opcode_Member, field.Key.Ident()), node.Range)
			self.compileExpr(field.Expression)
			self.insert(newPrimitiveInstruction(Opcode_Assign), node.Range)
		}
	case ast.FunctionLiteralExpressionKind:
		panic("TODO: function value")
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

		// TODO: assignment operators

		// TODO: handle other types of LHS
		if node.Lhs.Kind() == ast.IdentExpressionKind {
			lhs := node.Lhs.(ast.AnalyzedIdentExpression)
			name, found := self.getMangled(lhs.Ident.Ident())
			if !found {
				name = lhs.Ident.Ident()
			}

			if node.Operator != pAst.StdAssignOperatorKind {
				self.insert(newOneStringInstruction(Opcode_GetVarImm, name), node.Range)
				self.compileExpr(node.Rhs)
				self.arithmeticHelper(node.Operator.IntoInfixOperator(), node.Range)
			} else {
				self.compileExpr(node.Rhs)
			}

			self.insert(newOneStringInstruction(Opcode_SetVatImm, name), node.Range)
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
		self.insert(newOneStringInstruction(Opcode_Member, node.Member.Ident()), node.Range) // TODO: add monomorphization
	case ast.CastExpressionKind:
		node := node.(ast.AnalyzedCastExpression)
		self.compileExpr(node.Base)
		self.insert(newCastInstruction(node.AsType), node.Range) // TODO: add monomorphization
	case ast.BlockExpressionKind:
		self.compileBlock(node.(ast.AnalyzedBlockExpression).Block, true)
	case ast.IfExpressionKind:
		self.compileIfExpr(node.(ast.AnalyzedIfExpression))
	case ast.MatchExpressionKind:
		node := node.(ast.AnalyzedMatchExpression)

		// push the control value onto the stack
		self.compileExpr(node.ControlExpression)

		branches := make(map[int]string)
		after_branch := self.mangle("match_after")

		for i, option := range node.Arms {
			name := self.mangle(fmt.Sprintf("case_%d", i))
			branches[i] = name

			// Insert value to compare with
			self.compileExpr(option.Literal)

			// Compare control and branch value
			self.insert(newPrimitiveInstruction(Opcode_Eq_PopOnce), node.Range)

			// if true, jump to the label of this branch
			self.insert(newPrimitiveInstruction(Opcode_Not), node.Range)
			self.insert(newOneStringInstruction(Opcode_JumpIfFalse, name), node.Range)
		}

		default_branch := self.mangle("match_default")
		if node.DefaultArmAction != nil {
			self.insert(newOneStringInstruction(Opcode_Jump, default_branch), node.Range)
		} else {
			self.insert(newOneStringInstruction(Opcode_Jump, after_branch), node.Range)
		}

		// Each individual branch
		for i, option := range node.Arms {
			self.insert(newOneStringInstruction(Opcode_Label, branches[i]), node.Range)
			self.compileExpr(option.Action)
			self.insert(newOneStringInstruction(Opcode_Jump, after_branch), node.Range)
		}

		if node.DefaultArmAction != nil {
			self.insert(newOneStringInstruction(Opcode_Label, default_branch), node.Range)
			self.compileExpr(*node.DefaultArmAction)
			self.insert(newOneStringInstruction(Opcode_Jump, after_branch), node.Range)
		}
	case ast.TryExpressionKind:
		node := node.(ast.AnalyzedTryExpression)
		// TODO: name mangling
		exceptionLabel := self.mangle("exception_label")
		afterTryLabel := self.mangle("after_try_label")
		self.insert(newOneStringInstruction(Opcode_SetTryLabel, exceptionLabel), node.Range)
		self.compileBlock(node.TryBlock, true)
		self.insert(newOneStringInstruction(Opcode_Jump, afterTryLabel), node.Range)
		self.insert(newOneStringInstruction(Opcode_Label, exceptionLabel), node.Range)

		self.pushScope()
		defer self.popScope()

		self.insert(newOneStringInstruction(Opcode_SetVatImm, node.CatchIdent.Ident()), node.Range) // TODO: mangle names
		self.compileBlock(node.CatchBlock, false)
	default:
		panic("Unreachable")
	}
}
