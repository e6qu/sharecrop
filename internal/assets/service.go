package assets

import (
	"context"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
)

type Store interface {
	CreateCollectible(context.Context, Collectible) CreateStoreResult
	ListCollectibles(context.Context, core.UserID, core.Page) ListStoreResult
	ListCollectiblesByOwner(context.Context, string, string, core.Page) ListStoreResult
	FundCollectibleReward(context.Context, FundRewardStoreCommand) FundRewardResult
	RefundCollectibleReward(context.Context, RefundRewardStoreCommand) RefundRewardResult
	GiftCollectible(context.Context, GiftStoreCommand) GiftResult
	AwardOrganizationCollectible(context.Context, AwardOrganizationCollectibleStoreCommand) GiftResult
	TaskHeldCollectibles(context.Context, core.TaskID) TaskHeldCollectiblesResult
}

// AwardOrganizationCollectibleStoreCommand carries a validated request to
// transfer an organization-owned collectible to one of its active members.
type AwardOrganizationCollectibleStoreCommand struct {
	OrganizationID  core.OrganizationID
	CollectibleID   core.CollectibleID
	RecipientUserID core.UserID
	// Draft is the collectible_awarded event recorded inside the award
	// transaction.
	Draft event.Draft
}

// GiftStoreCommand carries a validated collectible tip (a voluntary transfer of
// an owned collectible from one user to another).
type GiftStoreCommand struct {
	FromUserID    core.UserID
	ToUserID      core.UserID
	CollectibleID core.CollectibleID
	// Draft is the collectible_awarded event recorded inside the gift
	// transaction; an idempotent replay (already-owned collectible) records
	// nothing.
	Draft event.Draft
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
	store    Store
	recorder event.Recorder
}

func NewService(store Store, recorder event.Recorder) Service {
	return Service{store: store, recorder: recorder}
}

