// Package agentbridge is the WASI bridge for internal/agent's Store (agent MCP
// credentials): hand-written per-type codecs (this file) plus a generated
// dispatcher and guest client (bridge_gen.go). Shared core types are serialized
// by internal/wasibridge/corewire; the agent value types it shares with the
// orgcred bridge (Label, State, ScopeSet, CreateStoreResult) live in
// internal/wasibridge/agentwire. Only what is specific to the agent store's
// Credential and result unions is here.
//
// Like auth, the credential SecretHash is an opaque stored string, so it
// round-trips through agent.SecretHashFromString rather than by re-hashing.
package agentbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasibridge/agentwire"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- value types specific to the agent store ----

func encodeSecretHash(hash agent.SecretHash) string { return hash.String() }

func decodeSecretHash(raw string) (agent.SecretHash, error) {
	return agent.SecretHashFromString(raw), nil
}

func encodeTaskIDPtr(value *core.TaskID) string {
	if value == nil {
		return ""
	}
	return corewire.EncodeTaskID(*value)
}

func decodeTaskIDPtr(raw string) (*core.TaskID, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := corewire.DecodeTaskID(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// ---- agent.WorkPolicy ----

type workPolicyWire struct {
	State string `json:"state"`
	// Allowance fields are meaningful only for the enabled state; zero /
	// empty means the allowance is absent.
	MaxTasksPerDay            int64    `json:"max_tasks_per_day,omitempty"`
	MaxConcurrentReservations int64    `json:"max_concurrent_reservations,omitempty"`
	MaxCreditsPerDay          int64    `json:"max_credits_per_day,omitempty"`
	MinRewardCredits          int64    `json:"min_reward_credits,omitempty"`
	TokenBudgetTokens         int64    `json:"token_budget_tokens,omitempty"`
	TokenBudgetNote           string   `json:"token_budget_note,omitempty"`
	TaskTypes                 []string `json:"task_types,omitempty"`
}

func encodeWorkPolicy(policy agent.WorkPolicy) workPolicyWire {
	wire := workPolicyWire{State: agent.WorkPolicyState(policy).String()}
	enabled, isEnabled := policy.(agent.WorkPolicyEnabled)
	if !isEnabled {
		return wire
	}
	wire.MaxTasksPerDay = enabled.Allowances.MaxTasksPerDay.Int64()
	if capped, matched := enabled.Allowances.ConcurrentReservations.(agent.ConcurrentReservationsCapped); matched {
		wire.MaxConcurrentReservations = capped.Limit.Int64()
	}
	if capped, matched := enabled.Allowances.DailySpend.(agent.DailySpendCapped); matched {
		wire.MaxCreditsPerDay = capped.Limit.Int64()
	}
	if floor, matched := enabled.Allowances.RewardFloor.(agent.RewardFloorAtLeast); matched {
		wire.MinRewardCredits = floor.Minimum.Int64()
	}
	if advised, matched := enabled.Allowances.TokenBudget.(agent.TokenBudgetAdvised); matched {
		wire.TokenBudgetTokens = advised.Tokens.Int64()
		wire.TokenBudgetNote = advised.Note.String()
	}
	if limited, matched := enabled.Allowances.TaskTypes.(agent.TaskTypesLimited); matched {
		for _, taskType := range limited.Values() {
			wire.TaskTypes = append(wire.TaskTypes, taskType.String())
		}
	}
	return wire
}

func decodeWorkPolicy(wire workPolicyWire) (agent.WorkPolicy, error) {
	stateResult := agent.ParseWorkSeekingState(wire.State)
	state, stateMatched := stateResult.(agent.WorkSeekingStateAccepted)
	if !stateMatched {
		return nil, fmt.Errorf("work policy state %q is invalid", wire.State)
	}
	if state.Value == agent.WorkSeekingDisabled {
		return agent.WorkPolicyDisabled{}, nil
	}

	budgetResult := agent.NewDailyTaskBudget(wire.MaxTasksPerDay)
	budget, budgetMatched := budgetResult.(agent.DailyTaskBudgetAccepted)
	if !budgetMatched {
		return nil, fmt.Errorf("work policy daily task budget %d is invalid", wire.MaxTasksPerDay)
	}
	allowances := agent.WorkAllowances{
		MaxTasksPerDay:         budget.Value,
		ConcurrentReservations: agent.ConcurrentReservationsUnlimited{},
		DailySpend:             agent.DailySpendUnlimited{},
		TaskTypes:              agent.AllTaskTypesAllowed{},
		RewardFloor:            agent.NoRewardFloor{},
		TokenBudget:            agent.NoTokenBudgetAdvisory{},
	}
	if wire.MaxConcurrentReservations != 0 {
		capResult := agent.NewConcurrentReservationCap(wire.MaxConcurrentReservations)
		capAccepted, capMatched := capResult.(agent.ConcurrentReservationCapAccepted)
		if !capMatched {
			return nil, fmt.Errorf("work policy concurrent reservation cap %d is invalid", wire.MaxConcurrentReservations)
		}
		allowances.ConcurrentReservations = agent.ConcurrentReservationsCapped{Limit: capAccepted.Value}
	}
	if wire.MaxCreditsPerDay != 0 {
		amountResult := ledger.NewCreditAmount(wire.MaxCreditsPerDay)
		amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
		if !amountMatched {
			return nil, fmt.Errorf("work policy daily spend cap %d is invalid", wire.MaxCreditsPerDay)
		}
		allowances.DailySpend = agent.DailySpendCapped{Limit: amount.Value}
	}
	if wire.MinRewardCredits != 0 {
		amountResult := ledger.NewCreditAmount(wire.MinRewardCredits)
		amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
		if !amountMatched {
			return nil, fmt.Errorf("work policy reward floor %d is invalid", wire.MinRewardCredits)
		}
		allowances.RewardFloor = agent.RewardFloorAtLeast{Minimum: amount.Value}
	}
	if wire.TokenBudgetTokens != 0 {
		tokensResult := agent.NewTokenBudgetTokens(wire.TokenBudgetTokens)
		tokens, tokensMatched := tokensResult.(agent.TokenBudgetTokensAccepted)
		if !tokensMatched {
			return nil, fmt.Errorf("work policy token budget %d is invalid", wire.TokenBudgetTokens)
		}
		noteResult := agent.NewTokenBudgetNote(wire.TokenBudgetNote)
		note, noteMatched := noteResult.(agent.TokenBudgetNoteAccepted)
		if !noteMatched {
			return nil, fmt.Errorf("work policy token budget note is invalid")
		}
		allowances.TokenBudget = agent.TokenBudgetAdvised{Tokens: tokens.Value, Note: note.Value}
	}
	if len(wire.TaskTypes) > 0 {
		types := make([]task.TaskType, 0, len(wire.TaskTypes))
		for _, rawType := range wire.TaskTypes {
			typeResult := task.ParseTaskType(rawType)
			typeAccepted, typeMatched := typeResult.(task.TaskTypeAccepted)
			if !typeMatched {
				return nil, fmt.Errorf("work policy task type %q is invalid", rawType)
			}
			types = append(types, typeAccepted.Value)
		}
		limitedResult := agent.NewTaskTypesLimited(types)
		limited, limitedMatched := limitedResult.(agent.TaskTypesLimitedAccepted)
		if !limitedMatched {
			return nil, fmt.Errorf("work policy task type restriction is invalid")
		}
		allowances.TaskTypes = limited.Value
	}
	return agent.WorkPolicyEnabled{Allowances: allowances}, nil
}

// ---- agent.Credential ----

type credentialWire struct {
	ID         string         `json:"id"`
	UserID     string         `json:"user_id"`
	Label      string         `json:"label"`
	Scopes     []string       `json:"scopes"`
	State      string         `json:"state"`
	ExpiresAt  string         `json:"expires_at,omitempty"`
	TaskID     string         `json:"task_id,omitempty"`
	WorkPolicy workPolicyWire `json:"work_policy"`
}

func encodeCredential(credential agent.Credential) credentialWire {
	return credentialWire{
		ID:         corewire.EncodeAgentCredentialID(credential.ID),
		UserID:     corewire.EncodeUserID(credential.UserID),
		Label:      agentwire.EncodeLabel(credential.Label),
		Scopes:     agentwire.EncodeScopeSet(credential.Scopes),
		State:      agentwire.EncodeState(credential.State),
		ExpiresAt:  corewire.EncodeTimePtr(credential.ExpiresAt),
		TaskID:     encodeTaskIDPtr(credential.TaskID),
		WorkPolicy: encodeWorkPolicy(credential.WorkPolicy),
	}
}

func decodeCredential(wire credentialWire) (agent.Credential, error) {
	id, err := corewire.DecodeAgentCredentialID(wire.ID)
	if err != nil {
		return agent.Credential{}, err
	}
	userID, err := corewire.DecodeUserID(wire.UserID)
	if err != nil {
		return agent.Credential{}, err
	}
	label, err := agentwire.DecodeLabel(wire.Label)
	if err != nil {
		return agent.Credential{}, err
	}
	scopes, err := agentwire.DecodeScopeSet(wire.Scopes)
	if err != nil {
		return agent.Credential{}, err
	}
	state, err := agentwire.DecodeState(wire.State)
	if err != nil {
		return agent.Credential{}, err
	}
	expiresAt, err := corewire.DecodeTimePtr(wire.ExpiresAt)
	if err != nil {
		return agent.Credential{}, err
	}
	taskID, err := decodeTaskIDPtr(wire.TaskID)
	if err != nil {
		return agent.Credential{}, err
	}
	workPolicy, err := decodeWorkPolicy(wire.WorkPolicy)
	if err != nil {
		return agent.Credential{}, err
	}
	return agent.Credential{
		ID:         id,
		UserID:     userID,
		Label:      label,
		Scopes:     scopes,
		State:      state,
		ExpiresAt:  expiresAt,
		TaskID:     taskID,
		WorkPolicy: workPolicy,
	}, nil
}

// ---- result unions specific to the agent store ----
//
// CreateStoreResult is shared with orgcred, so it is serialized by agentwire;
// the verify/revoke/list results carry an agent.Credential and stay here.

type credentialResultWire struct {
	Variant    string                  `json:"variant"`
	Credential *credentialWire         `json:"credential,omitempty"`
	Error      *domainwire.DomainError `json:"error,omitempty"`
}

func encodeVerifyResult(result agent.VerifyStoreResult) credentialResultWire {
	switch typed := result.(type) {
	case agent.VerifyStoreFound:
		credential := encodeCredential(typed.Value)
		return credentialResultWire{Variant: "found", Credential: &credential}
	case agent.VerifyStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return credentialResultWire{Variant: "rejected", Error: &reason}
	default:
		return credentialResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown agent result %T", result))}
	}
}

func decodeVerifyResult(wire credentialResultWire) (agent.VerifyStoreResult, error) {
	switch wire.Variant {
	case "found":
		credential, err := decodeCredentialPayload(wire.Credential)
		if err != nil {
			return nil, err
		}
		return agent.VerifyStoreFound{Value: credential}, nil
	case "rejected":
		return agent.VerifyStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown verify result variant %q", wire.Variant)
	}
}

func encodeRevokeResult(result agent.RevokeStoreResult) credentialResultWire {
	switch typed := result.(type) {
	case agent.RevokeStoreRevoked:
		credential := encodeCredential(typed.Value)
		return credentialResultWire{Variant: "revoked", Credential: &credential}
	case agent.RevokeStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return credentialResultWire{Variant: "rejected", Error: &reason}
	default:
		return credentialResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown agent result %T", result))}
	}
}

func decodeRevokeResult(wire credentialResultWire) (agent.RevokeStoreResult, error) {
	switch wire.Variant {
	case "revoked":
		credential, err := decodeCredentialPayload(wire.Credential)
		if err != nil {
			return nil, err
		}
		return agent.RevokeStoreRevoked{Value: credential}, nil
	case "rejected":
		return agent.RevokeStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown revoke result variant %q", wire.Variant)
	}
}

type listResultWire struct {
	Variant     string                  `json:"variant"`
	Credentials []credentialWire        `json:"credentials,omitempty"`
	Error       *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result agent.ListStoreResult) listResultWire {
	switch typed := result.(type) {
	case agent.ListStoreListed:
		credentials := make([]credentialWire, 0, len(typed.Values))
		for index := range typed.Values {
			credentials = append(credentials, encodeCredential(typed.Values[index]))
		}
		return listResultWire{Variant: "listed", Credentials: credentials}
	case agent.ListStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return listResultWire{Variant: "rejected", Error: &reason}
	default:
		return listResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown agent result %T", result))}
	}
}

func decodeListResult(wire listResultWire) (agent.ListStoreResult, error) {
	switch wire.Variant {
	case "listed":
		credentials := make([]agent.Credential, 0, len(wire.Credentials))
		for index := range wire.Credentials {
			credential, err := decodeCredential(wire.Credentials[index])
			if err != nil {
				return nil, err
			}
			credentials = append(credentials, credential)
		}
		return agent.ListStoreListed{Values: credentials}, nil
	case "rejected":
		return agent.ListStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list result variant %q", wire.Variant)
	}
}

func encodeUpdateWorkPolicyResult(result agent.UpdateWorkPolicyStoreResult) credentialResultWire {
	switch typed := result.(type) {
	case agent.UpdateWorkPolicyStoreUpdated:
		credential := encodeCredential(typed.Value)
		return credentialResultWire{Variant: "updated", Credential: &credential}
	case agent.UpdateWorkPolicyStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return credentialResultWire{Variant: "rejected", Error: &reason}
	default:
		return credentialResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown agent result %T", result))}
	}
}

func decodeUpdateWorkPolicyResult(wire credentialResultWire) (agent.UpdateWorkPolicyStoreResult, error) {
	switch wire.Variant {
	case "updated":
		credential, err := decodeCredentialPayload(wire.Credential)
		if err != nil {
			return nil, err
		}
		return agent.UpdateWorkPolicyStoreUpdated{Value: credential}, nil
	case "rejected":
		return agent.UpdateWorkPolicyStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown update work policy result variant %q", wire.Variant)
	}
}

// ---- agent.CredentialWorkActivity ----

type workActivityWire struct {
	CredentialID       string `json:"credential_id"`
	TasksToday         int64  `json:"tasks_today"`
	CreditsSpentToday  int64  `json:"credits_spent_today"`
	ActiveReservations int64  `json:"active_reservations"`
}

type workActivityResultWire struct {
	Variant string                  `json:"variant"`
	Values  []workActivityWire      `json:"values,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeWorkActivityResult(result agent.WorkActivityStoreResult) workActivityResultWire {
	switch typed := result.(type) {
	case agent.WorkActivityStoreListed:
		values := make([]workActivityWire, 0, len(typed.Values))
		for index := range typed.Values {
			values = append(values, workActivityWire{
				CredentialID:       corewire.EncodeAgentCredentialID(typed.Values[index].CredentialID),
				TasksToday:         typed.Values[index].TasksToday,
				CreditsSpentToday:  typed.Values[index].CreditsSpentToday,
				ActiveReservations: typed.Values[index].ActiveReservations,
			})
		}
		return workActivityResultWire{Variant: "listed", Values: values}
	case agent.WorkActivityStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return workActivityResultWire{Variant: "rejected", Error: &reason}
	default:
		return workActivityResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown agent result %T", result))}
	}
}

func decodeWorkActivityResult(wire workActivityResultWire) (agent.WorkActivityStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values := make([]agent.CredentialWorkActivity, 0, len(wire.Values))
		for index := range wire.Values {
			credentialID, err := corewire.DecodeAgentCredentialID(wire.Values[index].CredentialID)
			if err != nil {
				return nil, err
			}
			values = append(values, agent.CredentialWorkActivity{
				CredentialID:       credentialID,
				TasksToday:         wire.Values[index].TasksToday,
				CreditsSpentToday:  wire.Values[index].CreditsSpentToday,
				ActiveReservations: wire.Values[index].ActiveReservations,
			})
		}
		return agent.WorkActivityStoreListed{Values: values}, nil
	case "rejected":
		return agent.WorkActivityStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown work activity result variant %q", wire.Variant)
	}
}

func decodeCredentialPayload(wire *credentialWire) (agent.Credential, error) {
	if wire == nil {
		return agent.Credential{}, fmt.Errorf("credential result is missing its credential")
	}
	return decodeCredential(*wire)
}

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "agent bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
