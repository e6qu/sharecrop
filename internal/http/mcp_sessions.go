package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const mcpSessionHeader = "Mcp-Session-Id"
const mcpLastEventIDHeader = "Last-Event-ID"

type mcpHTTPSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*mcpHTTPSession
}

type mcpHTTPSession struct {
	id      string
	nextID  int64
	nextSub int64
	events  []mcpHTTPEvent
	closed  bool
	subject string
	subs    map[int64]chan mcpHTTPEvent
}

type mcpHTTPEvent struct {
	id      string
	payload []byte
}

func newMCPHTTPSessionStore() *mcpHTTPSessionStore {
	return &mcpHTTPSessionStore{sessions: make(map[string]*mcpHTTPSession)}
}

func (store *mcpHTTPSessionStore) create(id string, subject string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.sessions[id] = &mcpHTTPSession{id: id, subject: subject, events: make([]mcpHTTPEvent, 0), subs: make(map[int64]chan mcpHTTPEvent)}
}

func (store *mcpHTTPSessionStore) existsForSubject(id string, subject string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	session, found := store.sessions[id]
	return found && !session.closed && session.subject == subject
}

func (store *mcpHTTPSessionStore) terminate(id string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	session, found := store.sessions[id]
	if !found || session.closed {
		return false
	}
	session.closed = true
	for _, subscriber := range session.subs {
		close(subscriber)
	}
	delete(store.sessions, id)
	return true
}

func (store *mcpHTTPSessionStore) appendEvent(sessionID string, payload []byte) (string, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	session, found := store.sessions[sessionID]
	if !found || session.closed {
		return "", false
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
