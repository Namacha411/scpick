package remotefs

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestCopyWithProgressReportsBytes(t *testing.T) {
	content := strings.Repeat("x", 100*1024) // exceeds the 32KB copy buffer
	var dst bytes.Buffer
	var lastDone, lastTotal int64
	calls := 0

	err := copyWithProgress(&dst, strings.NewReader(content), int64(len(content)), func(done, total int64) {
		calls++
		lastDone, lastTotal = done, total
	})
	if err != nil {
		t.Fatalf("copyWithProgress: %v", err)
	}
	if dst.String() != content {
		t.Fatal("copied content does not match source")
	}
	if calls < 2 {
		t.Fatalf("expected multiple progress callbacks for a multi-chunk copy, got %d", calls)
	}
	if lastDone != int64(len(content)) || lastTotal != int64(len(content)) {
		t.Errorf("final progress = %d/%d, want %d/%d", lastDone, lastTotal, len(content), len(content))
	}
}

func TestCopyWithProgressNilCallback(t *testing.T) {
	content := "hello world"
	var dst bytes.Buffer
	if err := copyWithProgress(&dst, strings.NewReader(content), int64(len(content)), nil); err != nil {
		t.Fatalf("copyWithProgress: %v", err)
	}
	if dst.String() != content {
		t.Fatal("copied content does not match source")
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

func TestCopyWithProgressPropagatesReadError(t *testing.T) {
	wantErr := errors.New("boom")
	var dst bytes.Buffer
	err := copyWithProgress(&dst, errReader{wantErr}, 10, func(int64, int64) {})
	if !errors.Is(err, wantErr) {
		t.Fatalf("copyWithProgress error = %v, want %v", err, wantErr)
	}
}

type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

func TestCopyWithProgressPropagatesWriteError(t *testing.T) {
	wantErr := errors.New("disk full")
	err := copyWithProgress(errWriter{wantErr}, strings.NewReader("data"), 4, func(int64, int64) {})
	if !errors.Is(err, wantErr) {
		t.Fatalf("copyWithProgress error = %v, want %v", err, wantErr)
	}
}

var _ io.Reader = errReader{}
var _ io.Writer = errWriter{}
