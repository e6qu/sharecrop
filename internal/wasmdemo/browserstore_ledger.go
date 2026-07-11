package wasmdemo

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/task"
)

// LedgerBrowserStore implements ledger.Store against BrowserStorage. It
// shares the same underlying task record storedTaskRecord/loadStoredTaskRecord/
// saveStoredTaskRecord (browserstore_task_shared.go) that TaskBrowserStore
// uses, since the real Postgres LedgerStore reads/writes the tasks table
// directly for its own invariants (draft-only funding, reward_kind flips,
// cancel-on-refund) - splitting that into two independent stores would
// either violate those invariants or duplicate them, the exact problem this
// whole effort exists to eliminate.
//
// Escrow and ledger-entry persistence reuses the existing
// StoredLedgerEntry/SaveLedgerEntry/ListLedgerEntries/LedgerBalance helpers
// internal/wasmdemo already relies on for signup grants and task funding
// display; escrow state itself is tracked as a small additional record per
// task since those existing helpers only model entries, not held escrow.
//
// Submission-review flows (AcceptSubmission/RequestChanges/RejectSubmission)
// cover credit/collectible/bundle payouts, credit/collectible tips, and
// implementor bans (BanSelection writes a storedImplementorBan record that
// CreateReservation and CheckSubmissionEligibility enforce), same as
// internal/db.
type LedgerBrowserStore struct {
	storage BrowserStorage
	ids     InteractionIDSource
}

func NewLedgerBrowserStore(storage BrowserStorage, ids InteractionIDSource) LedgerBrowserStore {
	return LedgerBrowserStore{storage: storage, ids: ids}
}

// storedTaskFund mirrors internal/db's task_funds row: a stateless record that
// exists iff the task currently holds allocated credits. It is deleted when the
// task is awarded or refunded, never state-transitioned.
type storedTaskFund struct {
	TaskID          string `json:"task_id"`
	FunderOwnerKind string `json:"funder_owner_kind"`
	FunderOwnerID   string `json:"funder_owner_id"`
	CreditAmount    int64  `json:"credit_amount"`
}

func taskFundKey(taskID string) string { return "ledger:fund:" + taskID }

func putStoredTaskFundJSON(storage BrowserStorage, rawKey string, record storedTaskFund) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredTaskFundJSON(storage BrowserStorage, rawKey string) (storedTaskFund, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedTaskFund{}, found, ok
	}
	// A deleted fund is stored as an empty string (delete-by-tombstone,
	// matching the rest of the demo store), which reads back as "not found".
	if raw == "" {
		return storedTaskFund{}, false, true
	}
	var record storedTaskFund
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedTaskFund{}, false, false
	}
	return record, true, true
}

func (store LedgerBrowserStore) loadFund(taskID string) (storedTaskFund, bool, *core.DomainError) {
	record, found, ok := getStoredTaskFundJSON(store.storage, taskFundKey(taskID))
	if !ok {
		reason := invalidState("read task fund failed")
		return storedTaskFund{}, false, &reason
	}
	return record, found, nil
}

func (store LedgerBrowserStore) saveFund(record storedTaskFund) bool {
	return putStoredTaskFundJSON(store.storage, taskFundKey(record.TaskID), record)
}

// fundOwnerIndexKey indexes the tasks a given owner (user or organization)
// currently funds, so the allocated section of that owner's wallet can be
// summed - mirroring internal/db's sum(task_funds.credit_amount) joined on the
// funder account.
func fundOwnerIndexKey(ownerKind string, ownerID string) string {
	return "ledger:fund_index:" + ownerKind + ":" + ownerID
}

// clearFund removes a task's fund record and drops it from its funder's index,
// used on award/refund/cancel when the allocated credits change hands.
func (store LedgerBrowserStore) clearFund(record storedTaskFund) bool {
	if !removeFromStringIndex(store.storage, fundOwnerIndexKey(record.FunderOwnerKind, record.FunderOwnerID), record.TaskID) {
		return false
	}
	return putStorageString(store.storage, taskFundKey(record.TaskID), "")
}

// ledgerAllocated sums the credits an owner currently has allocated to tasks.
func ledgerAllocated(storage BrowserStorage, ownerKind string, ownerID string) (int64, *core.DomainError) {
	indexResult := loadStringIndex(storage, fundOwnerIndexKey(ownerKind, ownerID), "task fund")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return 0, &reason
	}
	var total int64
	for _, taskID := range loaded.values {
		record, found, ok := getStoredTaskFundJSON(storage, taskFundKey(taskID))
		if !ok {
			reason := invalidState("read task fund failed")
			return 0, &reason
		}
		if !found {
			continue
		}
		total += record.CreditAmount
	}
	return total, nil
}

// ledgerIdempotencyKeyMarker is a global (cross-owner, cross-command) key
// space mirroring internal/db's `exists(select 1 from ledger_entries where
// idempotency_key = $1)` checks - the real backend's idempotency keys live
// on ledger entries in one table, so a key spent by one ledger command is
// visible to every other. Markers are plain storage records outside the
// per-owner entry index, so they never appear in (or shift the pagination
// of) a listed ledger.
func ledgerIdempotencyKeyMarker(key string) string { return "ledger:idempotency_key:" + key }

// storedFundReceipt is the durable idempotency record for a fund or refund
// command. The real backend reconstructs a replayed fund/refund from its
// ledger entries; the demo store keeps the equivalent facts under the
// idempotency key so a replay returns the same TaskFund even after the
// stateless task_funds record has been consumed.
type storedFundReceipt struct {
	TaskID          string `json:"task_id"`
	FunderOwnerKind string `json:"funder_owner_kind"`
	FunderOwnerID   string `json:"funder_owner_id"`
	CreditAmount    int64  `json:"credit_amount"`
}

func (store LedgerBrowserStore) idempotencyKeyUsed(key string) (bool, *core.DomainError) {
	raw, found, ok := getStorageString(store.storage, ledgerIdempotencyKeyMarker(key))
	if !ok {
		reason := invalidState("check ledger idempotency failed")
		return false, &reason
	}
	return found && raw != "", nil
}

func (store LedgerBrowserStore) markFundReceipt(key string, receipt storedFundReceipt) bool {
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return false
	}
	return putStorageString(store.storage, ledgerIdempotencyKeyMarker(key), string(encoded))
}

