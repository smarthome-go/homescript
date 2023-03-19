package homescript

import (
	"github.com/smarthome-go/homescript/v2/homescript/errors"
)

type Block struct {
	Statements []Statement
	Expr       *Expression
	Span       errors.Span
}

func (self Block) IntoItemsList() []StatementOrExpr {
	list := make([]StatementOrExpr, 0)

	for _, statement := range self.Statements {
		list = append(list, StatementOrExpr{Statement: statement})
	}

	if self.Expr != nil {
		list = append(list, StatementOrExpr{Expression: self.Expr})
	}

	return list
}

type StatementOrExpr struct {
	Statement  Statement
	Expression *Expression
}

func (self StatementOrExpr) Span() errors.Span {
	if self.Statement != nil {
		return self.Statement.Span()
	} else {
		return self.Expression.Span
	}
}

func (self StatementOrExpr) IsStatement() bool {
	return self.Statement != nil
}

// ///// Statements ///////
type StatementKind uint8

const (
	LetStmtKind StatementKind = iota
	ImportStmtKind
	BreakStmtKind
	ContinueStmtKind
	ReturnStmtKind
	ExpressionStmtKind
)

func (self StatementKind) String() string {
	var value string
	switch self {
	case LetStmtKind:
		value = "let statement"
	case ImportStmtKind:
		value = "import statement"
	case BreakStmtKind:
		value = "break statement"
	case ContinueStmtKind:
		value = "continue statement"
	case ReturnStmtKind:
		value = "return statement"
	case ExpressionStmtKind:
		value = "expression statement"
	default:
		panic("BUG: A new statement kind was introduced without updating this code")
	}
	return value
}

type Statement interface {
	Kind() StatementKind
	Span() errors.Span
}

type LetStmt struct {
	Left struct {
		Identifier string
		Span       errors.Span
	}
	Right Expression
	Range errors.Span
}

func (self LetStmt) Kind() StatementKind { return LetStmtKind }
func (self LetStmt) Span() errors.Span   { return self.Range }

type ImportStmt struct {
	Function   string  // import `foo`
	RewriteAs  *string // as `bar`
	FromModule string  // from `baz`
	Range      errors.Span
}

func (self ImportStmt) Kind() StatementKind { return ImportStmtKind }
func (self ImportStmt) Span() errors.Span   { return self.Range }

type BreakStmt struct {
	Expression *Expression // Can be the return value of the loop
	Range      errors.Span
}

func (self BreakStmt) Kind() StatementKind { return BreakStmtKind }
func (self BreakStmt) Span() errors.Span   { return self.Range }

type ContinueStmt struct {
	Range errors.Span
}

func (self ContinueStmt) Kind() StatementKind { return ContinueStmtKind }
func (self ContinueStmt) Span() errors.Span   { return self.Range }

type ReturnStmt struct {
	Expression *Expression // Can be the return value of the function
	Range      errors.Span
}

func (self ReturnStmt) Kind() StatementKind { return ReturnStmtKind }
func (self ReturnStmt) Span() errors.Span   { return self.Range }

type ExpressionStmt struct {
	Expression Expression
	// Range ommitted because the expression is forwarded here
}

func (self ExpressionStmt) Kind() StatementKind { return ExpressionStmtKind }
func (self ExpressionStmt) Span() errors.Span   { return self.Expression.Span }

/////// Expressions ///////

// Expression
type Expression OrExpression

// Or expression
type OrExpression struct {
	Base      AndExpression
	Following []AndExpression
	Span      errors.Span
}

// And expression
type AndExpression struct {
	Base      EqExpression
	Following []EqExpression
	Span      errors.Span
}

// Equality expression
type EqExpression struct {
	Base  RelExpression
	Other *struct {
		// True corresponds to `!=` and false corresponds to `==`
		Inverted bool
		Node     RelExpression
	}
	Span errors.Span
}

// Relational expression
type RelExpression struct {
	Base  AddExpression
	Other *struct {
		RelOperator RelOperator
		Node        AddExpression
	}
	Span errors.Span
}

type RelOperator uint8

const (
	RelLessThan RelOperator = iota
	RelLessOrEqual
	RelGreaterThan
	RelGreaterOrEqual
)

