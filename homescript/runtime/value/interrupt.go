package value

import (
	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type VmInterruptKind uint8

const (
	Vm_TerminateInterruptKind VmInterruptKind = iota
	Vm_ExitInterruptKind
	Vm_NormalExceptionInterruptKind
	Vm_FatalExceptionInterruptKind
)

func (self VmInterruptKind) String() string {
	switch self {
	case Vm_TerminateInterruptKind:
		return "terminate"
	case Vm_ExitInterruptKind:
		return "exit"
	case Vm_NormalExceptionInterruptKind:
		return "exception"
	case Vm_FatalExceptionInterruptKind:
		return "fatal exception"
	default:
		panic("A new interrupt kind was added without updating this code")
	}
}

type VmInterrupt interface {
	Kind() VmInterruptKind
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

func (self Vm_ExitInterrupt) Kind() VmInterruptKind { return Vm_ExitInterruptKind }
func (self Vm_ExitInterrupt) Message() string       { return "<exit-interrupt>" }
func (self Vm_ExitInterrupt) GetSpan() errors.Span  { return self.Span }
func NewVMExitInterrupt(code int64, span errors.Span) *VmInterrupt {
	i := VmInterrupt(Vm_ExitInterrupt{Code: code, Span: span})
	return &i
}

//
// Termination interrupt
//

type VmTerminationInterrupt struct {
	Reason string
	Span   errors.Span
}

func (self VmTerminationInterrupt) Kind() VmInterruptKind { return Vm_TerminateInterruptKind }
func (self VmTerminationInterrupt) Message() string       { return self.Reason }
func (self VmTerminationInterrupt) GetSpan() errors.Span {
	return self.Span
}

func NewVMTerminationInterrupt(reason string, span errors.Span) *VmInterrupt {
	i := VmInterrupt(VmTerminationInterrupt{Reason: reason, Span: span})
	return &i
}

//
// Normal exception
//

type Vm_NormalException struct {
	MessageInternal string
	Span            errors.Span
}

func (self Vm_NormalException) Kind() VmInterruptKind { return Vm_NormalExceptionInterruptKind }
func (self Vm_NormalException) Message() string       { return self.MessageInternal }
func (self Vm_NormalException) GetSpan() errors.Span {
	return self.Span
}

func NewVMThrowInterrupt(span errors.Span, message string) *VmInterrupt {
	i := VmInterrupt(Vm_NormalException{MessageInternal: message, Span: span})
	return &i
}

//
// Fatal exception
//

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

type VmFatalException struct {
	ErrKind         VMFatalExceptionKind
	MessageInternal string
	Span            errors.Span
}

func (self VmFatalException) Kind() VmInterruptKind { return Vm_FatalExceptionInterruptKind }

func (self VmFatalException) Message() string {
	return self.MessageInternal
}

func (self VmFatalException) GetSpan() errors.Span {
	return self.Span
}

func NewVMFatalException(message string, kind VMFatalExceptionKind, span errors.Span) *VmInterrupt {
	i := VmInterrupt(VmFatalException{
		MessageInternal: message,
		ErrKind:         kind,
		Span:            span,
	})
	return &i
}
