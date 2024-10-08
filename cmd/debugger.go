package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/smarthome-go/homescript/v3/homescript"
	"github.com/smarthome-go/homescript/v3/homescript/compiler"
	"github.com/smarthome-go/homescript/v3/homescript/runtime"
)

//
// REPL commands.
//

type debuggerCommandKind uint8

const (
	nopDebuggerCommandKind debuggerCommandKind = iota
	runDebuggerCommandKind
	continueDebuggerCommandKind
	infoDebuggerCommandKind
	callStackDebuggerCommandKind
	breakpointDebuggerCommandKind
	speedDebuggerCommandKind
	singleStepDebuggerCommandKind
)

type debuggerCommand interface {
	Kind() debuggerCommandKind
}

//
// NOP Subcommand
//

type nopDebuggerCommand struct{}

func (c nopDebuggerCommand) Kind() debuggerCommandKind { return nopDebuggerCommandKind }

//
// Run Subcommand
//

type runDebuggerCommand struct{}

func (c runDebuggerCommand) Kind() debuggerCommandKind { return runDebuggerCommandKind }

//
// Continue Subcommand
//

type continueDebuggerCommand struct{}

func (c continueDebuggerCommand) Kind() debuggerCommandKind { return continueDebuggerCommandKind }

//
// Info Subcommand
//

type infoSubcommand uint8

const (
	memoryInfoSubcommand infoSubcommand = iota
	stackInfoSubcommand
)

type infoDebuggerCommand struct {
	Subcommand infoSubcommand
}

func (c infoDebuggerCommand) Kind() debuggerCommandKind { return infoDebuggerCommandKind }

//
// Callstack Subcommand
//

type callStackDebuggerCommand struct{}

func (c callStackDebuggerCommand) Kind() debuggerCommandKind { return callStackDebuggerCommandKind }

//
// Singlestep Subcommand
//

type singleStepDebuggerCommand struct{}

func (c singleStepDebuggerCommand) Kind() debuggerCommandKind { return singleStepDebuggerCommandKind }

//
// Breakpoint Subcommand
//

type breakpointDebuggerCommand struct {
	IsAsm          bool
	FunctionOrFile string
	IndexOrLine    uint
}

func (c breakpointDebuggerCommand) Kind() debuggerCommandKind { return breakpointDebuggerCommandKind }

//
// Speed Subcommand
//

type speedDebuggerCommand struct {
	Millis uint
}

func (c speedDebuggerCommand) Kind() debuggerCommandKind { return speedDebuggerCommandKind }

//
// END REPL commands.
//

func parseDebuggerInput(input string) (debuggerCommand, error) {
	tokensRaw := strings.Split(input, " ")

	tokens := make([]string, 0)
	for _, t := range tokensRaw {
		if len(strings.TrimSpace(t)) == 0 {
			continue
		}

		tokens = append(tokens, t)
	}

	// No action.
	if len(tokens) == 0 {
		return nopDebuggerCommand{}, nil
	}

	command := tokens[0]

	switch command {
	case "speed":
		return parseExecutionSpeed(tokens[1:])
	case "info", "i":
		return parseInfoInput(tokens[1:])
	case "run", "r":
		if err := ensureEOF(tokens[1:]); err != nil {
			return nil, err
		}
		return runDebuggerCommand{}, nil
	case "bt":
		if err := ensureEOF(tokens[1:]); err != nil {
			return nil, err
		}
		return callStackDebuggerCommand{}, nil
	case "break", "b":
		return parseBreakpointInput(tokens[1:])
	case "si":
		if err := ensureEOF(tokens[1:]); err != nil {
			return nil, err
		}
		return singleStepDebuggerCommand{}, nil
	case "c":
		if err := ensureEOF(tokens[1:]); err != nil {
			return nil, err
		}
		return continueDebuggerCommand{}, nil
	default:
		return nil, fmt.Errorf("Illegal command: %s", command)
	}
}

func parseInfoInput(tokens []string) (infoDebuggerCommand, error) {
	if len(tokens) == 0 {
		return infoDebuggerCommand{}, errors.New("Expected info subcommand, got nothing")
	}

	subcommand := tokens[0]
	switch subcommand {
	case "memory", "mem", "m":
		return infoDebuggerCommand{
			Subcommand: memoryInfoSubcommand,
		}, nil
	case "stack", "st", "s":
		return infoDebuggerCommand{
			Subcommand: stackInfoSubcommand,
		}, nil
	default:
		return infoDebuggerCommand{}, fmt.Errorf("Illegal subcommand: %s", subcommand)
	}
}

