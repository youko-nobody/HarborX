package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginRateLimiterAllowsWithinWindow(t *testing.T) {
	limiter := newLoginRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !limiter.allow("1.2.3.4") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
}

func TestLoginRateLimiterBlocksAfterMax(t *testing.T) {
	limiter := newLoginRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		limiter.allow("1.2.3.4")
	}
	if limiter.allow("1.2.3.4") {
		t.Fatal("attempt beyond max should be blocked")
	}
}

func TestLoginRateLimiterSeparateKeys(t *testing.T) {
	limiter := newLoginRateLimiter(2, time.Minute)
	limiter.allow("1.1.1.1")
	limiter.allow("1.1.1.1")
	if limiter.allow("1.1.1.1") {
		t.Fatal("first key should be exhausted")
	}
	if !limiter.allow("2.2.2.2") {
		t.Fatal("second key must not be affected")
	}
}

func TestLoginRateLimiterWindowExpiry(t *testing.T) {
	limiter := newLoginRateLimiter(1, 50*time.Millisecond)
	limiter.allow("1.2.3.4")
	if limiter.allow("1.2.3.4") {
		t.Fatal("should still be blocked inside the window")
	}
	time.Sleep(80 * time.Millisecond)
	if !limiter.allow("1.2.3.4") {
		t.Fatal("attempt after window expiry should be allowed")
	}
}

func TestClientIPFromForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	if got := clientIP(r); got != "203.0.113.9" {
		t.Fatalf("clientIP = %q, want first hop", got)
	}
}

func TestClientIPFromRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "192.168.1.50:51234"
	if got := clientIP(r); got != "192.168.1.50" {
		t.Fatalf("clientIP = %q, want host part of RemoteAddr", got)
	}
}
