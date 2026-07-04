package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const mcpSessionHeader = "Mcp-Session-Id"
const mcpLastEventIDHeader = "Last-Event-ID"

// mcpSessionTTL bounds how long an idle MCP session is retained in memory.
// Sessions are evicted lazily on the next store operation after they go idle,
// so abandoned sessions cannot accumulate without bound.
const mcpSessionTTL = 30 * time.Minute

// Bounds on concurrent MCP sessions to keep an authenticated agent (or a stolen
// credential) from exhausting memory by bursting initialize calls. Sessions are
// also evicted after mcpSessionTTL, so these caps only bound concurrency.
const maxMCPSessionsPerSubject = 16
const maxMCPSessionsTotal = 1024

type mcpHTTPSessionStore struct {
	mu          sync.Mutex
	sessions    map[string]*mcpHTTPSession
	ttl         time.Duration
	now         func() time.Time
	persistence MCPSessionPersistence
}

type MCPSessionPersistence interface {
	CreateMCPSession(context.Context, string, string, time.Time) error
	TouchMCPSession(context.Context, string, string, time.Time, time.Time) (bool, error)
	CloseMCPSession(context.Context, string, time.Time) (bool, error)
	ActiveMCPSessionCount(context.Context, time.Time) (int, error)
	ActiveMCPSessionCountForSubject(context.Context, string, time.Time) (int, error)
	AppendMCPEvent(context.Context, string, []byte, time.Time) (string, []byte, error)
	ListMCPEvents(context.Context, string, string, int) ([]string, [][]byte, error)
}

type mcpHTTPSession struct {
	id       string
	nextID   int64
	nextSub  int64
	events   []mcpHTTPEvent
	closed   bool
	subject  string
	subs     map[int64]chan mcpHTTPEvent
	lastSeen time.Time
}

type mcpHTTPEvent struct {
	id      string
	payload []byte
}

func newMCPHTTPSessionStore() *mcpHTTPSessionStore {
	return &mcpHTTPSessionStore{sessions: make(map[string]*mcpHTTPSession), ttl: mcpSessionTTL, now: time.Now}
}

func newPersistedMCPHTTPSessionStore(persistence MCPSessionPersistence) *mcpHTTPSessionStore {
	return &mcpHTTPSessionStore{sessions: make(map[string]*mcpHTTPSession), ttl: mcpSessionTTL, now: time.Now, persistence: persistence}
}

func NewPersistedMCPHTTPSessionStore(persistence MCPSessionPersistence) *mcpHTTPSessionStore {
	if persistence == nil {
		panic("MCP session persistence is required")
	}
	return newPersistedMCPHTTPSessionStore(persistence)
}

// evictExpiredLocked removes sessions idle for longer than the TTL. Callers must
// hold the store mutex.
func (store *mcpHTTPSessionStore) evictExpiredLocked() {
	cutoff := store.now().Add(-store.ttl)
	for id, session := range store.sessions {
		if session.lastSeen.Before(cutoff) {
			for _, subscriber := range session.subs {
				close(subscriber)
			}
			delete(store.sessions, id)
		}
	}
}

// create registers a new session and reports whether it was admitted. It is
// refused when the global ceiling or the per-subject cap is reached.
func (store *mcpHTTPSessionStore) create(id string, subject string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.evictExpiredLocked()
	now := store.now()
	if store.persistence != nil {
		cutoff := now.Add(-store.ttl)
		total, err := store.persistence.ActiveMCPSessionCount(context.Background(), cutoff)
		if err != nil {
			slog.Error("count active MCP sessions failed, refusing session creation", "error", err)
			return false
		}
		if total >= maxMCPSessionsTotal {
			return false
		}
		perSubject, err := store.persistence.ActiveMCPSessionCountForSubject(context.Background(), subject, cutoff)
		if err != nil {
			slog.Error("count subject MCP sessions failed, refusing session creation", "error", err)
			return false
		}
		if perSubject >= maxMCPSessionsPerSubject {
			return false
		}
		if err := store.persistence.CreateMCPSession(context.Background(), id, subject, now); err != nil {
			slog.Error("create MCP session failed", "error", err)
			return false
		}
	} else {
		if len(store.sessions) >= maxMCPSessionsTotal {
			return false
		}
		perSubject := 0
		for _, session := range store.sessions {
			if session.subject == subject {
				perSubject++
			}
		}
		if perSubject >= maxMCPSessionsPerSubject {
			return false
		}
	}
	store.sessions[id] = &mcpHTTPSession{id: id, subject: subject, events: make([]mcpHTTPEvent, 0), subs: make(map[int64]chan mcpHTTPEvent), lastSeen: now}
	return true
}

