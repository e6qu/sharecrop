package agent

import (
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/task"
)

// WorkSeekingState says whether a personal agent credential may seek work on
// its own (reserve tasks, submit directly to unreserved tasks, spend credits).
// Every newly minted credential starts work_seeking_disabled: an agent does
// not request work until its human explicitly enables it with a budget.
type WorkSeekingState struct {
	value string
}

var (
	WorkSeekingDisabled = WorkSeekingState{value: "work_seeking_disabled"}
	WorkSeekingEnabled  = WorkSeekingState{value: "work_seeking_enabled"}
)

type WorkSeekingStateResult interface {
	workSeekingStateResult()
}

type WorkSeekingStateAccepted struct {
	Value WorkSeekingState
}

type WorkSeekingStateRejected struct {
	Reason core.DomainError
}

func (WorkSeekingStateAccepted) workSeekingStateResult() {}

func (WorkSeekingStateRejected) workSeekingStateResult() {}

func ParseWorkSeekingState(raw string) WorkSeekingStateResult {
	switch raw {
	case WorkSeekingDisabled.value:
		return WorkSeekingStateAccepted{Value: WorkSeekingDisabled}
	case WorkSeekingEnabled.value:
		return WorkSeekingStateAccepted{Value: WorkSeekingEnabled}
	default:
		return WorkSeekingStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "work-seeking state is invalid")}
	}
}

func (state WorkSeekingState) String() string {
	return state.value
}

// WorkPolicy is the work-budget policy attached to a personal agent
// credential. It is a tagged union over the work-seeking state: a disabled
// credential carries no allowances at all, an enabled one carries the full
// allowance set its human configured.
type WorkPolicy interface {
	workPolicy()
}

// WorkPolicyDisabled is the default for every newly minted credential: the
// credential cannot reserve tasks, cannot submit to tasks it has not been
// handed, and cannot spend credits.
type WorkPolicyDisabled struct{}

// WorkPolicyEnabled allows work-seeking within the configured allowances.
type WorkPolicyEnabled struct {
	Allowances WorkAllowances
}

func (WorkPolicyDisabled) workPolicy() {}

func (WorkPolicyEnabled) workPolicy() {}

// State reports the sealed work-seeking state of a policy.
func WorkPolicyState(policy WorkPolicy) WorkSeekingState {
	if _, enabled := policy.(WorkPolicyEnabled); enabled {
		return WorkSeekingEnabled
	}
	return WorkSeekingDisabled
}

// WorkAllowances groups every allowance of an enabled work policy. The daily
// task budget is always present (a human enabling work-seeking must state a
// number); every other allowance is an explicit per-type option whose absent
// variant means "no limit configured".
type WorkAllowances struct {
	// MaxTasksPerDay caps how many tasks the credential may take on per UTC
	// calendar day. Reservations count; so does a direct submission to a task
	// the credential had not already reserved.
	MaxTasksPerDay DailyTaskBudget
	// ConcurrentReservations caps how many reservations established via this
	// credential may be active at once.
	ConcurrentReservations ConcurrentReservationAllowance
	// DailySpend caps how many credits the credential may spend (task
	// funding, tips, peer sends) per UTC calendar day.
	DailySpend DailySpendAllowance
	// TaskTypes restricts which task types the credential may reserve or
	// submit to.
	TaskTypes TaskTypeAllowance
	// RewardFloor skips tasks whose credit reward is below the configured
	// minimum, so the agent does not burn its daily budget on work its human
	// considers not worth taking.
	RewardFloor RewardFloorAllowance
	// TokenBudget is ADVISORY ONLY: the server stores it and hands it back to
	// the agent, but NEVER enforces it. It exists so a human can tell their
	// agent how many model tokens the day's work should cost, in the same
	// place as the enforced budgets.
	TokenBudget TokenBudgetAdvisory
}

// DailyTaskBudget is the required tasks-per-day cap of an enabled policy.
type DailyTaskBudget struct {
	value int64
}

const maxDailyTaskBudget = 10000

type DailyTaskBudgetResult interface {
	dailyTaskBudgetResult()
}

type DailyTaskBudgetAccepted struct {
	Value DailyTaskBudget
}

type DailyTaskBudgetRejected struct {
	Reason core.DomainError
}

func (DailyTaskBudgetAccepted) dailyTaskBudgetResult() {}

func (DailyTaskBudgetRejected) dailyTaskBudgetResult() {}

func NewDailyTaskBudget(value int64) DailyTaskBudgetResult {
	if value < 1 {
		return DailyTaskBudgetRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "daily task budget must be at least 1")}
	}
	if value > maxDailyTaskBudget {
		return DailyTaskBudgetRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "daily task budget is too large")}
	}
	return DailyTaskBudgetAccepted{Value: DailyTaskBudget{value: value}}
}