// Add expression
type AddExpression struct {
	Base      MulExpression
	Following []struct {
		AddOperator AddOperator
		Other       MulExpression
		Span        errors.Span
	}
	Span errors.Span
}

type AddOperator uint8

const (
	AddOpPlus AddOperator = iota
	AddOpMinus
)

// Mul expression
type MulExpression struct {
	Base      CastExpression
	Following []struct {
		MulOperator MulOperator
		Other       CastExpression
		Span        errors.Span
	}
	Span errors.Span
}

type MulOperator uint8

const (
	MulOpMul MulOperator = iota
	MulOpDiv
	MulOpIntDiv
	MulOpReminder
)

// Cast expression
type CastExpression struct {
	Base  UnaryExpression
	Other *ValueType // Casting is optional, otherwise, just the base is used
	Span  errors.Span
}

// Unary expression
// Is either unary or exp expression
// if unary = nil, then exp is not nil
// if exp is nil, then unary is not nil
type UnaryExpression struct {
	UnaryExpression *struct {
		UnaryOp         UnaryOp
		UnaryExpression UnaryExpression
	}
	ExpExpression *ExpExpression
	Span          errors.Span
}

type UnaryOp uint8

const (
	UnaryOpPlus UnaryOp = iota
	UnaryOpMinus
	UnaryOpNot
)

// Exp expression
type ExpExpression struct {
	Base  AssignExpression
	Other *UnaryExpression
	Span  errors.Span
}

// Assign expression
type AssignExpression struct {
	Base  CallExpression
	Other *struct {
		Operator   AssignOperator
		Expression Expression
	}
	Span errors.Span
}

type AssignOperator uint8

const (
	OpAssign AssignOperator = iota
	OpPlusAssign
	OpMinusAssign
	OpMulAssign
	OpDivAssign
	OpIntDivAssign
	OpReminderAssign
	OpPowerAssign
)

// Call expression
type CallExpression struct {
	Base  MemberExpression
	Parts []CallExprPart // Allows chaining of function calls ( like foo()()() )
	Span  errors.Span
}

// If member expr part is nil, args is used
// if args is nil, member expr part is used
type CallExprPart struct {
	MemberExpressionPart *string       // Optional: .identifier as a member
	Args                 *[]Expression // Optional: (arg1, arg2) to call the function
	Index                *Expression   // Optional: [42] to index a value
	Span                 errors.Span
}

// Member expression
type MemberExpression struct {
	Base Atom
	// Each member is either an identifier 'foo.bar.baz' where `foo` is the base and `bar` and `baz` are the members
	// However, each member can also be an index access: `[1]`, where 1 is an expression
	Members []struct {
		// Normal member identifier with a dot
		Identifier *string
		// List index access
		Index *Expression
		// Span is used in order to deliver better error messages
		Span errors.Span
	}
	Span errors.Span
}

///////////// ATOM /////////////

// Atom
type AtomKind uint8

const (
	AtomKindNumber AtomKind = iota
	AtomKindBoolean
	AtomKindString
	AtomKindListLiteral
	AtomKindObject
	AtomKindPair
	AtomKindNull
	AtomKindIdentifier
	AtomKindRange
	AtomKindEnum
	AtomKindEnumVariant
	AtomKindIfExpr
	AtomKindForExpr
	AtomKindWhileExpr
	AtomKindLoopExpr
	AtomKindFnExpr
	AtomKindTryExpr
	AtomKindExpression
)

type Atom interface {
	Kind() AtomKind
	Span() errors.Span
}

// Number
type AtomNumber struct {
	Num   float64
	Range errors.Span
}

func (self AtomNumber) Kind() AtomKind    { return AtomKindNumber }
func (self AtomNumber) Span() errors.Span { return self.Range }

// String
type AtomString struct {
	Content string
	Range   errors.Span
}

func (self AtomString) Kind() AtomKind    { return AtomKindString }
func (self AtomString) Span() errors.Span { return self.Range }

// Boolean
type AtomBoolean struct {
	Value bool
	Range errors.Span
}

func (self AtomBoolean) Kind() AtomKind    { return AtomKindBoolean }
func (self AtomBoolean) Span() errors.Span { return self.Range }

// Identifier
type AtomIdentifier struct {
	Identifier string
	Range      errors.Span
}