func ensureEOF(tokens []string) error {
	if len(tokens) != 0 {
		return fmt.Errorf("Got too many tokens: expected no more, got %d additional", len(tokens))
	}
	return nil
}

func parseExecutionSpeed(tokens []string) (speedDebuggerCommand, error) {
	if len(tokens) != 1 {
		return speedDebuggerCommand{}, fmt.Errorf("Expected exactly one argument: <speed>, got %d", len(tokens))
	}

	speedRaw := tokens[0]
	speed, err := strconv.ParseUint(speedRaw, 10, 64)
	if err != nil {
		return speedDebuggerCommand{}, err
	}

	return speedDebuggerCommand{
		Millis: uint(speed),
	}, nil
}

func parseBreakpointInput(tokens []string) (breakpointDebuggerCommand, error) {
	if len(tokens) != 3 {
		return breakpointDebuggerCommand{}, fmt.Errorf("Expected exactly two arguments: <source> <filename/function> <line>, got %d", len(tokens))
	}

	file := tokens[1]
	lineRaw := tokens[2]
	line, err := strconv.ParseUint(lineRaw, 10, 64)
	if err != nil {
		return breakpointDebuggerCommand{}, err
	}

	isAsm := false

	source := tokens[0]
	switch source {
	case "file":
	case "asm":
		isAsm = true
	default:
		return breakpointDebuggerCommand{}, fmt.Errorf("Expected <asm> or <file> for <source>, got %s", source)
	}

	return breakpointDebuggerCommand{
		IsAsm:          isAsm,
		FunctionOrFile: file,
		IndexOrLine:    uint(line),
	}, nil
}

type Breakpoint struct {
	Function string
	Index    uint
}

type Debugger struct {
	speedWait      time.Duration
	running        bool
	singleStep     bool
	breakpoints    map[Breakpoint]struct{}
	debuggerOutput *chan runtime.DebugOutput
	debuggerResume *chan struct{}
	core           *runtime.Core
	programIn      string
	programOut     compiler.CompileOutput
}

func NewDebugger(
	debuggerOutput *chan runtime.DebugOutput,
	debuggerResume *chan struct{},
	core *runtime.Core,
	programIn string,
	programOut compiler.CompileOutput,
) Debugger {
	return Debugger{
		breakpoints:    make(map[Breakpoint]struct{}),
		debuggerOutput: debuggerOutput,
		debuggerResume: debuggerResume,
		core:           core,
		programIn:      programIn,
		programOut:     programOut,
	}
}

func (d *Debugger) DebuggerMainloop() {
	for {
		select {
		case msg, open := <-*d.debuggerOutput:
			if !open {
				return
			}

			lineIdx := int(msg.CurrentCallFrame.InstructionPointer)
			programStr := d.programOut.AsmStringHighlight(true, &msg.CurrentCallFrame.Function, &lineIdx)

			stack := make([]string, 0)
			for _, v := range d.core.Stack {
				if v == nil || *v == nil {
					stack = append(stack, "<nil>")
					continue
				}

				d, i := (*v).Display()
				if i != nil {
					panic(*i)
				}

				stack = append(stack, d)
			}

			// If the current line is not a breakpoint, skip it.
			_, isBreakPoint := d.breakpoints[Breakpoint{
				Function: msg.CurrentCallFrame.Function,
				Index:    msg.CurrentCallFrame.InstructionPointer,
			}]

			if d.singleStep || !d.running {
				isBreakPoint = true
			}

			fmt.Printf(
				"\033[2J\033[H%s\n---------------------------\n%s\n",
				programStr,
				*(d.core.Executor).(homescript.TestingVmExecutor).PrintBuf,
			)

			for {
				if isBreakPoint {
					// Wait for keypress.
					scanner := bufio.NewScanner(os.Stdin)
					if !scanner.Scan() {
						continue
					}
					inputLn := scanner.Text()
					command, err := parseDebuggerInput(inputLn)

					if err != nil {
						fmt.Printf("ERROR: %s\n", err)
						continue
					}

					breakoutT, err := d.interpret(command)
					if err != nil {
						fmt.Printf("ERROR: %s\n", err)
						continue
					}

					if breakoutT {
						break
					}

					continue
				}

				break
			}

			if !d.singleStep {
				time.Sleep(d.speedWait)
			}

			*d.debuggerResume <- struct{}{}
		}
	}
}

