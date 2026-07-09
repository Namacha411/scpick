package transfer

import "testing"

func TestOverwriteGateRemembersAll(t *testing.T) {
	gate := &overwriteGate{}
	calls := 0
	confirm := func(string, int64, int64) OverwriteDecision {
		calls++
		return OverwriteAll
	}

	if d := gate.decide(confirm, "a", 1, 2); d != OverwriteYes {
		t.Errorf("first decide = %v, want OverwriteYes", d)
	}
	if d := gate.decide(confirm, "b", 1, 2); d != OverwriteYes {
		t.Errorf("second decide = %v, want OverwriteYes (from cached all)", d)
	}
	if calls != 1 {
		t.Errorf("confirm called %d times, want 1", calls)
	}
}

func TestOverwriteGatePassesThroughSkip(t *testing.T) {
	gate := &overwriteGate{}
	confirm := func(string, int64, int64) OverwriteDecision { return OverwriteSkip }
	if d := gate.decide(confirm, "a", 1, 2); d != OverwriteSkip {
		t.Errorf("decide = %v, want OverwriteSkip", d)
	}
	// A skip must not be remembered as "all".
	confirm2 := func(string, int64, int64) OverwriteDecision { return OverwriteYes }
	if d := gate.decide(confirm2, "b", 1, 2); d != OverwriteYes {
		t.Errorf("decide after skip = %v, want OverwriteYes", d)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0B"},
		{999, "999B"},
		{1024, "1.0KiB"},
		{1536, "1.5KiB"},
		{1024 * 1024, "1.0MiB"},
	}
	for _, tt := range tests {
		if got := formatBytes(tt.in); got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
