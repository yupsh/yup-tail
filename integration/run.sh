#!/bin/sh
# Integration checks for yup-tail, run inside a Debian (GNU coreutils) container.
#
# parity ...   — yup-tail must produce byte-identical output to GNU `tail` for
#                stdin piped through both.
# parityf ...  — same, but with a file operand (a fixture written to /work).
# assert WANT  — yup-tail must produce WANT exactly (used where yup-tail diverges
#                from GNU by design; see cmd-tail COMPATIBILITY.md).
#
# Follow mode (-f) is intentionally NOT exercised: the batch, EOF-terminated
# pipeline model has no streaming source, so -f is unimplemented.
set -eu

fails=0

# Multi-line sample fed to stdin parity cases (15 numbered lines).
sample='1
2
3
4
5
6
7
8
9
10
11
12
13
14
15'

parity() {
	ours=$(printf '%s\n' "$sample" | yup-tail "$@" 2>/dev/null || true)
	gnu=$(printf '%s\n' "$sample" | tail "$@" 2>/dev/null || true)
	if [ "$ours" = "$gnu" ]; then
		printf 'ok    parity  tail %s\n' "$*"
	else
		printf 'FAIL  parity  tail %s\n        gnu:  %s\n        ours: %s\n' "$*" "$gnu" "$ours"
		fails=$((fails + 1))
	fi
}

parityf() {
	ours=$(yup-tail "$@" 2>/dev/null || true)
	gnu=$(tail "$@" 2>/dev/null || true)
	if [ "$ours" = "$gnu" ]; then
		printf 'ok    parity  tail %s\n' "$*"
	else
		printf 'FAIL  parity  tail %s\n        gnu:  %s\n        ours: %s\n' "$*" "$gnu" "$ours"
		fails=$((fails + 1))
	fi
}

assert() {
	want=$1
	shift
	got=$(printf '%s\n' "$sample" | yup-tail "$@" 2>/dev/null || true)
	if [ "$got" = "$want" ]; then
		printf 'ok    assert  tail %s\n' "$*"
	else
		printf 'FAIL  assert  tail %s\n        want: %s\n        got:  %s\n' "$*" "$want" "$got"
		fails=$((fails + 1))
	fi
}

# Default: last 10 lines of stdin.
parity

# -n N: last N lines.
parity -n 3
parity -n 1
parity -n 20

# -n +N: output from line N onward (1-indexed).
parity -n +13
parity -n +1

# File operand: a single file behaves like stdin (no header) for -n.
printf '%s\n' "$sample" > /work.txt
parityf -n 4 /work.txt

# Documented divergence: -c N emits the trailing bytes as one record, and the
# line-oriented byte sink appends a record newline, so the RAW output is one
# byte longer than GNU `tail -c` (a trailing "\n"). A shell `$(...)` comparison
# hides this by stripping trailing newlines, so compare exact byte counts: GNU
# emits 6 bytes ("14\n15\n"), ours emits 7 ("14\n15\n\n").
ours_n=$(printf '%s\n' "$sample" | yup-tail -c 6 | wc -c | tr -d ' ')
gnu_n=$(printf '%s\n' "$sample" | tail -c 6 | wc -c | tr -d ' ')
if [ "$ours_n" = "7" ] && [ "$gnu_n" = "6" ]; then
	printf 'ok    assert  tail -c 6 (raw: 1 extra trailing newline vs GNU)\n'
else
	printf 'FAIL  assert  tail -c 6 (raw byte count)\n        gnu:  %s bytes\n        ours: %s bytes\n' "$gnu_n" "$ours_n"
	fails=$((fails + 1))
fi

# Documented divergence: multiple files are concatenated WITHOUT the GNU
# "==> FILE <==" header. GNU prints a header before each file; ours does not.
printf 'a\nb\n' > /a.txt
printf 'c\nd\n' > /b.txt
got=$(yup-tail /a.txt /b.txt 2>/dev/null || true)
want=$(printf 'a\nb\nc\nd')
if [ "$got" = "$want" ]; then
	printf 'ok    assert  tail (multi-file: concatenated, no header)\n'
else
	printf 'FAIL  assert  tail (multi-file)\n        want: %s\n        got:  %s\n' "$want" "$got"
	fails=$((fails + 1))
fi

if [ "$fails" -ne 0 ]; then
	printf '\n%s check(s) failed\n' "$fails"
	exit 1
fi
printf '\nall checks passed\n'
