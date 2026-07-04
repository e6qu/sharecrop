package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolListUsers          = "sharecrop.list_users"
	toolGetUserProfile     = "sharecrop.get_user_profile"
	toolGetUserWork        = "sharecrop.get_user_work"
	toolGetUserSubmissions = "sharecrop.get_user_submissions"
)

func usersToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolListUsers,
			Description: "List the user directory. query optionally filters by email.",
			Scope:       agent.ScopeUsersRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
		},
		{
			Name:        toolGetUserProfile,
			Description: "Get a user's public profile: the tasks they created.",
			Scope:       agent.ScopeUsersRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"string"}},"required":["user_id"]}`),
		},
		{
			Name:        toolGetUserWork,
			Description: "List tasks a user is currently assigned to or has reserved.",
			Scope:       agent.ScopeUsersRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"string"}},"required":["user_id"]}`),
		},
		{
			Name:        toolGetUserSubmissions,
			Description: "List a user's own submissions. Only the user themselves may read their submissions.",
			Scope:       agent.ScopeUsersRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"string"}},"required":["user_id"]}`),
		},
	}
}
