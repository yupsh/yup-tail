// Command yup-tail is the CLI wrapper around github.com/gloo-foo/cmd-tail.
package main

import (
	"strconv"
	"strings"

	clix "github.com/gloo-foo/cli"
	command "github.com/gloo-foo/cmd-tail"
	urf "github.com/urfave/cli/v3"
)

// version is the build version. It defaults to "dev" for local builds and is
// overridden at release time via the linker: -ldflags "-X main.version=<v>".
var version = "dev"

const (
	name      = "tail"
	flagLines = "lines"
	flagBytes = "bytes"
)

// Error is the sole error type the wrapper emits, so every failure path is
// testable with errors.Is.
type Error string

func (e Error) Error() string { return string(e) }

// ErrInvalidLines is returned when the --lines value is not a number (an
// optional leading "+" selecting from-line mode is permitted).
const ErrInvalidLines Error = "invalid number of lines"

// synopsis is the multi-line --help usage block; urfave/cli indents it three
// spaces, so the lines stay flush-left.
const synopsis = `tail [OPTIONS] [FILE...]

Print the last 10 lines of each FILE to standard output.
With more than one FILE, the files are concatenated (no "==> FILE <==" header).
With no FILE, or when FILE is -, read standard input.`

// spec declares the tail wrapper: a file-or-stdin filter with -n/-c selection.
var spec = clix.Spec{
	Name:     name,
	Summary:  "output the last part of files",
	Synopsis: synopsis,
	Build:    build,
	Flags:    flags(),
}

// flags returns fresh flag instances. It is a constructor rather than a shared
// slice because urfave/cli flag structs retain per-parse state, so each parse
// (including tests) must build its own.
func flags() []urf.Flag {
	return []urf.Flag{
		&urf.StringFlag{
			Name:    flagLines,
			Aliases: []string{"n"},
			Usage:   "output the last NUM lines; with +NUM, output from line NUM (default: 10)",
		},
		&urf.IntFlag{Name: flagBytes, Aliases: []string{"c"}, Usage: "output the last NUM bytes"},
	}
}

// build maps the invocation to tail's pipeline: a file-or-stdin source into the
// tail command configured by the flags. An invalid --lines value is a usage
// error.
func build(inv clix.Invocation) (clix.Source, clix.Command, error) {
	opts, err := options(inv.Args)
	if err != nil {
		return nil, nil, err
	}
	return clix.OperandsOrStdin(inv), command.Tail(opts...), nil
}

// options translates the selected flags into constructor option values. -c
// takes precedence over -n, matching cmd-tail's mode resolution.
func options(c *urf.Command) ([]any, error) {
	if c.IsSet(flagBytes) {
		return []any{command.TailBytes(c.Int(flagBytes))}, nil
	}
	if c.IsSet(flagLines) {
		return lineOption(lineSpec(c.String(flagLines)))
	}
	return nil, nil
}

// lineSpec is the --lines/-n value: a trailing line count ("10"), or a "+N"
// starting line.
type lineSpec string

// lineOption parses the --lines value: a leading "+" selects from-line mode
// (TailFromLine), otherwise the value is a trailing line count (TailLines).
func lineOption(value lineSpec) ([]any, error) {
	if from, ok := strings.CutPrefix(string(value), "+"); ok {
		return fromLineOption(startLine(from))
	}
	n, err := strconv.Atoi(string(value))
	if err != nil {
		return nil, ErrInvalidLines
	}
	return []any{command.TailLines(n)}, nil
}

// startLine is the textual 1-based starting line number following the "+" of a
// from-line --lines spec.
type startLine string

// fromLineOption parses the digits following a "+" prefix into from-line mode.
func fromLineOption(digits startLine) ([]any, error) {
	n, err := strconv.Atoi(string(digits))
	if err != nil {
		return nil, ErrInvalidLines
	}
	return []any{command.TailFromLine(n)}, nil
}

// runMain is an indirection seam so main's wiring is testable without spawning
// the process; a test swaps it and restores it.
var runMain = clix.Main

func main() { runMain(spec, version) }
