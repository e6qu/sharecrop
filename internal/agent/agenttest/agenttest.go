// Package agenttest holds test-support helpers for agent.Credential and the
// agent value types (Label, ScopeSet, State) that the orgcred credential shares.
// It is used by the agent and orgcred bridges' codec tests and their
// integration dual-run tests so those do not carry duplicate comparisons.
package agenttest

import (
	"fmt"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
)

// SharedFieldsDiff compares the label, scope set, state, and expiry that the
// agent and orgcred credentials have in common - agent's value types. It
// returns a description of the first difference, or "" if equal. Both
// credentials' own comparators call this for their shared fields.
func SharedFieldsDiff(gotLabel, wantLabel agent.Label, gotScopes, wantScopes agent.ScopeSet, gotState, wantState agent.State, gotExpires, wantExpires *time.Time) string {
	if gotLabel.String() != wantLabel.String() {
		return fmt.Sprintf("label: %s != %s", gotLabel, wantLabel)
	}
	if gotState.String() != wantState.String() {
		return fmt.Sprintf("state: %s != %s", gotState, wantState)
	}

	gotValues, wantValues := gotScopes.Values(), wantScopes.Values()
	if len(gotValues) != len(wantValues) {
		return fmt.Sprintf("scope count: %d != %d", len(gotValues), len(wantValues))
	}
	for index := range wantValues {
		if gotValues[index].String() != wantValues[index].String() {
			return fmt.Sprintf("scope %d: %s != %s", index, gotValues[index], wantValues[index])
		}
	}

	if (gotExpires == nil) != (wantExpires == nil) {
		return fmt.Sprintf("expires_at presence: got %v want %v", gotExpires, wantExpires)
	}
	if gotExpires != nil && !gotExpires.Equal(*wantExpires) {
		return fmt.Sprintf("expires_at: %s != %s", gotExpires, wantExpires)
	}
	return ""
}

// CredentialDiff returns a description of the first field in which got and want
// differ, or "" if they are equal.
func CredentialDiff(got, want agent.Credential) string {
	if got.ID != want.ID {
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	}
	if got.UserID != want.UserID {
		return fmt.Sprintf("user id: %s != %s", got.UserID, want.UserID)
	}
	if diff := SharedFieldsDiff(got.Label, want.Label, got.Scopes, want.Scopes, got.State, want.State, got.ExpiresAt, want.ExpiresAt); diff != "" {
		return diff
	}
	if (got.TaskID == nil) != (want.TaskID == nil) {
		return "task_id presence differs"
	}
	if got.TaskID != nil && *got.TaskID != *want.TaskID {
		return fmt.Sprintf("task_id: %s != %s", got.TaskID, want.TaskID)
	}
	return ""
}
