package value

import (
	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type VMFatalExceptionKind uint8

const (
	Vm_StackOverFlowErrorKind VMFatalExceptionKind = iota
	Vm_OutOfMemoryErrorKind
	Vm_ValueErrorKind
	Vm_ImportErrorKind
	Vm_HostErrorKind
	Vm_JsonErrorKind
	Vm_CastErrorKind
	Vm_IndexOutOfBoundsErrorKind
	Vm_UncaughtThrowKind
)

func (self VMFatalExceptionKind) String() string {
	switch self {
	case Vm_StackOverFlowErrorKind:
		return "StackOverFlow"
	case Vm_OutOfMemoryErrorKind:
		return "OutOfMemoryError"
	case Vm_ValueErrorKind:
		return "ValueError"
	case Vm_HostErrorKind:
		return "HostError"
	case Vm_CastErrorKind:
		return "CastError"
	case Vm_IndexOutOfBoundsErrorKind:
		return "IndexOutOfBounds"
	case Vm_UncaughtThrowKind:
		return "UncaughtThrow"
	default:
		panic("A new ErrorKind was added without updating this code")
	}
}

type Vm_InterruptKind uint8

const (
	Vm_TerminateInterruptKind Vm_InterruptKind = iota
	Vm_ExitInterruptKind
	Vm_ReturnInterruptKind
	Vm_BreakInterruptKind
	Vm_ContinueInterruptKind
	Vm_NormalExceptionInterruptKind
	Vm_FatalExceptionInterruptKind
)

func (self Vm_InterruptKind) String() string {
	switch self {
	case Vm_TerminateInterruptKind:
		return "terminate"
	case Vm_ExitInterruptKind:
		return "exit"
	case Vm_ReturnInterruptKind:
		return "return"
	case Vm_BreakInterruptKind:
		return "break"
	case Vm_ContinueInterruptKind:
		return "continue"
	case Vm_NormalExceptionInterruptKind:
		return "exception"
	case Vm_FatalExceptionInterruptKind:
		return "fatal exception"
	default:
		panic("A new interrupt kind was added without updating this code")
	}
}

type Vm_Interrupt interface {
	Kind() Vm_InterruptKind
	Message() string
	// This function may panic if the target value has no span
	GetSpan() errors.Span
}

//
// Exit interrupt
//

type Vm_ExitInterrupt struct {
	Code int64
	Span errors.Span
}

func (self Vm_ExitInterrupt) Kind() Vm_InterruptKind { return Vm_ExitInterruptKind }
func (self Vm_ExitInterrupt) Message() string        { return "<exit-interrupt>" }
func (self Vm_ExitInterrupt) GetSpan() errors.Span   { return self.Span }
func NewVMExitInterrupt(code int64, span errors.Span) *Vm_Interrupt {
	i := Vm_Interrupt(Vm_ExitInterrupt{Code: code, Span: span})
	return &i
}

//
// Throw interrupt, TODO: is normal exception
//

type Vm_ThrowInterrupt struct {
	MessageInternal string
	Span            errors.Span
}

func (self Vm_ThrowInterrupt) Kind() Vm_InterruptKind { return Vm_NormalExceptionInterruptKind }
func (self Vm_ThrowInterrupt) Message() string        { return self.MessageInternal }
func (self Vm_ThrowInterrupt) GetSpan() errors.Span {
	return self.Span
}

func NewVMThrowInterrupt(span errors.Span, message string) *Vm_Interrupt {
	i := Vm_Interrupt(Vm_ThrowInterrupt{MessageInternal: message, Span: span})
	return &i
}

//
// Runtime error
//

type VmFatalException struct {
	ErrKind         VMFatalExceptionKind
	MessageInternal string
	Span            errors.Span
}

func (self VmFatalException) Kind() Vm_InterruptKind { return Vm_FatalExceptionInterruptKind }

func (self VmFatalException) Message() string {
	return self.MessageInternal
}

func (self VmFatalException) GetSpan() errors.Span {
	return self.Span
}

func NewVMFatalException(message string, kind VMFatalExceptionKind, span errors.Span) *Vm_Interrupt {
	i := Vm_Interrupt(VmFatalException{
		MessageInternal: message,
		ErrKind:         kind,
		Span:            span,
	})
	return &i
}

//
// Termination interrupt
//

type VmTerminationInterrupt struct {
	Reason string
	Span   errors.Span
}

func (self VmTerminationInterrupt) Kind() Vm_InterruptKind { return Vm_TerminateInterruptKind }
func (self VmTerminationInterrupt) Message() string        { return self.Reason }
func (self VmTerminationInterrupt) GetSpan() errors.Span {
	return self.Span
}

func NewVMTerminationInterrupt(reason string, span errors.Span) *Vm_Interrupt {
	i := Vm_Interrupt(VmTerminationInterrupt{Reason: reason, Span: span})
	return &i
}