func (store LedgerBrowserStore) loadFundReceipt(key string) (storedFundReceipt, bool, *core.DomainError) {
	raw, found, ok := getStorageString(store.storage, ledgerIdempotencyKeyMarker(key))
	if !ok {
		reason := invalidState("read ledger idempotency failed")
		return storedFundReceipt{}, false, &reason
	}
	if !found || raw == "" {
		return storedFundReceipt{}, false, nil
	}
	var receipt storedFundReceipt
	if err := json.Unmarshal([]byte(raw), &receipt); err != nil {
		reason := invalidState("read ledger idempotency failed")
		return storedFundReceipt{}, false, &reason
	}
	return receipt, true, nil
}

// requireFundableTaskBrowser mirrors internal/db's requireFundableTask /
// requireCreditRewardFunding: the task must be draft, and a first-time
// credit funding may flip reward_kind (none -> credit, collectible ->
// bundle) - a task is always fundable by whoever is authorized to fund it,
// regardless of the reward kind it was created with.
func requireFundableTaskBrowser(record storedTaskRecord, amount int64) (storedTaskRecord, *core.DomainError) {
	if record.State != "draft" {
		reason := core.NewDomainError(core.ErrorCodeConflict, "only draft tasks can be funded")
		return record, &reason
	}
	switch record.RewardKind {
	case "credit", "bundle":
		if record.RewardCreditAmount != amount {
			reason := core.NewDomainError(core.ErrorCodeConflict, "funding amount must match the declared credit reward")
			return record, &reason
		}
	case "collectible":
		record.RewardKind = "bundle"
		record.RewardCreditAmount = amount
	default:
		record.RewardKind = "credit"
		record.RewardCreditAmount = amount
	}
	return record, nil
}

func (store LedgerBrowserStore) fund(ownerKind string, ownerID string, taskID core.TaskID, amount ledger.CreditAmount, entryID core.LedgerEntryID, key ledger.IdempotencyKey, insufficientMessage string, requireOwnership func(storedTaskRecord) *core.DomainError) ledger.FundResult {
	// Mirrors internal/db's findFundForKey: a key seen before replays the
	// recorded fund from its durable idempotency receipt.
	keyUsed, keyErr := store.idempotencyKeyUsed(key.String())
	if keyErr != nil {
		return ledger.FundRejected{Reason: *keyErr}
	}
	if keyUsed {
		receipt, receiptFound, receiptErr := store.loadFundReceipt(key.String())
		if receiptErr != nil {
			return ledger.FundRejected{Reason: *receiptErr}
		}
		if !receiptFound || receipt.TaskID != taskID.String() {
			return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "idempotency key was used for a different command")}
		}
		return ledger.TaskFunded{Fund: buildLedgerTaskFund(receipt.TaskID, receipt.CreditAmount)}
	}

	record, found, recordErr := loadStoredTaskRecord(store.storage, taskID.String())
	if recordErr != nil {
		return ledger.FundRejected{Reason: *recordErr}
	}
	if !found {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if reason := requireOwnership(record); reason != nil {
		return ledger.FundRejected{Reason: *reason}
	}

	updated, fundableErr := requireFundableTaskBrowser(record, amount.Int64())
	if fundableErr != nil {
		return ledger.FundRejected{Reason: *fundableErr}
	}

	_, fundFound, fundErr := store.loadFund(taskID.String())
	if fundErr != nil {
		return ledger.FundRejected{Reason: *fundErr}
	}
	if fundFound {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task is already funded")}
	}

	balanceResult := LedgerBalance(store.storage, ownerKind, ownerID)
	balanceStored, balanceMatched := balanceResult.(LedgerBalanceStored)
	if !balanceMatched {
		return ledger.FundRejected{Reason: invalidState("read account balance failed")}
	}
	if balanceStored.Amount < amount.Int64() {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, insufficientMessage)}
	}

	if !saveStoredTaskRecord(store.storage, updated) {
		return ledger.FundRejected{Reason: invalidState("update task reward kind failed")}
	}
	fundRecord := storedTaskFund{TaskID: taskID.String(), FunderOwnerKind: ownerKind, FunderOwnerID: ownerID, CreditAmount: amount.Int64()}
	if !store.saveFund(fundRecord) {
		return ledger.FundRejected{Reason: invalidState("insert task fund failed")}
	}
	if _, matched := appendStringIndex(store.storage, fundOwnerIndexKey(ownerKind, ownerID), taskID.String(), "task fund").(stringIndexStored); !matched {
		return ledger.FundRejected{Reason: invalidState("update task fund index failed")}
	}
	entryResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID: entryID.String(), OwnerKind: ownerKind, OwnerID: ownerID,
		Kind: ledger.EntryKindTaskFund.String(), Amount: -amount.Int64(), TaskID: taskID.String(),
		IdempotencyKey: key.String(),
	})
	if _, matched := entryResult.(LedgerEntryStored); !matched {
		return ledger.FundRejected{Reason: invalidState("insert task fund ledger entry failed")}
	}
	if !store.markFundReceipt(key.String(), storedFundReceipt{TaskID: taskID.String(), FunderOwnerKind: ownerKind, FunderOwnerID: ownerID, CreditAmount: amount.Int64()}) {
		return ledger.FundRejected{Reason: invalidState("record fund idempotency failed")}
	}

	return ledger.TaskFunded{Fund: buildLedgerTaskFund(taskID.String(), amount.Int64())}
}

func buildLedgerTaskFund(rawTaskID string, amount int64) ledger.TaskFund {
	amountResult := ledger.NewCreditAmount(amount)
	amountAccepted, _ := amountResult.(ledger.CreditAmountAccepted)
	taskIDResult := core.ParseTaskID(rawTaskID)
	taskID, _ := taskIDResult.(core.TaskIDCreated)
	return ledger.TaskFund{TaskID: taskID.Value, CreditAmount: amountAccepted.Value}
}

func (store LedgerBrowserStore) FundTask(_ context.Context, command ledger.FundStoreCommand) ledger.FundResult {
	return store.fund("user", command.FunderUserID.String(), command.TaskID, command.Amount, command.EntryID, command.IdempotencyKey, "insufficient credits to fund the task", func(record storedTaskRecord) *core.DomainError {
		if record.CreatedBy != command.FunderUserID.String() {
			reason := core.NewDomainError(core.ErrorCodePermissionDenied, "only the task owner can fund the task")
			return &reason
		}
		return nil
	})
}

