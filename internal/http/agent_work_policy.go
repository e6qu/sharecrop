package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/task"
)

// workPolicyRequest is the PUT /api/agent-credentials/{id}/work-policy body.
// Absent numeric fields decode as 0 and absent lists as empty, which mean
// "unlimited" / "no restriction" per the option semantics below; only
// max_tasks_per_day is required when work-seeking is enabled. Disabling
// work-seeking ignores every allowance field and clears the stored ones.
type workPolicyRequest struct {
	WorkSeeking string `json:"work_seeking"`
	// MaxTasksPerDay is required (1..10000) when work_seeking is
	// work_seeking_enabled.
	MaxTasksPerDay int64 `json:"max_tasks_per_day"`
	// MaxConcurrentReservations is optional: 0 leaves concurrency uncapped.
	MaxConcurrentReservations int64 `json:"max_concurrent_reservations"`
	// MaxCreditsPerDay is optional: 0 leaves daily spending uncapped.
	MaxCreditsPerDay int64 `json:"max_credits_per_day"`
	// TaskTypes is optional: empty allows every task type.
	TaskTypes []string `json:"task_types"`
	// MinRewardCredits is optional: 0 sets no reward floor.
	MinRewardCredits int64 `json:"min_reward_credits"`
	// TokenBudgetTokens and TokenBudgetNote are the advisory token budget:
	// stored and returned, never enforced. 0 tokens means no advisory budget;
	// a note requires a token count.
	TokenBudgetTokens int64  `json:"token_budget_tokens"`
	TokenBudgetNote   string `json:"token_budget_note"`
}

type workPolicyParseResult interface {
	workPolicyParseResult()
}

type workPolicyParsed struct {
	value agent.WorkPolicy
}

type workPolicyParseRejected struct {
	reason core.DomainError
}

func (workPolicyParsed) workPolicyParseResult() {}

func (workPolicyParseRejected) workPolicyParseResult() {}

