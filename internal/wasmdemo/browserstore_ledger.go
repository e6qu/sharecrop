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
// cover credit/collectible/bundle payouts and credit/collectible tips, same
// as internal/db. Implementor bans (BanSelection) are not modeled - no
// held ban state exists yet in the browser stores - so a ban selection is
// silently treated as "no ban" rather than rejected outright, since a ban
// is an additional restriction on top of the review outcome, not something
// the reviewer depends on succeeding.
type LedgerBrowserStore struct {
	storage BrowserStorage
	ids     InteractionIDSource
}

func NewLedgerBrowserStore(storage BrowserStorage, ids InteractionIDSource) LedgerBrowserStore {
	return LedgerBrowserStore{storage: storage, ids: ids}
}

type storedTaskEscrow struct {
	TaskID          string `json:"task_id"`
	FunderOwnerKind string `json:"funder_owner_kind"`
	FunderOwnerID   string `json:"funder_owner_id"`
	Amount          int64  `json:"amount"`
	State           string `json:"state"`
}

func taskEscrowKey(taskID string) string { return "ledger:escrow:" + taskID }

func putStoredTaskEscrowJSON(storage BrowserStorage, rawKey string, record storedTaskEscrow) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredTaskEscrowJSON(storage BrowserStorage, rawKey string) (storedTaskEscrow, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedTaskEscrow{}, found, ok
	}
	var record storedTaskEscrow
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedTaskEscrow{}, false, false
	}
	return record, true, true
}

func (store LedgerBrowserStore) loadEscrow(taskID string) (storedTaskEscrow, bool, *core.DomainError) {
	record, found, ok := getStoredTaskEscrowJSON(store.storage, taskEscrowKey(taskID))
	if !ok {
		reason := invalidState("read task escrow failed")
		return storedTaskEscrow{}, false, &reason
	}
	return record, found, nil
}

func (store LedgerBrowserStore) saveEscrow(record storedTaskEscrow) bool {
	return putStoredTaskEscrowJSON(store.storage, taskEscrowKey(record.TaskID), record)
}

// findEntryByIdempotencyKey scans a ledger owner's entries for one already
// recorded under this key, mirroring the real store's idempotency check.
func (store LedgerBrowserStore) findEntryByIdempotencyKey(ownerKind string, ownerID string, key string) (bool, *core.DomainError) {
	entriesResult := ListLedgerEntries(store.storage, ownerKind, ownerID, StoredListPage{limit: 1000000, offset: 0})
	entries, matched := entriesResult.(LedgerEntriesStored)
	if !matched {
		reason := invalidState("check funding idempotency failed")
		return false, &reason
	}
	for _, entry := range entries.Values {
		if entry.ID == "idempotency:"+key {
			return true, nil
		}
	}
	return false, nil
}

// requireFundableTask mirrors internal/db's requireFundableTask /
// requireCreditRewardFunding: the task must be draft, and a first-time
// credit funding may flip reward_kind (none -> credit, collectible ->
// bundle) - a task is always fundable by whoever is authorized to fund it,
// regardless of the reward kind it was created with.
func requireFundableTaskBrowser(storage BrowserStorage, record storedTaskRecord, amount int64) (storedTaskRecord, *core.DomainError) {
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
	existing, existingFound, existingErr := store.loadEscrow(taskID.String())
	if existingErr != nil {
		return ledger.FundRejected{Reason: *existingErr}
	}
	if existingFound {
		// A retry of the exact same request (matching idempotency key) replays
		// the original result instead of erroring - the real store's behavior.
		keyUsed, keyErr := store.findEntryByIdempotencyKey(ownerKind, ownerID, key.String())
		if keyErr != nil {
			return ledger.FundRejected{Reason: *keyErr}
		}
		if keyUsed {
			return ledger.TaskFunded{Escrow: buildLedgerTaskEscrow(existing)}
		}
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task is already funded")}
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

	updated, fundableErr := requireFundableTaskBrowser(store.storage, record, amount.Int64())
	if fundableErr != nil {
		return ledger.FundRejected{Reason: *fundableErr}
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
	escrow := storedTaskEscrow{TaskID: taskID.String(), FunderOwnerKind: ownerKind, FunderOwnerID: ownerID, Amount: amount.Int64(), State: ledger.EscrowStateHeld.String()}
	if !store.saveEscrow(escrow) {
		return ledger.FundRejected{Reason: invalidState("insert task escrow failed")}
	}
	entryResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID: entryID.String(), OwnerKind: ownerKind, OwnerID: ownerID,
		Kind: ledger.EntryKindTaskEscrow.String(), Amount: -amount.Int64(), TaskID: taskID.String(),
	})
	if _, matched := entryResult.(LedgerEntryStored); !matched {
		return ledger.FundRejected{Reason: invalidState("insert task escrow ledger entry failed")}
	}
	store.markIdempotencyKeyUsed(ownerKind, ownerID, key.String(), taskID.String())

	return ledger.TaskFunded{Escrow: buildLedgerTaskEscrow(escrow)}
}