func (store LedgerBrowserStore) FundTaskFromOrganization(_ context.Context, command ledger.OrganizationFundStoreCommand) ledger.FundResult {
	return store.fund("organization", command.OrganizationID.String(), command.TaskID, command.Amount, command.EntryID, command.IdempotencyKey, "insufficient organization credits to fund the task", func(record storedTaskRecord) *core.DomainError {
		if record.OwnerOrganizationID != command.OrganizationID.String() {
			reason := core.NewDomainError(core.ErrorCodePermissionDenied, "task is not owned by the organization")
			return &reason
		}
		return nil
	})
}

func (store LedgerBrowserStore) RefundTask(_ context.Context, command ledger.RefundStoreCommand) ledger.RefundResult {
	record, found, recordErr := loadStoredTaskRecord(store.storage, command.TaskID.String())
	if recordErr != nil {
		return ledger.RefundRejected{Reason: *recordErr}
	}
	if !found {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if reason := requireRefundAuthorizedBrowser(store.storage, record, command.RequesterUserID); reason != nil {
		return ledger.RefundRejected{Reason: *reason}
	}
	if record.State != "draft" && record.State != "open" {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only tasks that are not yet awarded can be refunded")}
	}

	keyUsed, keyErr := store.idempotencyKeyUsed(command.IdempotencyKey.String())
	if keyErr != nil {
		return ledger.RefundRejected{Reason: *keyErr}
	}

	fund, fundFound, fundErr := store.loadFund(command.TaskID.String())
	if fundErr != nil {
		return ledger.RefundRejected{Reason: *fundErr}
	}
	if !fundFound {
		// No allocated credits: replay a prior refund from its receipt, else
		// there is nothing to refund.
		if keyUsed {
			receipt, receiptFound, receiptErr := store.loadFundReceipt(command.IdempotencyKey.String())
			if receiptErr != nil {
				return ledger.RefundRejected{Reason: *receiptErr}
			}
			if receiptFound && receipt.TaskID == command.TaskID.String() {
				return ledger.TaskRefunded{Fund: buildLedgerTaskFund(receipt.TaskID, receipt.CreditAmount)}
			}
			return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "idempotency key was used for a different command")}
		}
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task has nothing to refund")}
	}
	if keyUsed {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "idempotency key was used for a different command")}
	}

	entryResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID: command.EntryID.String(), OwnerKind: fund.FunderOwnerKind, OwnerID: fund.FunderOwnerID,
		Kind: ledger.EntryKindTaskRefund.String(), Amount: fund.CreditAmount, TaskID: command.TaskID.String(),
		IdempotencyKey: command.IdempotencyKey.String(),
	})
	if _, matched := entryResult.(LedgerEntryStored); !matched {
		return ledger.RefundRejected{Reason: invalidState("insert task refund ledger entry failed")}
	}
	if !store.markFundReceipt(command.IdempotencyKey.String(), storedFundReceipt{TaskID: fund.TaskID, FunderOwnerKind: fund.FunderOwnerKind, FunderOwnerID: fund.FunderOwnerID, CreditAmount: fund.CreditAmount}) {
		return ledger.RefundRejected{Reason: invalidState("record refund idempotency failed")}
	}
	if !store.clearFund(fund) {
		return ledger.RefundRejected{Reason: invalidState("clear task fund failed")}
	}

	if reason := refundHeldCollectibleRewardBrowser(store.storage, store.ids, command.TaskID.String()); reason != nil {
		return ledger.RefundRejected{Reason: *reason}
	}

	record.State = "cancelled"
	if !saveStoredTaskRecord(store.storage, record) {
		return ledger.RefundRejected{Reason: invalidState("cancel task failed")}
	}

	// A refund cancels the task, so release the worker's reservation too
	// (same as the cancel path's releaseReservationsOnCancel).
	if reason := releaseReservationsOnCancel(store.storage, command.TaskID.String()); reason != nil {
		return ledger.RefundRejected{Reason: *reason}
	}

	return ledger.TaskRefunded{Fund: buildLedgerTaskFund(fund.TaskID, fund.CreditAmount)}
}

// requireRefundAuthorizedBrowser mirrors internal/db's lockTaskRefundable
// authorization: a refund is permitted for the task owner (its creator) or the
// user currently holding the active reservation (the implementor).
func requireRefundAuthorizedBrowser(storage BrowserStorage, record storedTaskRecord, requester core.UserID) *core.DomainError {
	if record.CreatedBy == requester.String() {
		return nil
	}
	implementor, err := activeImplementorUserID(storage, record.ID)
	if err != nil {
		return err
	}
	if implementor == requester.String() {
		return nil
	}
	reason := core.NewDomainError(core.ErrorCodePermissionDenied, "only the task owner or the active implementor can refund the task")
	return &reason
}

// activeImplementorUserID returns the user id holding the task's active
// user reservation, or "" if there is none.
func activeImplementorUserID(storage BrowserStorage, taskID string) (string, *core.DomainError) {
	indexResult := loadStringIndex(storage, reservationTaskIndexKey(taskID), "reservation")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return "", &reason
	}
	for _, id := range loaded.values {
		reservation, found, ok := getStoredReservationJSON(storage, reservationRecordKey(id))
		if !ok || !found {
			continue
		}
		if reservation.State == task.ReservationStateActive.String() && reservation.AssigneeKind == task.AssigneeScopeUser.String() {
			return reservation.AssigneeUserID, nil
		}
	}
	return "", nil
}

func (store LedgerBrowserStore) Balance(_ context.Context, owner core.UserID) ledger.BalanceResult {
	return store.balanceForOwner("user", owner.String(), "read balance failed")
}

func (store LedgerBrowserStore) OrganizationBalance(_ context.Context, organizationID core.OrganizationID) ledger.BalanceResult {
	return store.balanceForOwner("organization", organizationID.String(), "read organization balance failed")
}

func (store LedgerBrowserStore) TaskAllocatedCredits(_ context.Context, taskID core.TaskID) ledger.TaskAllocatedResult {
	record, found, domainErr := store.loadFund(taskID.String())
	if domainErr != nil {
		return ledger.TaskAllocatedRejected{Reason: *domainErr}
	}
	if !found {
		return ledger.TaskAllocatedFound{Amount: 0}
	}
	return ledger.TaskAllocatedFound{Amount: record.CreditAmount}
}

