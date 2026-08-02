package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolListNotifications          = "sharecrop.list_notifications"
	toolGetUnreadNotificationCount = "sharecrop.get_unread_notification_count"
	toolMarkNotificationRead       = "sharecrop.mark_notification_read"
)

func notificationsToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolListNotifications,
			Description: "List the agent's user's notifications. Pass state \"unread\" to list only unread notifications.",
			Access:      toolNeedsScope{Value: agent.ScopeNotificationsRead},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"state":{"type":"string","enum":["unread"]},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}}}`),
		},
		{
			Name:        toolGetUnreadNotificationCount,
			Description: "Report how many unread notifications the agent's user has.",
			Access:      toolNeedsScope{Value: agent.ScopeNotificationsRead},
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolMarkNotificationRead,
			Description: "Mark a notification as read.",
			Access:      toolNeedsScope{Value: agent.ScopeNotificationsManage},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"notification_id":{"type":"string"}},"required":["notification_id"]}`),
		},
	}
}
