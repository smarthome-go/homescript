package errors

// All ranges inclusive
type Span struct {
	Start Location
	End   Location
}

type Location struct {
	Line   uint
	Column uint
	Index  uint
}

type Error struct {
	Kind    ErrorKind
	Message string
	Span    Span
}

type ErrorKind uint8

const (
	SyntaxError ErrorKind = iota
	TypeError
	RuntimeError
	ValueError
	ThrowError
	StackOverflow
	ReferenceError
)

func NewError(span Span, message string, kind ErrorKind) *Error {
	return &Error{
		Span:    span,
		Message: message,
		Kind:    kind,
	}
}
