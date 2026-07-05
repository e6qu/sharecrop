package wasmdemo

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
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
// Submission-review flows (AcceptSubmission/RequestChanges/RejectSubmission
// - tips, partial-credit selection, bans) are not yet implemented; they
// return a clear "not yet implemented" rejection rather than silently doing
// the wrong thing.
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

// Submission-review flows are not yet implemented in the browser store -
// they involve tips, partial-credit selection, and bans on top of the same
// task/escrow mechanics above, deferred to keep this change reviewable.
func (store LedgerBrowserStore) AcceptSubmission(context.Context, ledger.AcceptStoreCommand) ledger.AcceptResult {
	return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "accepting a submission through the browser demo is not yet implemented")}
}

func (store LedgerBrowserStore) RequestChanges(context.Context, ledger.RequestChangesStoreCommand) ledger.RequestChangesResult {
	return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "requesting submission changes through the browser demo is not yet implemented")}
}

func (store LedgerBrowserStore) RejectSubmission(context.Context, ledger.RejectStoreCommand) ledger.RejectResult {
	return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "rejecting a submission through the browser demo is not yet implemented")}
}

// refundHeldCollectibleRewardBrowser mirrors internal/db's
// refundHeldCollectibleReward: a bundle reward's held collectible half
// returns to "minted" alongside the credit refund below, so a credit refund
// on a bundle task doesn't strand the collectible half in escrow.
func refundHeldCollectibleRewardBrowser(storage BrowserStorage, ids InteractionIDSource, taskID string) *core.DomainError {
	_, err := (AssetBrowserStore{storage: storage, ids: ids}).releaseHeldCollectibleReward(taskID)
	return err
}
