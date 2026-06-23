//go:build integration

package integration_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSignupGrantPersistsBalance(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "integration-signup")

	store := db.NewLedgerStore(pool)
	balance := mustBalance(t, store, owner)
	if balance.Int64() != 100 {
		t.Fatalf("signup balance = %d, want 100", balance.Int64())
	}

	listed, matched := store.ListEntries(context.Background(), owner, core.DefaultPage()).(ledger.EntriesListed)
	if !matched {
		t.Fatalf("list entries was rejected")
	}
	if len(listed.Values) != 1 {
		t.Fatalf("entry count = %d, want 1", len(listed.Values))
	}
	if listed.Values[0].Kind != ledger.EntryKindSignupGrant {
		t.Fatalf("entry kind = %q, want signup_grant", listed.Values[0].Kind.String())
	}
}

func TestFundAcceptRefundPersist(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)

	owner := createUser(t, pool, "integration-owner")
	worker := createUser(t, pool, "integration-worker")

	taskID := insertTask(t, pool, owner, "draft", 40)

	fundResult := store.FundTask(context.Background(), fundCommand(t, owner, taskID, 40, "fund-"+taskID.String()))
	if _, matched := fundResult.(ledger.TaskFunded); !matched {
		t.Fatalf("fund result = %T, want TaskFunded", fundResult)
	}
	if balance := mustBalance(t, store, owner); balance.Int64() != 60 {
		t.Fatalf("owner balance after funding = %d, want 60", balance.Int64())
	}

	// Funding the same task twice is rejected (single escrow per task).
	if _, matched := store.FundTask(context.Background(), fundCommand(t, owner, taskID, 10, "fund-again-"+taskID.String())).(ledger.FundRejected); !matched {
		t.Fatalf("second funding was not rejected")
	}

	setTaskState(t, pool, taskID, "open")
	submissionID := insertSubmission(t, pool, taskID, worker)

	acceptResult := store.AcceptSubmission(context.Background(), acceptCommand(t, owner, taskID, submissionID, "accept-"+submissionID.String()))
	accepted, matched := acceptResult.(ledger.SubmissionAccepted)
	if !matched {
		t.Fatalf("accept result = %T, want SubmissionAccepted", acceptResult)
	}
	if _, paid := accepted.Payout.(ledger.CreditPayout); !paid {
		t.Fatalf("payout = %T, want CreditPayout", accepted.Payout)
	}
	if balance := mustBalance(t, store, worker); balance.Int64() != 140 {
		t.Fatalf("worker balance after payout = %d, want 140", balance.Int64())
	}

	// Idempotent re-accept does not pay twice.
	store.AcceptSubmission(context.Background(), acceptCommand(t, owner, taskID, submissionID, "accept-"+submissionID.String()))
	if balance := mustBalance(t, store, worker); balance.Int64() != 140 {
		t.Fatalf("worker balance after idempotent accept = %d, want 140", balance.Int64())
	}

	// A second funded task refunds back to the owner.
	refundTaskID := insertTask(t, pool, owner, "draft", 20)
	store.FundTask(context.Background(), fundCommand(t, owner, refundTaskID, 20, "fund-"+refundTaskID.String()))
	refundResult := store.RefundTask(context.Background(), refundCommand(t, owner, refundTaskID, "refund-"+refundTaskID.String()))
	if _, matched := refundResult.(ledger.TaskRefunded); !matched {
		t.Fatalf("refund result = %T, want TaskRefunded", refundResult)
	}
	if balance := mustBalance(t, store, owner); balance.Int64() != 60 {
		t.Fatalf("owner balance after refund = %d, want 60", balance.Int64())
	}
}

func TestReviewAcceptCanPayPartialEscrowAndTip(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)

	owner := createUser(t, pool, "integration-review-owner")
	worker := createUser(t, pool, "integration-review-worker")
	taskID := insertTask(t, pool, owner, "draft", 40)
	store.FundTask(context.Background(), fundCommand(t, owner, taskID, 40, "fund-"+taskID.String()))
	setTaskState(t, pool, taskID, "open")
	submissionID := insertSubmission(t, pool, taskID, worker)

	command := acceptCommand(t, owner, taskID, submissionID, "accept-partial-"+submissionID.String())
	command.CreditSelection = ledger.PartialCreditReviewSelection{Amount: creditAmount(t, 25)}
	command.TipSelection = ledger.CreditTipSelection{Amount: creditAmount(t, 5)}

	result := store.AcceptSubmission(context.Background(), command)
	accepted, matched := result.(ledger.SubmissionAccepted)
	if !matched {
		t.Fatalf("accept result = %T (%s), want SubmissionAccepted", result, result.(ledger.AcceptRejected).Reason.Description())
	}
	payout, paid := accepted.Payout.(ledger.CreditPayout)
	if !paid || payout.Amount.Int64() != 25 {
		t.Fatalf("payout = %#v, want 25 credit payout", accepted.Payout)
	}
	tip, tipped := accepted.Tip.(ledger.CreditTip)
	if !tipped || tip.Amount.Int64() != 5 {
		t.Fatalf("tip = %#v, want 5 credit tip", accepted.Tip)
	}
	if balance := mustBalance(t, store, owner); balance.Int64() != 70 {
		t.Fatalf("owner balance = %d, want 70", balance.Int64())
	}
	if balance := mustBalance(t, store, worker); balance.Int64() != 130 {
		t.Fatalf("worker balance = %d, want 130", balance.Int64())
	}
}

