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
// Diagnostic.
//

type Diagnostic struct {
	Level   DiagnosticLevel `json:"level"`
	Message string          `json:"message"`
	Notes   []string        `json:"notes"`
	Span    errors.Span     `json:"span"`
}

func (d Diagnostic) WithContext(context string) Diagnostic {
	return Diagnostic{
		Level:   d.Level,
		Message: fmt.Sprintf("%s: %s", context, d.Message),
		Notes:   d.Notes,
		Span:    d.Span,
	}
}

func (d Diagnostic) Display(program string) string {
	singleMarker := "^"
	markerMul := ""
	var color uint8

	switch d.Level {
	case DiagnosticLevelHint:
		markerMul = "~"
		color = 5 // Magenta.
	case DiagnosticLevelInfo:
		markerMul = "~"
		color = 4 // Cyan.
	case DiagnosticLevelWarning:
		markerMul = "~"
		color = 3 // Yellow.
	case DiagnosticLevelError:
		markerMul = "^"
		color = 1 // Red.
	}

	notes := ""

	for _, note := range d.Notes {
		notes += fmt.Sprintf("%s - note:\x1b[0m %s\n", ansiCol(36, true), note)
	}

	// take special action if there is no useful span / the source code is empty
	if d.Span.Start.Line == 0 &&
		d.Span.Start.Column == 0 &&
		d.Span.End.Line == 0 &&
		d.Span.End.Column == 0 {
		return fmt.Sprintf(
			"%s%s\x1b[1;39m in %s\x1b[0m\n%s\n%s",
			ansiCol(color+30, true),
			d.Level,
			d.Span.Filename,
			d.Message,
			notes,
		)
	}

	lines := strings.Split(program, "\n")

	line1 := ""
	if d.Span.Start.Line > 1 {
		line1 = fmt.Sprintf("\n \x1b[90m%- 3d | \x1b[0m%s", d.Span.Start.Line-1, lines[d.Span.Start.Line-2])
	}
	line2 := fmt.Sprintf(" \x1b[90m%- 3d | \x1b[0m%s", d.Span.Start.Line, lines[d.Span.Start.Line-1])
	line3 := ""
	if int(d.Span.Start.Line) < len(lines) {
		line3 = fmt.Sprintf("\n \x1b[90m%- 3d | \x1b[0m%s", d.Span.Start.Line+1, lines[d.Span.Start.Line])
	}

	markers := ""
	if d.Span.Start.Line == d.Span.End.Line {
		if d.Span.Start.Column == d.Span.End.Column {
			// only one column difference
			markers = singleMarker
		} else {
			// multiple columns difference
			markers = strings.Repeat(markerMul, int(d.Span.End.Column-d.Span.Start.Column)+1) // This is required because token spans are inclusive
		}
	} else {
		// multiline span
		s := "s"
		if d.Span.End.Line-d.Span.Start.Line == 1 {
			s = ""
		}

		markers = fmt.Sprintf(
			"%s ...\n%s%s+ %d more line%s\x1b[0m",
			strings.Repeat(markerMul, len(lines[d.Span.Start.Line-1])-int(d.Span.Start.Column)+1),
			strings.Repeat(" ", int(d.Span.Start.Column)+6),
			ansiCol(32, true),
			d.Span.End.Line-d.Span.Start.Line,
			s,
		)
	}
	marker := fmt.Sprintf(
		"%s%s%s\x1b[0m",
		ansiCol(color+30, true),
		strings.Repeat(" ", int(d.Span.Start.Column+6)),
		markers,
	)

	return fmt.Sprintf(
		"%s%v\x1b[39m at %s:%d:%d\x1b[0m\n%s\n%s\n%s%s\n\n\x1b%s%s\x1b[0m\n%s",
		ansiCol(color+30, true),
		d.Level,
		d.Span.Filename,
		d.Span.Start.Line,
		d.Span.Start.Column,
		line1,
		line2,
		marker,
		line3,
		ansiCol(color+30, true),
		d.Message,
		notes,
	)
}

func ansiCol(color uint8, bold bool) string {
	if bold {
		return fmt.Sprintf("\x1b[1;%dm", color)
	}
	return fmt.Sprintf("\x1b[%dm", color)
}