func (self AtomIdentifier) Kind() AtomKind    { return AtomKindIdentifier }
func (self AtomIdentifier) Span() errors.Span { return self.Range }

// List literals
type AtomListLiteral struct {
	Values []Expression
	Range  errors.Span
}

func (self AtomListLiteral) Kind() AtomKind    { return AtomKindListLiteral }
func (self AtomListLiteral) Span() errors.Span { return self.Range }

// Pair
type AtomPair struct {
	Key       string
	ValueExpr Expression
	Range     errors.Span
}

func (self AtomPair) Kind() AtomKind    { return AtomKindPair }
func (self AtomPair) Span() errors.Span { return self.Range }

// Object
type AtomObject struct {
	Range  errors.Span
	Fields []AtomObjectField
}

type AtomObjectField struct {
	Span       errors.Span
	Identifier string
	IdentSpan  errors.Span
	Expression Expression
}

func (self AtomObject) Kind() AtomKind    { return AtomKindObject }
func (self AtomObject) Span() errors.Span { return self.Range }

// Null
type AtomNull struct {
	Range errors.Span
}

func (self AtomNull) Kind() AtomKind    { return AtomKindNull }
func (self AtomNull) Span() errors.Span { return self.Range }

// Range
type AtomRange struct {
	Start int
	End   int
	Range errors.Span
}

func (self AtomRange) Kind() AtomKind    { return AtomKindRange }
func (self AtomRange) Span() errors.Span { return self.Range }

// Enum
type AtomEnum struct {
	Name     string
	Variants []EnumVariant
	Range    errors.Span
}

type EnumVariant struct {
	Value string
	Span  errors.Span
}

func (self AtomEnum) Kind() AtomKind    { return AtomKindEnum }
func (self AtomEnum) Span() errors.Span { return self.Range }

// Enum Variant
type AtomEnumVariant struct {
	RefersToEnum string
	Name         string
	Range        errors.Span
}

func (self AtomEnumVariant) Kind() AtomKind    { return AtomKindEnumVariant }
func (self AtomEnumVariant) Span() errors.Span { return self.Range }

// If expression

type IfExpr struct {
	Condition  Expression
	Block      Block   // To be executed if condition is true
	ElseBlock  *Block  // Optional (else {...})
	ElseIfExpr *IfExpr // Optional (else if ... {...})
	Range      errors.Span
}

func (self IfExpr) Kind() AtomKind    { return AtomKindIfExpr }
func (self IfExpr) Span() errors.Span { return self.Range }

// For loop
type AtomFor struct {
	HeadIdentifier struct {
		Identifier string
		Span       errors.Span
	}
	IterExpr      Expression
	IterationCode Block
	Range         errors.Span
}

func (self AtomFor) Kind() AtomKind    { return AtomKindForExpr }
func (self AtomFor) Span() errors.Span { return self.Range }

// While loop
type AtomWhile struct {
	HeadCondition Expression
	IterationCode Block
	Range         errors.Span
}

func (self AtomWhile) Kind() AtomKind    { return AtomKindWhileExpr }
func (self AtomWhile) Span() errors.Span { return self.Range }

// Loop expression
type AtomLoop struct {
	IterationCode Block
	Range         errors.Span
}

func (self AtomLoop) Kind() AtomKind    { return AtomKindLoopExpr }
func (self AtomLoop) Span() errors.Span { return self.Range }

// Function declaration
type AtomFunction struct {
	Ident          *string
	ArgIdentifiers []struct {
		Identifier string
		Span       errors.Span
	}
	Body  Block
	Range errors.Span
}

func (self AtomFunction) Kind() AtomKind    { return AtomKindFnExpr }
func (self AtomFunction) Span() errors.Span { return self.Range }

// Try expression
type AtomTry struct {
	TryBlock        Block
	ErrorIdentifier string
	CatchBlock      Block
	Range           errors.Span
}

func (self AtomTry) Kind() AtomKind    { return AtomKindTryExpr }
func (self AtomTry) Span() errors.Span { return self.Range }

// Atom Expression
type AtomExpression struct {
	Expression Expression
	Range      errors.Span
}

func (self AtomExpression) Kind() AtomKind    { return AtomKindExpression }
func (self AtomExpression) Span() errors.Span { return self.Range }
