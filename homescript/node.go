package homescript

import "github.com/MikMuellerDev/homescript/homescript/error"

type Expressions []Expression

type Expression struct {
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
	Location error.Location
}

type NotExpr struct {
	Base    Atom
	Negated bool
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
type AtomIdentifier struct {
	Name     string
	Location error.Location
}

func (self AtomIdentifier) Kind() AtomKind { return AtomIdentifierKind }

// If
type AtomIf struct{ IfExpr IfExpr }

func (self AtomIf) Kind() AtomKind { return AtomIfKind }

// Call
type AtomCall struct {
	CallExpr CallExpr
}

func (self AtomCall) Kind() AtomKind { return AtomCallKind }

// Expression
type AtomExpr struct{ Expr Expression }

func (self AtomExpr) Kind() AtomKind { return AtomExpressionKind }

type IfExpr struct {
	Condition Expression
	Body      Expressions
	ElseBody  Expressions
}

type CallExpr struct {
	Name      string
	Arguments []Expression
	Location  error.Location
}
