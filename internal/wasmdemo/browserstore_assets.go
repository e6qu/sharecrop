package wasmdemo

import (
	"context"
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
)

// AssetBrowserStore implements assets.Store against BrowserStorage. Its
// reward-funding methods (FundCollectibleReward/RefundCollectibleReward)
// read/write the shared storedTaskRecord (browserstore_task_shared.go) the
// same way LedgerBrowserStore's credit funding does, since the real
// Postgres store touches the tasks table directly for both.
type AssetBrowserStore struct {
	storage BrowserStorage
	ids     InteractionIDSource
}

func NewAssetBrowserStore(storage BrowserStorage, ids InteractionIDSource) AssetBrowserStore {
	return AssetBrowserStore{storage: storage, ids: ids}
}

type storedCollectible struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	State          string `json:"state"`
	Policy         string `json:"policy"`
	OwnerKind      string `json:"owner_kind"`
	OwnerID        string `json:"owner_id"`
	OrganizationID string `json:"organization_id,omitempty"`
	Art            string `json:"art"`
}

func collectibleRecordKey(id string) string { return "assets:collectible:" + id }
func collectibleOwnerIndexKey(ownerKind string, ownerID string) string {
	return "assets:collectible_index:" + ownerKind + ":" + ownerID
}
func taskCollectibleRewardIndexKey(taskID string) string {
	return "assets:task_reward_index:" + taskID
}
func taskCollectibleRewardRecordKey(taskID string, collectibleID string) string {
	return "assets:task_reward:" + taskID + ":" + collectibleID
}

// storedTaskFundCollectible mirrors internal/db's task_fund_collectibles row: a
// stateless record that exists iff the collectible is currently held for the
// task's reward. It is deleted on award or refund. The escrowed collectible
// keeps recording its funder as owner, so a refund returns it there.
type storedTaskFundCollectible struct {
	TaskID        string `json:"task_id"`
	CollectibleID string `json:"collectible_id"`
}

func putStoredTaskFundCollectibleJSON(storage BrowserStorage, rawKey string, record storedTaskFundCollectible) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredTaskFundCollectibleJSON(storage BrowserStorage, rawKey string) (storedTaskFundCollectible, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedTaskFundCollectible{}, found, ok
	}
	if raw == "" {
		return storedTaskFundCollectible{}, false, true
	}
	var record storedTaskFundCollectible
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedTaskFundCollectible{}, false, false
	}
	return record, true, true
}

// loadTaskFundCollectibles returns every held collectible-reward record on a
// task in escrow order, failing loudly on a broken index or a missing record.
func loadTaskFundCollectibles(storage BrowserStorage, taskID string) ([]storedTaskFundCollectible, *core.DomainError) {
	indexResult := loadStringIndex(storage, taskCollectibleRewardIndexKey(taskID), "task collectible reward")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return nil, &reason
	}
	rewards := make([]storedTaskFundCollectible, 0, len(loaded.values))
	for _, collectibleID := range loaded.values {
		record, found, ok := getStoredTaskFundCollectibleJSON(storage, taskCollectibleRewardRecordKey(taskID, collectibleID))
		if !ok {
			reason := invalidState("read collectible reward failed")
			return nil, &reason
		}
		if !found {
			// A deleted (awarded/refunded) reward leaves a tombstone in the
			// index; skip it rather than failing.
			continue
		}
		rewards = append(rewards, record)
	}
	return rewards, nil
}

// countHeldCollectibleRewards counts the collectibles currently held for a
// task's reward, used by the display count and the open/cancel invariants.
func countHeldCollectibleRewards(storage BrowserStorage, taskID string) (int, *core.DomainError) {
	rewards, err := loadTaskFundCollectibles(storage, taskID)
	if err != nil {
		return 0, err
	}
	return len(rewards), nil
}

func putStoredCollectibleJSON(storage BrowserStorage, rawKey string, record storedCollectible) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredCollectibleJSON(storage BrowserStorage, rawKey string) (storedCollectible, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedCollectible{}, found, ok
	}
	var record storedCollectible
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedCollectible{}, false, false
	}
	return record, true, true
}

func (store AssetBrowserStore) loadCollectible(id string) (storedCollectible, bool, *core.DomainError) {
	record, found, ok := getStoredCollectibleJSON(store.storage, collectibleRecordKey(id))
	if !ok {
		reason := invalidState("collectible lookup failed")
		return storedCollectible{}, false, &reason
	}
	return record, found, nil
}

func (store AssetBrowserStore) saveCollectible(record storedCollectible) bool {
	return putStoredCollectibleJSON(store.storage, collectibleRecordKey(record.ID), record)
}

