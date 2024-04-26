package runtime

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

const stackTraceLineLength = 24

func (self *VM) SourceMap(frame CallFrame) errors.Span {
	instructionsOfCurrFn := len(self.Program.SourceMap[frame.Function])
	if instructionsOfCurrFn == 0 {
		panic(fmt.Sprintf("Empty function: `%s`", frame.Function))
	}

	if frame.InstructionPointer >= uint(len(self.Program.SourceMap[frame.Function])) {
		return self.Program.SourceMap[frame.Function][instructionsOfCurrFn-1]
	}

	return self.Program.SourceMap[frame.Function][frame.InstructionPointer]
}

func formatStackTrace(message string, trace []string, lineLen int) string {
	const headline = "Stacktrace"
	padding := strings.Repeat("=", (lineLen-len(headline))/2)

	return fmt.Sprintf(
		"%s\n%s %s %s\n%s",
		message,
		padding,
		headline,
		padding,
		strings.Join(trace, "\n"),
	)
}

func (self Core) fatalErr(
	message string,
	kind value.VMFatalExceptionKind,
	span errors.Span,
) *value.VmInterrupt {
	trace, lineLen := self.unwind()

	// TODO: add stack unwinding to the runtime err itself
	return value.NewVMFatalException(
		formatStackTrace(message, trace, lineLen),
		kind,
		span,
	)
}

func (self Core) normalException(
	message string,
	span errors.Span,
) *value.VmInterrupt {
	trace, lineLen := self.unwind()

	// TODO: add stack unwinding to the runtime err itself
	return value.NewVMThrowInterrupt(
		span,
		formatStackTrace(message, trace, lineLen),
	)
}

type Fragment struct {
	Left  string
	Right string
}

func (self Core) unwind() (trace []string, lineLen int) {
	type CallFrameInfo struct {
		frame CallFrame
		count uint
		index uint
	}

	filtered := make([]CallFrameInfo, 0)

	var prev CallFrame
	for i, frame := range self.CallStack {
		if prev != frame {
			prev = frame
			filtered = append(filtered, CallFrameInfo{
				frame: frame,
				count: 1,
				index: uint(i),
			})
		} else {
			filtered[len(filtered)-1].count++
		}
	}

	stackTraceLineLengthUsed := stackTraceLineLength

	fragments := make([]Fragment, 0)

	for _, frame := range filtered {
		span := self.parent.SourceMap(frame.frame)

		additions := ""

		if frame.count > 1 {
			additions = fmt.Sprintf("    (%dx)", frame.count-1)
		}

		left := fmt.Sprintf(
			"%05d: %s()",
			frame.index,
			frame.frame.Function,
		)

		for stackTraceLineLengthUsed-utf8.RuneCountInString(left) < 0 {
			stackTraceLineLengthUsed += 4
		}

		fragments = append(fragments, Fragment{
			Left: left,
			Right: fmt.Sprintf(
				"%s:%d:%d%s",
				span.Filename,
				span.Start.Line,
				span.Start.Column,
				additions,
			),
		})
	}

	output := make([]string, len(fragments))

	for i := len(fragments) - 1; i >= 0; i-- {
		fragment := fragments[i]
		output[len(fragments)-1-i] = fmt.Sprintf(
			"%s%s%s",
			fragment.Left,
			strings.Repeat(" ", stackTraceLineLengthUsed-utf8.RuneCountInString(fragment.Left)),
			fragment.Right,
		)
	}

	return output, utf8.RuneCountInString(output[0])
}
