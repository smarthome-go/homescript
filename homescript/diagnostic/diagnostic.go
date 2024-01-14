package diagnostic

import (
	"fmt"
	"strings"

	"github.com/smarthome-go/homescript/v3/homescript/errors"
)

type DiagnosticLevel uint8

const (
	DiagnosticLevelHint DiagnosticLevel = iota
	DiagnosticLevelInfo
	DiagnosticLevelWarning
	DiagnosticLevelError
)

func (self DiagnosticLevel) String() string {
	switch self {
	case DiagnosticLevelHint:
		return "Hint"
	case DiagnosticLevelInfo:
		return "Info"
	case DiagnosticLevelWarning:
		return "Warning"
	case DiagnosticLevelError:
		return "Error"
	default:
		panic("A new diagnostic level was added without updating this code")
	}
}

//
// Diagnostic
//

type Diagnostic struct {
	Level   DiagnosticLevel `json:"level"`
	Message string          `json:"message"`
	Notes   []string        `json:"notes"`
	Span    errors.Span     `json:"span"`
}

func (self Diagnostic) Display(program string) string {
	singleMarker := "^"
	markerMul := ""
	var color uint8 = 0

	switch self.Level {
	case DiagnosticLevelHint:
		markerMul = "~"
		color = 5 // magenta
	case DiagnosticLevelInfo:
		markerMul = "~"
		color = 4 // blue
	case DiagnosticLevelWarning:
		markerMul = "~"
		color = 3 // yellow
	case DiagnosticLevelError:
		markerMul = "^"
		color = 1 // red
	}

	notes := ""

	for _, note := range self.Notes {
		notes += fmt.Sprintf("%s - note:\x1b[0m %s\n", ansiCol(36, true), note)
	}

	// take special action if there is no useful span / the source code is empty
	if self.Span.Start.Line == 0 &&
		self.Span.Start.Column == 0 &&
		self.Span.End.Line == 0 &&
		self.Span.End.Column == 0 {
		return fmt.Sprintf(
			"%s%s\x1b[1;39m in %s\x1b[0m\n%s\n%s",
			ansiCol(color+30, true),
			self.Level,
			self.Span.Filename,
			self.Message,
			notes,
		)
	}

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

	markers := ""
	if self.Span.Start.Line == self.Span.End.Line {
		if self.Span.Start.Column == self.Span.End.Column {
			// only one column difference
			markers = singleMarker
		} else {
			// multiple columns difference
			markers = strings.Repeat(markerMul, int(self.Span.End.Column-self.Span.Start.Column)+1) // This is required because token spans are inclusive
		}
	} else {
		// multiline span
		s := "s"
		if self.Span.End.Line-self.Span.Start.Line == 1 {
			s = ""
		}

		markers = fmt.Sprintf(
			"%s ...\n%s%s+ %d more line%s\x1b[0m",
			strings.Repeat(markerMul, len(lines[self.Span.Start.Line-1])-int(self.Span.Start.Column)+1),
			strings.Repeat(" ", int(self.Span.Start.Column)+6),
			ansiCol(32, true),
			self.Span.End.Line-self.Span.Start.Line,
			s,
		)
	}
	marker := fmt.Sprintf(
		"%s%s%s\x1b[0m",
		ansiCol(color+30, true),
		strings.Repeat(" ", int(self.Span.Start.Column+6)),
		markers,
	)

	return fmt.Sprintf(
		"%s%v\x1b[39m at %s:%d:%d\x1b[0m\n%s\n%s\n%s%s\n\n\x1b%s%s\x1b[0m\n%s",
		ansiCol(color+30, true),
		self.Level,
		self.Span.Filename,
		self.Span.Start.Line,
		self.Span.Start.Column,
		line1,
		line2,
		marker,
		line3,
		ansiCol(color+30, true),
		self.Message,
		notes,
	)
}

func ansiCol(color uint8, bold bool) string {
	if bold {
		return fmt.Sprintf("\x1b[1;%dm", color)
	}
	return fmt.Sprintf("\x1b[%dm", color)
}