func parseStoredCollectible(record storedCollectible) (assets.Collectible, *core.DomainError) {
	idResult := core.ParseCollectibleID(record.ID)
	id, idMatched := idResult.(core.CollectibleIDCreated)
	if !idMatched {
		reason := idResult.(core.CollectibleIDRejected).Reason
		return assets.Collectible{}, &reason
	}
	nameResult := assets.NewCollectibleName(record.Name)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		reason := nameResult.(assets.CollectibleNameRejected).Reason
		return assets.Collectible{}, &reason
	}
	kindResult := assets.ParseCollectibleKind(record.Kind)
	kind, kindMatched := kindResult.(assets.CollectibleKindAccepted)
	if !kindMatched {
		reason := kindResult.(assets.CollectibleKindRejected).Reason
		return assets.Collectible{}, &reason
	}
	stateResult := assets.ParseCollectibleState(record.State)
	state, stateMatched := stateResult.(assets.CollectibleStateAccepted)
	if !stateMatched {
		reason := stateResult.(assets.CollectibleStateRejected).Reason
		return assets.Collectible{}, &reason
	}
	policyResult := assets.ParseTransferPolicy(record.Policy)
	policy, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		reason := policyResult.(assets.TransferPolicyRejected).Reason
		return assets.Collectible{}, &reason
	}
	return assets.Collectible{
		ID: id.Value, Name: name.Value, Kind: kind.Value, State: state.Value, Policy: policy.Value,
		OwnerKind: record.OwnerKind, OwnerID: record.OwnerID, OrganizationID: record.OrganizationID, Art: record.Art,
	}, nil
}

func (store AssetBrowserStore) CreateCollectible(_ context.Context, collectible assets.Collectible) assets.CreateStoreResult {
	record := storedCollectible{
		ID: collectible.ID.String(), Name: collectible.Name.String(), Kind: collectible.Kind.String(), State: collectible.State.String(),
		Policy: collectible.Policy.String(), OwnerKind: collectible.OwnerKind, OwnerID: collectible.OwnerID,
		OrganizationID: collectible.OrganizationID, Art: collectible.Art,
	}
	if !store.saveCollectible(record) {
		return assets.CreateStoreRejected{Reason: invalidState("insert collectible failed")}
	}
	if _, matched := appendStringIndex(store.storage, collectibleOwnerIndexKey(record.OwnerKind, record.OwnerID), record.ID, "collectible").(stringIndexStored); !matched {
		return assets.CreateStoreRejected{Reason: invalidState("update collectible index failed")}
	}
	return assets.CreateStoreAccepted{}
}

func (store AssetBrowserStore) listByOwner(ownerKind string, ownerID string, page core.Page) assets.ListStoreResult {
	indexResult := loadStringIndex(store.storage, collectibleOwnerIndexKey(ownerKind, ownerID), "collectible")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return assets.ListStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}
	// Newest first, matching internal/db's `order by created_at desc, id`.
	values := make([]assets.Collectible, 0, len(loaded.values))
	for index := len(loaded.values) - 1; index >= 0; index-- {
		record, found, err := store.loadCollectible(loaded.values[index])
		if err != nil {
			return assets.ListStoreRejected{Reason: *err}
		}
		if !found || record.OwnerKind != ownerKind || record.OwnerID != ownerID {
			continue
		}
		value, parseErr := parseStoredCollectible(record)
		if parseErr != nil {
			return assets.ListStoreRejected{Reason: *parseErr}
		}
		values = append(values, value)
	}
	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	return assets.ListStoreListed{Values: values[start:end]}
}

func (store AssetBrowserStore) ListCollectibles(_ context.Context, owner core.UserID, page core.Page) assets.ListStoreResult {
	return store.listByOwner(assets.CollectibleOwnerKindUser, owner.String(), page)
}

func (store AssetBrowserStore) ListCollectiblesByOwner(_ context.Context, ownerKind string, ownerID string, page core.Page) assets.ListStoreResult {
	return store.listByOwner(ownerKind, ownerID, page)
}

