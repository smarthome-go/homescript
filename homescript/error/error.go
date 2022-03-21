package error

type Error struct {
	ErrorType ErrorType
	TypeName  string
	Location  Location
	Message   string
}

type ErrorType uint8

const (
	SyntaxError ErrorType = iota
	TypeError
	ReferenceError
	ValueError
)

func NewError(errorType ErrorType, location Location, message string) *Error {
	var typeName string
	switch errorType {
	case SyntaxError:
		typeName = "SyntaxError"
	case TypeError:
		typeName = "TypeError"
	case ReferenceError:
		typeName = "ReferenceError"
	case ValueError:
		typeName = "ValueError"
	default:
		panic(0)
	}
	return &Error{ErrorType: errorType, TypeName: typeName, Location: location, Message: message}
}
