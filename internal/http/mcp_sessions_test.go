package httpserver

import (
	"strconv"
	"testing"
	"time"
)

func TestMCPSessionStoreEvictsIdleSessions(t *testing.T) {
	current := time.Unix(0, 0).UTC()
	store := newMCPHTTPSessionStore()
	store.now = func() time.Time { return current }

	store.create("session-a", "subject-1")
	if !store.existsForSubject("session-a", "subject-1") {
		t.Fatalf("session-a should exist immediately after creation")
	}

	// Advance past the TTL and trigger a store operation; the idle session is evicted.
	current = current.Add(mcpSessionTTL + time.Minute)
	if store.existsForSubject("session-a", "subject-1") {
		t.Fatalf("idle session-a should have been evicted after the TTL")
	}
}

func TestMCPSessionStoreCapsSessionsPerSubject(t *testing.T) {
	current := time.Unix(0, 0).UTC()
	store := newMCPHTTPSessionStore()
	store.now = func() time.Time { return current }

	for i := 0; i < maxMCPSessionsPerSubject; i++ {
		if !store.create("session-"+strconv.Itoa(i), "subject-cap") {
			t.Fatalf("session %d should be admitted within the per-subject cap", i)
		}
	}
	if store.create("session-overflow", "subject-cap") {
		t.Fatalf("session beyond the per-subject cap should be refused")
	}
	// A different subject is unaffected by another subject's cap.
	if !store.create("session-other", "subject-other") {
		t.Fatalf("a different subject should still be admitted")
	}
}

func TestMCPSessionStoreKeepsActiveSessions(t *testing.T) {
	current := time.Unix(0, 0).UTC()
	store := newMCPHTTPSessionStore()
	store.now = func() time.Time { return current }

	store.create("session-b", "subject-2")

	// Touch the session just before each TTL boundary so it stays active.
	for step := 0; step < 5; step++ {
		current = current.Add(mcpSessionTTL - time.Minute)
		if !store.existsForSubject("session-b", "subject-2") {
			t.Fatalf("active session-b should survive step %d", step)
		}
	}
}