func (store AssetBrowserStore) FundCollectibleReward(_ context.Context, command assets.FundRewardStoreCommand) assets.FundRewardResult {
	collectible, found, err := store.loadCollectible(command.CollectibleID.String())
	if err != nil {
		return assets.FundRewardRejected{Reason: *err}
	}
	if !found || collectible.OwnerKind != assets.CollectibleOwnerKindUser || collectible.OwnerID != command.FunderUserID.String() {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "only the collectible owner can fund a task with it")}
	}
	if collectible.State != assets.CollectibleStateMinted.String() {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to escrow")}
	}
	policyResult := assets.ParseTransferPolicy(collectible.Policy)
	policyAccepted, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		return assets.FundRewardRejected{Reason: policyResult.(assets.TransferPolicyRejected).Reason}
	}
	if denied, matched := assets.AllowsRewardPayout(policyAccepted.Value).(assets.RewardDenied); matched {
		return assets.FundRewardRejected{Reason: denied.Reason}
	}

	record, taskFound, taskErr := loadStoredTaskRecord(store.storage, command.TaskID.String())
	if taskErr != nil {
		return assets.FundRewardRejected{Reason: *taskErr}
	}
	if !taskFound || record.CreatedBy != command.FunderUserID.String() {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "only the task owner can fund the task")}
	}
	if record.State != "draft" {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only draft tasks can be funded")}
	}

	// Mirror internal/db's (task_id, collectible_id) uniqueness constraint: the
	// same collectible can never be escrowed twice on the same task,
	// whatever reward state the earlier escrow reached.
	_, rewardExists, rewardOK := getStoredTaskFundCollectibleJSON(store.storage, taskCollectibleRewardRecordKey(command.TaskID.String(), command.CollectibleID.String()))
	if !rewardOK {
		return assets.FundRewardRejected{Reason: invalidState("read collectible reward failed")}
	}
	if rewardExists {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "collectible is already escrowed on this task")}
	}

	switch record.RewardKind {
	case "none":
		record.RewardKind = "collectible"
	case "credit":
		record.RewardKind = "bundle"
	}
	if !saveStoredTaskRecord(store.storage, record) {
		return assets.FundRewardRejected{Reason: invalidState("update task reward kind failed")}
	}

	collectible.State = assets.CollectibleStateEscrowed.String()
	if !store.saveCollectible(collectible) {
		return assets.FundRewardRejected{Reason: invalidState("escrow collectible failed")}
	}
	reward := storedTaskFundCollectible{
		TaskID: command.TaskID.String(), CollectibleID: command.CollectibleID.String(),
	}
	if !putStoredTaskFundCollectibleJSON(store.storage, taskCollectibleRewardRecordKey(reward.TaskID, reward.CollectibleID), reward) {
		return assets.FundRewardRejected{Reason: invalidState("insert collectible reward failed")}
	}
	if _, matched := appendStringIndex(store.storage, taskCollectibleRewardIndexKey(command.TaskID.String()), command.CollectibleID.String(), "task collectible reward").(stringIndexStored); !matched {
		return assets.FundRewardRejected{Reason: invalidState("insert collectible reward failed")}
	}

	awarded, parseErr := parseStoredCollectible(collectible)
	if parseErr != nil {
		return assets.FundRewardRejected{Reason: *parseErr}
	}
	return assets.RewardFunded{Value: awarded}
}

func (store AssetBrowserStore) RefundCollectibleReward(_ context.Context, command assets.RefundRewardStoreCommand) assets.RefundRewardResult {
	record, found, taskErr := loadStoredTaskRecord(store.storage, command.TaskID.String())
	if taskErr != nil {
		return assets.RefundRewardRejected{Reason: *taskErr}
	}
	if !found {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if reason := requireRefundAuthorizedBrowser(store.storage, record, command.RequesterUserID); reason != nil {
		return assets.RefundRewardRejected{Reason: *reason}
	}
	if record.State != "draft" && record.State != "open" {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only tasks that are not yet awarded can be refunded")}
	}
	if record.RewardKind == "bundle" {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "bundled rewards must be refunded together")}
	}

	rewards, rewardsErr := loadTaskFundCollectibles(store.storage, command.TaskID.String())
	if rewardsErr != nil {
		return assets.RefundRewardRejected{Reason: *rewardsErr}
	}
	if len(rewards) == 0 {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task has no collectible reward to refund")}
	}

	refunded, refundErr := store.releaseHeldCollectibleReward(command.TaskID.String())
	if refundErr != nil {
		return assets.RefundRewardRejected{Reason: *refundErr}
	}

	record.State = "cancelled"
	if !saveStoredTaskRecord(store.storage, record) {
		return assets.RefundRewardRejected{Reason: invalidState("cancel task failed")}
	}
	return assets.RewardRefunded{Values: refunded}
}

// deleteTaskFundCollectible removes a held collectible-reward record and its
// index entry, used on award/refund when the collectible changes hands.
func deleteTaskFundCollectible(storage BrowserStorage, taskID string, collectibleID string) bool {
	if !removeFromStringIndex(storage, taskCollectibleRewardIndexKey(taskID), collectibleID) {
		return false
	}
	return putStorageString(storage, taskCollectibleRewardRecordKey(taskID, collectibleID), "")
}

