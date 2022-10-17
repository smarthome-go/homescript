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
	ValueError
	ReferenceError
	StackOverflow
	// Can be caught using `catch`
	ThrowError
	RuntimeError
)

func (self ErrorKind) String() string {
	switch self {
	case SyntaxError:
		return "SyntaxError"
	case TypeError:
		return "TypeError"
	case RuntimeError:
		return "RuntimeError"
	case ValueError:
		return "ValueError"
	case ThrowError:
		return "ThrowError"
	case StackOverflow:
		return "StackOverflow"
	case ReferenceError:
		return "ReferenceError"
	default:
		panic("BUG: a new error kind was introduced without udating this code")
	}
}

func NewError(span Span, message string, kind ErrorKind) *Error {
	return &Error{
		Span:    span,
		Message: message,
		Kind:    kind,
	}
}