func (store *mcpHTTPSessionStore) existsForSubject(id string, subject string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.evictExpiredLocked()
	session, found := store.sessions[id]
	if !found || session.closed || session.subject != subject {
		if store.persistence == nil {
			return false
		}
		now := store.now()
		exists, err := store.persistence.TouchMCPSession(context.Background(), id, subject, now, now.Add(-store.ttl))
		if err != nil {
			slog.Error("touch MCP session failed, treating session as invalid", "error", err)
			return false
		}
		if !exists {
			return false
		}
		store.sessions[id] = &mcpHTTPSession{id: id, subject: subject, events: make([]mcpHTTPEvent, 0), subs: make(map[int64]chan mcpHTTPEvent), lastSeen: now}
		return true
	}
	session.lastSeen = store.now()
	if store.persistence != nil {
		if _, err := store.persistence.TouchMCPSession(context.Background(), id, subject, session.lastSeen, session.lastSeen.Add(-store.ttl)); err != nil {
			slog.Error("touch MCP session failed, treating session as invalid", "error", err)
			return false
		}
	}
	return true
}

func (store *mcpHTTPSessionStore) terminate(id string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	session, found := store.sessions[id]
	if !found || session.closed {
		if store.persistence == nil {
			return false
		}
		closed, err := store.persistence.CloseMCPSession(context.Background(), id, store.now())
		if err != nil {
			slog.Error("close MCP session failed", "error", err)
			return false
		}
		return closed
	}
	session.closed = true
	for _, subscriber := range session.subs {
		close(subscriber)
	}
	delete(store.sessions, id)
	if store.persistence != nil {
		if _, err := store.persistence.CloseMCPSession(context.Background(), id, store.now()); err != nil {
			slog.Error("close MCP session failed", "error", err)
			return false
		}
	}
	return true
}

func (store *mcpHTTPSessionStore) appendEvent(sessionID string, payload []byte) (string, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	session, found := store.sessions[sessionID]
	if !found || session.closed {
		return "", false
	}
	session.lastSeen = store.now()
	if store.persistence != nil {
		if _, err := store.persistence.TouchMCPSession(context.Background(), sessionID, session.subject, session.lastSeen, session.lastSeen.Add(-store.ttl)); err != nil {
			slog.Error("touch MCP session failed", "error", err)
			return "", false
		}
	}
	copied := make([]byte, len(payload))
	copy(copied, payload)
	var event mcpHTTPEvent
	if store.persistence != nil {
		eventID, eventPayload, err := store.persistence.AppendMCPEvent(context.Background(), sessionID, copied, session.lastSeen)
		if err != nil {
			slog.Error("append MCP event failed", "error", err)
			return "", false
		}
		event = mcpHTTPEvent{id: eventID, payload: eventPayload}
	} else {
		session.nextID++
		eventID := session.id + "-" + strconv.FormatInt(session.nextID, 10)
		event = mcpHTTPEvent{id: eventID, payload: copied}
	}
	session.events = append(session.events, event)
	if len(session.events) > 100 {
		session.events = session.events[len(session.events)-100:]
	}
	for _, subscriber := range session.subs {
		select {
		case subscriber <- event:
		default:
		}
	}
	return event.id, true
}

