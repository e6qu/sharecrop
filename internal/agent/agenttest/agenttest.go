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
	if diff := WorkPolicyDiff(got.WorkPolicy, want.WorkPolicy); diff != "" {
		return diff
	}
	return ""
}

// WorkPolicyDiff compares two work policies (a missing policy compares equal
// to the disabled default), returning the first difference or "".
func WorkPolicyDiff(got, want agent.WorkPolicy) string {
	if agent.WorkPolicyState(got) != agent.WorkPolicyState(want) {
		return fmt.Sprintf("work policy state: %s != %s", agent.WorkPolicyState(got), agent.WorkPolicyState(want))
	}
	gotEnabled, gotIsEnabled := got.(agent.WorkPolicyEnabled)
	wantEnabled, wantIsEnabled := want.(agent.WorkPolicyEnabled)
	if !gotIsEnabled || !wantIsEnabled {
		return ""
	}
	gotAllowances, wantAllowances := gotEnabled.Allowances, wantEnabled.Allowances
	if gotAllowances.MaxTasksPerDay.Int64() != wantAllowances.MaxTasksPerDay.Int64() {
		return fmt.Sprintf("max tasks per day: %d != %d", gotAllowances.MaxTasksPerDay.Int64(), wantAllowances.MaxTasksPerDay.Int64())
	}
	if fmt.Sprintf("%#v", gotAllowances.ConcurrentReservations) != fmt.Sprintf("%#v", wantAllowances.ConcurrentReservations) {
		return fmt.Sprintf("concurrent reservations: %#v != %#v", gotAllowances.ConcurrentReservations, wantAllowances.ConcurrentReservations)
	}
	if fmt.Sprintf("%#v", gotAllowances.DailySpend) != fmt.Sprintf("%#v", wantAllowances.DailySpend) {
		return fmt.Sprintf("daily spend: %#v != %#v", gotAllowances.DailySpend, wantAllowances.DailySpend)
	}
	if fmt.Sprintf("%#v", gotAllowances.RewardFloor) != fmt.Sprintf("%#v", wantAllowances.RewardFloor) {
		return fmt.Sprintf("reward floor: %#v != %#v", gotAllowances.RewardFloor, wantAllowances.RewardFloor)
	}
	if fmt.Sprintf("%#v", gotAllowances.TokenBudget) != fmt.Sprintf("%#v", wantAllowances.TokenBudget) {
		return fmt.Sprintf("token budget: %#v != %#v", gotAllowances.TokenBudget, wantAllowances.TokenBudget)
	}
	gotTypes, gotLimited := gotAllowances.TaskTypes.(agent.TaskTypesLimited)
	wantTypes, wantLimited := wantAllowances.TaskTypes.(agent.TaskTypesLimited)
	if gotLimited != wantLimited {
		return fmt.Sprintf("task type restriction presence: %T != %T", gotAllowances.TaskTypes, wantAllowances.TaskTypes)
	}
	if gotLimited {
		gotValues, wantValues := gotTypes.Values(), wantTypes.Values()
		if len(gotValues) != len(wantValues) {
			return fmt.Sprintf("task type count: %d != %d", len(gotValues), len(wantValues))
		}
		for index := range wantValues {
			if gotValues[index] != wantValues[index] {
				return fmt.Sprintf("task type %d: %s != %s", index, gotValues[index].String(), wantValues[index].String())
			}
		}
	}
	return ""
}