func (d *Debugger) interpret(command debuggerCommand) (breakOut bool, err error) {
	switch c := command.(type) {
	case speedDebuggerCommand:
		d.speedWait = time.Millisecond * time.Duration(c.Millis)
		fmt.Printf("New speed %v\n", d.speedWait)
	case nopDebuggerCommand:
		return false, nil
	case runDebuggerCommand:
		if d.running {
			return false, errors.New("Already running")
		}
		d.running = true
		return true, nil
	case breakpointDebuggerCommand:
		breakPoint := Breakpoint{
			Function: "",
			Index:    0,
		}

		if c.IsAsm {
			mangled, found := d.programOut.Mappings.Functions[c.FunctionOrFile]
			if !found {
				return false, fmt.Errorf("Illegal function name '%s'", c.FunctionOrFile)
			}

			breakPoint.Function = mangled

			breakPoint.Index = c.IndexOrLine
			instr := d.programOut.Functions[mangled]

			if c.IndexOrLine < 0 || int(c.IndexOrLine) >= len(instr) {
				return false, fmt.Errorf("Illegal instruction index, maximum is %d", len(instr))
			}
		} else {
			panic("TODO: not supported")
		}

		d.breakpoints[breakPoint] = struct{}{}

		fmt.Printf("Breakpoint set to %s:%d\n", c.FunctionOrFile, c.IndexOrLine)
	case infoDebuggerCommand:
		if !d.running {
			return false, errors.New("Not running")
		}

		switch c.Subcommand {
		case memoryInfoSubcommand:
			used := 0
			outp := make([]string, 0)
			for idx, v := range d.core.Memory {
				if v == nil {
					continue
				}

				disp, i := (*v).Display()
				if i != nil {
					panic(i)
				}

				outp = append(outp, fmt.Sprintf("%02d | %s", idx, disp))
				used++
			}

			fmt.Printf("MP at:      %03d\n", d.core.MemoryPointer)
			fmt.Printf("MAX memory: %03d\n", d.core.Limits.MaxMemorySize)
			fmt.Printf("Used	    %03d (%d%%)\n", used, int(float64(used)/float64(vmLimits.MaxMemorySize)*100))
			fmt.Println(strings.Join(outp, "\n"))
		case stackInfoSubcommand:
			stack := make([]string, 0)
			for _, v := range d.core.Stack {
				d, i := (*v).Display()
				if i != nil {
					panic(*i)
				}

				stack = append(stack, d)
			}
			stackStr := fmt.Sprintf("[%s]", strings.Join(stack, ", "))
			fmt.Println(stackStr)
		}
	case callStackDebuggerCommand:
		callstack := d.core.CallStack

		for idx, frame := range callstack {
			source := d.programOut.SourceMap[frame.Function][frame.InstructionPointer]
			fmt.Printf("%d | %s:%d (%s:%d:%d)\n", idx, frame.Function, frame.InstructionPointer, source.Filename, source.Start.Line, source.Start.Column)
		}

	case continueDebuggerCommand:
		if !d.running {
			return false, errors.New("Not running")
		}

		return true, nil
	case singleStepDebuggerCommand:
		if !d.running {
			d.running = true
		}
		d.singleStep = !d.singleStep
		fmt.Printf("Single step mode is now: %v\n", d.singleStep)
	case nil:
		panic("THIS IS NIL")
	default:
		panic("TODO: unhandled command")
	}

	return false, nil
}

