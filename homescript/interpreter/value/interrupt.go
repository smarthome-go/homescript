package value

import (
	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type RuntimeErrorKind uint8

const (
	StackOverFlowErrorKind RuntimeErrorKind = iota
	OutOfMemoryErrorKind
	ValueErrorKind
	ImportErrorKind
	HostErrorKind
	JsonErrorKind
	CastErrorKind
	IndexOutOfBoundsErrorKind
	UncaughtThrowKind
)

func (self RuntimeErrorKind) String() string {
	switch self {
	case StackOverFlowErrorKind:
		return "StackOverFlow"
	case OutOfMemoryErrorKind:
		return "OutOfMemoryError"
	case ValueErrorKind:
		return "ValueError"
	case ImportErrorKind:
		return "ImportError"
	case HostErrorKind:
		return "HostError"
	case JsonErrorKind:
		return "JsonError"
	case CastErrorKind:
		return "CastError"
	case IndexOutOfBoundsErrorKind:
		return "IndexOutOfBounds"
	case UncaughtThrowKind:
		return "UncaughtThrow"
	default:
		panic("A new ErrorKind was added without updating this code")
	}
}

type InterruptKind uint8

const (
	TerminateInterruptKind InterruptKind = iota
	ExitInterruptKind
	ReturnInterruptKind
	BreakInterruptKind
	ContinueInterruptKind
	NormalExceptionInterruptKind
	FatalExceptionInterruptKind
)

func (self InterruptKind) String() string {
	switch self {
	case TerminateInterruptKind:
		return "terminate"
	case ExitInterruptKind:
		return "exit"
	case ReturnInterruptKind:
		return "return"
	case BreakInterruptKind:
		return "break"
	case ContinueInterruptKind:
		return "continue"
	case NormalExceptionInterruptKind:
		return "exception"
	case FatalExceptionInterruptKind:
		return "fatal exception"
	default:
		panic("A new interrupt kind was added without updating this code")
	}
}

type Interrupt interface {
	Kind() InterruptKind
	Message() string
	// This function may panic if the target value has no span
	GetSpan() errors.Span
}

//
// Exit interrupt
//

type ExitInterrupt struct {
	Code int64
	Span errors.Span
}

func (self ExitInterrupt) Kind() InterruptKind  { return ExitInterruptKind }
func (self ExitInterrupt) Message() string      { return "<exit-interrupt>" }
func (self ExitInterrupt) GetSpan() errors.Span { return self.Span }
func NewExitInterrupt(code int64, span errors.Span) *Interrupt {
	i := Interrupt(ExitInterrupt{Code: code, Span: span})
	return &i
}

//
// Break interrupt
//

type BreakInterrupt struct{}

func (self BreakInterrupt) Kind() InterruptKind { return BreakInterruptKind }
func (self BreakInterrupt) Message() string     { return "<break-interrupt>" }
func (self BreakInterrupt) GetSpan() errors.Span {
	panic("This interrupt kind does not contain a span")
}
func NewBreakInterrupt() *Interrupt {
	i := Interrupt(BreakInterrupt{})
	return &i
}

//
// Continue interrupt
//

type ContinueInterrupt struct{}

func (self ContinueInterrupt) Kind() InterruptKind { return ContinueInterruptKind }
func (self ContinueInterrupt) Message() string     { return "<continue-interrupt>" }
func (self ContinueInterrupt) GetSpan() errors.Span {
	panic("This interrupt kind does not contain a span")
}
func NewContinueInterrupt() *Interrupt {
	i := Interrupt(ContinueInterrupt{})
	return &i
}

//
// Return interrupt
//

type ReturnInterrupt struct {
	ReturnValue Value
}

func (self ReturnInterrupt) Kind() InterruptKind { return ReturnInterruptKind }

func (self ReturnInterrupt) Message() string {
	return "<return-interrupt>"
}

func (self ReturnInterrupt) GetSpan() errors.Span {
	panic("This interrupt kind does not contain a span")
}

func NewReturnInterrupt(value Value) *Interrupt {
	i := Interrupt(ReturnInterrupt{ReturnValue: value})
	return &i
}

//
// Throw interrupt
//

type ThrowInterrupt struct {
	MessageInternal string
	Span            errors.Span
}

func (self ThrowInterrupt) Kind() InterruptKind { return NormalExceptionInterruptKind }
func (self ThrowInterrupt) Message() string     { return self.MessageInternal }
func (self ThrowInterrupt) GetSpan() errors.Span {
	return self.Span
}

func NewThrowInterrupt(span errors.Span, message string) *Interrupt {
	i := Interrupt(ThrowInterrupt{MessageInternal: message, Span: span})
	return &i
}

//
// Runtime error
//

type RuntimeErr struct {
	ErrKind         RuntimeErrorKind
	MessageInternal string
	Span            errors.Span
}

func (self RuntimeErr) Kind() InterruptKind { return FatalExceptionInterruptKind }

func (self RuntimeErr) Message() string {
	return self.MessageInternal
}

func (self RuntimeErr) GetSpan() errors.Span {
	return self.Span
}

func NewRuntimeErr(message string, kind RuntimeErrorKind, span errors.Span) *Interrupt {
	i := Interrupt(RuntimeErr{
		MessageInternal: message,
		ErrKind:         kind,
		Span:            span,
	})
	return &i
}

//
// Termination interrupt
//

type TerminationInterrupt struct {
	Reason string
	Span   errors.Span
}

func (self TerminationInterrupt) Kind() InterruptKind { return TerminateInterruptKind }
func (self TerminationInterrupt) Message() string     { return self.Reason }
func (self TerminationInterrupt) GetSpan() errors.Span {
	return self.Span
}

func NewTerminationInterrupt(reason string, span errors.Span) *Interrupt {
	i := Interrupt(TerminationInterrupt{Reason: reason, Span: span})
	return &i
}