func (store LedgerBrowserStore) balanceForOwner(ownerKind string, ownerID string, failureMessage string) ledger.BalanceResult {
	result := LedgerBalance(store.storage, ownerKind, ownerID)
	stored, matched := result.(LedgerBalanceStored)
	if !matched {
		return ledger.BalanceRejected{Reason: invalidState(failureMessage)}
	}
	allocated, allocatedErr := ledgerAllocated(store.storage, ownerKind, ownerID)
	if allocatedErr != nil {
		return ledger.BalanceRejected{Reason: *allocatedErr}
	}
	return ledger.BalanceFound{Value: ledger.NewBalance(stored.Amount, allocated)}
}

func (store LedgerBrowserStore) listEntries(ownerKind string, ownerID string, page core.Page) ledger.ListEntriesResult {
	result := ListLedgerEntries(store.storage, ownerKind, ownerID, StoredListPage{limit: page.Limit(), offset: page.Offset()})
	stored, matched := result.(LedgerEntriesStored)
	if !matched {
		return ledger.ListEntriesRejected{Reason: invalidState("read ledger entries failed")}
	}
	values := make([]ledger.LedgerEntry, 0, len(stored.Values))
	for _, entry := range stored.Values {
		// Idempotency markers now live in their own key space (see
		// ledgerIdempotencyKeyMarker) and never enter the entry index; this
		// skip only shields listings against markers persisted by pre-fix
		// demo storage.
		if strings.HasPrefix(entry.ID, "idempotency:") {
			continue
		}
		parsed, parseErr := parseStoredLedgerEntry(entry)
		if parseErr != nil {
			return ledger.ListEntriesRejected{Reason: *parseErr}
		}
		values = append(values, parsed)
	}
	return ledger.EntriesListed{Values: values}
}

func parseStoredLedgerEntry(entry StoredLedgerEntry) (ledger.LedgerEntry, *core.DomainError) {
	idResult := core.ParseLedgerEntryID(entry.ID)
	id, idMatched := idResult.(core.LedgerEntryIDCreated)
	if !idMatched {
		reason := idResult.(core.LedgerEntryIDRejected).Reason
		return ledger.LedgerEntry{}, &reason
	}
	kindResult := ledger.ParseEntryKind(entry.Kind)
	kind, kindMatched := kindResult.(ledger.EntryKindAccepted)
	if !kindMatched {
		reason := kindResult.(ledger.EntryKindRejected).Reason
		return ledger.LedgerEntry{}, &reason
	}
	amountResult := ledger.ParseSignedAmount(entry.Amount)
	amount, amountMatched := amountResult.(ledger.SignedAmountAccepted)
	if !amountMatched {
		reason := amountResult.(ledger.SignedAmountRejected).Reason
		return ledger.LedgerEntry{}, &reason
	}
	var taskRef ledger.TaskReference = ledger.NoTaskReference{}
	if entry.TaskID != "" {
		taskIDResult := core.ParseTaskID(entry.TaskID)
		taskID, taskIDMatched := taskIDResult.(core.TaskIDCreated)
		if !taskIDMatched {
			reason := taskIDResult.(core.TaskIDRejected).Reason
			return ledger.LedgerEntry{}, &reason
		}
		taskRef = ledger.TaskReferenced{TaskID: taskID.Value}
	}
	return ledger.LedgerEntry{ID: id.Value, Kind: kind.Value, Amount: amount.Value, TaskRef: taskRef}, nil
}

func (store LedgerBrowserStore) ListEntries(_ context.Context, owner core.UserID, page core.Page) ledger.ListEntriesResult {
	return store.listEntries("user", owner.String(), page)
}

func (store LedgerBrowserStore) ListOrganizationEntries(_ context.Context, organizationID core.OrganizationID, page core.Page) ledger.ListEntriesResult {
	return store.listEntries("organization", organizationID.String(), page)
}

// reviewEscrowOutcome mirrors internal/db's payReviewEscrow: pays out the
// task's held credit escrow (full or partial) to the worker, refunding the
// leftover remainder to the funder when closeTask is true (accept - the
// task is closing, nothing more will be paid from this escrow) or leaving
// the remainder held when false (reject - a future submission could still
// be accepted against it).
func (store LedgerBrowserStore) reviewEscrowOutcome(taskID core.TaskID, workerID core.UserID, payoutEntryID core.LedgerEntryID, refundEntryID core.LedgerEntryID, key ledger.IdempotencyKey, selection ledger.CreditReviewSelection, closeTask bool) (ledger.PayoutOutcome, *core.DomainError) {
	if _, noPayout := selection.(ledger.NoCreditReviewSelection); noPayout {
		return ledger.NoPayout{}, nil
	}
	fund, found, err := store.loadFund(taskID.String())
	if err != nil {
		return nil, err
	}
	_, partial := selection.(ledger.PartialCreditReviewSelection)
	if !found {
		if partial {
			reason := core.NewDomainError(core.ErrorCodeConflict, "credit reward fund is missing")
			return nil, &reason
		}
		return ledger.NoPayout{}, nil
	}

	payoutAmount := fund.CreditAmount
	if selected, matched := selection.(ledger.PartialCreditReviewSelection); matched {
		payoutAmount = selected.Amount.Int64()
	}
	if payoutAmount > fund.CreditAmount {
		reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "credit payout cannot exceed allocated credits")
		return nil, &reason
	}

	entryResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID: payoutEntryID.String(), OwnerKind: "user", OwnerID: workerID.String(),
		Kind: ledger.EntryKindTaskPayout.String(), Amount: payoutAmount, TaskID: taskID.String(),
		IdempotencyKey: key.String(),
	})
	if _, matched := entryResult.(LedgerEntryStored); !matched {
		reason := invalidState("insert task payout ledger entry failed")
		return nil, &reason
	}

	remaining := fund.CreditAmount - payoutAmount
	if closeTask {
		if remaining > 0 {
			refundResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
				ID: refundEntryID.String(), OwnerKind: fund.FunderOwnerKind, OwnerID: fund.FunderOwnerID,
				Kind: ledger.EntryKindTaskRefund.String(), Amount: remaining, TaskID: taskID.String(),
			})
			if _, matched := refundResult.(LedgerEntryStored); !matched {
				reason := invalidState("insert partial accept refund ledger entry failed")
				return nil, &reason
			}
		}
		if !store.clearFund(fund) {
			reason := invalidState("clear task fund failed")
			return nil, &reason
		}
	} else if remaining == 0 {
		if !store.clearFund(fund) {
			reason := invalidState("clear task fund failed")
			return nil, &reason
		}
	} else {
		fund.CreditAmount = remaining
		if !store.saveFund(fund) {
			reason := invalidState("update task fund after review payout failed")
			return nil, &reason
		}
	}

	amountResult := ledger.NewCreditAmount(payoutAmount)
	amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		reason := amountResult.(ledger.CreditAmountRejected).Reason
		return nil, &reason
	}
	return ledger.CreditPayout{WorkerUserID: workerID, Amount: amount.Value}, nil
}

