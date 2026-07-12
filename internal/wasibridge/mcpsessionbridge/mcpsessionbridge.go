// Package mcpsessionbridge is the WASI bridge for internal/http's
// MCPSessionPersistence (the store behind runtime.MCPSessions). Like the rate
// limiter it is hand-written, not generated: its methods return multi-value
// tuples ((bool, error), (string, []byte, error), ...), which the code generator
// (built for a single union result) does not model. MCP Streamable HTTP sessions
// and their replay events must be shared across every request, so a pooled guest
// has to reach one Postgres store on the host - a per-instance in-memory copy
// would strand sessions on whichever instance created them. internal/http is
// package httpserver.
package mcpsessionbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	httpserver "github.com/e6qu/sharecrop/internal/http"
)

// Invoker sends a call to the host and returns the serialized result (rpc.Invoke
// on the guest; a test stand-in otherwise).
type Invoker func(method string, args []byte) ([]byte, error)

// GuestMCPSessionPersistence implements httpserver.MCPSessionPersistence by
// forwarding each call to the host's shared Postgres-backed store.
type GuestMCPSessionPersistence struct {
	invoke Invoker
}

// NewGuestMCPSessionPersistence builds the guest-side persistence over an invoker.
func NewGuestMCPSessionPersistence(invoke Invoker) GuestMCPSessionPersistence {
	return GuestMCPSessionPersistence{invoke: invoke}
}

// ---- wire structs ----

type createArgs struct {
	ID      string    `json:"id"`
	Subject string    `json:"subject"`
	Now     time.Time `json:"now"`
}

type touchArgs struct {
	ID      string    `json:"id"`
	Subject string    `json:"subject"`
	Now     time.Time `json:"now"`
	Cutoff  time.Time `json:"cutoff"`
}

type closeArgs struct {
	ID  string    `json:"id"`
	Now time.Time `json:"now"`
}

type countArgs struct {
	Cutoff time.Time `json:"cutoff"`
}

type countForSubjectArgs struct {
	Subject string    `json:"subject"`
	Cutoff  time.Time `json:"cutoff"`
}

type appendArgs struct {
	SessionID string    `json:"session_id"`
	Payload   []byte    `json:"payload"`
	Now       time.Time `json:"now"`
}

type listArgs struct {
	SessionID   string `json:"session_id"`
	LastEventID string `json:"last_event_id"`
	Limit       int    `json:"limit"`
}

type errorResult struct {
	Error string `json:"error,omitempty"`
}

type boolResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type countResult struct {
	Count int    `json:"count"`
	Error string `json:"error,omitempty"`
}

type appendResult struct {
	EventID string `json:"event_id"`
	Payload []byte `json:"payload"`
	Error   string `json:"error,omitempty"`
}

type listResult struct {
	EventIDs []string `json:"event_ids"`
	Payloads [][]byte `json:"payloads"`
	Error    string   `json:"error,omitempty"`
}

func resultError(message string) error {
	if message == "" {
		return nil
	}
	return errors.New(message)
}

// ---- guest client ----

func (g GuestMCPSessionPersistence) CreateMCPSession(_ context.Context, id string, subject string, now time.Time) error {
	encoded, err := json.Marshal(createArgs{ID: id, Subject: subject, Now: now})
	if err != nil {
		return err
	}
	raw, err := g.invoke("mcpsession.CreateMCPSession", encoded)
	if err != nil {
		return err
	}
	var result errorResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return err
	}
	return resultError(result.Error)
}

func (g GuestMCPSessionPersistence) TouchMCPSession(_ context.Context, id string, subject string, now time.Time, cutoff time.Time) (bool, error) {
	encoded, err := json.Marshal(touchArgs{ID: id, Subject: subject, Now: now, Cutoff: cutoff})
	if err != nil {
		return false, err
	}
	raw, err := g.invoke("mcpsession.TouchMCPSession", encoded)
	if err != nil {
		return false, err
	}
	var result boolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return false, err
	}
	return result.OK, resultError(result.Error)
}

func (g GuestMCPSessionPersistence) CloseMCPSession(_ context.Context, id string, now time.Time) (bool, error) {
	encoded, err := json.Marshal(closeArgs{ID: id, Now: now})
	if err != nil {
		return false, err
	}
	raw, err := g.invoke("mcpsession.CloseMCPSession", encoded)
	if err != nil {
		return false, err
	}
	var result boolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return false, err
	}
	return result.OK, resultError(result.Error)
}

