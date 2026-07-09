package transfer

import "testing"

func TestRemoteParent(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/", "/"},
		{"/home/user", "/home"},
		{"/home/user/docs", "/home/user"},
	}
	for _, tt := range tests {
		if got := remoteParent(tt.in); got != tt.want {
			t.Errorf("remoteParent(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
