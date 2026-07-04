package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolListNotifications    = "sharecrop.list_notifications"
	toolMarkNotificationRead = "sharecrop.mark_notification_read"
)

func notificationsToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolListNotifications,
			Description: "List the agent's user's notifications.",
			Scope:       agent.ScopeNotificationsRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolMarkNotificationRead,
			Description: "Mark a notification as read.",
			Scope:       agent.ScopeNotificationsManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"notification_id":{"type":"string"}},"required":["notification_id"]}`),
		},
	}
}