func (budget DailyTaskBudget) Int64() int64 {
	return budget.value
}

// ConcurrentReservationAllowance caps active reservations established via the
// credential: absent means no cap.
type ConcurrentReservationAllowance interface {
	concurrentReservationAllowance()
}

type ConcurrentReservationsUnlimited struct{}

type ConcurrentReservationsCapped struct {
	Limit ConcurrentReservationCap
}

func (ConcurrentReservationsUnlimited) concurrentReservationAllowance() {}

func (ConcurrentReservationsCapped) concurrentReservationAllowance() {}

// ConcurrentReservationCap is a validated positive reservation count.
type ConcurrentReservationCap struct {
	value int64
}

const maxConcurrentReservationCap = 1000

type ConcurrentReservationCapResult interface {
	concurrentReservationCapResult()
}

type ConcurrentReservationCapAccepted struct {
	Value ConcurrentReservationCap
}

type ConcurrentReservationCapRejected struct {
	Reason core.DomainError
}

func (ConcurrentReservationCapAccepted) concurrentReservationCapResult() {}

func (ConcurrentReservationCapRejected) concurrentReservationCapResult() {}

func NewConcurrentReservationCap(value int64) ConcurrentReservationCapResult {
	if value < 1 {
		return ConcurrentReservationCapRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "concurrent reservation cap must be at least 1")}
	}
	if value > maxConcurrentReservationCap {
		return ConcurrentReservationCapRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "concurrent reservation cap is too large")}
	}
	return ConcurrentReservationCapAccepted{Value: ConcurrentReservationCap{value: value}}
}

func (cap ConcurrentReservationCap) Int64() int64 {
	return cap.value
}

// DailySpendAllowance caps credits spent per UTC day via the credential:
// absent means no cap.
type DailySpendAllowance interface {
	dailySpendAllowance()
}

type DailySpendUnlimited struct{}

type DailySpendCapped struct {
	Limit ledger.CreditAmount
}

func (DailySpendUnlimited) dailySpendAllowance() {}

func (DailySpendCapped) dailySpendAllowance() {}

// TaskTypeAllowance restricts which task types the credential may work:
// absent means every type is allowed.
type TaskTypeAllowance interface {
	taskTypeAllowance()
}

type AllTaskTypesAllowed struct{}

type TaskTypesLimited struct {
	values []task.TaskType
}

func (AllTaskTypesAllowed) taskTypeAllowance() {}

func (TaskTypesLimited) taskTypeAllowance() {}

type TaskTypesLimitedResult interface {
	taskTypesLimitedResult()
}

type TaskTypesLimitedAccepted struct {
	Value TaskTypesLimited
}

type TaskTypesLimitedRejected struct {
	Reason core.DomainError
}

func (TaskTypesLimitedAccepted) taskTypesLimitedResult() {}

func (TaskTypesLimitedRejected) taskTypesLimitedResult() {}

// NewTaskTypesLimited builds a de-duplicated, non-empty task-type restriction.
func NewTaskTypesLimited(types []task.TaskType) TaskTypesLimitedResult {
	unique := make([]task.TaskType, 0, len(types))
	for _, candidate := range types {
		if containsTaskType(unique, candidate) {
			continue
		}
		unique = append(unique, candidate)
	}
	if len(unique) == 0 {
		return TaskTypesLimitedRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "a task type restriction needs at least one task type")}
	}
	return TaskTypesLimitedAccepted{Value: TaskTypesLimited{values: unique}}
}

func (limited TaskTypesLimited) Values() []task.TaskType {
	copied := make([]task.TaskType, len(limited.values))
	copy(copied, limited.values)
	return copied
}

func (limited TaskTypesLimited) Allows(taskType task.TaskType) bool {
	return containsTaskType(limited.values, taskType)
}

func containsTaskType(types []task.TaskType, taskType task.TaskType) bool {
	for _, existing := range types {
		if existing == taskType {
			return true
		}
	}
	return false
}

// RewardFloorAllowance skips tasks rewarding fewer credits than the floor:
// absent means no floor.
type RewardFloorAllowance interface {
	rewardFloorAllowance()
}

type NoRewardFloor struct{}

type RewardFloorAtLeast struct {
	Minimum ledger.CreditAmount
}

func (NoRewardFloor) rewardFloorAllowance() {}

func (RewardFloorAtLeast) rewardFloorAllowance() {}

// TokenBudgetAdvisory is the stored-but-never-enforced token budget: absent,
// or a positive token count with an optional free-form note. The server does
// not meter model tokens; agents read this to pace themselves.
type TokenBudgetAdvisory interface {
	tokenBudgetAdvisory()
}

type NoTokenBudgetAdvisory struct{}