func (g GuestMCPSessionPersistence) ActiveMCPSessionCount(_ context.Context, cutoff time.Time) (int, error) {
	encoded, err := json.Marshal(countArgs{Cutoff: cutoff})
	if err != nil {
		return 0, err
	}
	raw, err := g.invoke("mcpsession.ActiveMCPSessionCount", encoded)
	if err != nil {
		return 0, err
	}
	var result countResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return 0, err
	}
	return result.Count, resultError(result.Error)
}

func (g GuestMCPSessionPersistence) ActiveMCPSessionCountForSubject(_ context.Context, subject string, cutoff time.Time) (int, error) {
	encoded, err := json.Marshal(countForSubjectArgs{Subject: subject, Cutoff: cutoff})
	if err != nil {
		return 0, err
	}
	raw, err := g.invoke("mcpsession.ActiveMCPSessionCountForSubject", encoded)
	if err != nil {
		return 0, err
	}
	var result countResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return 0, err
	}
	return result.Count, resultError(result.Error)
}

func (g GuestMCPSessionPersistence) AppendMCPEvent(_ context.Context, sessionID string, payload []byte, now time.Time) (string, []byte, error) {
	encoded, err := json.Marshal(appendArgs{SessionID: sessionID, Payload: payload, Now: now})
	if err != nil {
		return "", nil, err
	}
	raw, err := g.invoke("mcpsession.AppendMCPEvent", encoded)
	if err != nil {
		return "", nil, err
	}
	var result appendResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", nil, err
	}
	return result.EventID, result.Payload, resultError(result.Error)
}

func (g GuestMCPSessionPersistence) ListMCPEvents(_ context.Context, sessionID string, lastEventID string, limit int) ([]string, [][]byte, error) {
	encoded, err := json.Marshal(listArgs{SessionID: sessionID, LastEventID: lastEventID, Limit: limit})
	if err != nil {
		return nil, nil, err
	}
	raw, err := g.invoke("mcpsession.ListMCPEvents", encoded)
	if err != nil {
		return nil, nil, err
	}
	var result listResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, nil, err
	}
	return result.EventIDs, result.Payloads, resultError(result.Error)
}

var _ httpserver.MCPSessionPersistence = GuestMCPSessionPersistence{}

// ---- host dispatcher ----

// Dispatch services one MCP-session call against the real store on the host.
// method is "mcpsession.<op>".
func Dispatch(ctx context.Context, store httpserver.MCPSessionPersistence, method string, args []byte) ([]byte, error) {
	op, found := trimPrefix(method, "mcpsession.")
	if !found {
		return nil, fmt.Errorf("mcpsession bridge: unknown method %q", method)
	}
	switch op {
	case "CreateMCPSession":
		var in createArgs
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, err
		}
		return json.Marshal(errorResult{Error: errorString(store.CreateMCPSession(ctx, in.ID, in.Subject, in.Now))})
	case "TouchMCPSession":
		var in touchArgs
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, err
		}
		ok, err := store.TouchMCPSession(ctx, in.ID, in.Subject, in.Now, in.Cutoff)
		return json.Marshal(boolResult{OK: ok, Error: errorString(err)})
	case "CloseMCPSession":
		var in closeArgs
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, err
		}
		ok, err := store.CloseMCPSession(ctx, in.ID, in.Now)
		return json.Marshal(boolResult{OK: ok, Error: errorString(err)})
	case "ActiveMCPSessionCount":
		var in countArgs
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, err
		}
		count, err := store.ActiveMCPSessionCount(ctx, in.Cutoff)
		return json.Marshal(countResult{Count: count, Error: errorString(err)})
	case "ActiveMCPSessionCountForSubject":
		var in countForSubjectArgs
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, err
		}
		count, err := store.ActiveMCPSessionCountForSubject(ctx, in.Subject, in.Cutoff)
		return json.Marshal(countResult{Count: count, Error: errorString(err)})
	case "AppendMCPEvent":
		var in appendArgs
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, err
		}
		eventID, payload, err := store.AppendMCPEvent(ctx, in.SessionID, in.Payload, in.Now)
		return json.Marshal(appendResult{EventID: eventID, Payload: payload, Error: errorString(err)})
	case "ListMCPEvents":
		var in listArgs
		if err := json.Unmarshal(args, &in); err != nil {
			return nil, err
		}
		eventIDs, payloads, err := store.ListMCPEvents(ctx, in.SessionID, in.LastEventID, in.Limit)
		return json.Marshal(listResult{EventIDs: eventIDs, Payloads: payloads, Error: errorString(err)})
	default:
		return nil, fmt.Errorf("mcpsession bridge: unknown op %q", op)
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func trimPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return "", false
}
