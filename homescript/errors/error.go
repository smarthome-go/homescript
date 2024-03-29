package errors

import (
	"fmt"
	"strings"
)

type Error struct {
	Kind    ErrorKind
	Message string
	Span    Span
}

type Span struct {
	Start    Location `json:"start"`
	End      Location `json:"end"`
	Filename string   `json:"filename"`
}

// A location is always inclusive
type Location struct {
	Line   uint `json:"line"`
	Column uint `json:"column"`
	Index  uint `json:"index"`
}

func (self Location) Until(end Location, filename string) Span {
	return Span{
		Start:    self,
		End:      end,
		Filename: filename,
	}
}

func (self *Location) Advance(newline bool) {
	self.Index += 1
	if newline {
		self.Column = 1
		self.Line += 1
	} else {
		self.Column += 1
	}
}

type ErrorKind uint8

const (
	SyntaxError ErrorKind = iota
	TypeError
	ValueError
	ReferenceError
	StackOverflow
	OutOfBoundsError
	ImportError
)

func (self ErrorKind) String() string {
	switch self {
	case SyntaxError:
		return "SyntaxError"
	case TypeError:
		return "TypeError"
	case ValueError:
		return "ValueError"
	case StackOverflow:
		return "StackOverflow"
	case OutOfBoundsError:
		return "OutOfBoundsError"
	case ImportError:
		return "ImportError"
	case ReferenceError:
		return "ReferenceError"
	default:
		panic("BUG: a new error kind was introduced without updating this code")
	}
}

func NewError(span Span, message string, kind ErrorKind) *Error {
	return &Error{
		Span:    span,
		Message: message,
		Kind:    kind,
	}
}

func NewSyntaxError(span Span, message string) *Error {
	return NewError(span, message, SyntaxError)
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
		self.Span.Filename,
		self.Span.Start.Line,
		self.Span.Start.Column,
		line1,
		line2,
		marker,
		line3,
		self.Message,
	)
}
