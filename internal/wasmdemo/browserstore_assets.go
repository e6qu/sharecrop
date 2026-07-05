package wasmdemo

import (
	"context"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
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

func (store AssetBrowserStore) loadCollectible(id string) (storedCollectible, bool, *core.DomainError) {
	var record storedCollectible
	found, ok := getTaskJSON(store.storage, collectibleRecordKey(id), &record)
	if !ok {
		reason := invalidState("collectible lookup failed")
		return storedCollectible{}, false, &reason
	}
	return record, found, nil
}

func (store AssetBrowserStore) saveCollectible(record storedCollectible) bool {
	return putTaskJSON(store.storage, collectibleRecordKey(record.ID), record)
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
	if !found || record.CreatedBy != command.RequesterUserID.String() {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "only the task owner can refund the task")}
	}
	if record.State != "draft" && record.State != "open" {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only draft or open tasks can be refunded")}
	}
	if record.RewardKind == "bundle" {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "bundled rewards must be refunded together")}
	}

	refunded, refundErr := store.releaseHeldCollectibleReward(command.TaskID.String())
	if refundErr != nil {
		return assets.RefundRewardRejected{Reason: *refundErr}
	}
	if len(refunded) == 0 {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task has no collectible reward to refund")}
	}

	record.State = "cancelled"
	if !saveStoredTaskRecord(store.storage, record) {
		return assets.RefundRewardRejected{Reason: invalidState("cancel task failed")}
	}
	return assets.RewardRefunded{Values: refunded}
}

// releaseHeldCollectibleReward returns every held collectible reward on a
// task to "minted" and reports the released collectibles - shared by
// RefundCollectibleReward (a genuine collectible-reward refund request) and
// LedgerBrowserStore.RefundTask (a credit refund on a bundle task, which
// must also release any collectible half).
func (store AssetBrowserStore) releaseHeldCollectibleReward(taskID string) ([]assets.Collectible, *core.DomainError) {
	indexResult := loadStringIndex(store.storage, taskCollectibleRewardIndexKey(taskID), "task collectible reward")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return nil, &reason
	}
	released := make([]assets.Collectible, 0, len(loaded.values))
	for _, collectibleID := range loaded.values {
		collectible, found, err := store.loadCollectible(collectibleID)
		if err != nil {
			return nil, err
		}
		if !found || collectible.State != assets.CollectibleStateEscrowed.String() {
			continue
		}
		collectible.State = assets.CollectibleStateMinted.String()
		if !store.saveCollectible(collectible) {
			reason := invalidState("return collectible failed")
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

	collectible.OwnerKind = assets.CollectibleOwnerKindUser
	collectible.OwnerID = command.ToUserID.String()
	if !store.saveCollectible(collectible) {
		return assets.GiftRejected{Reason: invalidState("transfer collectible failed")}
	}
	value, parseErr := parseStoredCollectible(collectible)
	if parseErr != nil {
		return assets.GiftRejected{Reason: *parseErr}
	}
	return assets.CollectibleGifted{Value: value}
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

	var membershipID string
	memberFound, ok := getTaskJSON(store.storage, orgActiveMembershipKey(command.OrganizationID.String(), command.RecipientUserID.String()), &membershipID)
	if !ok {
		return assets.GiftRejected{Reason: invalidState("check organization membership failed")}
	}
	isActiveMember := false
	if memberFound {
		var membership storedMembership
		membershipFound, membershipOK := getTaskJSON(store.storage, orgMembershipKey(membershipID), &membership)
		isActiveMember = membershipOK && membershipFound && membership.Status == org.MembershipStatusActive.String()
	}
	if !isActiveMember {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "recipient is not an active member of this organization")}
	}

	collectible.OwnerKind = assets.CollectibleOwnerKindUser
	collectible.OwnerID = command.RecipientUserID.String()
	collectible.OrganizationID = ""
	if !store.saveCollectible(collectible) {
		return assets.GiftRejected{Reason: invalidState("transfer organization collectible failed")}
	}
	value, parseErr := parseStoredCollectible(collectible)
	if parseErr != nil {
		return assets.GiftRejected{Reason: *parseErr}
	}
	return assets.CollectibleGifted{Value: value}
}
