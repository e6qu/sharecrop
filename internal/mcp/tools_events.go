package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const toolListEvents = "sharecrop.list_events"

func eventsToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolListEvents,
			Description: "List the credential's domain-event feed as cursor-paged rows, oldest first: a personal agent credential reads its owner's feed, an organization credential reads the organization's events. Pass the previous result's next_cursor back as after to read only newer events; next_cursor is empty when the page is empty. The tool answers immediately (no long-poll), so poll it - or use webhooks - to follow review outcomes, reservations, and payouts.",
			Scope:       agent.ScopeNotificationsRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"after":{"type":"string"},"limit":{"type":"integer","minimum":1}}}`),
		},
	}
}
