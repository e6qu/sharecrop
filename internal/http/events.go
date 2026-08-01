package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/webhook"
)

// encodeEventPayload renders one feed event as the single-line JSON document
// an SSE `data:` field carries.
func encodeEventPayload(payload webhook.EventPayload) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// eventListResponse is the live feed page: the caller's visible events after
// the requested cursor, in the shared feed/webhook wire shape
// (webhook.EventPayload), plus the cursor to resume from. next_cursor is the
// last event's cursor, or the empty string when the page is empty.
type eventListResponse struct {
	Events     []webhook.EventPayload `json:"events"`
	NextCursor string                 `json:"next_cursor"`
}

func (eventListResponse) writableResponse() {}

// Feed listings take only `after` (a cursor) and `limit`; they page by
// cursor, never by offset, so the offset query parameter is not accepted.
const (
	eventCursorQueryParameter = "after"
	eventLimitQueryParameter  = "limit"
)

// SSE stream pacing: poll the event store for new rows every pollInterval,
// and end each connection cleanly before API Gateway's 30-second integration
// cap can kill it mid-write; the client reconnects with Last-Event-ID.
const (
	eventStreamPollInterval = 5 * time.Second
	eventStreamMaxDuration  = 25 * time.Second
)

// Long-poll pacing for GET /api/events?wait=N: the handler re-checks the
// store every tick until an event arrives or the wait elapses. The wait is
// capped below the API Gateway edge's 30-second request cap so the response
// always leaves cleanly.
const (
	eventFeedWaitQueryParameter = "wait"
	eventFeedLongPollTick       = 500 * time.Millisecond
	eventFeedMaxWait            = 25 * time.Second
)

type eventFeedQueryResult interface {
	eventFeedQueryResult()
}

type eventFeedQueryAccepted struct {
	filter event.CursorFilter
	page   core.Page
}

type eventFeedQueryRejected struct {
	reason string
}

func (eventFeedQueryAccepted) eventFeedQueryResult() {}

func (eventFeedQueryRejected) eventFeedQueryResult() {}

// parseEventFeedQuery reads the feed parameters strictly: an absent `after`
// means the start of the stream, an absent `limit` means the default page
// size, and malformed values are rejected rather than coerced.
func parseEventFeedQuery(r *http.Request) eventFeedQueryResult {
	query := r.URL.Query()

	filter := event.CursorFilter(event.FromStart{})
	if rawAfter := query.Get(eventCursorQueryParameter); rawAfter != "" {
		parsed, matched := event.ParseCursor(rawAfter).(event.CursorParsed)
		if !matched {
			return eventFeedQueryRejected{reason: "after query parameter is invalid"}
		}
		filter = event.After{Cursor: parsed.Value}
	}

	limit := core.DefaultPage().Limit()
	if rawLimit := query.Get(eventLimitQueryParameter); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return eventFeedQueryRejected{reason: "limit query parameter is invalid"}
		}
		limit = parsed
	}
	pageResult := core.NewPage(limit, 0)
	accepted, matched := pageResult.(core.PageAccepted)
	if !matched {
		return eventFeedQueryRejected{reason: pageResult.(core.PageRejected).Reason.Description()}
	}
	return eventFeedQueryAccepted{filter: filter, page: accepted.Value}
}

// eventFeedRecipient is the resolved owner of the feed being read: the events
// endpoint always serves a recipient's own event stream, never someone
// else's. Each variant knows which store read serves it.
type eventFeedRecipient interface {
	listEvents(ctx context.Context, store event.Store, filter event.CursorFilter, page core.Page) event.ListStoreResult
}

type userFeedRecipient struct {
	id core.UserID
}

func (recipient userFeedRecipient) listEvents(ctx context.Context, store event.Store, filter event.CursorFilter, page core.Page) event.ListStoreResult {
	return store.ListForRecipient(ctx, recipient.id, filter, page)
}

type organizationFeedRecipient struct {
	id core.OrganizationID
}

func (recipient organizationFeedRecipient) listEvents(ctx context.Context, store event.Store, filter event.CursorFilter, page core.Page) event.ListStoreResult {
	return store.ListForOrganization(ctx, recipient.id, filter, page)
}

// eventFeedActorResult resolves who is polling the event feed: a user
// session, an org credential holding notifications_read (the feed is the
// organization's own event stream), or a personal agent credential holding
// notifications_read (the feed is the owning user's event stream). The scope
// is notifications_read because the feed carries the same recipient-scoped
// facts the notification inbox is built from; a credential never sees more
// than its owner would.
type eventFeedActorResult interface {
	eventFeedActorResult()
}

