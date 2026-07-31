package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "version",
			args:       []string{"--version"},
			wantCode:   0,
			wantStdout: "scpick ",
		},
		{
			name:       "help long",
			args:       []string{"--help"},
			wantCode:   0,
			wantStderr: "Usage:",
		},
		{
			name:       "help short",
			args:       []string{"-h"},
			wantCode:   0,
			wantStderr: "Usage:",
		},
		{
			name:     "unknown flag",
			args:     []string{"--bogus"},
			wantCode: 2,
		},
		{
			name:     "unexpected positional arg",
			args:     []string{"somehost"},
			wantCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			gotCode := run(tt.args, &stdout, &stderr)

			if gotCode != tt.wantCode {
				t.Errorf("run(%v) exit code = %d, want %d (stdout=%q stderr=%q)", tt.args, gotCode, tt.wantCode, stdout.String(), stderr.String())
			}
			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("run(%v) stdout = %q, want substring %q", tt.args, stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("run(%v) stderr = %q, want substring %q", tt.args, stderr.String(), tt.wantStderr)
			}
		})
	}
}