// markIdempotencyKeyUsed records the key as its own zero-amount ledger entry
// so findEntryByIdempotencyKey can recognize a repeat call. Kept separate
// from the real accounting entry so the idempotency marker never affects a
// displayed balance.
func (store LedgerBrowserStore) markIdempotencyKeyUsed(ownerKind string, ownerID string, key string, taskID string) {
	SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID: "idempotency:" + key, OwnerKind: ownerKind, OwnerID: ownerID, Kind: "idempotency_marker", Amount: 0, TaskID: taskID,
	})
}

func buildLedgerTaskEscrow(record storedTaskEscrow) ledger.TaskEscrow {
	amountResult := ledger.NewCreditAmount(record.Amount)
	amount, _ := amountResult.(ledger.CreditAmountAccepted)
	stateResult := ledger.ParseEscrowState(record.State)
	state, _ := stateResult.(ledger.EscrowStateAccepted)
	taskIDResult := core.ParseTaskID(record.TaskID)
	taskID, _ := taskIDResult.(core.TaskIDCreated)
	return ledger.TaskEscrow{TaskID: taskID.Value, Amount: amount.Value, State: state.Value}
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
	if record.CreatedBy != command.RequesterUserID.String() {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "only the task owner can refund the task")}
	}
	if record.State != "draft" && record.State != "open" {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only draft or open tasks can be refunded")}
	}

	escrow, escrowFound, escrowErr := store.loadEscrow(command.TaskID.String())
	if escrowErr != nil {
		return ledger.RefundRejected{Reason: *escrowErr}
	}
	if !escrowFound {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task has no escrow to refund")}
	}
	if escrow.State == ledger.EscrowStateRefunded.String() {
		return ledger.TaskRefunded{Escrow: buildLedgerTaskEscrow(escrow)}
	}
	if escrow.State != ledger.EscrowStateHeld.String() {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task escrow is not held")}
	}

	entryResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID: command.EntryID.String(), OwnerKind: escrow.FunderOwnerKind, OwnerID: escrow.FunderOwnerID,
		Kind: ledger.EntryKindTaskRefund.String(), Amount: escrow.Amount, TaskID: command.TaskID.String(),
	})
	if _, matched := entryResult.(LedgerEntryStored); !matched {
		return ledger.RefundRejected{Reason: invalidState("insert task refund ledger entry failed")}
	}

	escrow.State = ledger.EscrowStateRefunded.String()
	if !store.saveEscrow(escrow) {
		return ledger.RefundRejected{Reason: invalidState("update task escrow failed")}
	}

	if reason := refundHeldCollectibleRewardBrowser(store.storage, store.ids, command.TaskID.String()); reason != nil {
		return ledger.RefundRejected{Reason: *reason}
	}

	record.State = "cancelled"
	if !saveStoredTaskRecord(store.storage, record) {
		return ledger.RefundRejected{Reason: invalidState("cancel task failed")}
	}

	return ledger.TaskRefunded{Escrow: buildLedgerTaskEscrow(escrow)}
}

func (store LedgerBrowserStore) Balance(_ context.Context, owner core.UserID) ledger.BalanceResult {
	result := LedgerBalance(store.storage, "user", owner.String())
	stored, matched := result.(LedgerBalanceStored)
	if !matched {
		return ledger.BalanceRejected{Reason: invalidState("read balance failed")}
	}
	return ledger.BalanceFound{Value: ledger.NewBalance(stored.Amount)}
}

