package assets

import (
	"context"

	"github.com/e6qu/sharecrop/internal/core"
)

type Store interface {
	CreateCollectible(context.Context, Collectible) CreateStoreResult
	ListCollectibles(context.Context, core.UserID, core.Page) ListStoreResult
	FundCollectibleReward(context.Context, FundRewardStoreCommand) FundRewardResult
	RefundCollectibleReward(context.Context, RefundRewardStoreCommand) RefundRewardResult
	GiftCollectible(context.Context, GiftStoreCommand) GiftResult
}

// GiftStoreCommand carries a validated collectible tip (a voluntary transfer of
// an owned collectible from one user to another).
type GiftStoreCommand struct {
	FromUserID    core.UserID
	ToUserID      core.UserID
	CollectibleID core.CollectibleID
}

// FundRewardStoreCommand carries a validated collectible-reward funding request.
type FundRewardStoreCommand struct {
	FunderUserID  core.UserID
	TaskID        core.TaskID
	CollectibleID core.CollectibleID
}

// RefundRewardStoreCommand carries a validated collectible-reward refund request.
type RefundRewardStoreCommand struct {
	RequesterUserID core.UserID
	TaskID          core.TaskID
}

type Service struct {
	store Store
}

func NewService(store Store) Service {
	return Service{store: store}
}

type MintResult interface {
	mintResult()
}

type CollectibleMinted struct {
	Value Collectible
}

type MintRejected struct {
	Reason core.DomainError
}

func (CollectibleMinted) mintResult() {}

func (MintRejected) mintResult() {}

func (service Service) Mint(ctx context.Context, owner core.UserID, name CollectibleName, kind CollectibleKind, policy TransferPolicy, art string) MintResult {
	idResult := core.NewCollectibleID()
	idCreated, matched := idResult.(core.CollectibleIDCreated)
	if !matched {
		return MintRejected{Reason: idResult.(core.CollectibleIDRejected).Reason}
	}

	collectible := Collectible{
		ID:      idCreated.Value,
		Name:    name,
		Kind:    kind,
		State:   CollectibleStateMinted,
		Policy:  policy,
		OwnerID: owner,
		Art:     art,
	}
	storeResult := service.store.CreateCollectible(ctx, collectible)
	if rejected, rejectedMatched := storeResult.(CreateStoreRejected); rejectedMatched {
		return MintRejected{Reason: rejected.Reason}
	}
	return CollectibleMinted{Value: collectible}
}

type ListResult interface {
	listResult()
}

type CollectiblesListed struct {
	Values []Collectible
}

type ListRejected struct {
	Reason core.DomainError
}

func (CollectiblesListed) listResult() {}

func (ListRejected) listResult() {}

func (service Service) ListCollectibles(ctx context.Context, owner core.UserID, page core.Page) ListResult {
	storeResult := service.store.ListCollectibles(ctx, owner, page)
	listed, matched := storeResult.(ListStoreListed)
	if !matched {
		return ListRejected{Reason: storeResult.(ListStoreRejected).Reason}
	}
	return CollectiblesListed{Values: listed.Values}
}

type FundRewardResult interface {
	fundRewardResult()
}

type RewardFunded struct {
	Value Collectible
}

type FundRewardRejected struct {
	Reason core.DomainError
}

func (RewardFunded) fundRewardResult() {}

func (FundRewardRejected) fundRewardResult() {}

type GiftResult interface {
	giftResult()
}

type CollectibleGifted struct {
	Value Collectible
}

type GiftRejected struct {
	Reason core.DomainError
}

func (CollectibleGifted) giftResult() {}

func (GiftRejected) giftResult() {}

// GiftCollectible transfers an owned, transferable collectible to another user
// (a review tip). Ownership, availability, and transfer policy are enforced in
// the store transaction.
func (service Service) GiftCollectible(ctx context.Context, from core.UserID, to core.UserID, collectibleID core.CollectibleID) GiftResult {
	return service.store.GiftCollectible(ctx, GiftStoreCommand{
		FromUserID:    from,
		ToUserID:      to,
		CollectibleID: collectibleID,
	})
}

func (service Service) FundReward(ctx context.Context, funder core.UserID, taskID core.TaskID, collectibleID core.CollectibleID) FundRewardResult {
	return service.store.FundCollectibleReward(ctx, FundRewardStoreCommand{
		FunderUserID:  funder,
		TaskID:        taskID,
		CollectibleID: collectibleID,
	})
}

type RefundRewardResult interface {
	refundRewardResult()
}

type RewardRefunded struct {
	Values []Collectible
}

type RefundRewardRejected struct {
	Reason core.DomainError
}

func (RewardRefunded) refundRewardResult() {}

func (RefundRewardRejected) refundRewardResult() {}

func (service Service) RefundReward(ctx context.Context, requester core.UserID, taskID core.TaskID) RefundRewardResult {
	return service.store.RefundCollectibleReward(ctx, RefundRewardStoreCommand{
		RequesterUserID: requester,
		TaskID:          taskID,
	})
}