// func TestingDebugConsumerAsm(
// 	debuggerOutput *chan runtime.DebugOutput,
// 	debuggerResume *chan struct{},
// 	core *runtime.Core,
// 	program compiler.CompileOutput,
// ) {
// 	for {
// 		select {
// 		case msg, open := <-*debuggerOutput:
// 			if !open {
// 				return
// 			}
//
// 			lineIdx := int(msg.CurrentCallFrame.InstructionPointer)
// 			programStr := program.AsmStringHighlight(true, &msg.CurrentCallFrame.Function, &lineIdx)
//
// 			// coreInfo := fmt.Sprintf("Corenum %d | I: %v | IP: %d | FP: %s MP=%d | CLSTCK: %v | STCKSS=%d | STCK: [%s] | MEM: [%s] | GLOB:  [%s]\n", core.Corenum, i, core.callFrame().InstructionPointer, self.callFrame().Function, self.MemoryPointer, self.CallStack, len(self.Stack), strings.Join(stack, ", "), strings.Join(mem, ", "), strings.Join(globals, ", ")),
//
// 			stack := make([]string, 0)
// 			for _, v := range core.Stack {
// 				d, i := (*v).Display()
// 				if i != nil {
// 					panic(*i)
// 				}
//
// 				stack = append(stack, d)
// 			}
// 			coreInfo := fmt.Sprintf("[%s]", strings.Join(stack, ", "))
//
// 			fmt.Printf(
// 				"\033[2J\033[H%s\n---------------------------\n%s\n---------------------------\n%s\n",
// 				coreInfo,
// 				programStr,
// 				*(core.Executor).(homescript.TestingVmExecutor).PrintBuf,
// 			)
//
// 		start:
// 			// Wait for keypress.
// 			inputLn := make([]byte, 10)
// 			fmt.Scanln(&inputLn)
// 			inputStr := string(inputLn)
//
// 			tokens := strings.Split(inputStr, " ")
//
// 			switch len(tokens) {
// 			case 0:
// 				// Do nothing
// 			default:
// 				switch tokens[0] {
// 				case "bt":
// 				case "i":
// 					if len(tokens) != 2 {
// 						fmt.Printf
// 					}
//
// 					switch to
//
// 					for idx, v := range core.Memory {
// 						if v == nil {
// 							continue
// 						}
//
// 						v, i := (*v).Display()
// 						if i != nil {
// 							panic(i)
// 						}
//
// 						fmt.Printf("%02d | %s\n", idx, v)
// 					}
//
// 					goto start
// 				}
// 			}
//
// 			*debuggerResume <- struct{}{}
// 		}
// 	}
// }
//
// func TestingDebugConsumerCode(
// 	debuggerOutput *chan runtime.DebugOutput,
// 	debuggerResume *chan struct{},
// 	core *runtime.Core,
// ) {
// 	const sleepTime = 500 * time.Millisecond
//
// 	hits := make(map[uint]int)
// 	colors := []int{0, 10, 2, 12, 4, 14, 3, 11, 1}
//
// 	for {
// 		select {
// 		case msg, open := <-*debuggerOutput:
// 			if !open {
// 				return
// 			}
//
// 			// Read input file
// 			program, err := os.ReadFile(msg.CurrentSpan.Filename)
// 			if err != nil {
// 				fmt.Printf("Debugger: cannot open input file `%s.hms`: %s\n", msg.CurrentSpan.Filename, err.Error())
// 				return
// 			}
//
// 			programStr := string(program)
// 			lines := strings.Split(programStr, "\n")
//
// 			lineIdx := msg.CurrentSpan.Start.Line - 1
// 			hits[lineIdx]++
//
// 			// Highlight active line
// 			for idx := range lines {
// 				lineHit := hits[uint(idx)]
// 				sumHits := 0
// 				for _, lineHitsI := range hits {
// 					sumHits += lineHitsI
// 				}
//
// 				cpuTimePercent := (float64(lineHit) / float64(sumHits))
//
// 				color := colors[int(cpuTimePercent*float64(len(colors)-1))]
//
// 				if idx == int(lineIdx) {
// 					lines[idx] = fmt.Sprintf("\x1b[4m\x1b[1;3%dm%s\x1b[0m       (%s)", color, lines[lineIdx], msg.CurrentInstruction)
// 				} else {
// 					lines[idx] = fmt.Sprintf("\x1b[1;3%dm%s\x1b[1;0m", color, lines[idx])
// 				}
// 			}
//
// 			fmt.Printf("\033[2J\033[H%s\n---------------------------\n%s\n", *(core.Executor).(homescript.TestingVmExecutor).PrintBuf, strings.Join(lines, "\n"))
//
// 			time.Sleep(sleepTime)
//
// 			*debuggerResume <- struct{}{}
// 		}
// 	}
// }
