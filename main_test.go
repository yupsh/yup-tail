package main

import (
	"context"
	"errors"
	"testing"

	clix "github.com/gloo-foo/cli"
	"github.com/spf13/afero"
	urf "github.com/urfave/cli/v3"
)

// parse runs args through a bare command carrying the wrapper's flags and
// returns the parsed accessor, so flag-dependent helpers are tested against real
// parsed flags.
func parse(t *testing.T, args ...string) *urf.Command {
	t.Helper()
	var got *urf.Command
	app := &urf.Command{
		Name:   name,
		Flags:  flags(),
		Action: func(_ context.Context, c *urf.Command) error { got = c; return nil },
	}
	if err := app.Run(context.Background(), args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return got
}

func TestOptions(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    int
		wantErr bool
	}{
		{"none", []string{name}, 0, false},
		{"bytes", []string{name, "-c", "5"}, 1, false},
		{"lines", []string{name, "-n", "10"}, 1, false},
		{"from-line", []string{name, "-n", "+5"}, 1, false},
		{"invalid-lines", []string{name, "-n", "abc"}, 0, true},
		{"invalid-from-line", []string{name, "-n", "+abc"}, 0, true},
		{"bytes-over-lines", []string{name, "-c", "5", "-n", "3"}, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := options(parse(t, tc.args...))
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidLines) {
					t.Fatalf("err=%v, want ErrInvalidLines", err)
				}
				return
			}
			if err != nil || len(opts) != tc.want {
				t.Fatalf("options len=%d err=%v, want len=%d", len(opts), err, tc.want)
			}
		})
	}
}

func TestBuild_Success(t *testing.T) {
	src, filter, err := build(clix.Invocation{Args: parse(t, name, "-n", "10"), Fs: afero.NewMemMapFs()})
	if err != nil || src == nil || filter == nil {
		t.Fatalf("build: src=%v filter=%v err=%v", src, filter, err)
	}
}

func TestBuild_InvalidLinesIsError(t *testing.T) {
	src, filter, err := build(clix.Invocation{Args: parse(t, name, "-n", "abc"), Fs: afero.NewMemMapFs()})
	if !errors.Is(err, ErrInvalidLines) {
		t.Fatalf("err=%v, want ErrInvalidLines", err)
	}
	if src != nil || filter != nil {
		t.Fatalf("src=%v filter=%v, want both nil on error", src, filter)
	}
	if err.Error() != string(ErrInvalidLines) {
		t.Fatalf("message=%q, want %q", err.Error(), string(ErrInvalidLines))
	}
}

func Test_main(t *testing.T) {
	orig := runMain
	t.Cleanup(func() { runMain = orig })
	var gotName clix.Name
	runMain = func(s clix.Spec, _ clix.Version) { gotName = s.Name }
	main()
	if gotName != name {
		t.Fatalf("main used spec %q, want %s", gotName, name)
	}
}
