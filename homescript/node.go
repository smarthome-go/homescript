package homescript

type Block []Statement

/////// Statements ///////
type StatementKind uint8

const (
	LetStmtKind StatementKind = iota
	ImportStmtKind
	BreakStmtKind
	ContinueStmtKind
	ReturnStmtKind
	ExpressionStmtKind
)

type Statement interface {
	Kind() StatementKind
	Span() Span
}

type LetStmt struct {
	Left  string
	Right Expression
	Range Span
}

func (self LetStmt) Kind() StatementKind { return LetStmtKind }
func (self LetStmt) Span() Span          { return self.Range }

type ImportStmt struct {
	Function   string  // import `foo`
	RewriteAs  *string // as `bar`
	FromModule string  // from `baz`
	Range      Span
}

func (self ImportStmt) Kind() StatementKind { return ImportStmtKind }
func (self ImportStmt) Span() Span          { return self.Range }

type BreakStmt struct {
	Expression *Expression // Can be the return value of the loop
	Range      Span
}

func (self BreakStmt) Kind() StatementKind { return BreakStmtKind }
func (self BreakStmt) Span() Span          { return self.Range }

type ContinueStmt struct {
	Range Span
}

func (self ContinueStmt) Kind() StatementKind { return ContinueStmtKind }
func (self ContinueStmt) Span() Span          { return self.Range }

type ReturnStmt struct {
	Expression *Expression // Can be the return value of the function
	Range      Span
}

func (self ReturnStmt) Kind() StatementKind { return ReturnStmtKind }
func (self ReturnStmt) Span() Span          { return self.Range }

type ExpressionStmt struct {
	Expression Expression
	Range      Span
}

func (self ExpressionStmt) Kind() StatementKind { return ReturnStmtKind }
func (self ExpressionStmt) Span() Span          { return self.Range }

/////// Expressions ///////

// Expression
type Expression OrExpression

// Or expression
type OrExpression struct {
	Base      AndExpression
	Following []AndExpression
	Span      Span
}

// And expression
type AndExpression struct {
	Base      EqExpression
	Following []EqExpression
	Span      Span
}

// Equality expression
type EqExpression struct {
	Base  RelExpression
	Other *struct {
		// True corresponds to `!=` and false corresponds to `==`
		Inverted bool
		Other    RelExpression
	}
	Span Span
}

// Relational expression
type RelExpression struct {
	Base  AddExpression
	Other *struct {
		RelOperator RelOperator
		Other       AddExpression
	}
	Span Span
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
	Span Span
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
		MulOperator
		Other CastExpression
	}
	Span Span
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
	Other TypeName
	Span  Span
}

type TypeName uint8

const (
	NullTypeName TypeName = iota
	NumberTypeName
	StringTypeName
	BoolTypeName
)

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
	Span          Span
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
	Span  Span
}

// Assign expression
type AssignExpression struct {
	Base  CallExpression
	Other *struct {
		Operator   AssignOperator
		Expression AssignExpression
	}
	Span Span
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
	Other *struct {
		Args  []AssignExpression
		Parts []CallExprPart // Allows chaining of member expressions like `a.b.c()`
	}
	Span Span
}

// If member expr part is nil, args is used
// if args is nil, member expr part is used
type CallExprPart struct {
	MemberExpressionPart *string             // Optional: `.identifier`
	Args                 *[]AssignExpression // Optional: (arg1, arg2) to call the function
	Span                 Span
}

// Member expression
type MemberExpression struct {
	Base  Atom
	Parts []Atom // Each part is an atom (identifier) 'foo.bar.baz' where `foo` is the base and `bar` and `baz` are the parts
	Span  Span
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
	Span() Span
}

// Number
type AtomNumber struct {
	Num   float64
	Range Span
}

func (self AtomNumber) Kind() AtomKind { return AtomKindNumber }
func (self AtomNumber) Span() Span     { return self.Range }

// String
type AtomString struct {
	Content string
	Range   Span
}

func (self AtomString) Kind() AtomKind { return AtomKindString }
func (self AtomString) Span() Span     { return self.Range }

// Boolean
type AtomBoolean struct {
	Value bool
	Range Span
}

func (self AtomBoolean) Kind() AtomKind { return AtomKindBoolean }
func (self AtomBoolean) Span() Span     { return self.Range }

// Identifier
type AtomIdentifier struct {
	Identifier string
	Range      Span
}

func (self AtomIdentifier) Kind() AtomKind { return AtomKindIdentifier }
func (self AtomIdentifier) Span() Span     { return self.Range }

// Pair
type AtomPair struct {
	Key   string
	Value Atom
	Range Span
}

func (self AtomPair) Kind() AtomKind { return AtomKindPair }
func (self AtomPair) Span() Span     { return self.Range }

// Null
type AtomNull struct{}

func (self AtomNull) Kind() AtomKind { return AtomKindNull }
func (self AtomNull) Span() Span     { return Span{} }

// For loop
type AtomFor struct {
	HeadIdentifier     string
	IterationSpecifier AssignExpression
	IterationCode      Block
	Range              Span
}

func (self AtomFor) Kind() AtomKind { return AtomKindForExpr }
func (self AtomFor) Span() Span     { return self.Range }

// While loop
type AtomWhile struct {
	HeadCondition AssignExpression
	IterationCode Block
	Range         Span
}

func (self AtomWhile) Kind() AtomKind { return AtomKindWhileExpr }
func (self AtomWhile) Span() Span     { return self.Range }

// Loop expression
type AtomLoop struct {
	IterationCode Block
	Range         Span
}

func (self AtomLoop) Kind() AtomKind { return AtomKindLoopExpr }
func (self AtomLoop) Span() Span     { return self.Range }

// Function declaration
type AtomFunction struct {
	ArgIdentifiers []string
	Body           Block
	Range          Span
}

func (self AtomFunction) Kind() AtomKind { return AtomKindFnExpr }
func (self AtomFunction) Span() Span     { return self.Range }

// Try expression
type AtomTry struct {
	TryBlock        Block
	ErrorIdentifier string
	CatchBlock      Block
	Range           Span
}

func (self AtomTry) Kind() AtomKind { return AtomKindTryExpr }
func (self AtomTry) Span() Span     { return self.Range }

// Atom Expression
type AtomExpression struct {
	Expression AssignExpression
	Range      Span
}

func (self AtomExpression) Kind() AtomKind { return AtomKindExpression }
func (self AtomExpression) Span() Span     { return self.Range }
