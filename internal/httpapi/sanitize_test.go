package httpapi

import "testing"

func TestSanitizeErrorRedactsDBLeakage(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"database is locked", "internal error"},
		{"unique constraint failed: users.username", "resource conflict"},
		{"FOREIGN KEY constraint failed", "internal error"},
		{"no such table: api_tokens", "internal error"},
		{"open /etc/harborx/somefile: permission denied", "internal error"},
		{"password is too short", "password is too short"},
		{"username is required", "username is required"},
	}
	for _, tt := range tests {
		got := sanitizeError(tt.in)
		if got != tt.out {
			t.Fatalf("sanitizeError(%q) = %q, want %q", tt.in, got, tt.out)
		}
	}
}
