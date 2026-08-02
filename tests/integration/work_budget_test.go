//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/event/eventtest"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Day-counter key prefixes as stored in work_day_counters; pinned here so a
// drifting key format breaks a test instead of silently orphaning budgets.
const (
	workBudgetTaskKeyPrefix  = "agent_tasks:"
	workBudgetSpendKeyPrefix = "agent_spend:"
	workBudgetPeerKeyPrefix  = "peer_send:"
)

func newWorkBudgetTaskService(pool *pgxpool.Pool) task.Service {
	return task.NewService(db.NewTaskStore(pool), org.NewService(db.NewOrgStore(pool)), nil, eventtest.NewRecorder())
}

func mintAgentCredential(t *testing.T, pool *pgxpool.Pool, owner core.UserID) agent.Credential {
	t.Helper()
	service := agent.NewService(db.NewAgentStore(pool))
	label, matched := agent.NewLabel("Budget test agent").(agent.LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	scopes := agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead, agent.ScopeSubmissionsWrite, agent.ScopeSubmissionsRead, agent.ScopeLedgerWrite})
	created, createdMatched := service.Create(context.Background(), owner, label.Value, scopes, nil, nil).(agent.CredentialCreated)
	if !createdMatched {
		t.Fatalf("create credential rejected")
	}
	return created.Value
}

func configureWorkPolicy(t *testing.T, pool *pgxpool.Pool, owner core.UserID, id core.AgentCredentialID, policy agent.WorkPolicy) agent.Credential {
	t.Helper()
	service := agent.NewService(db.NewAgentStore(pool))
	result := service.ConfigureWorkPolicy(context.Background(), owner, id, policy)
	configured, matched := result.(agent.WorkPolicyConfigured)
	if !matched {
		t.Fatalf("configure work policy rejected: %s", result.(agent.ConfigureWorkPolicyRejected).Reason.Description())
	}
	return configured.Value
}

func workAllowances(t *testing.T, dailyTasks int64) agent.WorkAllowances {
	t.Helper()
	budget, matched := agent.NewDailyTaskBudget(dailyTasks).(agent.DailyTaskBudgetAccepted)
	if !matched {
		t.Fatalf("daily task budget rejected")
	}
	return agent.WorkAllowances{
		MaxTasksPerDay:         budget.Value,
		ConcurrentReservations: agent.ConcurrentReservationsUnlimited{},
		DailySpend:             agent.DailySpendUnlimited{},
		TaskTypes:              agent.AllTaskTypesAllowed{},
		RewardFloor:            agent.NoRewardFloor{},
		TokenBudget:            agent.NoTokenBudgetAdvisory{},
	}
}

