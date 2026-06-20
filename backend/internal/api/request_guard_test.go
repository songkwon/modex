package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestLimiterPermitMatchesAllow(t *testing.T) {
	limiter := newRequestLimiter()
	policy := limitPolicy{requests: 1, window: time.Minute}
	if !limiter.permit(context.Background(), "client", policy) {
		t.Fatal("first request should be permitted")
	}
	if limiter.permit(context.Background(), "client", policy) {
		t.Fatal("second request in window should be denied")
	}
}

func TestAcquireDeploySlotBoundsConcurrency(t *testing.T) {
	s := &Server{deploy: &localDeployLimiter{sem: make(chan struct{}, 1)}}
	release, ok := s.acquireDeploySlot(context.Background())
	if !ok {
		t.Fatal("first slot should be acquired")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := s.acquireDeploySlot(cancelled); ok {
		t.Fatal("slot should not be acquired while the only slot is held")
	}
	release()
	again, ok := s.acquireDeploySlot(context.Background())
	if !ok {
		t.Fatal("slot should be reusable after release")
	}
	again()
}

func TestAcquireDeploySlotUnboundedWhenDisabled(t *testing.T) {
	s := &Server{}
	if _, ok := s.acquireDeploySlot(context.Background()); !ok {
		t.Fatal("nil semaphore should always grant a slot")
	}
}

func TestRequestLimiterResetsAfterWindow(t *testing.T) {
	limiter := newRequestLimiter()
	policy := limitPolicy{requests: 2, window: time.Minute}
	now := time.Now()
	if !limiter.allow("client", policy, now) {
		t.Fatal("first request should be allowed")
	}
	if !limiter.allow("client", policy, now) {
		t.Fatal("second request should be allowed")
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
