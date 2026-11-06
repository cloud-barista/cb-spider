package resources

import (
	"testing"
	"time"
)

func TestEndpointRetryTrackerBackoffAndBudget(t *testing.T) {
	tr := &endpointRetryTracker{states: map[string]*endpointRetryState{}}
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)

	// 1st retry is immediate.
	if ok, n, _ := tr.acquire("ap-seoul", "cls-a", now); !ok || n != 1 {
		t.Fatalf("1st retry: got granted=%v attempt=%d, want true/1", ok, n)
	}
	// 2nd retry waits for the 5-minute backoff.
	if ok, n, next := tr.acquire("ap-seoul", "cls-a", now.Add(4*time.Minute)); ok || n != 1 || !next.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("early 2nd retry: got granted=%v attempt=%d next=%v", ok, n, next)
	}
	if ok, n, _ := tr.acquire("ap-seoul", "cls-a", now.Add(5*time.Minute)); !ok || n != 2 {
		t.Fatalf("2nd retry: got granted=%v attempt=%d, want true/2", ok, n)
	}
	// 3rd retry waits 10 minutes after the 2nd.
	if ok, _, _ := tr.acquire("ap-seoul", "cls-a", now.Add(14*time.Minute)); ok {
		t.Fatalf("early 3rd retry should not be granted")
	}
	if ok, n, _ := tr.acquire("ap-seoul", "cls-a", now.Add(15*time.Minute)); !ok || n != 3 {
		t.Fatalf("3rd retry: got granted=%v attempt=%d, want true/3", ok, n)
	}
	// Budget exhausted: no retry, zero nextAt.
	if ok, n, next := tr.acquire("ap-seoul", "cls-a", now.Add(time.Hour)); ok || n != maxEndpointRetries || !next.IsZero() {
		t.Fatalf("exhausted: got granted=%v attempt=%d next=%v", ok, n, next)
	}

	// Other clusters are independent.
	if ok, _, _ := tr.acquire("ap-seoul", "cls-b", now.Add(time.Hour)); !ok {
		t.Fatalf("independent cluster should get its own budget")
	}

	// reset (endpoint Created / cluster deleted) grants a fresh budget.
	tr.reset("ap-seoul", "cls-a")
	if ok, n, _ := tr.acquire("ap-seoul", "cls-a", now.Add(time.Hour)); !ok || n != 1 {
		t.Fatalf("after reset: got granted=%v attempt=%d, want true/1", ok, n)
	}
}

func TestEndpointRetryTrackerRelease(t *testing.T) {
	tr := &endpointRetryTracker{states: map[string]*endpointRetryState{}}
	now := time.Now()

	if ok, _, _ := tr.acquire("ap-seoul", "cls-a", now); !ok {
		t.Fatalf("1st retry should be granted")
	}
	// The teardown request failed: the attempt is given back and may be retried right away.
	tr.release("ap-seoul", "cls-a")
	if ok, n, _ := tr.acquire("ap-seoul", "cls-a", now); !ok || n != 1 {
		t.Fatalf("after release: got granted=%v attempt=%d, want true/1", ok, n)
	}
}