// insertOpenWorkTask creates a public open task with a chosen participation
// policy, task type, and credit reward (0 = no credit reward), accepting an
// object response with one required string field ("answer").
func insertOpenWorkTask(t *testing.T, pool *pgxpool.Pool, owner core.UserID, participation string, taskType task.TaskType, rewardCredits int64) core.TaskID {
	t.Helper()
	taskID := newTaskID(t)
	schemaJSON := `{"kind":"object","fields":[{"name":"answer","presence":"required","schema":{"kind":"string"}}]}`
	rewardKind := "none"
	var rewardAmount *int64
	if rewardCredits > 0 {
		rewardKind = "credit"
		rewardAmount = &rewardCredits
	}
	_, err := pool.Exec(context.Background(), `
		insert into tasks (id, owner_kind, user_id, title, description, task_type, reward_kind, reward_credit_amount, state, participation_policy, response_schema_json, data_payload_kind, created_by_user_id)
		values ($1, 'user', $2, 'Work budget task', 'Answer with a string.', $3, $4, $5, 'open', $6, $7::jsonb, 'none', $2)
	`, taskID.String(), owner.String(), taskType.String(), rewardKind, rewardAmount, participation, schemaJSON)
	if err != nil {
		t.Fatalf("insert work task: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		insert into task_visibility_scopes (task_id, visibility_kind, scope_key)
		values ($1, 'public', 'public')
	`, taskID.String()); err != nil {
		t.Fatalf("insert public visibility: %v", err)
	}
	return taskID
}

// shiftDayCounterToYesterday rewrites a day counter's window so the next
// consumption sees a fresh UTC day, the same way the rate-limit tests age
// bucket rows directly instead of injecting a clock.
func shiftDayCounterToYesterday(t *testing.T, pool *pgxpool.Pool, key string) {
	t.Helper()
	tag, err := pool.Exec(context.Background(), "update work_day_counters set day = '2000-01-01' where key = $1", key)
	if err != nil {
		t.Fatalf("shift day counter: %v", err)
	}
	if tag.RowsAffected() == 0 {
		t.Fatalf("no day counter row for key %q", key)
	}
}

func assertBudgetExceeded(t *testing.T, reason core.DomainError, context string) {
	t.Helper()
	if reason.Code() != core.ErrorCodeBudgetExceeded {
		t.Fatalf("%s: code = %s (%s), want budget_exceeded", context, reason.Code().String(), reason.Description())
	}
}

func TestWorkPolicyStoreRoundTrip(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "work-policy-roundtrip")
	credential := mintAgentCredential(t, pool, owner)
	if agent.WorkPolicyState(credential.WorkPolicy) != agent.WorkSeekingDisabled {
		t.Fatalf("fresh credential is not work_seeking_disabled")
	}

	allowances := workAllowances(t, 7)
	capValue, capMatched := agent.NewConcurrentReservationCap(3).(agent.ConcurrentReservationCapAccepted)
	if !capMatched {
		t.Fatalf("cap rejected")
	}
	spendCap, spendMatched := ledger.NewCreditAmount(80).(ledger.CreditAmountAccepted)
	if !spendMatched {
		t.Fatalf("spend cap rejected")
	}
	floor, floorMatched := ledger.NewCreditAmount(12).(ledger.CreditAmountAccepted)
	if !floorMatched {
		t.Fatalf("floor rejected")
	}
	tokens, tokensMatched := agent.NewTokenBudgetTokens(750_000).(agent.TokenBudgetTokensAccepted)
	if !tokensMatched {
		t.Fatalf("token budget rejected")
	}
	note, noteMatched := agent.NewTokenBudgetNote("advisory only").(agent.TokenBudgetNoteAccepted)
	if !noteMatched {
		t.Fatalf("token note rejected")
	}
	limited, limitedMatched := agent.NewTaskTypesLimited([]task.TaskType{task.TaskTypeResearch, task.TaskTypeTroubleshooting}).(agent.TaskTypesLimitedAccepted)
	if !limitedMatched {
		t.Fatalf("task type restriction rejected")
	}
	allowances.ConcurrentReservations = agent.ConcurrentReservationsCapped{Limit: capValue.Value}
	allowances.DailySpend = agent.DailySpendCapped{Limit: spendCap.Value}
	allowances.RewardFloor = agent.RewardFloorAtLeast{Minimum: floor.Value}
	allowances.TokenBudget = agent.TokenBudgetAdvised{Tokens: tokens.Value, Note: note.Value}
	allowances.TaskTypes = limited.Value

	configureWorkPolicy(t, pool, owner, credential.ID, agent.WorkPolicyEnabled{Allowances: allowances})

	store := db.NewAgentStore(pool)
	verified, verifiedMatched := store.VerifyCredential(context.Background(), agent.SecretHashFromString(mustCredentialHash(t, pool, credential.ID))).(agent.VerifyStoreFound)
	if !verifiedMatched {
		t.Fatalf("verify credential rejected")
	}
	enabled, enabledMatched := verified.Value.WorkPolicy.(agent.WorkPolicyEnabled)
	if !enabledMatched {
		t.Fatalf("verified policy = %T, want enabled", verified.Value.WorkPolicy)
	}
	if enabled.Allowances.MaxTasksPerDay.Int64() != 7 {
		t.Fatalf("daily budget = %d, want 7", enabled.Allowances.MaxTasksPerDay.Int64())
	}
	if capped, ok := enabled.Allowances.ConcurrentReservations.(agent.ConcurrentReservationsCapped); !ok || capped.Limit.Int64() != 3 {
		t.Fatalf("concurrent cap = %#v, want 3", enabled.Allowances.ConcurrentReservations)
	}
	if capped, ok := enabled.Allowances.DailySpend.(agent.DailySpendCapped); !ok || capped.Limit.Int64() != 80 {
		t.Fatalf("spend cap = %#v, want 80", enabled.Allowances.DailySpend)
	}
	if floorAllowance, ok := enabled.Allowances.RewardFloor.(agent.RewardFloorAtLeast); !ok || floorAllowance.Minimum.Int64() != 12 {
		t.Fatalf("reward floor = %#v, want 12", enabled.Allowances.RewardFloor)
	}
	if advised, ok := enabled.Allowances.TokenBudget.(agent.TokenBudgetAdvised); !ok || advised.Tokens.Int64() != 750_000 || advised.Note.String() != "advisory only" {
		t.Fatalf("token budget = %#v, want advisory 750000 / note", enabled.Allowances.TokenBudget)
	}
	if types, ok := enabled.Allowances.TaskTypes.(agent.TaskTypesLimited); !ok || len(types.Values()) != 2 || !types.Allows(task.TaskTypeTroubleshooting) {
		t.Fatalf("task types = %#v, want research+troubleshooting", enabled.Allowances.TaskTypes)
	}

	// Disabling drops every allowance.
	disabled := configureWorkPolicy(t, pool, owner, credential.ID, agent.WorkPolicyDisabled{})
	if agent.WorkPolicyState(disabled.WorkPolicy) != agent.WorkSeekingDisabled {
		t.Fatalf("policy did not disable")
	}
	var taskTypeRows int
	if err := pool.QueryRow(context.Background(), "select count(*) from agent_credential_work_task_types where credential_id = $1", credential.ID.String()).Scan(&taskTypeRows); err != nil {
		t.Fatalf("count work task types: %v", err)
	}
	if taskTypeRows != 0 {
		t.Fatalf("disable left %d task type rows", taskTypeRows)
	}
}

// mustCredentialHash reads the stored token hash so the test can re-verify a
// credential without holding its plaintext secret.
func mustCredentialHash(t *testing.T, pool *pgxpool.Pool, id core.AgentCredentialID) string {
	t.Helper()
	var hash string
	if err := pool.QueryRow(context.Background(), "select token_hash from agent_credentials where id = $1", id.String()).Scan(&hash); err != nil {
		t.Fatalf("read credential hash: %v", err)
	}
	return hash
}

func TestWorkPolicyRejectsTaskScopedAndRevokedCredentials(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "work-policy-guard")
	agentService := agent.NewService(db.NewAgentStore(pool))
	taskID := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeGeneral, 0)

	label, _ := agent.NewLabel("Task-scoped token").(agent.LabelAccepted)
	scopes := agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead})
	scoped, scopedMatched := agentService.Create(context.Background(), owner, label.Value, scopes, nil, &taskID).(agent.CredentialCreated)
	if !scopedMatched {
		t.Fatalf("create task-scoped credential rejected")
	}
	policy := agent.WorkPolicyEnabled{Allowances: workAllowances(t, 1)}
	if _, rejected := agentService.ConfigureWorkPolicy(context.Background(), owner, scoped.Value.ID, policy).(agent.ConfigureWorkPolicyRejected); !rejected {
		t.Fatalf("task-scoped credential accepted a work policy")
	}

	revokable := mintAgentCredential(t, pool, owner)
	if _, revoked := agentService.Revoke(context.Background(), owner, revokable.ID).(agent.CredentialRevoked); !revoked {
		t.Fatalf("revoke rejected")
	}
	if _, rejected := agentService.ConfigureWorkPolicy(context.Background(), owner, revokable.ID, policy).(agent.ConfigureWorkPolicyRejected); !rejected {
		t.Fatalf("revoked credential accepted a work policy")
	}
}

func TestReserveDefaultDenyForFreshCredential(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "budget-deny-owner")
	worker := createUser(t, pool, "budget-deny-worker")
	taskID := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeGeneral, 0)
	credential := mintAgentCredential(t, pool, worker)

	service := newWorkBudgetTaskService(pool)
	result := service.Reserve(context.Background(), testWorkerSubject(worker), credential.TaskWorkerOrigin(), taskID)
	rejected, matched := result.(task.ReservationRejected)
	if !matched {
		t.Fatalf("fresh credential reserved a task: %T", result)
	}
	if rejected.Reason.Code() != core.ErrorCodePermissionDenied {
		t.Fatalf("code = %s, want permission_denied", rejected.Reason.Code().String())
	}

	// The same worker acting through their session reserves without a gate.
	if _, ok := service.Reserve(context.Background(), testWorkerSubject(worker), task.WorkerIsUser{}, taskID).(task.ReservationCreated); !ok {
		t.Fatalf("user session reserve was refused")
	}
}

func testWorkerSubject(id core.UserID) auth.UserSubject {
	return auth.UserSubject{ID: id}
}

// createOrganizationFor creates an organization owned by the given user
// through the real org store (which writes the account and, only for a
// verified creator, the grant).
func createOrganizationFor(t *testing.T, pool *pgxpool.Pool, createdBy core.UserID, name string) core.OrganizationID {
	t.Helper()
	organizationID, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	membershipID, membershipMatched := core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated)
	if !membershipMatched {
		t.Fatalf("membership id rejected")
	}
	orgName, nameMatched := org.NewOrganizationName(name).(org.OrganizationNameAccepted)
	if !nameMatched {
		t.Fatalf("organization name rejected")
	}
	result := db.NewOrgStore(pool).CreateOrganization(context.Background(), organizationID.Value, orgName.Value, createdBy, membershipID.Value)
	if _, accepted := result.(org.CreateOrganizationStoreAccepted); !accepted {
		t.Fatalf("create organization rejected: %#v", result)
	}
	return organizationID.Value
}

func mustOrganizationBalance(t *testing.T, store db.LedgerStore, organizationID core.OrganizationID) ledger.Balance {
	t.Helper()
	result := store.OrganizationBalance(context.Background(), organizationID)
	found, matched := result.(ledger.BalanceFound)
	if !matched {
		t.Fatalf("organization balance rejected")
	}
	return found.Value
}

func TestDailyTaskBudgetBoundaryAndReset(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "budget-day-owner")
	worker := createUser(t, pool, "budget-day-worker")
	credential := mintAgentCredential(t, pool, worker)
	enabled := configureWorkPolicy(t, pool, worker, credential.ID, agent.WorkPolicyEnabled{Allowances: workAllowances(t, 5)})
	origin := enabled.TaskWorkerOrigin()
	service := newWorkBudgetTaskService(pool)

	// The 5th of 5 succeeds; the 6th is refused with budget_exceeded.
	for index := 0; index < 5; index++ {
		taskID := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeGeneral, 0)
		if result := service.Reserve(context.Background(), testWorkerSubject(worker), origin, taskID); result == nil {
			t.Fatalf("reserve %d returned nil", index)
		} else if _, ok := result.(task.ReservationCreated); !ok {
			t.Fatalf("reserve %d refused: %s", index+1, result.(task.ReservationRejected).Reason.Description())
		}
	}
	sixthTask := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeGeneral, 0)
	refused, refusedMatched := service.Reserve(context.Background(), testWorkerSubject(worker), origin, sixthTask).(task.ReservationRejected)
	if !refusedMatched {
		t.Fatalf("6th reserve was not refused")
	}
	assertBudgetExceeded(t, refused.Reason, "6th reserve")

	// A new UTC day resets the window.
	shiftDayCounterToYesterday(t, pool, workBudgetTaskKeyPrefix+credential.ID.String())
	if _, ok := service.Reserve(context.Background(), testWorkerSubject(worker), origin, sixthTask).(task.ReservationCreated); !ok {
		t.Fatalf("reserve after day reset was refused")
	}
}

func TestConcurrentReservationCap(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "budget-concurrent-owner")
	worker := createUser(t, pool, "budget-concurrent-worker")
	credential := mintAgentCredential(t, pool, worker)
	allowances := workAllowances(t, 10)
	capValue, capMatched := agent.NewConcurrentReservationCap(1).(agent.ConcurrentReservationCapAccepted)
	if !capMatched {
		t.Fatalf("cap rejected")
	}
	allowances.ConcurrentReservations = agent.ConcurrentReservationsCapped{Limit: capValue.Value}
	enabled := configureWorkPolicy(t, pool, worker, credential.ID, agent.WorkPolicyEnabled{Allowances: allowances})
	origin := enabled.TaskWorkerOrigin()
	service := newWorkBudgetTaskService(pool)

	firstTask := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeGeneral, 0)
	secondTask := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeGeneral, 0)
	first, firstMatched := service.Reserve(context.Background(), testWorkerSubject(worker), origin, firstTask).(task.ReservationCreated)
	if !firstMatched {
		t.Fatalf("first reserve refused")
	}
	refused, refusedMatched := service.Reserve(context.Background(), testWorkerSubject(worker), origin, secondTask).(task.ReservationRejected)
	if !refusedMatched {
		t.Fatalf("second concurrent reserve was not refused")
	}
	assertBudgetExceeded(t, refused.Reason, "second concurrent reserve")

	// Releasing the held reservation frees the slot.
	taskStore := db.NewTaskStore(pool)
	if _, ok := taskStore.ChangeReservationState(context.Background(), firstTask, first.Value.ID, task.ReservationStateCancelledByWorker, event.NoEvent{}).(task.ChangeReservationStateStoreAccepted); !ok {
		t.Fatalf("cancel first reservation rejected")
	}
	if _, ok := service.Reserve(context.Background(), testWorkerSubject(worker), origin, secondTask).(task.ReservationCreated); !ok {
		t.Fatalf("reserve after release was refused")
	}
}

func TestTypeRestrictionAndRewardFloorFilters(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "budget-filter-owner")
	worker := createUser(t, pool, "budget-filter-worker")
	credential := mintAgentCredential(t, pool, worker)
	allowances := workAllowances(t, 10)
	limited, limitedMatched := agent.NewTaskTypesLimited([]task.TaskType{task.TaskTypeResearch}).(agent.TaskTypesLimitedAccepted)
	if !limitedMatched {
		t.Fatalf("restriction rejected")
	}
	floor, floorMatched := ledger.NewCreditAmount(25).(ledger.CreditAmountAccepted)
	if !floorMatched {
		t.Fatalf("floor rejected")
	}
	allowances.TaskTypes = limited.Value
	allowances.RewardFloor = agent.RewardFloorAtLeast{Minimum: floor.Value}
	enabled := configureWorkPolicy(t, pool, worker, credential.ID, agent.WorkPolicyEnabled{Allowances: allowances})
	origin := enabled.TaskWorkerOrigin()
	service := newWorkBudgetTaskService(pool)

	wrongType := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeCodeReview, 50)
	if rejected, ok := service.Reserve(context.Background(), testWorkerSubject(worker), origin, wrongType).(task.ReservationRejected); !ok || rejected.Reason.Code() != core.ErrorCodePermissionDenied {
		t.Fatalf("disallowed task type was not refused with permission_denied")
	}
	lowReward := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeResearch, 20)
	if rejected, ok := service.Reserve(context.Background(), testWorkerSubject(worker), origin, lowReward).(task.ReservationRejected); !ok || rejected.Reason.Code() != core.ErrorCodePermissionDenied {
		t.Fatalf("below-floor task was not refused with permission_denied")
	}
	matching := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeResearch, 25)
	if _, ok := service.Reserve(context.Background(), testWorkerSubject(worker), origin, matching).(task.ReservationCreated); !ok {
		t.Fatalf("matching task was refused")
	}
}

func TestDirectSubmissionConsumesDailyTaskBudget(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "budget-submit-owner")
	worker := createUser(t, pool, "budget-submit-worker")
	credential := mintAgentCredential(t, pool, worker)
	enabled := configureWorkPolicy(t, pool, worker, credential.ID, agent.WorkPolicyEnabled{Allowances: workAllowances(t, 1)})
	origin := enabled.TaskWorkerOrigin()

	taskStore := db.NewTaskStore(pool)
	service := submission.NewService(db.NewSubmissionStore(pool), taskStore, nil, eventtest.NewRecorder())
	firstTask := insertOpenWorkTask(t, pool, owner, "open", task.TaskTypeGeneral, 0)
	secondTask := insertOpenWorkTask(t, pool, owner, "open", task.TaskTypeGeneral, 0)

	source, sourceMatched := submission.NewResponseSource(`{"answer":"done"}`).(submission.ResponseSourceAccepted)
	if !sourceMatched {
		t.Fatalf("response source rejected")
	}
	first := service.Submit(context.Background(), origin, submission.SubmitCommand{TaskID: firstTask, SubmitterID: worker, ResponseSource: source.Value})
	if _, ok := first.(submission.SubmissionCreated); !ok {
		t.Fatalf("first direct submission refused: %#v", first)
	}
	second := service.Submit(context.Background(), origin, submission.SubmitCommand{TaskID: secondTask, SubmitterID: worker, ResponseSource: source.Value})
	refused, refusedMatched := second.(submission.SubmitRejected)
	if !refusedMatched {
		t.Fatalf("second direct submission was not refused")
	}
	assertBudgetExceeded(t, refused.Reason, "second direct submission")

	// The same submission through the user's own session is never budgeted.
	sessionSubmit := service.Submit(context.Background(), task.WorkerIsUser{}, submission.SubmitCommand{TaskID: secondTask, SubmitterID: worker, ResponseSource: source.Value})
	if _, ok := sessionSubmit.(submission.SubmissionCreated); !ok {
		t.Fatalf("user session submission was refused: %#v", sessionSubmit)
	}
}

func TestSpendCapAcrossFundTipAndSend(t *testing.T) {
	pool := newPool(t)
	spender := createUser(t, pool, "budget-spend-owner")
	worker := createUser(t, pool, "budget-spend-worker")
	receiver := createUser(t, pool, "budget-spend-receiver")
	credential := mintAgentCredential(t, pool, spender)
	allowances := workAllowances(t, 10)
	capAmount, capMatched := ledger.NewCreditAmount(50).(ledger.CreditAmountAccepted)
	if !capMatched {
		t.Fatalf("cap rejected")
	}
	allowances.DailySpend = agent.DailySpendCapped{Limit: capAmount.Value}
	enabled := configureWorkPolicy(t, pool, spender, credential.ID, agent.WorkPolicyEnabled{Allowances: allowances})
	spendOrigin := enabled.SpendOrigin()

	store := db.NewLedgerStore(pool)
	taskID := insertTask(t, pool, spender, "draft", 20)

	// Fund 20 under the credential (store command carries the charge exactly
	// as the service builds it).
	fund := fundCommand(t, spender, taskID, 20, "budget-fund-"+taskID.String())
	fund.Spend = ledger.ChargeSpendBudget{CredentialID: credential.ID, DayLimit: capAmount.Value, Amount: creditAmount(t, 20)}
	if _, ok := store.FundTask(context.Background(), fund).(ledger.TaskFunded); !ok {
		t.Fatalf("capped fund of 20 was refused")
	}

	// Tip 20 on accept under the credential: 40 of 50 spent.
	setTaskState(t, pool, taskID, "open")
	submissionID := insertSubmission(t, pool, taskID, worker)
	accept := acceptCommand(t, spender, taskID, submissionID, "budget-accept-"+submissionID.String())
	accept.TipSelection = ledger.CreditTipSelection{Amount: creditAmount(t, 20)}
	accept.Spend = ledger.ChargeSpendBudget{CredentialID: credential.ID, DayLimit: capAmount.Value, Amount: creditAmount(t, 20)}
	if result := store.AcceptSubmission(context.Background(), accept); result == nil {
		t.Fatalf("accept returned nil")
	} else if _, ok := result.(ledger.SubmissionAccepted); !ok {
		t.Fatalf("capped tip of 20 was refused: %#v", result)
	}

	// A 15-credit send would take the day to 55 of 50: refused through the
	// ledger service with the credential's spend origin.
	service := newDBLedgerService(pool)
	refused, refusedMatched := service.SendCredits(context.Background(), spender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		creditAmount(t, 15), ledger.NoTransferNote{}, idempotencyKey(t, "budget-send-over-"+credential.ID.String()), spendOrigin).(ledger.SendRejected)
	if !refusedMatched {
		t.Fatalf("over-cap send was not refused")
	}
	assertBudgetExceeded(t, refused.Reason, "over-cap send")

	// A 10-credit send lands exactly on the cap and succeeds.
	if _, ok := service.SendCredits(context.Background(), spender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		creditAmount(t, 10), ledger.NoTransferNote{}, idempotencyKey(t, "budget-send-exact-"+credential.ID.String()), spendOrigin).(ledger.CreditsSent); !ok {
		t.Fatalf("at-cap send was refused")
	}
}

func TestPeerTransferVelocityCeiling(t *testing.T) {
	pool := newPool(t)
	sender := createUser(t, pool, "velocity-sender")
	receiver := createUser(t, pool, "velocity-receiver")
	admin := createUser(t, pool, "velocity-admin")
	service := newDBLedgerService(pool)

	// An admin grant of 600 credits is exempt from the velocity ceiling.
	grantNote, _ := ledger.NewGrantNote("velocity test bankroll").(ledger.GrantNoteAccepted)
	if result := service.GrantCredits(context.Background(), admin, ledger.GrantToUser{ID: sender}, creditAmount(t, 600), grantNote.Value, idempotencyKey(t, "velocity-grant-"+sender.String())); result == nil {
		t.Fatalf("grant returned nil")
	} else if _, ok := result.(ledger.CreditsGranted); !ok {
		t.Fatalf("600-credit admin grant was refused: %#v", result)
	}

	// 450 of the 500/day ceiling.
	if _, ok := service.SendCredits(context.Background(), sender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		creditAmount(t, 450), ledger.NoTransferNote{}, idempotencyKey(t, "velocity-send-1-"+sender.String()), ledger.SpendByUser{}).(ledger.CreditsSent); !ok {
		t.Fatalf("450-credit send was refused")
	}
	// 100 more would cross the ceiling.
	refused, refusedMatched := service.SendCredits(context.Background(), sender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		creditAmount(t, 100), ledger.NoTransferNote{}, idempotencyKey(t, "velocity-send-2-"+sender.String()), ledger.SpendByUser{}).(ledger.SendRejected)
	if !refusedMatched {
		t.Fatalf("over-ceiling send was not refused")
	}
	assertBudgetExceeded(t, refused.Reason, "over-ceiling send")

	// 50 still fits under the ceiling today; a new UTC day fully resets it.
	if _, ok := service.SendCredits(context.Background(), sender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		creditAmount(t, 50), ledger.NoTransferNote{}, idempotencyKey(t, "velocity-send-3-"+sender.String()), ledger.SpendByUser{}).(ledger.CreditsSent); !ok {
		t.Fatalf("under-ceiling send was refused")
	}
	var senderAccount string
	if err := pool.QueryRow(context.Background(), "select id::text from credit_accounts where owner_kind = 'user' and user_id = $1", sender.String()).Scan(&senderAccount); err != nil {
		t.Fatalf("read sender account: %v", err)
	}
	shiftDayCounterToYesterday(t, pool, workBudgetPeerKeyPrefix+senderAccount)
	if _, ok := service.SendCredits(context.Background(), sender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		creditAmount(t, 100), ledger.NoTransferNote{}, idempotencyKey(t, "velocity-send-4-"+sender.String()), ledger.SpendByUser{}).(ledger.CreditsSent); !ok {
		t.Fatalf("send after day reset was refused")
	}
}

func TestVerificationGatedSignupGrant(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)

	// Registration alone yields an empty account.
	unverified := createUnverifiedUser(t, pool, "grant-gate")
	if balance := mustBalance(t, store, unverified); balance.Spendable() != 0 {
		t.Fatalf("unverified balance = %d, want 0", balance.Spendable())
	}

	// An organization created by the unverified user gets an account, no grant.
	organizationID := createOrganizationFor(t, pool, unverified, "Unverified Org")
	if balance := mustOrganizationBalance(t, store, organizationID); balance.Spendable() != 0 {
		t.Fatalf("unverified creator's org balance = %d, want 0", balance.Spendable())
	}

	// First verification lands the grant; re-verifying is idempotent.
	verifyUserEmail(t, pool, unverified)
	if balance := mustBalance(t, store, unverified); balance.Spendable() != 100 {
		t.Fatalf("verified balance = %d, want 100", balance.Spendable())
	}
	verifyUserEmail(t, pool, unverified)
	if balance := mustBalance(t, store, unverified); balance.Spendable() != 100 {
		t.Fatalf("re-verified balance = %d, want exactly 100", balance.Spendable())
	}

	// An organization created by a now-verified user gets its grant.
	verifiedOrganizationID := createOrganizationFor(t, pool, unverified, "Verified Org")
	if balance := mustOrganizationBalance(t, store, verifiedOrganizationID); balance.Spendable() != 100 {
		t.Fatalf("verified creator's org balance = %d, want 100", balance.Spendable())
	}
}

func TestOpsCountersReadModel(t *testing.T) {
	pool := newPool(t)
	counters := db.NewOpsCountersStoreFromHandle(db.NewPGX(pool))

	before, beforeMatched := counters.Count(context.Background()).(db.OpsCountersCounted)
	if !beforeMatched {
		t.Fatalf("ops counters rejected")
	}

	// One fresh signup grant, one peer transfer, one budget refusal.
	sender := createUser(t, pool, "ops-sender")
	receiver := createUser(t, pool, "ops-receiver")
	service := newDBLedgerService(pool)
	if _, ok := service.SendCredits(context.Background(), sender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		creditAmount(t, 5), ledger.NoTransferNote{}, idempotencyKey(t, "ops-send-"+sender.String()), ledger.SpendByUser{}).(ledger.CreditsSent); !ok {
		t.Fatalf("ops peer send refused")
	}

	worker := createUser(t, pool, "ops-worker")
	owner := createUser(t, pool, "ops-owner")
	credential := mintAgentCredential(t, pool, worker)
	enabled := configureWorkPolicy(t, pool, worker, credential.ID, agent.WorkPolicyEnabled{Allowances: workAllowances(t, 1)})
	taskService := newWorkBudgetTaskService(pool)
	firstTask := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeGeneral, 0)
	secondTask := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeGeneral, 0)
	if _, ok := taskService.Reserve(context.Background(), testWorkerSubject(worker), enabled.TaskWorkerOrigin(), firstTask).(task.ReservationCreated); !ok {
		t.Fatalf("ops first reserve refused")
	}
	if _, ok := taskService.Reserve(context.Background(), testWorkerSubject(worker), enabled.TaskWorkerOrigin(), secondTask).(task.ReservationRejected); !ok {
		t.Fatalf("ops second reserve was not refused")
	}

	after, afterMatched := counters.Count(context.Background()).(db.OpsCountersCounted)
	if !afterMatched {
		t.Fatalf("ops counters rejected after activity")
	}
	// createUser verifies 4 accounts above, each landing a signup grant.
	if after.Value.DaySignupGrants < before.Value.DaySignupGrants+4 {
		t.Fatalf("day signup grants = %d, want at least %d", after.Value.DaySignupGrants, before.Value.DaySignupGrants+4)
	}
	if after.Value.DayPeerTransfers < before.Value.DayPeerTransfers+1 {
		t.Fatalf("day peer transfers = %d, want at least %d", after.Value.DayPeerTransfers, before.Value.DayPeerTransfers+1)
	}
	if after.Value.DayPeerTransferCredits < before.Value.DayPeerTransferCredits+5 {
		t.Fatalf("day peer transfer credits = %d, want at least %d", after.Value.DayPeerTransferCredits, before.Value.DayPeerTransferCredits+5)
	}
	if after.Value.DayBudgetRefusals < before.Value.DayBudgetRefusals+1 {
		t.Fatalf("day budget refusals = %d, want at least %d", after.Value.DayBudgetRefusals, before.Value.DayBudgetRefusals+1)
	}
	if after.Value.OutboxRecordedBacklog < 0 || after.Value.OutboxDispatchFailed < 0 || after.Value.WebhookDeliveriesPending < 0 || after.Value.WebhookDeliveriesDead < 0 {
		t.Fatalf("negative aggregate in %+v", after.Value)
	}
	switch after.Value.OldestPending.(type) {
	case db.NoPendingWebhookDeliveries, db.OldestPendingWebhookWaiting:
	default:
		t.Fatalf("oldest pending = %T, want a declared variant", after.Value.OldestPending)
	}
}

// TestReadWorkActivityPerCredential pins the consumption read model behind
// credential listings: one owner query returns, per credential, today's
// consumed daily-task units, today's spent credits, and the still-active
// reservations attributed to the credential — and leaves untouched
// credentials at zero.
func TestReadWorkActivityPerCredential(t *testing.T) {
	pool := newPool(t)
	worker := createUser(t, pool, "activity-worker")
	owner := createUser(t, pool, "activity-owner")
	receiver := createUser(t, pool, "activity-receiver")

	active := mintAgentCredential(t, pool, worker)
	idle := mintAgentCredential(t, pool, worker)
	allowances := workAllowances(t, 10)
	capAmount, capMatched := ledger.NewCreditAmount(50).(ledger.CreditAmountAccepted)
	if !capMatched {
		t.Fatalf("cap rejected")
	}
	allowances.DailySpend = agent.DailySpendCapped{Limit: capAmount.Value}
	enabled := configureWorkPolicy(t, pool, worker, active.ID, agent.WorkPolicyEnabled{Allowances: allowances})

	// One reservation consumes a daily-task unit and stays active.
	taskService := newWorkBudgetTaskService(pool)
	taskID := insertOpenWorkTask(t, pool, owner, "reservation_required", task.TaskTypeGeneral, 0)
	if _, ok := taskService.Reserve(context.Background(), testWorkerSubject(worker), enabled.TaskWorkerOrigin(), taskID).(task.ReservationCreated); !ok {
		t.Fatalf("activity reserve refused")
	}
	// A 10-credit send through the credential consumes the spend budget.
	if _, ok := newDBLedgerService(pool).SendCredits(context.Background(), worker, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		creditAmount(t, 10), ledger.NoTransferNote{}, idempotencyKey(t, "activity-send-"+active.ID.String()), enabled.SpendOrigin()).(ledger.CreditsSent); !ok {
		t.Fatalf("activity send refused")
	}

	store := db.NewAgentStore(pool)
	listed, listedMatched := store.ReadWorkActivity(context.Background(), worker).(agent.WorkActivityStoreListed)
	if !listedMatched {
		t.Fatalf("read work activity rejected")
	}
	byID := func(id core.AgentCredentialID) agent.CredentialWorkActivity {
		for _, value := range listed.Values {
			if value.CredentialID == id {
				return value
			}
		}
		t.Fatalf("credential %s missing from activity read", id)
		return agent.CredentialWorkActivity{}
	}
	got := byID(active.ID)
	if got.TasksToday != 1 || got.CreditsSpentToday != 10 || got.ActiveReservations != 1 {
		t.Fatalf("active credential activity = %+v, want tasks 1, spend 10, reservations 1", got)
	}
	untouched := byID(idle.ID)
	if untouched.TasksToday != 0 || untouched.CreditsSpentToday != 0 || untouched.ActiveReservations != 0 {
		t.Fatalf("idle credential activity = %+v, want zeros", untouched)
	}

	// A fresh UTC day resets the day totals; the reservation stays active.
	shiftDayCounterToYesterday(t, pool, workBudgetTaskKeyPrefix+active.ID.String())
	shiftDayCounterToYesterday(t, pool, workBudgetSpendKeyPrefix+active.ID.String())
	reset, resetMatched := store.ReadWorkActivity(context.Background(), worker).(agent.WorkActivityStoreListed)
	if !resetMatched {
		t.Fatalf("read work activity after reset rejected")
	}
	listed = reset
	got = byID(active.ID)
	if got.TasksToday != 0 || got.CreditsSpentToday != 0 || got.ActiveReservations != 1 {
		t.Fatalf("post-reset activity = %+v, want tasks 0, spend 0, reservations 1", got)
	}
}
