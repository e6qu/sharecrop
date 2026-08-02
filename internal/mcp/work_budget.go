package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// toolGetMyBudget is the self-introspection tool: an agent reads the work
// policy its human configured for the credential it is currently using, plus
// what it has already consumed today.
const toolGetMyBudget = "sharecrop.get_my_budget"

// WorkBudgetView is what sharecrop.get_my_budget may report about the calling
// credential. It is a sealed union over the credential kinds MCP
// authenticates: an organization-wide credential (which carries no work
// policy at all), a task-scoped worker credential (which works inside the
// reservation it was issued for and is never budgeted), or a personal agent
// credential together with the work policy its owner configured.
type WorkBudgetView interface {
	workBudgetView()
}

// OrganizationCredentialBudget is an organization-wide credential: work
// policies are a property of personal agent credentials only.
type OrganizationCredentialBudget struct{}

// TaskScopedCredentialBudget is a credential restricted to one task.
type TaskScopedCredentialBudget struct {
	TaskID core.TaskID
}

// PersonalCredentialBudget is an unscoped personal agent credential and the
// work policy stored on it.
type PersonalCredentialBudget struct {
	CredentialID core.AgentCredentialID
	Policy       agent.WorkPolicy
}

func (OrganizationCredentialBudget) workBudgetView() {}

func (TaskScopedCredentialBudget) workBudgetView() {}

func (PersonalCredentialBudget) workBudgetView() {}

// Credential-kind discriminators on the get_my_budget result.
const (
	personalCredentialKind   = "personal_agent_credential"
	taskScopedCredentialKind = "task_scoped_worker_credential"
)

// workSeekingDisabledGuidance tells an agent refused for work-seeking what is
// actually wrong: this is a configuration state only its human can change, so
// retrying the same call is futile.
const workSeekingDisabledGuidance = "your operator has not enabled work-seeking for this credential; call sharecrop.get_my_budget to read its work policy, and ask your operator to enable work-seeking with a daily task budget before trying again"

func budgetToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolGetMyBudget,
			Description: "Read the work budget your human configured for the credential you are calling with, plus what you have already consumed today. Requires no scope: a credential may always inspect its own limits. credential_kind is \"personal_agent_credential\" (the budgeted case) or \"task_scoped_worker_credential\" (a credential issued for one task, which works inside its existing reservation and carries no budget); an organization credential has no work policy and is refused. For a personal credential the result carries work_seeking (work_seeking_enabled or work_seeking_disabled - a credential starts disabled and cannot reserve tasks or submit to tasks it was not handed until its human enables it), max_tasks_per_day with tasks_used_today and tasks_remaining_today, max_concurrent_reservations with active_reservations, max_credits_per_day with credits_spent_today, task_types (the task types you may work; empty means every type), min_reward_credits (skip tasks paying less; 0 means no floor), token_budget_tokens with token_budget_note, and resets_at. Conventions: every allowance except max_tasks_per_day uses 0 (or an empty list) to mean \"not configured\", so no limit applies; when work_seeking is work_seeking_disabled every allowance is 0 and no work may be taken at all. resets_at is the next UTC midnight, when the daily task and credit counters return to zero. The token budget is ADVISORY: Sharecrop never meters model tokens, so metering your own usage against it is your responsibility.",
			Access:      toolNeedsNoScope{},
			InputSchema: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{}}`),
		},
	}
}

// myBudgetPayload is get_my_budget's result for a personal agent credential.
// Derived fields (tasks_remaining_today, resets_at) are computed here so an
// agent loop does not have to redo the arithmetic or the calendar rule.
type myBudgetPayload struct {
	CredentialKind            string   `json:"credential_kind"`
	WorkSeeking               string   `json:"work_seeking"`
	MaxTasksPerDay            int64    `json:"max_tasks_per_day"`
	TasksUsedToday            int64    `json:"tasks_used_today"`
	TasksRemainingToday       int64    `json:"tasks_remaining_today"`
	MaxConcurrentReservations int64    `json:"max_concurrent_reservations"`
	ActiveReservations        int64    `json:"active_reservations"`
	MaxCreditsPerDay          int64    `json:"max_credits_per_day"`
	CreditsSpentToday         int64    `json:"credits_spent_today"`
	TaskTypes                 []string `json:"task_types"`
	MinRewardCredits          int64    `json:"min_reward_credits"`
	TokenBudgetTokens         int64    `json:"token_budget_tokens"`
	TokenBudgetNote           string   `json:"token_budget_note"`
	ResetsAt                  string   `json:"resets_at"`
}

// taskScopedBudgetPayload is get_my_budget's result for a task-scoped worker
// credential: it reports what the credential is rather than a budget, since
// enforcement happened when the reservation it was issued for was granted.
type taskScopedBudgetPayload struct {
	CredentialKind string `json:"credential_kind"`
	TaskID         string `json:"task_id"`
	Guidance       string `json:"guidance"`
}

func (myBudgetPayload) payloadValue() {}

func (taskScopedBudgetPayload) payloadValue() {}

// callGetMyBudget answers only for the credential that authenticated this
// call: the policy travels on the caller's own CallerCredential, and the
// consumption read is keyed by that same credential id, so no other
// credential's numbers can be reached through this tool.
func (server Server) callGetMyBudget(ctx context.Context, subject auth.Subject, credential CallerCredential) toolResult {
	switch view := credential.Budget.(type) {
	case PersonalCredentialBudget:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.personalBudgetResult(ctx, userActor, view)
	case TaskScopedCredentialBudget:
		return marshalPayload(taskScopedBudgetPayload{
			CredentialKind: taskScopedCredentialKind,
			TaskID:         view.TaskID.String(),
			Guidance:       "this credential was issued for one task and works inside the reservation it already holds; it carries no work policy and cannot reserve further tasks",
		})
	case OrganizationCredentialBudget:
		return toolFailed{code: core.ErrorCodePermissionDenied, message: "organization credentials do not carry work policies; work budgets are configured per personal agent credential"}
	default:
		return toolFailed{code: core.ErrorCodeInvalidState, message: "this credential's work policy is unavailable"}
	}
}

func (server Server) personalBudgetResult(ctx context.Context, subject auth.UserSubject, view PersonalCredentialBudget) toolResult {
	activityResult := server.services.AgentWorkActivity(ctx, subject.ID, view.CredentialID)
	listed, matched := activityResult.(agent.WorkActivityListed)
	if !matched {
		reason := activityResult.(agent.WorkActivityRejected).Reason
		return toolFailed{code: reason.Code(), message: reason.Description()}
	}
	return marshalPayload(budgetPayload(view.Policy, usageOf(listed.Values), time.Now()))
}

// usageOf reduces the activity read to this credential's consumption. The
// read is already keyed by the calling credential, and a credential that has
// consumed nothing today has no stored row at all, so an empty read is a
// factual triple of zeros rather than a missing answer.
func usageOf(values []agent.CredentialWorkActivity) agent.CredentialWorkActivity {
	if len(values) == 0 {
		return agent.CredentialWorkActivity{}
	}
	return values[0]
}

// budgetPayload flattens the sealed policy and the day's consumption into the
// agent-facing result, applying the documented "0 means not configured"
// convention for every allowance a human left unset.
func budgetPayload(policy agent.WorkPolicy, usage agent.CredentialWorkActivity, now time.Time) myBudgetPayload {
	payload := myBudgetPayload{
		CredentialKind:     personalCredentialKind,
		WorkSeeking:        agent.WorkPolicyState(policy).String(),
		TasksUsedToday:     usage.TasksToday,
		ActiveReservations: usage.ActiveReservations,
		CreditsSpentToday:  usage.CreditsSpentToday,
		TaskTypes:          []string{},
		ResetsAt:           nextUTCMidnight(now).Format(time.RFC3339),
	}
	enabled, isEnabled := policy.(agent.WorkPolicyEnabled)
	if !isEnabled {
		return payload
	}
	payload.MaxTasksPerDay = enabled.Allowances.MaxTasksPerDay.Int64()
	payload.TasksRemainingToday = remainingToday(payload.MaxTasksPerDay, usage.TasksToday)
	if capped, matched := enabled.Allowances.ConcurrentReservations.(agent.ConcurrentReservationsCapped); matched {
		payload.MaxConcurrentReservations = capped.Limit.Int64()
	}
	if capped, matched := enabled.Allowances.DailySpend.(agent.DailySpendCapped); matched {
		payload.MaxCreditsPerDay = capped.Limit.Int64()
	}
	if limited, matched := enabled.Allowances.TaskTypes.(agent.TaskTypesLimited); matched {
		for _, taskType := range limited.Values() {
			payload.TaskTypes = append(payload.TaskTypes, taskType.String())
		}
	}
	if floor, matched := enabled.Allowances.RewardFloor.(agent.RewardFloorAtLeast); matched {
		payload.MinRewardCredits = floor.Minimum.Int64()
	}
	if advised, matched := enabled.Allowances.TokenBudget.(agent.TokenBudgetAdvised); matched {
		payload.TokenBudgetTokens = advised.Tokens.Int64()
		payload.TokenBudgetNote = advised.Note.String()
	}
	return payload
}

// remainingToday never reports a negative remainder: an over-consumed budget
// (the day limit was lowered after work was taken) has nothing left, not a
// debt.
func remainingToday(limit int64, used int64) int64 {
	if used >= limit {
		return 0
	}
	return limit - used
}

// nextUTCMidnight is the instant the daily counters reset: budget windows are
// UTC calendar days (see internal/db's work day counters).
func nextUTCMidnight(now time.Time) time.Time {
	day := now.UTC()
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
}

// workSeekingIsDisabled reports whether the caller is a personal credential
// whose owner has not enabled work-seeking - the state that makes every
// reserve and direct-submit attempt fail the same way until a human changes
// it.
func workSeekingIsDisabled(credential CallerCredential) bool {
	personal, matched := credential.Budget.(PersonalCredentialBudget)
	if !matched {
		return false
	}
	return agent.WorkPolicyState(personal.Policy) == agent.WorkSeekingDisabled
}

// withWorkSeekingGuidance adds the operator-facing guidance line to a worker
// tool's permission refusal when the calling credential has work-seeking
// disabled. Every other failure, and every other tool, is returned untouched.
func withWorkSeekingGuidance(credential CallerCredential, name string, outcome toolResult) toolResult {
	if name != toolReserveTask && name != toolSubmitResponse {
		return outcome
	}
	failure, isFailure := outcome.(toolFailed)
	if !isFailure || failure.code != core.ErrorCodePermissionDenied {
		return outcome
	}
	if !workSeekingIsDisabled(credential) {
		return outcome
	}
	failure.guidance = workSeekingDisabledGuidance
	return failure
}
