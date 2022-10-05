package homescript

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
	Start   Location
	End     Location
	Kind    ErrorKind
	Message string
}

type ErrorKind uint8

const (
	SyntaxError ErrorKind = iota
)

func newError(start Location, end Location, message string, kind ErrorKind) *Error {
	return &Error{
		Start:   start,
		End:     end,
		Message: message,
		Kind:    kind,
	}
}