// releaseHeldCollectibleReward returns every held collectible reward on a
// task to "minted" (its escrowed owner is already the funder) and deletes the
// reward records - shared by RefundCollectibleReward (a genuine collectible
// -reward refund request) and LedgerBrowserStore.RefundTask (a credit refund
// on a bundle task, which must also release its collectible half).
func (store AssetBrowserStore) releaseHeldCollectibleReward(taskID string) ([]assets.Collectible, *core.DomainError) {
	rewards, rewardsErr := loadTaskFundCollectibles(store.storage, taskID)
	if rewardsErr != nil {
		return nil, rewardsErr
	}
	released := make([]assets.Collectible, 0, len(rewards))
	for _, reward := range rewards {
		collectible, found, err := store.loadCollectible(reward.CollectibleID)
		if err != nil {
			return nil, err
		}
		if !found {
			reason := invalidState("return collectible failed")
			return nil, &reason
		}
		collectible.State = assets.CollectibleStateMinted.String()
		if !store.saveCollectible(collectible) {
			reason := invalidState("return collectible failed")
			return nil, &reason
		}
		if !deleteTaskFundCollectible(store.storage, reward.TaskID, reward.CollectibleID) {
			reason := invalidState("clear collectible reward failed")
			return nil, &reason
		}
		value, parseErr := parseStoredCollectible(collectible)
		if parseErr != nil {
			return nil, parseErr
		}
		released = append(released, value)
	}
	return released, nil
}

// payOutHeldCollectibleReward transfers every held collectible reward on a
// task to the accepted worker (escrowed -> minted, owner = worker) and deletes
// the reward records, mirroring internal/db's payOutCollectible - used by
// LedgerBrowserStore.AcceptSubmission for collectible/bundle reward tasks.
func (store AssetBrowserStore) payOutHeldCollectibleReward(taskID string, workerUserID string) ([]core.CollectibleID, *core.DomainError) {
	rewards, rewardsErr := loadTaskFundCollectibles(store.storage, taskID)
	if rewardsErr != nil {
		return nil, rewardsErr
	}
	awarded := make([]core.CollectibleID, 0, len(rewards))
	for _, reward := range rewards {
		collectible, found, err := store.loadCollectible(reward.CollectibleID)
		if err != nil {
			return nil, err
		}
		if !found {
			reason := invalidState("award collectible failed")
			return nil, &reason
		}
		previousOwnerKind, previousOwnerID := collectible.OwnerKind, collectible.OwnerID
		collectible.State = assets.CollectibleStateMinted.String()
		collectible.OwnerKind = assets.CollectibleOwnerKindUser
		collectible.OwnerID = workerUserID
		if !store.saveCollectible(collectible) {
			reason := invalidState("award collectible failed")
			return nil, &reason
		}
		if !store.moveCollectibleOwnerIndex(collectible.ID, previousOwnerKind, previousOwnerID, collectible.OwnerKind, collectible.OwnerID) {
			reason := invalidState("update collectible index failed")
			return nil, &reason
		}
		if !deleteTaskFundCollectible(store.storage, reward.TaskID, reward.CollectibleID) {
			reason := invalidState("clear collectible reward failed")
			return nil, &reason
		}
		idResult := core.ParseCollectibleID(reward.CollectibleID)
		id, idMatched := idResult.(core.CollectibleIDCreated)
		if !idMatched {
			reason := idResult.(core.CollectibleIDRejected).Reason
			return nil, &reason
		}
		awarded = append(awarded, id.Value)
	}
	return awarded, nil
}

