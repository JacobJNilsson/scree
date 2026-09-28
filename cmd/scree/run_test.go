package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantStdout string
		wantStderr string
		wantCode   int
	}{
		{name: "version", args: []string{"version"}, wantStdout: "scree 0.1.0-dev\n"},
		{name: "missing subcommand", wantStderr: usage + "\n", wantCode: 1},
		{name: "unknown subcommand", args: []string{"audit"}, wantStderr: usage + "\n", wantCode: 1},
		{name: "extra argument", args: []string{"version", "x"}, wantStderr: usage + "\n", wantCode: 1},
		{name: "help flag", args: []string{"-h"}, wantStderr: usage + "\n", wantCode: 1},
		{
			name:       "unknown flag",
			args:       []string{"-x"},
			wantStderr: "flag provided but not defined: -x\n" + usage + "\n",
			wantCode:   1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, &stdout, &stderr)
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d", code, tc.wantCode)
			}
			if got := stdout.String(); got != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", got, tc.wantStdout)
			}
			if got := stderr.String(); got != tc.wantStderr {
				t.Errorf("stderr = %q, want %q", got, tc.wantStderr)
			}
		})
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestRunVersionWriteFailure(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"version"}, failingWriter{}, &stderr); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