// creditTipOutcome mirrors internal/db's payCreditTip: debits the requester
// and credits the worker for a voluntary tip on top of the review outcome.
func (store LedgerBrowserStore) creditTipOutcome(taskID core.TaskID, requesterID core.UserID, workerID core.UserID, debitEntryID core.LedgerEntryID, creditEntryID core.LedgerEntryID, selection ledger.TipSelection) (ledger.TipOutcome, *core.DomainError) {
	tip, matched := selection.(ledger.CreditTipSelection)
	if !matched {
		return ledger.NoTip{}, nil
	}
	balanceResult := LedgerBalance(store.storage, "user", requesterID.String())
	balanceStored, balanceMatched := balanceResult.(LedgerBalanceStored)
	if !balanceMatched {
		reason := invalidState("read requester balance failed")
		return nil, &reason
	}
	if balanceStored.Amount < tip.Amount.Int64() {
		reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "insufficient credits to tip the implementor")
		return nil, &reason
	}
	debitResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID: debitEntryID.String(), OwnerKind: "user", OwnerID: requesterID.String(),
		Kind: ledger.EntryKindTaskTip.String(), Amount: -tip.Amount.Int64(), TaskID: taskID.String(),
	})
	if _, matched := debitResult.(LedgerEntryStored); !matched {
		reason := invalidState("insert task tip debit failed")
		return nil, &reason
	}
	creditResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID: creditEntryID.String(), OwnerKind: "user", OwnerID: workerID.String(),
		Kind: ledger.EntryKindTaskTip.String(), Amount: tip.Amount.Int64(), TaskID: taskID.String(),
	})
	if _, matched := creditResult.(LedgerEntryStored); !matched {
		reason := invalidState("insert task tip credit failed")
		return nil, &reason
	}
	return ledger.CreditTip{WorkerUserID: workerID, Amount: tip.Amount}, nil
}

// collectibleTipOutcome mirrors internal/db's payCollectibleTip, reusing
// AssetBrowserStore.GiftCollectible directly since a voluntary tip is
// exactly a gift from requester to worker (same ownership/policy/idempotent
// -replay checks).
func (store LedgerBrowserStore) collectibleTipOutcome(requesterID core.UserID, workerID core.UserID, selection ledger.CollectibleTipSelection) (ledger.TipOutcome, *core.DomainError) {
	selected, matched := selection.(ledger.CollectibleTipSelected)
	if !matched {
		return ledger.NoTip{}, nil
	}
	assetStore := AssetBrowserStore{storage: store.storage, ids: store.ids}
	giftResult := assetStore.GiftCollectible(context.Background(), assets.GiftStoreCommand{FromUserID: requesterID, ToUserID: workerID, CollectibleID: selected.ID})
	if rejected, matched := giftResult.(assets.GiftRejected); matched {
		reason := rejected.Reason
		return nil, &reason
	}
	return ledger.CollectibleTip{WorkerUserID: workerID, CollectibleID: selected.ID}, nil
}

func combinePayoutsBrowser(first ledger.PayoutOutcome, second ledger.PayoutOutcome) ledger.PayoutOutcome {
	credit, hasCredit := first.(ledger.CreditPayout)
	collectible, hasCollectible := second.(ledger.CollectiblePayout)
	if hasCredit && hasCollectible && credit.WorkerUserID == collectible.WorkerUserID {
		return ledger.BundlePayout{WorkerUserID: credit.WorkerUserID, Amount: credit.Amount, CollectibleIDs: collectible.CollectibleIDs}
	}
	if _, firstNone := first.(ledger.NoPayout); firstNone {
		return second
	}
	return first
}

func combineTipsBrowser(first ledger.TipOutcome, second ledger.TipOutcome) ledger.TipOutcome {
	credit, hasCredit := first.(ledger.CreditTip)
	collectible, hasCollectible := second.(ledger.CollectibleTip)
	if hasCredit && hasCollectible && credit.WorkerUserID == collectible.WorkerUserID {
		return ledger.BundleTip{WorkerUserID: credit.WorkerUserID, Amount: credit.Amount, CollectibleID: collectible.CollectibleID}
	}
	if _, firstNone := first.(ledger.NoTip); firstNone {
		return second
	}
	return first
}

// reactivateWorkerReservation and cancelWorkerReservation mirror internal/db's
// reservation side effects of RequestChanges/RejectSubmission - a worker's
// submitted reservation goes back to active on changes requested, or is
// cancelled outright on rejection, so a fresh reservation could be made.
func reactivateWorkerReservation(storage BrowserStorage, taskID string, workerID string, fromState string, toState string) *core.DomainError {
	indexResult := loadStringIndex(storage, reservationTaskIndexKey(taskID), "reservation")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return &reason
	}
	for _, id := range loaded.values {
		record, found, ok := getStoredReservationJSON(storage, reservationRecordKey(id))
		if !ok || !found {
			continue
		}
		if record.AssigneeKind != task.AssigneeScopeUser.String() || record.AssigneeUserID != workerID {
			continue
		}
		if record.State != fromState {
			continue
		}
		record.State = toState
		if !putStoredReservationJSON(storage, reservationRecordKey(id), record) {
			reason := invalidState("update task reservation failed")
			return &reason
		}
	}
	return nil
}

