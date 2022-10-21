package homescript

import (
	"github.com/smarthome-go/homescript/homescript/errors"
)

type Block []Statement

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
	Left  string
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
	}
	Span errors.Span
}

type MulOperator uint8

const (
	MulOpMul MulOperator = iota
	MulOpDiv
	MullOpReminder
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
	MemberExpressionPart *string       // Optional: `.identifier`
	Args                 *[]Expression // Optional: (arg1, arg2) to call the function
	Span                 errors.Span
}

// Member expression
type MemberExpression struct {
	Base    Atom
	Members []string // Each member is an identifier 'foo.bar.baz' where `foo` is the base and `bar` and `baz` are the members
	Span    errors.Span
}

///////////// ATOM /////////////

// Atom
type AtomKind uint8

const (
	AtomKindNumber AtomKind = iota
	AtomKindBoolean
	AtomKindString
	AtomKindPair
	AtomKindNull
	AtomKindIdentifier
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

// Pair
type AtomPair struct {
	Key       string
	ValueExpr Expression
	Range     errors.Span
}

func (self AtomPair) Kind() AtomKind    { return AtomKindPair }
func (self AtomPair) Span() errors.Span { return self.Range }

// Null
type AtomNull struct{}

func (self AtomNull) Kind() AtomKind    { return AtomKindNull }
func (self AtomNull) Span() errors.Span { return errors.Span{} }

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
	HeadIdentifier string
	RangeLowerExpr Expression
	RangeUpperExpr Expression
	IterationCode  Block
	Range          errors.Span
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
	ArgIdentifiers []string
	Body           Block
	Range          errors.Span
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
