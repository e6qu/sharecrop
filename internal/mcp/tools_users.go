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
			Access:      toolNeedsScope{Value: agent.ScopeUsersRead},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}}}`),
		},
		{
			Name:        toolGetUserProfile,
			Description: "Get a user's public profile: the tasks they created.",
			Access:      toolNeedsScope{Value: agent.ScopeUsersRead},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["user_id"]}`),
		},
		{
			Name:        toolGetUserWork,
			Description: "List tasks a user is currently assigned to or has reserved.",
			Access:      toolNeedsScope{Value: agent.ScopeUsersRead},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["user_id"]}`),
		},
		{
			Name:        toolGetUserSubmissions,
			Description: "List a user's own submissions. Only the user themselves may read their submissions.",
			Access:      toolNeedsScope{Value: agent.ScopeUsersRead},
			InputSchema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"string"},"limit":{"type":"integer","minimum":1},"offset":{"type":"integer","minimum":0}},"required":["user_id"]}`),
		},
	}
}