type eventFeedActorAccepted struct {
	recipient eventFeedRecipient
}

// eventFeedActorRejection wraps the shared user/org rejection so listEvents
// can reuse writeActorRejection's 401/403 split.
type eventFeedActorRejection struct {
	value actorResult
}

func (eventFeedActorAccepted) eventFeedActorResult() {}

func (eventFeedActorRejection) eventFeedActorResult() {}

func (server Server) resolveEventFeedActor(r *http.Request) eventFeedActorResult {
	sharedResult := server.requireUserOrOrgSubject(r, agent.ScopeNotificationsRead)
	if accepted, matched := sharedResult.(actorAccepted); matched {
		switch subject := accepted.actor.(type) {
		case auth.UserSubject:
			return eventFeedActorAccepted{recipient: userFeedRecipient{id: subject.ID}}
		case auth.OrgSubject:
			return eventFeedActorAccepted{recipient: organizationFeedRecipient{id: subject.ID}}
		}
		return eventFeedActorRejection{value: actorRejected{reason: "a user access token or an organization credential is required"}}
	}
	rawToken, hasBearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !hasBearer || !agent.HasSecretPrefix(rawToken) {
		return eventFeedActorRejection{value: sharedResult}
	}
	verifyResult := server.verifyAgent(r)
	verified, verifiedMatched := verifyResult.(agent.CredentialVerified)
	if !verifiedMatched {
		return eventFeedActorRejection{value: actorRejected{reason: verifyResult.(agent.VerifyRejected).Reason.Description()}}
	}
	if _, granted := verified.Credential.Scopes.Allows(agent.ScopeNotificationsRead).(agent.ScopeGranted); !granted {
		return eventFeedActorRejection{value: actorScopeDenied{reason: "the agent credential is missing the " + agent.ScopeNotificationsRead.String() + " scope"}}
	}
	return eventFeedActorAccepted{recipient: userFeedRecipient{id: verified.Subject.ID}}
}

type eventFeedWaitResult interface {
	eventFeedWaitResult()
}

type eventFeedWaitAccepted struct {
	value time.Duration
}

type eventFeedWaitRejected struct {
	reason string
}

func (eventFeedWaitAccepted) eventFeedWaitResult() {}

func (eventFeedWaitRejected) eventFeedWaitResult() {}

// parseEventFeedWait reads the optional long-poll wait: whole seconds, absent
// or 0 means respond immediately, negative or malformed is rejected, and
// anything above the cap is clamped to it (the edge would cut a longer hold).
func parseEventFeedWait(r *http.Request) eventFeedWaitResult {
	rawWait := r.URL.Query().Get(eventFeedWaitQueryParameter)
	if rawWait == "" {
		return eventFeedWaitAccepted{value: 0}
	}
	seconds, err := strconv.Atoi(rawWait)
	if err != nil || seconds < 0 {
		return eventFeedWaitRejected{reason: "wait query parameter is invalid"}
	}
	wait := time.Duration(seconds) * time.Second
	if wait > eventFeedMaxWait {
		wait = eventFeedMaxWait
	}
	return eventFeedWaitAccepted{value: wait}
}

func (server Server) listEvents(w http.ResponseWriter, r *http.Request) {
	actorResult := server.resolveEventFeedActor(r)
	actor, matched := actorResult.(eventFeedActorAccepted)
	if !matched {
		writeActorRejection(w, actorResult.(eventFeedActorRejection).value)
		return
	}

	queryResult := parseEventFeedQuery(r)
	query, queryMatched := queryResult.(eventFeedQueryAccepted)
	if !queryMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, queryResult.(eventFeedQueryRejected).reason)
		return
	}
	waitResult := parseEventFeedWait(r)
	wait, waitMatched := waitResult.(eventFeedWaitAccepted)
	if !waitMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, waitResult.(eventFeedWaitRejected).reason)
		return
	}

	// The WASI guest bridge buffers whole responses and cannot hold a request
	// cheaply (a held request pins a pooled guest worker), so long-poll
	// degrades to an immediate response there — the same transport probe the
	// SSE stream uses before entering its live loop.
	_, transportCanHold := w.(http.Flusher)

	deadline := time.Now().Add(wait.value)
	for {
		result := actor.recipient.listEvents(r.Context(), server.eventStore, query.filter, query.page)
		listed, listedMatched := result.(event.ListStoreAccepted)
		if !listedMatched {
			writeDomainError(w, result.(event.ListStoreRejected).Reason)
			return
		}
		if len(listed.Values) > 0 || wait.value == 0 || !transportCanHold || !time.Now().Before(deadline) {
			writeEventFeedPage(w, listed.Values)
			return
		}
		tick := time.NewTimer(eventFeedLongPollTick)
		select {
		case <-tick.C:
		case <-r.Context().Done():
			tick.Stop()
			return
		}
	}
}

