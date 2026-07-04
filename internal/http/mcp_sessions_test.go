package httpserver

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

// failingMCPSessionPersistence returns errFailingPersistence from every
// method, simulating a transient database outage.
type failingMCPSessionPersistence struct{}

var errFailingPersistence = errors.New("simulated persistence failure")

func (failingMCPSessionPersistence) CreateMCPSession(context.Context, string, string, time.Time) error {
	return errFailingPersistence
}

func (failingMCPSessionPersistence) TouchMCPSession(context.Context, string, string, time.Time, time.Time) (bool, error) {
	return false, errFailingPersistence
}

func (failingMCPSessionPersistence) CloseMCPSession(context.Context, string, time.Time) (bool, error) {
	return false, errFailingPersistence
}

func (failingMCPSessionPersistence) ActiveMCPSessionCount(context.Context, time.Time) (int, error) {
	return 0, errFailingPersistence
}

func (failingMCPSessionPersistence) ActiveMCPSessionCountForSubject(context.Context, string, time.Time) (int, error) {
	return 0, errFailingPersistence
}

func (failingMCPSessionPersistence) AppendMCPEvent(context.Context, string, []byte, time.Time) (string, []byte, error) {
	return "", nil, errFailingPersistence
}

func (failingMCPSessionPersistence) ListMCPEvents(context.Context, string, string, int) ([]string, [][]byte, error) {
	return nil, nil, errFailingPersistence
}

// TestMCPSessionStoreDegradesGracefullyOnPersistenceFailure is a regression
// test for a real bug: every persisted-store method used to panic on every
// database error, crashing the entire process on a transient DB hiccup —
// serious given many concurrent MCP SSE sessions each poll the database
// independently, so one blip could take down every session at once. Each
// operation must now fail closed (report "not found"/"refused"/zero) rather
// than panic.
func TestMCPSessionStoreDegradesGracefullyOnPersistenceFailure(t *testing.T) {
	store := newPersistedMCPHTTPSessionStore(failingMCPSessionPersistence{})

	if store.create("session-a", "subject-1") {
		t.Fatalf("create should refuse the session when persistence fails, not panic")
	}
	if store.existsForSubject("session-a", "subject-1") {
		t.Fatalf("existsForSubject should report false when persistence fails, not panic")
	}

	// appendEvent/terminate/replayAndSubscribe all short-circuit on "session
	// not found" before ever reaching a persistence call, since create()
	// above never inserted one — inject an in-memory session directly (same
	// package as the store) so these exercise their own persistence-error
	// paths instead of the already-covered not-found path.
	store.mu.Lock()
	store.sessions["session-a"] = &mcpHTTPSession{id: "session-a", subject: "subject-1", subs: make(map[int64]chan mcpHTTPEvent)}
	store.mu.Unlock()

	if _, ok := store.appendEvent("session-a", []byte("{}")); ok {
		t.Fatalf("appendEvent should report failure when persistence fails, not panic")
	}
	if _, _, _, ok := store.replayAndSubscribe("session-a", ""); ok {
		t.Fatalf("replayAndSubscribe should report failure when persistence fails, not panic")
	}
	if store.terminate("session-a") {
		t.Fatalf("terminate should report failure when persistence fails, not panic")
	}
	if count := store.activeSessionCount(); count != 0 {
		t.Fatalf("activeSessionCount should degrade to 0 when persistence fails, got %d", count)
	}
}

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