func (store *mcpHTTPSessionStore) replayAndSubscribe(sessionID string, lastEventID string) ([]mcpHTTPEvent, <-chan mcpHTTPEvent, func(), bool) {
	store.mu.Lock()
	session, found := store.sessions[sessionID]
	if !found || session.closed {
		store.mu.Unlock()
		return nil, nil, func() {}, false
	}
	session.lastSeen = store.now()
	var events []mcpHTTPEvent
	if store.persistence != nil {
		eventIDs, payloads, err := store.persistence.ListMCPEvents(context.Background(), sessionID, lastEventID, 100)
		if err != nil {
			slog.Error("list MCP events failed", "error", err)
			store.mu.Unlock()
			return nil, nil, func() {}, false
		}
		events = make([]mcpHTTPEvent, 0, len(eventIDs))
		for index := range eventIDs {
			events = append(events, mcpHTTPEvent{id: eventIDs[index], payload: payloads[index]})
		}
		nextLastEventID := lastEventID
		if len(events) > 0 {
			nextLastEventID = events[len(events)-1].id
		}
		subscriber, cancel := store.pollPersistedEvents(sessionID, nextLastEventID)
		store.mu.Unlock()
		return events, subscriber, cancel, true
	} else {
		start := 0
		if lastEventID != "" {
			start = len(session.events)
			for index := range session.events {
				if session.events[index].id == lastEventID {
					start = index + 1
					break
				}
			}
		}
		events = make([]mcpHTTPEvent, len(session.events[start:]))
		copy(events, session.events[start:])
	}
	session.nextSub++
	subID := session.nextSub
	subscriber := make(chan mcpHTTPEvent, 16)
	session.subs[subID] = subscriber
	store.mu.Unlock()

	cancel := func() {
		store.mu.Lock()
		defer store.mu.Unlock()
		current, stillFound := store.sessions[sessionID]
		if !stillFound {
			return
		}
		if ch, subFound := current.subs[subID]; subFound {
			delete(current.subs, subID)
			close(ch)
		}
	}
	return events, subscriber, cancel, true
}

func (store *mcpHTTPSessionStore) pollPersistedEvents(sessionID string, lastEventID string) (<-chan mcpHTTPEvent, func()) {
	subscriber := make(chan mcpHTTPEvent, 16)
	done := make(chan struct{})
	cancelOnce := sync.Once{}
	go func() {
		defer close(subscriber)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		currentLastEventID := lastEventID
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				eventIDs, payloads, err := store.persistence.ListMCPEvents(context.Background(), sessionID, currentLastEventID, 100)
				if err != nil {
					// A transient DB error here must not crash the process:
					// this runs in a background goroutine per live SSE
					// subscriber, so with many concurrent streams a single
					// blip would otherwise take down every session at once.
					// Skip this tick and retry on the next one.
					slog.Error("poll MCP events failed, will retry next tick", "session_id", sessionID, "error", err)
					continue
				}
				for index := range eventIDs {
					event := mcpHTTPEvent{id: eventIDs[index], payload: payloads[index]}
					currentLastEventID = event.id
					select {
					case subscriber <- event:
					case <-done:
						return
					default:
					}
				}
			}
		}
	}()
	cancel := func() {
		cancelOnce.Do(func() {
			close(done)
		})
	}
	return subscriber, cancel
}

func (store *mcpHTTPSessionStore) activeSessionCount() int {
	if store.persistence != nil {
		count, err := store.persistence.ActiveMCPSessionCount(context.Background(), store.now().Add(-store.ttl))
		if err != nil {
			slog.Error("count active MCP sessions failed", "error", err)
			return 0
		}
		return count
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.evictExpiredLocked()
	return len(store.sessions)
}

func (store *mcpHTTPSessionStore) storageKind() string {
	if store.persistence != nil {
		return "postgres_session_replay_polling_stream"
	}
	return "process_memory"
}

func newMCPHTTPSessionID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return ""
	}
	return hex.EncodeToString(value)
}

func writeSSEEvent(w http.ResponseWriter, event mcpHTTPEvent) {
	_, _ = fmt.Fprintf(w, "id: %s\n", event.id)
	_, _ = fmt.Fprint(w, "event: message\n")
	writeSSEData(w, string(event.payload))
	_, _ = fmt.Fprint(w, "\n")
}

func writeSSEData(w http.ResponseWriter, payload string) {
	lines := strings.Split(payload, "\n")
	for _, line := range lines {
		_, _ = fmt.Fprintf(w, "data: %s\n", line)
	}
}
