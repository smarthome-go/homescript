package runtime

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
	"github.com/smarthome-go/homescript/v3/homescript/runtime/value"
)

func (self *VM) SourceMap(frame CallFrame) errors.Span {
	if frame.InstructionPointer >= uint(len(self.Program.SourceMap[frame.Function])) {
		return self.Program.SourceMap[frame.Function][len(self.Program.SourceMap[frame.Function])-1]
	}

	return self.Program.SourceMap[frame.Function][frame.InstructionPointer]
}

func (self Core) fatalErr(
	message string,
	kind value.VMFatalExceptionKind,
	span errors.Span,
) *value.VmInterrupt {
	trace := self.unwind()

	// TODO: add stack unwinding to the runtime err itself
	return value.NewVMFatalException(
		fmt.Sprintf("%s\n=== Stacktrace ===\n%s", message, strings.Join(trace, "\n")),
		kind,
		span,
	)
}

func (self Core) normalException(
	message string,
	span errors.Span,
) *value.VmInterrupt {
	trace := self.unwind()

	// TODO: add stack unwinding to the runtime err itself
	return value.NewVMThrowInterrupt(
		span,
		fmt.Sprintf("%s\n=== Stacktrace ===\n%s", message, strings.Join(trace, "\n")),
	)
}

func (self Core) unwind() []string {
	output := make([]string, 0)

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

	for _, frame := range filtered {
		span := self.parent.SourceMap(frame.frame)

		additions := ""

		if frame.count > 1 {
			additions = fmt.Sprintf("    (%dx)", frame.count-1)
		}

		output = append(
			output,
			fmt.Sprintf(
				"%05d: %s() %s:%d:%d%s",
				frame.index,
				frame.frame.Function,
				span.Filename,
				span.Start.Line,
				span.Start.Column,
				additions,
			),
		)
	}

	return output
}