// collectibleAwardedDraft builds the collectible_awarded draft a transfer
// records inside its store transaction.
func collectibleAwardedDraft(actor event.Actor, subject event.Subject, recipients event.Recipients) (event.Draft, *core.DomainError) {
	draftResult := event.NewDraft(event.KindCollectibleAwarded, actor, subject, event.EmptyMetadata(), recipients)
	created, matched := draftResult.(event.DraftCreated)
	if !matched {
		reason := draftResult.(event.DraftRejected).Reason
		return event.Draft{}, &reason
	}
	return created.Value, nil
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

func (service Service) Mint(ctx context.Context, ownerKind string, ownerID string, organizationID string, name CollectibleName, kind CollectibleKind, policy TransferPolicy, art string) MintResult {
	idResult := core.NewCollectibleID()
	idCreated, matched := idResult.(core.CollectibleIDCreated)
	if !matched {
		return MintRejected{Reason: idResult.(core.CollectibleIDRejected).Reason}
	}
	scopeID := strings.TrimSpace(organizationID)
	if ownerKind == CollectibleOwnerKindOrganization && scopeID == "" {
		scopeID = ownerID
	}

	collectible := Collectible{
		ID:             idCreated.Value,
		Name:           name,
		Kind:           kind,
		State:          CollectibleStateMinted,
		Policy:         policy,
		OwnerKind:      ownerKind,
		OwnerID:        ownerID,
		OrganizationID: scopeID,
		Art:            art,
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

// TaskHeldCollectiblesResult reports the individual collectibles currently held
// as a task's reward (the stateless task_fund_collectibles rows). Collectibles
// are non-fungible, so this returns each held collectible's id, not a count -
// empty when the task holds no collectible funding.
type TaskHeldCollectiblesResult interface {
	taskHeldCollectiblesResult()
}

type TaskHeldCollectiblesFound struct {
	IDs []core.CollectibleID
}

type TaskHeldCollectiblesRejected struct {
	Reason core.DomainError
}

func (TaskHeldCollectiblesFound) taskHeldCollectiblesResult()    {}
func (TaskHeldCollectiblesRejected) taskHeldCollectiblesResult() {}

// TaskHeldCollectibles reports the individual collectibles currently held as
// the given task's reward.
func (service Service) TaskHeldCollectibles(ctx context.Context, taskID core.TaskID) TaskHeldCollectiblesResult {
	return service.store.TaskHeldCollectibles(ctx, taskID)
}

func (service Service) ListCollectibles(ctx context.Context, owner core.UserID, page core.Page) ListResult {
	storeResult := service.store.ListCollectibles(ctx, owner, page)
	listed, matched := storeResult.(ListStoreListed)
	if !matched {
		return ListRejected{Reason: storeResult.(ListStoreRejected).Reason}
	}
	return CollectiblesListed{Values: listed.Values}
}

// ListByOwner lists the collectibles held by one owner entity (a user, team, or
// organization).
func (service Service) ListByOwner(ctx context.Context, ownerKind string, ownerID string, page core.Page) ListResult {
	storeResult := service.store.ListCollectiblesByOwner(ctx, ownerKind, ownerID, page)
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
	// RecordedEvents are the event drafts recorded inside the transfer
	// transaction; empty on an idempotent replay (the collectible already
	// belonged to the recipient).
	RecordedEvents []event.Draft
}

type GiftRejected struct {
	Reason core.DomainError
}

func (CollectibleGifted) giftResult() {}

func (GiftRejected) giftResult() {}

// AwardOrganizationCollectible transfers a collectible owned by an
// organization to one of its active members. Caller-permission (an org
// admin/owner) is checked by the HTTP layer before this is called;
// ownership, org match, and membership are enforced in the store
// transaction.
func (service Service) AwardOrganizationCollectible(ctx context.Context, organizationID core.OrganizationID, collectibleID core.CollectibleID, recipientUserID core.UserID) GiftResult {
	// The awarding admin's user id never reaches this service (the HTTP
	// layer authorizes the award), so the organization's award is recorded
	// as the system actor with the recipient as the audience.
	subject := event.NoSubjectRefs()
	subject.Collectible = event.CollectibleSubject{ID: collectibleID}
	subject.Organization = event.OrganizationSubject{ID: organizationID}
	draft, draftProblem := collectibleAwardedDraft(event.ActorSystem{}, subject, event.NewRecipients(recipientUserID))
	if draftProblem != nil {
		return GiftRejected{Reason: *draftProblem}
	}

	result := service.store.AwardOrganizationCollectible(ctx, AwardOrganizationCollectibleStoreCommand{
		OrganizationID:  organizationID,
		CollectibleID:   collectibleID,
		RecipientUserID: recipientUserID,
		Draft:           draft,
	})
	if gifted, matched := result.(CollectibleGifted); matched {
		service.recorder.Dispatch(ctx, gifted.RecordedEvents...)
	}
	return result
}

// GiftCollectible transfers an owned, transferable collectible to another user
// (a review tip). Ownership, availability, and transfer policy are enforced in
// the store transaction.
func (service Service) GiftCollectible(ctx context.Context, from core.UserID, to core.UserID, collectibleID core.CollectibleID) GiftResult {
	subject := event.NoSubjectRefs()
	subject.Collectible = event.CollectibleSubject{ID: collectibleID}
	draft, draftProblem := collectibleAwardedDraft(event.ActorUser{ID: from}, subject, event.NewRecipients(to, from))
	if draftProblem != nil {
		return GiftRejected{Reason: *draftProblem}
	}

	result := service.store.GiftCollectible(ctx, GiftStoreCommand{
		FromUserID:    from,
		ToUserID:      to,
		CollectibleID: collectibleID,
		Draft:         draft,
	})
	if gifted, matched := result.(CollectibleGifted); matched {
		service.recorder.Dispatch(ctx, gifted.RecordedEvents...)
	}
	return result
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