func TestRequestChangesStoresNoteAndReactivatesReservation(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)

	owner := createUser(t, pool, "integration-changes-owner")
	worker := createUser(t, pool, "integration-changes-worker")
	taskID := insertTask(t, pool, owner, "open", 20)
	submissionID := insertSubmission(t, pool, taskID, worker)
	insertSubmittedReservation(t, pool, taskID, worker)
	note := reviewNote(t, "Use the latest endpoint response.")

	result := store.RequestChanges(context.Background(), ledger.RequestChangesStoreCommand{
		RequesterUserID: owner,
		TaskID:          taskID,
		SubmissionID:    submissionID,
		ReviewNote:      note,
	})
	if _, matched := result.(ledger.ChangesRequested); !matched {
		t.Fatalf("request changes result = %T, want ChangesRequested", result)
	}

	var submissionState string
	var storedNote string
	var reservationState string
	if err := pool.QueryRow(context.Background(), "select state, review_note from submissions where id = $1", submissionID.String()).Scan(&submissionState, &storedNote); err != nil {
		t.Fatalf("read submission review state: %v", err)
	}
	if err := pool.QueryRow(context.Background(), "select state from task_reservations where task_id = $1 and user_id = $2", taskID.String(), worker.String()).Scan(&reservationState); err != nil {
		t.Fatalf("read reservation state: %v", err)
	}
	if submissionState != "changes_requested" || storedNote != note.String() || reservationState != "active" {
		t.Fatalf("state/note/reservation = %q/%q/%q", submissionState, storedNote, reservationState)
	}
}

func TestRejectCanPayPartialTipAndBanImplementor(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)

	owner := createUser(t, pool, "integration-reject-owner")
	worker := createUser(t, pool, "integration-reject-worker")
	taskID := insertTask(t, pool, owner, "draft", 40)
	store.FundTask(context.Background(), fundCommand(t, owner, taskID, 40, "fund-"+taskID.String()))
	setTaskState(t, pool, taskID, "open")
	submissionID := insertSubmission(t, pool, taskID, worker)
	note := reviewNote(t, "The data is stale.")

	result := store.RejectSubmission(context.Background(), ledger.RejectStoreCommand{
		PayoutEntryID:    newEntryID(t),
		TipDebitEntryID:  newEntryID(t),
		TipCreditEntryID: newEntryID(t),
		RequesterUserID:  owner,
		TaskID:           taskID,
		SubmissionID:     submissionID,
		IdempotencyKey:   idempotencyKey(t, "reject-"+submissionID.String()),
		ReviewNote:       note,
		CreditSelection:  ledger.PartialCreditReviewSelection{Amount: creditAmount(t, 10)},
		TipSelection:     ledger.CreditTipSelection{Amount: creditAmount(t, 3)},
		BanSelection:     ledger.BanImplementorSelection{},
	})
	rejected, matched := result.(ledger.SubmissionRejected)
	if !matched {
		t.Fatalf("reject result = %T (%s), want SubmissionRejected", result, result.(ledger.RejectRejected).Reason.Description())
	}
	payout, paid := rejected.Payout.(ledger.CreditPayout)
	if !paid || payout.Amount.Int64() != 10 {
		t.Fatalf("payout = %#v, want 10 credit payout", rejected.Payout)
	}
	if balance := mustBalance(t, store, worker); balance.Int64() != 113 {
		t.Fatalf("worker balance = %d, want 113", balance.Int64())
	}
	if balance := mustBalance(t, store, owner); balance.Int64() != 57 {
		t.Fatalf("owner balance = %d, want 57", balance.Int64())
	}

	eligibility := db.NewTaskStore(pool).CheckSubmissionEligibility(context.Background(), taskID, worker)
	if _, banned := eligibility.(task.SubmissionEligibilityRejected); !banned {
		t.Fatalf("banned implementor remained eligible: %T", eligibility)
	}
}

