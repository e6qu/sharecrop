package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
			panic(fmt.Sprintf("count active MCP sessions failed: %v", err))
		}
		if total >= maxMCPSessionsTotal {
			return false
		}
		perSubject, err := store.persistence.ActiveMCPSessionCountForSubject(context.Background(), subject, cutoff)
		if err != nil {
			panic(fmt.Sprintf("count subject MCP sessions failed: %v", err))
		}
		if perSubject >= maxMCPSessionsPerSubject {
			return false
		}
		if err := store.persistence.CreateMCPSession(context.Background(), id, subject, now); err != nil {
			panic(fmt.Sprintf("create MCP session failed: %v", err))
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
			panic(fmt.Sprintf("touch MCP session failed: %v", err))
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
			panic(fmt.Sprintf("touch MCP session failed: %v", err))
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
			panic(fmt.Sprintf("close MCP session failed: %v", err))
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
			panic(fmt.Sprintf("close MCP session failed: %v", err))
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
			panic(fmt.Sprintf("touch MCP session failed: %v", err))
		}
	}
	session.nextID++
	eventID := session.id + "-" + strconv.FormatInt(session.nextID, 10)
	copied := make([]byte, len(payload))
	copy(copied, payload)
	event := mcpHTTPEvent{id: eventID, payload: copied}
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
	return eventID, true
}

func (store *mcpHTTPSessionStore) replayAndSubscribe(sessionID string, lastEventID string) ([]mcpHTTPEvent, <-chan mcpHTTPEvent, func(), bool) {
	store.mu.Lock()
	session, found := store.sessions[sessionID]
	if !found || session.closed {
		store.mu.Unlock()
		return nil, nil, func() {}, false
	}
	session.lastSeen = store.now()
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
	events := make([]mcpHTTPEvent, len(session.events[start:]))
	copy(events, session.events[start:])
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

func (store *mcpHTTPSessionStore) activeSessionCount() int {
	if store.persistence != nil {
		count, err := store.persistence.ActiveMCPSessionCount(context.Background(), store.now().Add(-store.ttl))
		if err != nil {
			panic(fmt.Sprintf("count active MCP sessions failed: %v", err))
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
		return "postgres_session_process_stream"
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