type TokenBudgetAdvised struct {
	Tokens TokenBudgetTokens
	Note   TokenBudgetNote
}

func (NoTokenBudgetAdvisory) tokenBudgetAdvisory() {}

func (TokenBudgetAdvised) tokenBudgetAdvisory() {}

// TokenBudgetTokens is a validated positive advisory token count.
type TokenBudgetTokens struct {
	value int64
}

type TokenBudgetTokensResult interface {
	tokenBudgetTokensResult()
}

type TokenBudgetTokensAccepted struct {
	Value TokenBudgetTokens
}

type TokenBudgetTokensRejected struct {
	Reason core.DomainError
}

func (TokenBudgetTokensAccepted) tokenBudgetTokensResult() {}

func (TokenBudgetTokensRejected) tokenBudgetTokensResult() {}

func NewTokenBudgetTokens(value int64) TokenBudgetTokensResult {
	if value < 1 {
		return TokenBudgetTokensRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "advisory token budget must be a positive number of tokens")}
	}
	return TokenBudgetTokensAccepted{Value: TokenBudgetTokens{value: value}}
}

func (tokens TokenBudgetTokens) Int64() int64 {
	return tokens.value
}

// TokenBudgetNote is the free-form note travelling with an advisory token
// budget. It may be empty; it is capped so the stored row stays small.
type TokenBudgetNote struct {
	value string
}

type TokenBudgetNoteResult interface {
	tokenBudgetNoteResult()
}

type TokenBudgetNoteAccepted struct {
	Value TokenBudgetNote
}

type TokenBudgetNoteRejected struct {
	Reason core.DomainError
}

func (TokenBudgetNoteAccepted) tokenBudgetNoteResult() {}

func (TokenBudgetNoteRejected) tokenBudgetNoteResult() {}

func NewTokenBudgetNote(raw string) TokenBudgetNoteResult {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) > 500 {
		return TokenBudgetNoteRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "advisory token budget note is too long")}
	}
	return TokenBudgetNoteAccepted{Value: TokenBudgetNote{value: trimmed}}
}

func (note TokenBudgetNote) String() string {
	return note.value
}

// TaskWorkerOrigin adapts the credential into the task package's worker
// origin: a task-scoped credential works within its already-granted
// reservation, an unscoped one carries its work-policy snapshot for the task
// service to enforce.
func (credential Credential) TaskWorkerOrigin() task.WorkerOrigin {
	if credential.TaskID != nil {
		return task.WorkerViaTaskCredential{CredentialID: credential.ID}
	}
	return task.WorkerViaCredential{CredentialID: credential.ID, Policy: credential.workPolicySnapshot()}
}

func (credential Credential) workPolicySnapshot() task.CredentialWorkPolicy {
	enabled, isEnabled := credential.WorkPolicy.(WorkPolicyEnabled)
	if !isEnabled {
		return task.CredentialWorkDisabled{}
	}
	snapshot := task.CredentialWorkBudget{
		DailyTaskLimit: enabled.Allowances.MaxTasksPerDay.Int64(),
		Concurrent:     task.NoReservationConcurrencyCap{},
		Types:          task.AnyWorkTaskType{},
		RewardFloor:    task.NoWorkRewardFloor{},
	}
	if capped, matched := enabled.Allowances.ConcurrentReservations.(ConcurrentReservationsCapped); matched {
		snapshot.Concurrent = task.ReservationConcurrencyCapAtMost{Limit: capped.Limit.Int64()}
	}
	if limited, matched := enabled.Allowances.TaskTypes.(TaskTypesLimited); matched {
		snapshot.Types = task.OnlyWorkTaskTypes{Values: limited.Values()}
	}
	if floor, matched := enabled.Allowances.RewardFloor.(RewardFloorAtLeast); matched {
		snapshot.RewardFloor = task.WorkRewardFloorAtLeast{MinimumCredits: floor.Minimum.Int64()}
	}
	return snapshot
}

// SpendOrigin adapts the credential into the ledger package's spend origin.
// Spending (funding a task, tipping, sending credits) is asking for work
// rather than seeking it, so the work-seeking state does not gate it: a
// disabled policy, or an enabled one without a spend cap, spends without a
// daily cap (the credential's scopes still bound what it can reach). Only an
// enabled policy with a configured daily spend cap is charged.
func (credential Credential) SpendOrigin() ledger.SpendOrigin {
	origin := ledger.SpendViaWorkCredential{CredentialID: credential.ID, Cap: ledger.NoSpendDayCap{}}
	if enabled, isEnabled := credential.WorkPolicy.(WorkPolicyEnabled); isEnabled {
		if capped, matched := enabled.Allowances.DailySpend.(DailySpendCapped); matched {
			origin.Cap = ledger.SpendDayCapAtMost{Limit: capped.Limit}
		}
	}
	return origin
}