// writeEventFeedPage renders one feed page in the shared feed/webhook wire
// shape with the cursor to resume from.
func writeEventFeedPage(w http.ResponseWriter, values []event.StoredEvent) {
	response := eventListResponse{Events: make([]webhook.EventPayload, 0, len(values))}
	for index := range values {
		response.Events = append(response.Events, webhook.EventPayloadFromStored(values[index]))
	}
	if len(values) > 0 {
		response.NextCursor = values[len(values)-1].Cursor.String()
	}
	writeJSON(w, http.StatusOK, response)
}

// streamEvents serves the live feed as Server-Sent Events. The resume cursor
// comes from the Last-Event-ID header when the client reconnects (taking
// precedence over ?after) or from ?after on a fresh connection.
func (server Server) streamEvents(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	queryResult := parseEventFeedQuery(r)
	query, queryMatched := queryResult.(eventFeedQueryAccepted)
	if !queryMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, queryResult.(eventFeedQueryRejected).reason)
		return
	}
	filter := query.filter
	if rawLastEventID := r.Header.Get("Last-Event-ID"); rawLastEventID != "" {
		parsed, parsedMatched := event.ParseCursor(rawLastEventID).(event.CursorParsed)
		if !parsedMatched {
			writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "Last-Event-ID header is invalid")
			return
		}
		filter = event.After{Cursor: parsed.Value}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	filter, wrote := server.writeEventBatch(w, r, actor.subject.ID, filter, query.page)
	if !wrote {
		return
	}
	flusher, canStream := w.(http.Flusher)
	if !canStream {
		// The transport cannot stream - it buffers the whole response and
		// returns it as one unit (the WASI guest bridge). Blocking here to wait
		// for live events would never return and would pin the worker forever,
		// so send the replayed events and stop. The client's EventSource
		// reconnects with Last-Event-ID and picks up later events, turning
		// server push into short polling over a transport that cannot do better.
		return
	}
	flusher.Flush()

	deadline := time.NewTimer(eventStreamMaxDuration)
	defer deadline.Stop()
	poll := time.NewTicker(eventStreamPollInterval)
	defer poll.Stop()
	for {
		select {
		case <-poll.C:
			filter, wrote = server.writeEventBatch(w, r, actor.subject.ID, filter, query.page)
			if !wrote {
				return
			}
			flusher.Flush()
		case <-deadline.C:
			// End the connection cleanly before API Gateway's 30-second cap
			// can cut it mid-write; the client reconnects with Last-Event-ID.
			return
		case <-r.Context().Done():
			return
		}
	}
}

// writeEventBatch writes every visible event after the filter as one SSE
// event each (`id:` carrying the cursor) and returns the advanced filter. It
// reports false when the store rejected the read, after emitting an SSE
// error event so the client does not mistake the close for a clean end.
func (server Server) writeEventBatch(w http.ResponseWriter, r *http.Request, recipient core.UserID, filter event.CursorFilter, page core.Page) (event.CursorFilter, bool) {
	result := server.eventStore.ListForRecipient(r.Context(), recipient, filter, page)
	listed, matched := result.(event.ListStoreAccepted)
	if !matched {
		_, _ = w.Write([]byte("event: error\ndata: " + result.(event.ListStoreRejected).Reason.Description() + "\n\n"))
		return filter, false
	}
	for index := range listed.Values {
		payload := webhook.EventPayloadFromStored(listed.Values[index])
		encoded, err := encodeEventPayload(payload)
		if err != nil {
			_, _ = w.Write([]byte("event: error\ndata: event could not be encoded\n\n"))
			return filter, false
		}
		_, _ = w.Write([]byte("id: " + listed.Values[index].Cursor.String() + "\ndata: " + encoded + "\n\n"))
		filter = event.After{Cursor: listed.Values[index].Cursor}
	}
	return filter, true
}
