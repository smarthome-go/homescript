package errors

import (
	"fmt"
	"strings"
)

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

func (self Error) Display(program string) string {
	lines := strings.Split(program, "\n")

	line1 := ""
	if self.Span.Start.Line > 1 {
		line1 = fmt.Sprintf("\n \x1b[90m%- 3d | \x1b[0m%s", self.Span.Start.Line-1, lines[self.Span.Start.Line-2])
	}
	line2 := fmt.Sprintf(" \x1b[90m%- 3d | \x1b[0m%s", self.Span.Start.Line, lines[self.Span.Start.Line-1])
	line3 := ""
	if int(self.Span.Start.Line) < len(lines) {
		line3 = fmt.Sprintf("\n \x1b[90m%- 3d | \x1b[0m%s", self.Span.Start.Line+1, lines[self.Span.Start.Line])
	}

	markers := "^"
	if self.Span.Start.Line == self.Span.End.Line {
		markers = strings.Repeat("^", int(self.Span.End.Column-self.Span.Start.Column)+1) // This is required because token spans are inclusive
	}
	marker := fmt.Sprintf("%s\x1b[1;31m%s\x1b[0m", strings.Repeat(" ", int(self.Span.Start.Column+6)), markers)

	return fmt.Sprintf(
		"\x1b[1;36m%v\x1b[39m at %s:%d:%d\x1b[0m\n%s\n%s\n%s%s\n\n\x1b[1;31m%s\x1b[0m\n",
		self.Kind,
		"file",
		self.Span.Start.Line,
		self.Span.Start.Column,
		line1,
		line2,
		marker,
		line3,
		self.Message,
	)
}
