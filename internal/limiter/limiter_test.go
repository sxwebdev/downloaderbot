package limiter

import (
	"context"
	"strings"
	"testing"
)

func TestLimiter_AllowAndExceed(t *testing.T) {
	// 3 requests per second (per key)
	l, err := New("3-S")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	for i := range 3 {
		if err := l.Allow(ctx, "user-1"); err != nil {
			t.Fatalf("unexpected error on call %d: %v", i+1, err)
		}
	}

	if err := l.Allow(ctx, "user-1"); err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("expected rate-limit error, got %v", err)
	}

	// different key still allowed
	if err := l.Allow(ctx, "user-2"); err != nil {
		t.Fatalf("different key should be allowed: %v", err)
	}
}

func TestLimiter_InvalidFormat(t *testing.T) {
	if _, err := New("garbage"); err == nil {
		t.Fatal("expected error for invalid rate format")
	}
}
