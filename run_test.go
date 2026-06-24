package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		version    string
		args       []string
		stdin      string
		files      map[string]string
		wantOut    string
		wantCode   int
		wantErrSub string
	}{
		{
			name:    "default count",
			args:    []string{"tail"},
			stdin:   "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
			wantOut: "3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n",
		},
		{
			name:    "explicit lines",
			args:    []string{"tail", "-n", "3"},
			stdin:   "a\nb\nc\nd\ne\n",
			wantOut: "c\nd\ne\n",
		},
		{
			name:    "file source",
			args:    []string{"tail", "-n", "2", "/in.txt"},
			files:   map[string]string{"/in.txt": "one\ntwo\nthree\n"},
			wantOut: "two\nthree\n",
		},
		{
			name:    "from line plus N",
			args:    []string{"tail", "-n", "+3"},
			stdin:   "a\nb\nc\nd\ne\n",
			wantOut: "c\nd\ne\n",
		},
		{
			// cmd-tail's byte mode emits the trailing bytes as one stream
			// item (here "rld\n"), and the line-oriented byte sink appends a
			// record newline — so -c output ends with an extra "\n" vs GNU.
			// See cmd-tail COMPATIBILITY.md.
			name:    "bytes",
			args:    []string{"tail", "-c", "4"},
			stdin:   "hello\nworld\n",
			wantOut: "rld\n\n",
		},
		{
			name:       "invalid lines value errors",
			args:       []string{"tail", "-n", "x"},
			wantCode:   1,
			wantErrSub: "invalid number of lines",
		},
		{
			name:       "invalid from-line value errors",
			args:       []string{"tail", "-n", "+x"},
			wantCode:   1,
			wantErrSub: "invalid number of lines",
		},
		{
			name:    "version flag reports injected version",
			version: "1.2.3",
			args:    []string{"tail", "--version"},
			wantOut: "tail version 1.2.3\n",
		},
		{
			name:       "unknown flag errors",
			args:       []string{"tail", "--nope"},
			wantCode:   1,
			wantErrSub: "tail:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for path, content := range tc.files {
				if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
					t.Fatalf("write fixture %s: %v", path, err)
				}
			}

			var out, errOut bytes.Buffer
			code := run(tc.version, tc.args, strings.NewReader(tc.stdin), &out, &errOut, fs)

			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr=%q)", code, tc.wantCode, errOut.String())
			}
			if tc.wantErrSub == "" && out.String() != tc.wantOut {
				t.Fatalf("stdout = %q, want %q", out.String(), tc.wantOut)
			}
			if tc.wantErrSub != "" && !strings.Contains(errOut.String(), tc.wantErrSub) {
				t.Fatalf("stderr = %q, want substring %q", errOut.String(), tc.wantErrSub)
			}
		})
	}
}

func Test_main(t *testing.T) {
	origExit, origRun := osExit, runCLI
	t.Cleanup(func() { osExit, runCLI = origExit, origRun })

	gotCode := -1
	osExit = func(code int) { gotCode = code }
	runCLI = func(string, []string, io.Reader, io.Writer, io.Writer, afero.Fs) int { return 7 }

	main()

	if gotCode != 7 {
		t.Fatalf("main propagated exit code %d, want 7", gotCode)
	}
}