// TestConcurrentAcceptKeepsSingleAcceptedSubmission proves that the
// transactional acceptance path keeps at most one accepted submission per task
// even when two acceptances race. Authorization and state are validated inside
// the transaction under a FOR UPDATE task-row lock, with a unique partial index
// as the final backstop, so the checks are not moved out to the service layer
// where they could drift from the write.
func TestConcurrentAcceptKeepsSingleAcceptedSubmission(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)

	owner := createUser(t, pool, "integration-concurrent-owner")
	workerA := createUser(t, pool, "integration-concurrent-worker-a")
	workerB := createUser(t, pool, "integration-concurrent-worker-b")

	taskID := insertTask(t, pool, owner, "draft", 40)
	store.FundTask(context.Background(), fundCommand(t, owner, taskID, 40, "fund-"+taskID.String()))
	setTaskState(t, pool, taskID, "open")
	submissionA := insertSubmission(t, pool, taskID, workerA)
	submissionB := insertSubmission(t, pool, taskID, workerB)

	commandA := acceptCommand(t, owner, taskID, submissionA, "accept-"+submissionA.String())
	commandB := acceptCommand(t, owner, taskID, submissionB, "accept-"+submissionB.String())

	results := make(chan ledger.AcceptResult, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, command := range []ledger.AcceptStoreCommand{commandA, commandB} {
		wg.Add(1)
		go func(command ledger.AcceptStoreCommand) {
			defer wg.Done()
			<-start
			results <- store.AcceptSubmission(context.Background(), command)
		}(command)
	}
	close(start)
	wg.Wait()
	close(results)

	accepted := 0
	rejected := 0
	for result := range results {
		switch result.(type) {
		case ledger.SubmissionAccepted:
			accepted++
		case ledger.AcceptRejected:
			rejected++
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Fatalf("accepted=%d rejected=%d, want exactly one accepted and one rejected", accepted, rejected)
	}

	var acceptedCount int
	if err := pool.QueryRow(context.Background(), "select count(*) from submissions where task_id = $1 and state = 'accepted'", taskID.String()).Scan(&acceptedCount); err != nil {
		t.Fatalf("count accepted submissions: %v", err)
	}
	if acceptedCount != 1 {
		t.Fatalf("accepted submission count = %d, want 1", acceptedCount)
	}

	var taskState string
	if err := pool.QueryRow(context.Background(), "select state from tasks where id = $1", taskID.String()).Scan(&taskState); err != nil {
		t.Fatalf("read task state: %v", err)
	}
	if taskState != "closed" {
		t.Fatalf("task state = %q, want closed", taskState)
	}

	// Exactly one worker was paid the 40-credit escrow.
	balanceA := mustBalance(t, store, workerA).Int64()
	balanceB := mustBalance(t, store, workerB).Int64()
	if balanceA+balanceB != 240 {
		t.Fatalf("worker balances = %d and %d, want one 140 and one 100", balanceA, balanceB)
	}
}

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pool, err := db.Open(ctx, requireEnv(t, "DATABASE_URL"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.MigrateUp(ctx, pool, requireEnv(t, "SHARECROP_MIGRATIONS_DIR")); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return pool
}

func createUser(t *testing.T, pool *pgxpool.Pool, prefix string) core.UserID {
	t.Helper()
	userID := newUserID(t)
	email, matched := auth.NewEmailAddress(prefix + "-" + userID.String() + "@example.com").(auth.EmailAddressAccepted)
	if !matched {
		t.Fatalf("email rejected")
	}
	secret, secretMatched := auth.NewPasswordSecret("correct horse battery staple").(auth.PasswordSecretAccepted)
	if !secretMatched {
		t.Fatalf("password secret rejected")
	}
	hash, hashMatched := auth.HashPassword(secret.Value).(auth.PasswordHashCreated)
	if !hashMatched {
		t.Fatalf("password hash rejected")
	}
	result := db.NewAuthStore(pool).CreateUserCredential(context.Background(), userID, email.Value, hash.Value)
	if _, accepted := result.(auth.StoreUserAccepted); !accepted {
		t.Fatalf("create user credential rejected")
	}
	return userID
}

func insertTask(t *testing.T, pool *pgxpool.Pool, owner core.UserID, state string, rewardAmount int64) core.TaskID {
	t.Helper()
	taskID := newTaskID(t)
	_, err := pool.Exec(context.Background(), `
		insert into tasks (id, owner_kind, user_id, title, description, reward_kind, reward_credit_amount, state, response_schema_json, data_payload_kind, created_by_user_id)
		values ($1, 'user', $2, 'Integration task', 'Integration task description', 'credit', $3, $4, '{}'::jsonb, 'none', $2)
	`, taskID.String(), owner.String(), rewardAmount, state)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}
	return taskID
}

func setTaskState(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID, state string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "update tasks set state = $2 where id = $1", taskID.String(), state); err != nil {
		t.Fatalf("set task state: %v", err)
	}
}