// requireTaskReviewPermission mirrors internal/db's lockTaskForReview: the
// task's literal creator is always authorized; for an organization-owned
// task, a member holding the review-submissions permission is authorized
// too, so an org's designated reviewer (not just its task's creator) can
// accept/reject/request changes on its behalf.
func requireTaskReviewPermission(storage BrowserStorage, record storedTaskRecord, requesterID core.UserID, action string) *core.DomainError {
	if record.CreatedBy == requesterID.String() {
		return nil
	}
	if record.OwnerOrganizationID != "" {
		organizationIDResult := core.ParseOrganizationID(record.OwnerOrganizationID)
		if organizationID, matched := organizationIDResult.(core.OrganizationIDCreated); matched {
			rolesResult := (OrgBrowserStore{storage: storage}).FindMemberRoles(context.Background(), organizationID.Value, requesterID)
			if found, matched := rolesResult.(org.MemberRolesFound); matched {
				if _, granted := org.CheckPermission(found.Roles, org.PermissionReviewSubmissions).(org.PermissionGranted); granted {
					return nil
				}
			}
		}
	}
	reason := core.NewDomainError(core.ErrorCodePermissionDenied, "only the task owner or an organization reviewer can "+action+" the task")
	return &reason
}

func (store LedgerBrowserStore) AcceptSubmission(_ context.Context, command ledger.AcceptStoreCommand) ledger.AcceptResult {
	taskRecord, taskFound, taskErr := loadStoredTaskRecord(store.storage, command.TaskID.String())
	if taskErr != nil {
		return ledger.AcceptRejected{Reason: *taskErr}
	}
	if !taskFound {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if err := requireTaskReviewPermission(store.storage, taskRecord, command.RequesterUserID, "accept submissions for"); err != nil {
		return ledger.AcceptRejected{Reason: *err}
	}

	submission, submissionFound, ok := getStoredSubmissionJSON(store.storage, submissionRecordKey(command.SubmissionID.String()))
	if !ok {
		return ledger.AcceptRejected{Reason: invalidState("read submission failed")}
	}
	if !submissionFound || submission.TaskID != command.TaskID.String() {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found for the task")}
	}
	workerIDResult := core.ParseUserID(submission.SubmitterID)
	workerID, workerIDMatched := workerIDResult.(core.UserIDCreated)
	if !workerIDMatched {
		return ledger.AcceptRejected{Reason: workerIDResult.(core.UserIDRejected).Reason}
	}

	if submission.State == "accepted" {
		if submission.AcceptedIdempotencyKey != command.IdempotencyKey.String() {
			return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "submission was already accepted with a different idempotency key")}
		}
		return store.idempotentAcceptOutcome(command, workerID.Value)
	}
	if submission.State != "submitted" {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only valid submissions can be accepted")}
	}
	if taskRecord.State != "open" {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can accept submissions")}
	}

	payout, payoutErr := store.reviewEscrowOutcome(command.TaskID, workerID.Value, command.PayoutEntryID, command.RefundEntryID, command.IdempotencyKey, command.CreditSelection, true)
	if payoutErr != nil {
		return ledger.AcceptRejected{Reason: *payoutErr}
	}
	assetStore := AssetBrowserStore{storage: store.storage, ids: store.ids}
	collectibleIDs, collectibleErr := assetStore.payOutHeldCollectibleReward(command.TaskID.String(), workerID.Value.String())
	if collectibleErr != nil {
		return ledger.AcceptRejected{Reason: *collectibleErr}
	}
	var collectiblePayout ledger.PayoutOutcome = ledger.NoPayout{}
	if len(collectibleIDs) > 0 {
		collectiblePayout = ledger.CollectiblePayout{WorkerUserID: workerID.Value, CollectibleIDs: collectibleIDs}
	}
	outcome := combinePayoutsBrowser(payout, collectiblePayout)
	if _, noPayout := outcome.(ledger.NoPayout); noPayout {
		if taskRecord.RewardKind == "credit" || taskRecord.RewardKind == "bundle" {
			return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward fund is missing")}
		}
	}

	tip, tipErr := store.creditTipOutcome(command.TaskID, command.RequesterUserID, workerID.Value, command.TipDebitEntryID, command.TipCreditEntryID, command.TipSelection)
	if tipErr != nil {
		return ledger.AcceptRejected{Reason: *tipErr}
	}
	collectibleTip, collectibleTipErr := store.collectibleTipOutcome(command.RequesterUserID, workerID.Value, command.CollectibleTip)
	if collectibleTipErr != nil {
		return ledger.AcceptRejected{Reason: *collectibleTipErr}
	}
	tipOutcome := combineTipsBrowser(tip, collectibleTip)

	submission.State = "accepted"
	submission.AcceptedIdempotencyKey = command.IdempotencyKey.String()
	if !putStoredSubmissionJSON(store.storage, submissionRecordKey(submission.ID), submission) {
		return ledger.AcceptRejected{Reason: invalidState("accept submission failed")}
	}
	taskRecord.State = "closed"
	if !saveStoredTaskRecord(store.storage, taskRecord) {
		return ledger.AcceptRejected{Reason: invalidState("close task failed")}
	}

	return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: outcome, Tip: tipOutcome}
}

// idempotentAcceptOutcome mirrors internal/db's idempotentAccept: a retried
// accept with the original key reconstructs the original payout (the released
// escrow amount plus the released collectible rewards) instead of reporting
// NoPayout, so a retry after a lost response reads the same as the first
// call. Tips are not replayed, matching the real store.
func (store LedgerBrowserStore) idempotentAcceptOutcome(command ledger.AcceptStoreCommand, workerID core.UserID) ledger.AcceptResult {
	outcome := ledger.PayoutOutcome(ledger.NoPayout{})
	payoutAmount, payoutFound, payoutErr := store.taskPayoutForKey(workerID.String(), command.TaskID.String(), command.IdempotencyKey.String())
	if payoutErr != nil {
		return ledger.AcceptRejected{Reason: *payoutErr}
	}
	if payoutFound {
		amountResult := ledger.NewCreditAmount(payoutAmount)
		amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
		if amountMatched {
			outcome = ledger.CreditPayout{WorkerUserID: workerID, Amount: amount.Value}
		}
	}
	return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: outcome, Tip: ledger.NoTip{}}
}

