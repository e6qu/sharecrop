// Package agenttest holds test-support helpers for agent.Credential, shared by
// the agent bridge's codec tests and the integration dual-run test so the two
// do not carry duplicate comparisons.
package agenttest

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/agent"
)

// CredentialDiff returns a description of the first field in which got and want
// differ, or "" if they are equal. Timestamps compare by instant; nullable
// fields compare presence then value.
func CredentialDiff(got, want agent.Credential) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.UserID != want.UserID:
		return fmt.Sprintf("user id: %s != %s", got.UserID, want.UserID)
	case got.Label.String() != want.Label.String():
		return fmt.Sprintf("label: %s != %s", got.Label, want.Label)
	case got.State.String() != want.State.String():
		return fmt.Sprintf("state: %s != %s", got.State, want.State)
	}

	gotScopes, wantScopes := got.Scopes.Values(), want.Scopes.Values()
	if len(gotScopes) != len(wantScopes) {
		return fmt.Sprintf("scope count: %d != %d", len(gotScopes), len(wantScopes))
	}
	for index := range wantScopes {
		if gotScopes[index].String() != wantScopes[index].String() {
			return fmt.Sprintf("scope %d: %s != %s", index, gotScopes[index], wantScopes[index])
		}
	}

	if (got.ExpiresAt == nil) != (want.ExpiresAt == nil) {
		return fmt.Sprintf("expires_at presence: got %v want %v", got.ExpiresAt, want.ExpiresAt)
	}
	if got.ExpiresAt != nil && !got.ExpiresAt.Equal(*want.ExpiresAt) {
		return fmt.Sprintf("expires_at: %s != %s", got.ExpiresAt, want.ExpiresAt)
	}
	if (got.TaskID == nil) != (want.TaskID == nil) {
		return "task_id presence differs"
	}
	if got.TaskID != nil && *got.TaskID != *want.TaskID {
		return fmt.Sprintf("task_id: %s != %s", got.TaskID, want.TaskID)
	}
	return ""
}