func insertSubmission(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID, worker core.UserID) core.SubmissionID {
	t.Helper()
	submissionID, matched := core.NewSubmissionID().(core.SubmissionIDCreated)
	if !matched {
		t.Fatalf("submission id rejected")
	}
	_, err := pool.Exec(context.Background(), `
		insert into submissions (id, task_id, user_id, state, response_json)
		values ($1, $2, $3, 'submitted', '{}'::jsonb)
	`, submissionID.Value.String(), taskID.String(), worker.String())
	if err != nil {
		t.Fatalf("insert submission: %v", err)
	}
	return submissionID.Value
}

func insertSubmittedReservation(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID, worker core.UserID) {
	t.Helper()
	reservationID := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	_, err := pool.Exec(context.Background(), `
		insert into task_reservations (id, task_id, assignee_kind, user_id, state, requested_by_user_id, expires_at)
		values ($1, $2, 'user', $3, 'submitted', $3, now() + interval '48 hours')
	`, reservationID.Value.String(), taskID.String(), worker.String())
	if err != nil {
		t.Fatalf("insert submitted reservation: %v", err)
	}
}

func fundCommand(t *testing.T, owner core.UserID, taskID core.TaskID, amount int64, key string) ledger.FundStoreCommand {
	t.Helper()
	return ledger.FundStoreCommand{
		EntryID:        newEntryID(t),
		FunderUserID:   owner,
		TaskID:         taskID,
		Amount:         creditAmount(t, amount),
		IdempotencyKey: idempotencyKey(t, key),
	}
}

func acceptCommand(t *testing.T, owner core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key string) ledger.AcceptStoreCommand {
	t.Helper()
	return ledger.AcceptStoreCommand{
		PayoutEntryID:    newEntryID(t),
		RefundEntryID:    newEntryID(t),
		TipDebitEntryID:  newEntryID(t),
		TipCreditEntryID: newEntryID(t),
		RequesterUserID:  owner,
		TaskID:           taskID,
		SubmissionID:     submissionID,
		IdempotencyKey:   idempotencyKey(t, key),
	}
}

func refundCommand(t *testing.T, owner core.UserID, taskID core.TaskID, key string) ledger.RefundStoreCommand {
	t.Helper()
	return ledger.RefundStoreCommand{
		EntryID:         newEntryID(t),
		RequesterUserID: owner,
		TaskID:          taskID,
		IdempotencyKey:  idempotencyKey(t, key),
	}
}

func mustBalance(t *testing.T, store db.LedgerStore, owner core.UserID) ledger.Balance {
	t.Helper()
	result := store.Balance(context.Background(), owner)
	found, matched := result.(ledger.BalanceFound)
	if !matched {
		t.Fatalf("balance was rejected")
	}
	return found.Value
}

func creditAmount(t *testing.T, value int64) ledger.CreditAmount {
	t.Helper()
	accepted, matched := ledger.NewCreditAmount(value).(ledger.CreditAmountAccepted)
	if !matched {
		t.Fatalf("credit amount rejected")
	}
	return accepted.Value
}

func idempotencyKey(t *testing.T, raw string) ledger.IdempotencyKey {
	t.Helper()
	accepted, matched := ledger.NewIdempotencyKey(raw).(ledger.IdempotencyKeyAccepted)
	if !matched {
		t.Fatalf("idempotency key rejected")
	}
	return accepted.Value
}

func reviewNote(t *testing.T, raw string) submission.ReviewNote {
	t.Helper()
	accepted, matched := submission.NewRequiredReviewNote(raw).(submission.ReviewNoteAccepted)
	if !matched {
		t.Fatalf("review note rejected")
	}
	return accepted.Value
}

func newUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

func newTaskID(t *testing.T) core.TaskID {
	t.Helper()
	created, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	return created.Value
}

func newEntryID(t *testing.T) core.LedgerEntryID {
	t.Helper()
	created, matched := core.NewLedgerEntryID().(core.LedgerEntryIDCreated)
	if !matched {
		t.Fatalf("ledger entry id rejected")
	}
	return created.Value
}

func requireEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if strings.TrimSpace(value) == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}