// parseWorkPolicyRequest converts the request body into the domain policy,
// running every allowance through its validating constructor.
func parseWorkPolicyRequest(request workPolicyRequest) workPolicyParseResult {
	stateResult := agent.ParseWorkSeekingState(request.WorkSeeking)
	state, stateMatched := stateResult.(agent.WorkSeekingStateAccepted)
	if !stateMatched {
		return workPolicyParseRejected{reason: stateResult.(agent.WorkSeekingStateRejected).Reason}
	}
	if state.Value == agent.WorkSeekingDisabled {
		return workPolicyParsed{value: agent.WorkPolicyDisabled{}}
	}

	budgetResult := agent.NewDailyTaskBudget(request.MaxTasksPerDay)
	budget, budgetMatched := budgetResult.(agent.DailyTaskBudgetAccepted)
	if !budgetMatched {
		return workPolicyParseRejected{reason: budgetResult.(agent.DailyTaskBudgetRejected).Reason}
	}
	allowances := agent.WorkAllowances{
		MaxTasksPerDay:         budget.Value,
		ConcurrentReservations: agent.ConcurrentReservationsUnlimited{},
		DailySpend:             agent.DailySpendUnlimited{},
		TaskTypes:              agent.AllTaskTypesAllowed{},
		RewardFloor:            agent.NoRewardFloor{},
		TokenBudget:            agent.NoTokenBudgetAdvisory{},
	}

	if request.MaxConcurrentReservations != 0 {
		capResult := agent.NewConcurrentReservationCap(request.MaxConcurrentReservations)
		capAccepted, capMatched := capResult.(agent.ConcurrentReservationCapAccepted)
		if !capMatched {
			return workPolicyParseRejected{reason: capResult.(agent.ConcurrentReservationCapRejected).Reason}
		}
		allowances.ConcurrentReservations = agent.ConcurrentReservationsCapped{Limit: capAccepted.Value}
	}
	if request.MaxCreditsPerDay != 0 {
		amountResult := ledger.NewCreditAmount(request.MaxCreditsPerDay)
		amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
		if !amountMatched {
			return workPolicyParseRejected{reason: amountResult.(ledger.CreditAmountRejected).Reason}
		}
		allowances.DailySpend = agent.DailySpendCapped{Limit: amount.Value}
	}
	if len(request.TaskTypes) > 0 {
		types := make([]task.TaskType, 0, len(request.TaskTypes))
		for _, rawType := range request.TaskTypes {
			typeResult := task.ParseTaskType(rawType)
			typeAccepted, typeMatched := typeResult.(task.TaskTypeAccepted)
			if !typeMatched {
				return workPolicyParseRejected{reason: typeResult.(task.TaskTypeRejected).Reason}
			}
			types = append(types, typeAccepted.Value)
		}
		limitedResult := agent.NewTaskTypesLimited(types)
		limited, limitedMatched := limitedResult.(agent.TaskTypesLimitedAccepted)
		if !limitedMatched {
			return workPolicyParseRejected{reason: limitedResult.(agent.TaskTypesLimitedRejected).Reason}
		}
		allowances.TaskTypes = limited.Value
	}
	if request.MinRewardCredits != 0 {
		amountResult := ledger.NewCreditAmount(request.MinRewardCredits)
		amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
		if !amountMatched {
			return workPolicyParseRejected{reason: amountResult.(ledger.CreditAmountRejected).Reason}
		}
		allowances.RewardFloor = agent.RewardFloorAtLeast{Minimum: amount.Value}
	}
	if request.TokenBudgetTokens == 0 && request.TokenBudgetNote != "" {
		return workPolicyParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "token_budget_note requires token_budget_tokens")}
	}
	if request.TokenBudgetTokens != 0 {
		tokensResult := agent.NewTokenBudgetTokens(request.TokenBudgetTokens)
		tokens, tokensMatched := tokensResult.(agent.TokenBudgetTokensAccepted)
		if !tokensMatched {
			return workPolicyParseRejected{reason: tokensResult.(agent.TokenBudgetTokensRejected).Reason}
		}
		noteResult := agent.NewTokenBudgetNote(request.TokenBudgetNote)
		note, noteMatched := noteResult.(agent.TokenBudgetNoteAccepted)
		if !noteMatched {
			return workPolicyParseRejected{reason: noteResult.(agent.TokenBudgetNoteRejected).Reason}
		}
		allowances.TokenBudget = agent.TokenBudgetAdvised{Tokens: tokens.Value, Note: note.Value}
	}

	return workPolicyParsed{value: agent.WorkPolicyEnabled{Allowances: allowances}}
}

// configureAgentWorkPolicy handles PUT /api/agent-credentials/{id}/work-policy:
// the owning user session enables work-seeking with a budget or disables it
// again. The response is the full credential, including the stored policy and
// today's consumption.
func (server Server) configureAgentWorkPolicy(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	credentialIDResult := core.ParseAgentCredentialID(r.PathValue("credential_id"))
	credentialID, credentialMatched := credentialIDResult.(core.AgentCredentialIDCreated)
	if !credentialMatched {
		writeDomainError(w, credentialIDResult.(core.AgentCredentialIDRejected).Reason)
		return
	}

	var request workPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	parseResult := parseWorkPolicyRequest(request)
	parsed, parsedMatched := parseResult.(workPolicyParsed)
	if !parsedMatched {
		writeDomainError(w, parseResult.(workPolicyParseRejected).reason)
		return
	}

	result := server.agentService.ConfigureWorkPolicy(r.Context(), actor.subject.ID, credentialID.Value, parsed.value)
	configured, matched := result.(agent.WorkPolicyConfigured)
	if !matched {
		writeDomainError(w, result.(agent.ConfigureWorkPolicyRejected).Reason)
		return
	}

	response, activityErr := server.credentialResponseWithActivity(r.Context(), actor.subject.ID, configured.Value)
	if activityErr != nil {
		writeDomainError(w, *activityErr)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
