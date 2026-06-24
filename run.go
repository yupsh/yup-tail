package main

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	command "github.com/gloo-foo/cmd-tail"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

const (
	flagLines = "lines"
	flagBytes = "bytes"
)

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `tail [OPTIONS] [FILE...]

Print the last 10 lines of each FILE to standard output.
With more than one FILE, the files are concatenated (no "==> FILE <==" header).
With no FILE, or when FILE is -, read standard input.`

// Error is the sole error type the wrapper emits, so every failure path is
// testable with errors.Is.
type Error string

func (e Error) Error() string { return string(e) }

// ErrInvalidLines is returned when the --lines value is not a number (an
// optional leading "+" selecting from-line mode is permitted).
const ErrInvalidLines Error = "invalid number of lines"

// init replaces urfave/cli's default --version/-v flag with a --version-only
// flag, freeing the single-letter -v for command flags (e.g. grep -v) while
// still exposing the injected build version.
func init() {
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
}

// run builds and executes the tail CLI against the injected version, I/O, and
// filesystem, returning the process exit code.
func run(version string, args []string, stdin io.Reader, stdout, stderr io.Writer, fs afero.Fs) int {
	cmd := newApp(version, stdin, stdout, fs)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), args); err != nil {
		_, _ = fmt.Fprintf(stderr, "tail: %v\n", err)
		return 1
	}
	return 0
}

func newApp(version string, stdin io.Reader, stdout io.Writer, fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:            "tail",
		Version:         version,
		Usage:           "output the last part of files",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: flagLines, Aliases: []string{"n"}, Usage: "output the last NUM lines; with +NUM, output from line NUM (default: 10)"},
			&cli.IntFlag{Name: flagBytes, Aliases: []string{"c"}, Usage: "output the last NUM bytes"},
		},
		Action: action(stdin, stdout, fs),
	}
}

func action(stdin io.Reader, stdout io.Writer, fs afero.Fs) cli.ActionFunc {
	return func(_ context.Context, c *cli.Command) error {
		opts, err := options(c)
		if err != nil {
			return err
		}
		_, err = gloo.Run(source(c, stdin, fs), gloo.ByteWriteTo(stdout), command.Tail(opts...))
		return err
	}
}

func source(c *cli.Command, stdin io.Reader, fs afero.Fs) any {
	if c.NArg() == 0 {
		return gloo.ByteReaderSource([]io.Reader{stdin})
	}
	files := make([]gloo.File, c.NArg())
	for i := range files {
		files[i] = gloo.File(c.Args().Get(i))
	}
	return gloo.ByteFileSource(fs, files)
}

// options translates the selected flags into constructor option values. -c
// takes precedence over -n, matching cmd-tail's mode resolution.
func options(c *cli.Command) ([]any, error) {
	if c.IsSet(flagBytes) {
		return []any{command.TailBytes(c.Int(flagBytes))}, nil
	}
	if c.IsSet(flagLines) {
		return lineOption(c.String(flagLines))
	}
	return nil, nil
}

// lineOption parses the --lines value: a leading "+" selects from-line mode
// (TailFromLine), otherwise the value is a trailing line count (TailLines).
func lineOption(value string) ([]any, error) {
	if from, ok := strings.CutPrefix(value, "+"); ok {
		return fromLineOption(from)
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return nil, ErrInvalidLines
	}
	return []any{command.TailLines(n)}, nil
}

// fromLineOption parses the digits following a "+" prefix into from-line mode.
func fromLineOption(digits string) ([]any, error) {
	n, err := strconv.Atoi(digits)
	if err != nil {
		return nil, ErrInvalidLines
	}
	return []any{command.TailFromLine(n)}, nil
}
