package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/event"
)

// eventFeedItem is one feed row: the event's identity and kind, the acting
// user (empty for system actors), the read-time display enrichments, and the
// most specific subject reference flattened to a (kind, id) pair with the
// same precedence the notification inbox uses.
type eventFeedItem struct {
	ID               string `json:"id"`
	Kind             string `json:"kind"`
	ActorID          string `json:"actor_id"`
	ActorDisplayName string `json:"actor_display_name"`
	SubjectKind      string `json:"subject_kind"`
	SubjectID        string `json:"subject_id"`
	// TaskTitle is the referenced task's title when the store resolved it;
	// empty when the event references no task.
	TaskTitle  string `json:"task_title"`
	OccurredAt string `json:"occurred_at"`
}

// eventFeedPayload is one cursor page of the feed. NextCursor is the last
// row's cursor (pass it back as the after argument), or the empty string
// when the page is empty.
type eventFeedPayload struct {
	Events     []eventFeedItem `json:"events"`
	NextCursor string          `json:"next_cursor"`
}

func (eventFeedItem) payloadValue() {}

func (eventFeedPayload) payloadValue() {}

func storedEventToFeedItem(stored event.StoredEvent) eventFeedItem {
	item := eventFeedItem{
		ID:         stored.Event.ID.String(),
		Kind:       stored.Event.Kind.String(),
		OccurredAt: stored.Event.OccurredAt.UTC().Format(time.RFC3339Nano),
	}
	if actor, matched := stored.Event.Actor.(event.ActorUser); matched {
		item.ActorID = actor.ID.String()
	}
	if named, matched := stored.ActorName.(event.ActorNamed); matched {
		item.ActorDisplayName = named.DisplayName.String()
	}
	if titled, matched := stored.TaskTitle.(event.TaskTitled); matched {
		item.TaskTitle = titled.Title
	}
	subject := event.NotificationSubjectFor(stored.Event.Subject)
	item.SubjectKind = subject.Kind
	item.SubjectID = subject.ID
	return item
}

// callListEvents serves the credential's event feed over MCP through the
// same store path the REST feed (GET /api/events) uses. It pages by cursor,
// never by offset, and answers immediately: MCP tools are request/response,
// so the REST feed's long-poll wait parameter deliberately has no MCP
// counterpart.
func (server Server) callListEvents(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	var args struct {
		After string `json:"after"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	filter := event.CursorFilter(event.FromStart{})
	if args.After != "" {
		parsed, matched := event.ParseCursor(args.After).(event.CursorParsed)
		if !matched {
			return toolProtocolError{code: codeInvalidParams, message: "after must be a next_cursor value from a previous list_events result"}
		}
		filter = event.After{Cursor: parsed.Value}
	}
	page, pageProblem := parseMCPPage(args.Limit, 0)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListEvents(ctx, subject, filter, page)
	listed, matched := result.(event.ListStoreAccepted)
	if !matched {
		return toolFailed{code: result.(event.ListStoreRejected).Reason.Code(), message: result.(event.ListStoreRejected).Reason.Description()}
	}
	items := make([]eventFeedItem, 0, len(listed.Values))
	for index := range listed.Values {
		items = append(items, storedEventToFeedItem(listed.Values[index]))
	}
	payload := eventFeedPayload{Events: items}
	if len(listed.Values) > 0 {
		payload.NextCursor = listed.Values[len(listed.Values)-1].Cursor.String()
	}
	return marshalPayload(payload)
}
