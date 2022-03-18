package homescript

type Expressions []Expression

type Expression OrExpr

type OrExpr struct {
	Base      AndExpr
	Following []AndExpr
}

type AndExpr struct {
	Base      EqExpr
	Following []EqExpr
}

type EqExpr struct {
	Base  RelExpr
	Other *struct {
		TokenType
		RelExpr
	}
}

type RelExpr struct {
	Base  NotExpr
	Other *struct {
		TokenType
		NotExpr
	}
}

type NotExpr struct {
	Negated bool
	Base    Atom
}

///////////// Atom /////////////

type AtomKind uint8

const (
	AtomNumberKind AtomKind = iota
	AtomStringKind
	AtomBooleanKind
	AtomIdentifierKind
	AtomIfKind
	AtomCallKind
	AtomExpressionKind
)

type Atom interface {
	Kind() AtomKind
}

// Number
type AtomNumber struct{ Num int }

func (self AtomNumber) Kind() AtomKind { return AtomNumberKind }

// String
type AtomString struct{ Content string }

func (self AtomString) Kind() AtomKind { return AtomStringKind }

// Boolean
type AtomBoolean struct{ Value bool }

func (self AtomBoolean) Kind() AtomKind { return AtomBooleanKind }

// Identifier
type AtomIdentifier struct{ Name string }

func (self AtomIdentifier) Kind() AtomKind { return AtomIdentifierKind }

// If
type AtomIf struct{ IfExpression IfExpression }

func (self AtomIf) Kind() AtomKind { return AtomIfKind }

// Call
type AtomCall struct {
	CallExpression CallExpression
}

func (self AtomCall) Kind() AtomKind { return AtomCallKind }

// Expression
type AtomExpression struct{ Expression Expression }

func (self AtomExpression) Kind() AtomKind { return AtomExpressionKind }

type IfExpression struct {
	Condition Expression
	Body      Expressions
	ElseBody  Expressions
}

type CallExpression struct {
	Name      string
	Arguments Expressions
}