func (store LedgerBrowserStore) OrganizationBalance(_ context.Context, organizationID core.OrganizationID) ledger.BalanceResult {
	result := LedgerBalance(store.storage, "organization", organizationID.String())
	stored, matched := result.(LedgerBalanceStored)
	if !matched {
		return ledger.BalanceRejected{Reason: invalidState("read organization balance failed")}
	}
	return ledger.BalanceFound{Value: ledger.NewBalance(stored.Amount)}
}

func (store LedgerBrowserStore) listEntries(ownerKind string, ownerID string, page core.Page) ledger.ListEntriesResult {
	result := ListLedgerEntries(store.storage, ownerKind, ownerID, StoredListPage{limit: page.Limit(), offset: page.Offset()})
	stored, matched := result.(LedgerEntriesStored)
	if !matched {
		return ledger.ListEntriesRejected{Reason: invalidState("read ledger entries failed")}
	}
	values := make([]ledger.LedgerEntry, 0, len(stored.Values))
	for _, entry := range stored.Values {
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
	escrow, found, err := store.loadEscrow(taskID.String())
	if err != nil {
		return nil, err
	}
	_, partial := selection.(ledger.PartialCreditReviewSelection)
	if !found {
		if partial {
			reason := core.NewDomainError(core.ErrorCodeConflict, "credit reward escrow is missing")
			return nil, &reason
		}
		return ledger.NoPayout{}, nil
	}
	if escrow.State != ledger.EscrowStateHeld.String() {
		if partial {
			reason := core.NewDomainError(core.ErrorCodeConflict, "credit reward escrow is not held")
			return nil, &reason
		}
		return ledger.NoPayout{}, nil
	}

	payoutAmount := escrow.Amount
	if selected, matched := selection.(ledger.PartialCreditReviewSelection); matched {
		payoutAmount = selected.Amount.Int64()
	}
	if payoutAmount > escrow.Amount {
		reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "credit payout cannot exceed held escrow")
		return nil, &reason
	}

	entryResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID: payoutEntryID.String(), OwnerKind: "user", OwnerID: workerID.String(),
		Kind: ledger.EntryKindTaskPayout.String(), Amount: payoutAmount, TaskID: taskID.String(),
	})
	if _, matched := entryResult.(LedgerEntryStored); !matched {
		reason := invalidState("insert task payout ledger entry failed")
		return nil, &reason
	}

	remaining := escrow.Amount - payoutAmount
	if closeTask {
		if remaining > 0 {
			refundResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
				ID: refundEntryID.String(), OwnerKind: escrow.FunderOwnerKind, OwnerID: escrow.FunderOwnerID,
				Kind: ledger.EntryKindTaskRefund.String(), Amount: remaining, TaskID: taskID.String(),
			})
			if _, matched := refundResult.(LedgerEntryStored); !matched {
				reason := invalidState("insert partial accept refund ledger entry failed")
				return nil, &reason
			}
		}
		escrow.Amount = payoutAmount
		escrow.State = ledger.EscrowStateReleased.String()
	} else {
		escrow.State = ledger.EscrowStateHeld.String()
		escrow.Amount = remaining
		if remaining == 0 {
			escrow.State = ledger.EscrowStateReleased.String()
			escrow.Amount = payoutAmount
		}
	}
	if !store.saveEscrow(escrow) {
		reason := invalidState("update task escrow after review payout failed")
		return nil, &reason
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
		return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
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
			return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward escrow is missing")}
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
	ID        string `json:"id"`
	OwnerKind string `json:"owner_kind"`
	OwnerID   string `json:"owner_id"`
	Kind      string `json:"kind"`
	Amount    int64  `json:"amount"`
	TaskID    string `json:"task_id"`
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
		ID:        strings.TrimSpace(entry.ID),
		OwnerKind: strings.TrimSpace(entry.OwnerKind),
		OwnerID:   strings.TrimSpace(entry.OwnerID),
		Kind:      strings.TrimSpace(entry.Kind),
		Amount:    entry.Amount,
		TaskID:    strings.TrimSpace(entry.TaskID),
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