func (store AssetBrowserStore) GiftCollectible(_ context.Context, command assets.GiftStoreCommand) assets.GiftResult {
	collectible, found, err := store.loadCollectible(command.CollectibleID.String())
	if err != nil {
		return assets.GiftRejected{Reason: *err}
	}
	if !found {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to tip")}
	}
	// Idempotent replay: already gifted to the recipient.
	if collectible.OwnerKind == assets.CollectibleOwnerKindUser && collectible.OwnerID == command.ToUserID.String() {
		value, parseErr := parseStoredCollectible(collectible)
		if parseErr != nil {
			return assets.GiftRejected{Reason: *parseErr}
		}
		return assets.CollectibleGifted{Value: value}
	}
	if collectible.OwnerKind != assets.CollectibleOwnerKindUser || collectible.OwnerID != command.FromUserID.String() {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to tip")}
	}
	if collectible.State != assets.CollectibleStateMinted.String() {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to tip")}
	}
	policyResult := assets.ParseTransferPolicy(collectible.Policy)
	policyAccepted, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		return assets.GiftRejected{Reason: policyResult.(assets.TransferPolicyRejected).Reason}
	}
	if denied, matched := assets.AllowsTip(policyAccepted.Value).(assets.RewardDenied); matched {
		return assets.GiftRejected{Reason: denied.Reason}
	}
	if policyAccepted.Value == assets.TransferPolicyTransferableWithinOrg {
		if collectible.OrganizationID == "" {
			return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "within-organization collectible has no organization")}
		}
		for _, userID := range []string{command.FromUserID.String(), command.ToUserID.String()} {
			active, ok := isActiveOrgMember(store.storage, collectible.OrganizationID, userID)
			if !ok {
				return assets.GiftRejected{Reason: invalidState("check collectible organization membership failed")}
			}
			if !active {
				return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "within-organization collectible can only be tipped between organization members")}
			}
		}
	}

	previousOwnerKind, previousOwnerID := collectible.OwnerKind, collectible.OwnerID
	collectible.OwnerKind = assets.CollectibleOwnerKindUser
	collectible.OwnerID = command.ToUserID.String()
	if !store.saveCollectible(collectible) {
		return assets.GiftRejected{Reason: invalidState("transfer collectible failed")}
	}
	if !store.moveCollectibleOwnerIndex(collectible.ID, previousOwnerKind, previousOwnerID, collectible.OwnerKind, collectible.OwnerID) {
		return assets.GiftRejected{Reason: invalidState("update collectible index failed")}
	}
	value, parseErr := parseStoredCollectible(collectible)
	if parseErr != nil {
		return assets.GiftRejected{Reason: *parseErr}
	}
	return assets.CollectibleGifted{Value: value}
}

// moveCollectibleOwnerIndex keeps collectibleOwnerIndexKey in sync with a
// collectible's current owner: every method that changes ownership
// (GiftCollectible, AwardOrganizationCollectible, reward payout) must remove
// the id from its previous owner's index and add it to the new owner's,
// otherwise the collectible silently disappears from every owner's listing
// (listByOwner filters out index entries whose current owner has since
// changed - it never re-homes them).
func (store AssetBrowserStore) moveCollectibleOwnerIndex(collectibleID string, previousOwnerKind string, previousOwnerID string, newOwnerKind string, newOwnerID string) bool {
	if previousOwnerKind == newOwnerKind && previousOwnerID == newOwnerID {
		return true
	}
	if !removeFromStringIndex(store.storage, collectibleOwnerIndexKey(previousOwnerKind, previousOwnerID), collectibleID) {
		return false
	}
	_, matched := appendStringIndex(store.storage, collectibleOwnerIndexKey(newOwnerKind, newOwnerID), collectibleID, "collectible").(stringIndexStored)
	return matched
}

func (store AssetBrowserStore) AwardOrganizationCollectible(_ context.Context, command assets.AwardOrganizationCollectibleStoreCommand) assets.GiftResult {
	collectible, found, err := store.loadCollectible(command.CollectibleID.String())
	if err != nil {
		return assets.GiftRejected{Reason: *err}
	}
	if !found || collectible.OwnerKind != assets.CollectibleOwnerKindOrganization || collectible.OrganizationID != command.OrganizationID.String() {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible does not belong to this organization")}
	}
	if collectible.State != assets.CollectibleStateMinted.String() {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to award")}
	}

	isActiveMember, ok := isActiveOrgMember(store.storage, command.OrganizationID.String(), command.RecipientUserID.String())
	if !ok {
		return assets.GiftRejected{Reason: invalidState("check organization membership failed")}
	}
	if !isActiveMember {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "recipient is not an active member of this organization")}
	}

	previousOwnerKind, previousOwnerID := collectible.OwnerKind, collectible.OwnerID
	collectible.OwnerKind = assets.CollectibleOwnerKindUser
	collectible.OwnerID = command.RecipientUserID.String()
	collectible.OrganizationID = ""
	if !store.saveCollectible(collectible) {
		return assets.GiftRejected{Reason: invalidState("transfer organization collectible failed")}
	}
	if !store.moveCollectibleOwnerIndex(collectible.ID, previousOwnerKind, previousOwnerID, collectible.OwnerKind, collectible.OwnerID) {
		return assets.GiftRejected{Reason: invalidState("update collectible index failed")}
	}
	value, parseErr := parseStoredCollectible(collectible)
	if parseErr != nil {
		return assets.GiftRejected{Reason: *parseErr}
	}
	return assets.CollectibleGifted{Value: value}
}
