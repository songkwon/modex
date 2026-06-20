package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestLimiterResetsAfterWindow(t *testing.T) {
	limiter := newRequestLimiter()
	policy := limitPolicy{requests: 2, window: time.Minute}
	now := time.Now()
	if !limiter.allow("client", policy, now) || !limiter.allow("client", policy, now) {
		t.Fatal("initial requests should be allowed")
	}
	if limiter.allow("client", policy, now) {
		t.Fatal("request above limit should be denied")
	}
	if !limiter.allow("client", policy, now.Add(time.Minute)) {
		t.Fatal("request after window should be allowed")
	}
}

func TestClientIPOnlyTrustsForwardedHeaderWhenConfigured(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.10:1234"
	r.Header.Set("X-Forwarded-For", "198.51.100.2")
	t.Setenv("TRUST_PROXY_HEADERS", "false")
	if got := clientIP(r); got != "192.0.2.10" {
		t.Fatalf("clientIP = %q", got)
	}
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	if got := clientIP(r); got != "198.51.100.2" {
		t.Fatalf("trusted clientIP = %q", got)
	}
}