// taskPayoutForKey reconstructs the credit payout an accept recorded, from the
// durable task_payout ledger entry on the worker's account keyed on the
// accept's idempotency key - mirroring internal/db's idempotentAccept.
func (store LedgerBrowserStore) taskPayoutForKey(workerID string, taskID string, key string) (int64, bool, *core.DomainError) {
	result := ListLedgerEntries(store.storage, "user", workerID, StoredListPage{limit: 1000000, offset: 0})
	stored, matched := result.(LedgerEntriesStored)
	if !matched {
		reason := invalidState("read ledger entries failed")
		return 0, false, &reason
	}
	for _, entry := range stored.Values {
		if entry.Kind == ledger.EntryKindTaskPayout.String() && entry.TaskID == taskID && entry.IdempotencyKey == key {
			return entry.Amount, true, nil
		}
	}
	return 0, false, nil
}

func (store LedgerBrowserStore) RequestChanges(_ context.Context, command ledger.RequestChangesStoreCommand) ledger.RequestChangesResult {
	taskRecord, taskFound, taskErr := loadStoredTaskRecord(store.storage, command.TaskID.String())
	if taskErr != nil {
		return ledger.RequestChangesRejected{Reason: *taskErr}
	}
	if !taskFound {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if err := requireTaskReviewPermission(store.storage, taskRecord, command.RequesterUserID, "review submissions for"); err != nil {
		return ledger.RequestChangesRejected{Reason: *err}
	}
	if taskRecord.State != "open" {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can request submission changes")}
	}

	submission, submissionFound, ok := getStoredSubmissionJSON(store.storage, submissionRecordKey(command.SubmissionID.String()))
	if !ok {
		return ledger.RequestChangesRejected{Reason: invalidState("read submission failed")}
	}
	if !submissionFound || submission.TaskID != command.TaskID.String() {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found for the task")}
	}
	if submission.State != "submitted" {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only submitted work can receive requested changes")}
	}

	submission.State = "changes_requested"
	submission.ReviewNote = command.ReviewNote.String()
	if !putStoredSubmissionJSON(store.storage, submissionRecordKey(submission.ID), submission) {
		return ledger.RequestChangesRejected{Reason: invalidState("request submission changes failed")}
	}
	if err := reactivateWorkerReservation(store.storage, command.TaskID.String(), submission.SubmitterID, "submitted", "active"); err != nil {
		return ledger.RequestChangesRejected{Reason: *err}
	}

	return ledger.ChangesRequested{TaskID: command.TaskID, SubmissionID: command.SubmissionID, ReviewNote: command.ReviewNote.String()}
}

func (store LedgerBrowserStore) RejectSubmission(_ context.Context, command ledger.RejectStoreCommand) ledger.RejectResult {
	taskRecord, taskFound, taskErr := loadStoredTaskRecord(store.storage, command.TaskID.String())
	if taskErr != nil {
		return ledger.RejectRejected{Reason: *taskErr}
	}
	if !taskFound {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if err := requireTaskReviewPermission(store.storage, taskRecord, command.RequesterUserID, "review submissions for"); err != nil {
		return ledger.RejectRejected{Reason: *err}
	}
	if taskRecord.State != "open" {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can reject submissions")}
	}

	submission, submissionFound, ok := getStoredSubmissionJSON(store.storage, submissionRecordKey(command.SubmissionID.String()))
	if !ok {
		return ledger.RejectRejected{Reason: invalidState("read submission failed")}
	}
	if !submissionFound || submission.TaskID != command.TaskID.String() {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found for the task")}
	}
	workerIDResult := core.ParseUserID(submission.SubmitterID)
	workerID, workerIDMatched := workerIDResult.(core.UserIDCreated)
	if !workerIDMatched {
		return ledger.RejectRejected{Reason: workerIDResult.(core.UserIDRejected).Reason}
	}

	if submission.State == "rejected" {
		if submission.ReviewIdempotencyKey == command.IdempotencyKey.String() {
			return ledger.SubmissionRejected{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
		}
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "submission was already rejected with a different idempotency key")}
	}
	if submission.State != "submitted" {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only submitted work can be rejected")}
	}

	payout, payoutErr := store.reviewEscrowOutcome(command.TaskID, workerID.Value, command.PayoutEntryID, core.LedgerEntryID{}, command.IdempotencyKey, command.CreditSelection, false)
	if payoutErr != nil {
		return ledger.RejectRejected{Reason: *payoutErr}
	}
	tip, tipErr := store.creditTipOutcome(command.TaskID, command.RequesterUserID, workerID.Value, command.TipDebitEntryID, command.TipCreditEntryID, command.TipSelection)
	if tipErr != nil {
		return ledger.RejectRejected{Reason: *tipErr}
	}

	submission.State = "rejected"
	submission.ReviewNote = command.ReviewNote.String()
	submission.ReviewIdempotencyKey = command.IdempotencyKey.String()
	if !putStoredSubmissionJSON(store.storage, submissionRecordKey(submission.ID), submission) {
		return ledger.RejectRejected{Reason: invalidState("reject submission failed")}
	}
	if err := reactivateWorkerReservation(store.storage, command.TaskID.String(), submission.SubmitterID, "active", "cancelled_by_requester"); err != nil {
		return ledger.RejectRejected{Reason: *err}
	}
	if err := reactivateWorkerReservation(store.storage, command.TaskID.String(), submission.SubmitterID, "submitted", "cancelled_by_requester"); err != nil {
		return ledger.RejectRejected{Reason: *err}
	}

	if _, ban := command.BanSelection.(ledger.BanImplementorSelection); ban {
		banRecord := storedImplementorBan{
			TaskID: command.TaskID.String(), AssigneeKind: task.AssigneeScopeUser.String(),
			UserID: submission.SubmitterID, BannedBy: command.RequesterUserID.String(),
		}
		if !saveImplementorBan(store.storage, banRecord) {
			return ledger.RejectRejected{Reason: invalidState("ban task implementor failed")}
		}
	}

	return ledger.SubmissionRejected{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: payout, Tip: tip}
}

// refundHeldCollectibleRewardBrowser mirrors internal/db's
// refundHeldCollectibleReward: a bundle reward's held collectible half
// returns to "minted" alongside the credit refund below, so a credit refund
// on a bundle task doesn't strand the collectible half in escrow.
func refundHeldCollectibleRewardBrowser(storage BrowserStorage, ids InteractionIDSource, taskID string) *core.DomainError {
	_, err := (AssetBrowserStore{storage: storage, ids: ids}).releaseHeldCollectibleReward(taskID)
	return err
}

type StoredLedgerEntry struct {
	ID             string `json:"id"`
	OwnerKind      string `json:"owner_kind"`
	OwnerID        string `json:"owner_id"`
	Kind           string `json:"kind"`
	Amount         int64  `json:"amount"`
	TaskID         string `json:"task_id"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type LedgerStorageResult interface {
	ledgerStorageResult()
}

type LedgerEntryStored struct {
	Value StoredLedgerEntry
}

type LedgerEntriesStored struct {
	Values []StoredLedgerEntry
}

type LedgerBalanceStored struct {
	Amount int64
}

type LedgerStorageRejected struct {
	Reason string
}

func (LedgerEntryStored) ledgerStorageResult()     {}
func (LedgerEntriesStored) ledgerStorageResult()   {}
func (LedgerBalanceStored) ledgerStorageResult()   {}
func (LedgerStorageRejected) ledgerStorageResult() {}

func SaveLedgerEntry(storage BrowserStorage, entry StoredLedgerEntry) LedgerStorageResult {
	cleaned := cleanStoredLedgerEntry(entry)
	if reason := validateStoredLedgerEntry(cleaned); reason != "" {
		return LedgerStorageRejected{Reason: reason}
	}
	keyResult := NewStorageKey("ledger:" + cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return LedgerStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return LedgerStorageRejected{Reason: "ledger entry encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return LedgerStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendStringIndex(storage, "ledger:index:"+cleaned.OwnerKind+":"+cleaned.OwnerID, cleaned.ID, "ledger entry")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return LedgerStorageRejected{Reason: indexResult.(stringIndexRejected).reason}
	}
	return LedgerEntryStored{Value: cleaned}
}

func ListLedgerEntries(storage BrowserStorage, ownerKind string, ownerID string, page StoredListPage) LedgerStorageResult {
	cleanKind := strings.TrimSpace(ownerKind)
	cleanID := strings.TrimSpace(ownerID)
	if !validStoredLedgerOwnerKind(cleanKind) {
		return LedgerStorageRejected{Reason: "ledger owner kind is invalid"}
	}
	if cleanID == "" {
		return LedgerStorageRejected{Reason: "ledger owner id is required"}
	}
	idsResult := loadStringIndex(storage, "ledger:index:"+cleanKind+":"+cleanID, "ledger entry")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return LedgerStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	start, end := pageBounds(len(ids.values), page)
	values := make([]StoredLedgerEntry, 0, end-start)
	for index := start; index < end; index++ {
		loadResult := loadLedgerEntry(storage, ids.values[index])
		loaded, loadedMatched := loadResult.(LedgerEntryStored)
		if !loadedMatched {
			return loadResult
		}
		if loaded.Value.OwnerKind != cleanKind || loaded.Value.OwnerID != cleanID {
			return LedgerStorageRejected{Reason: "ledger index contains mismatched record"}
		}
		values = append(values, loaded.Value)
	}
	return LedgerEntriesStored{Values: values}
}

// LedgerBalance sums an owner's ledger entries directly. It used to add a
// hardcoded +100 baseline for user-kind owners, on the assumption that the
// signup grant was implicit and never had its own ledger entry - true only
// for the pre-cutover demo's simplified auth handler. Now that every real
// user (via AuthBrowserStore.insertSignupGrant) and organization (via
// OrgBrowserStore.insertOrganizationCreditGrant) gets an explicit signup-
// grant ledger entry, the baseline would double-count it.
func LedgerBalance(storage BrowserStorage, ownerKind string, ownerID string) LedgerStorageResult {
	entriesResult := ListLedgerEntries(storage, ownerKind, ownerID, StoredListPage{limit: 1000000, offset: 0})
	entries, entriesMatched := entriesResult.(LedgerEntriesStored)
	if !entriesMatched {
		return entriesResult
	}
	var amount int64
	for index := range entries.Values {
		amount += entries.Values[index].Amount
	}
	return LedgerBalanceStored{Amount: amount}
}

func loadLedgerEntry(storage BrowserStorage, ledgerID string) LedgerStorageResult {
	keyResult := NewStorageKey("ledger:" + strings.TrimSpace(ledgerID))
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return LedgerStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return LedgerStorageRejected{Reason: storageReadReason(readResult, "ledger entry")}
	}
	var entry StoredLedgerEntry
	if err := json.Unmarshal([]byte(read.Value), &entry); err != nil {
		return LedgerStorageRejected{Reason: "ledger entry decoding failed"}
	}
	cleaned := cleanStoredLedgerEntry(entry)
	if cleaned.ID != strings.TrimSpace(ledgerID) {
		return LedgerStorageRejected{Reason: "ledger storage key contains mismatched record"}
	}
	if reason := validateStoredLedgerEntry(cleaned); reason != "" {
		return LedgerStorageRejected{Reason: reason}
	}
	return LedgerEntryStored{Value: cleaned}
}

func cleanStoredLedgerEntry(entry StoredLedgerEntry) StoredLedgerEntry {
	return StoredLedgerEntry{
		ID:             strings.TrimSpace(entry.ID),
		OwnerKind:      strings.TrimSpace(entry.OwnerKind),
		OwnerID:        strings.TrimSpace(entry.OwnerID),
		Kind:           strings.TrimSpace(entry.Kind),
		Amount:         entry.Amount,
		TaskID:         strings.TrimSpace(entry.TaskID),
		IdempotencyKey: strings.TrimSpace(entry.IdempotencyKey),
	}
}

func validateStoredLedgerEntry(entry StoredLedgerEntry) string {
	if entry.ID == "" {
		return "ledger entry id is required"
	}
	if !validStoredLedgerOwnerKind(entry.OwnerKind) {
		return "ledger owner kind is invalid"
	}
	if entry.OwnerID == "" {
		return "ledger owner id is required"
	}
	if entry.Kind == "" {
		return "ledger entry kind is required"
	}
	return ""
}

func validStoredLedgerOwnerKind(value string) bool {
	switch value {
	case "user", "organization":
		return true
	default:
		return false
	}
}
