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
	ListCatalogEntries(context.Context) CatalogListResult
	AddCatalogEntry(context.Context, CatalogEntry) CatalogMutationResult
	WithdrawCatalogEntry(context.Context, CatalogSlug) CatalogMutationResult
	// ReleaseCatalogEntry returns a withdrawn entry to the available state,
	// so it can be awarded again.
	ReleaseCatalogEntry(context.Context, CatalogSlug) CatalogMutationResult
	// DeleteCatalogEntry removes a withdrawn entry that no instance (live or
	// withdrawn) references.
	DeleteCatalogEntry(context.Context, CatalogSlug) CatalogMutationResult
	// AwardFromCatalog mints a fresh instance of an available catalog entry,
	// enforcing the entry's kind semantics (unique: one live instance ever;
	// edition: numbered against the cap; badge: uncapped).
	AwardFromCatalog(context.Context, AwardFromCatalogStoreCommand) CreateStoreResult
	WithdrawCollectible(context.Context, WithdrawCollectibleStoreCommand) WithdrawResult
	// ReleaseCollectible returns a withdrawn instance to its holder's
	// inventory (state -> minted, owner unchanged), re-validating the entry's
	// kind semantics under the entry row lock: a unique whose live slot was
	// re-minted while this instance was withdrawn is refused with a conflict.
	ReleaseCollectible(context.Context, ReleaseCollectibleStoreCommand) ReleaseResult
	// DeleteWithdrawnCollectible hard-deletes an instance, allowed only from
	// the withdrawn state and only when no task escrow references it.
	DeleteWithdrawnCollectible(context.Context, core.CollectibleID) DeleteCollectibleResult
	TransferCollectibleToOrganization(context.Context, TransferToOrganizationStoreCommand) GiftResult
	TransferCollectibleFromOrganization(context.Context, TransferFromOrganizationStoreCommand) GiftResult
}

// AwardFromCatalogStoreCommand mints one instance of a catalog entry for a
// recipient. Issuer is the acting (admin) user recorded as the instance's
// provenance.
type AwardFromCatalogStoreCommand struct {
	NewID          core.CollectibleID
	Slug           CatalogSlug
	OwnerKind      string
	OwnerID        string
	OrganizationID string
	Issuer         core.UserID
}

// WithdrawCollectibleStoreCommand removes a catalog-minted instance from its
// holder (state -> withdrawn). The store merges the holder into the draft's
// recipients inside the withdraw transaction.
type WithdrawCollectibleStoreCommand struct {
	CollectibleID core.CollectibleID
	// Draft is the collectible_withdrawn event recorded inside the withdraw
	// transaction.
	Draft event.Draft
}

// TransferToOrganizationStoreCommand donates a user-owned collectible to an
// organization.
type TransferToOrganizationStoreCommand struct {
	FromUserID     core.UserID
	OrganizationID core.OrganizationID
	CollectibleID  core.CollectibleID
	// Draft is the collectible_awarded event recorded inside the transfer
	// transaction.
	Draft event.Draft
}

// TransferFromOrganizationStoreCommand moves an organization-owned
// collectible to a user. The store authorizes the acting member
// in-transaction: they must hold the organization's manage_collectibles
// permission.
type TransferFromOrganizationStoreCommand struct {
	OrganizationID core.OrganizationID
	ActingUserID   core.UserID
	ToUserID       core.UserID
	CollectibleID  core.CollectibleID
	// Draft is the collectible_awarded event recorded inside the transfer
	// transaction.
	Draft event.Draft
}

// ReleaseCollectibleStoreCommand returns a withdrawn instance to its holder
// (state -> minted). The store merges the holder into the draft's recipients
// inside the release transaction.
type ReleaseCollectibleStoreCommand struct {
	CollectibleID core.CollectibleID
	// Draft is the collectible_released event recorded inside the release
	// transaction.
	Draft event.Draft
}

type ReleaseResult interface {
	releaseResult()
}

type CollectibleReleased struct {
	Value Collectible
	// RecordedEvents are the drafts recorded inside the release transaction
	// (the collectible_released event to the holder).
	RecordedEvents []event.Draft
}

type ReleaseRejected struct {
	Reason core.DomainError
}

func (CollectibleReleased) releaseResult() {}

func (ReleaseRejected) releaseResult() {}

type WithdrawResult interface {
	withdrawResult()
}

type CollectibleWithdrawn struct {
	Value Collectible
	// RecordedEvents are the drafts recorded inside the withdraw
	// transaction (the collectible_withdrawn event to the former holder).
	RecordedEvents []event.Draft
}

type WithdrawRejected struct {
	Reason core.DomainError
}

func (CollectibleWithdrawn) withdrawResult() {}

func (WithdrawRejected) withdrawResult() {}

type DeleteCollectibleResult interface {
	deleteCollectibleResult()
}

type CollectibleDeleted struct{}

type DeleteCollectibleRejected struct {
	Reason core.DomainError
}

func (CollectibleDeleted) deleteCollectibleResult() {}

func (DeleteCollectibleRejected) deleteCollectibleResult() {}

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

// Mint creates a custom collectible. issuer is the acting user, recorded as
// provenance; the store enforces the kind semantics for custom mints (one
// live unique per (issuer, name); editions numbered per (issuer, name)).
func (service Service) Mint(ctx context.Context, issuer core.UserID, ownerKind string, ownerID string, organizationID string, name CollectibleName, kind CollectibleKind, policy TransferPolicy, art string) MintResult {
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
		Catalog:        NoCatalogRef{},
		Edition:        NoEditionNumber{},
		Issuer:         IssuedBy{ID: issuer},
	}
	storeResult := service.store.CreateCollectible(ctx, collectible)
	created, createdMatched := storeResult.(CreateStoreAccepted)
	if !createdMatched {
		return MintRejected{Reason: storeResult.(CreateStoreRejected).Reason}
	}
	return CollectibleMinted{Value: created.Value}
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

// ListCatalog lists every catalog entry, including withdrawn ones (read
// boundaries decide whether to filter to available entries).
func (service Service) ListCatalog(ctx context.Context) CatalogListResult {
	return service.store.ListCatalogEntries(ctx)
}

// AddCatalogEntry creates a new catalog entry. The art must name a sprite
// from the fixed art registry; slug uniqueness is enforced by the store.
// Platform-admin authorization is the calling boundary's responsibility.
func (service Service) AddCatalogEntry(ctx context.Context, entry CatalogEntry) CatalogMutationResult {
	if !KnownSpriteSlug(entry.Art) {
		return CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "catalog art must name a sprite from the fixed art registry")}
	}
	if err := validateCatalogCap(entry); err != nil {
		return CatalogMutationRejected{Reason: *err}
	}
	return service.store.AddCatalogEntry(ctx, entry)
}

// validateCatalogCap enforces the kind/cap coherence rules: uniques are
// capped at exactly 1, editions need a positive cap, badges are uncapped.
func validateCatalogCap(entry CatalogEntry) *core.DomainError {
	switch entry.Kind {
	case CollectibleKindUnique:
		if capped, matched := entry.Cap.(EditionCapOf); !matched || capped.Limit != 1 {
			reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "unique catalog entries are capped at exactly one instance")
			return &reason
		}
	case CollectibleKindEdition:
		if capped, matched := entry.Cap.(EditionCapOf); !matched || capped.Limit < 1 {
			reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "edition catalog entries need a positive edition cap")
			return &reason
		}
	case CollectibleKindBadge:
		if _, matched := entry.Cap.(NoEditionCap); !matched {
			reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "badge catalog entries are uncapped")
			return &reason
		}
	}
	return nil
}

// WithdrawCatalogEntry marks an entry no longer awardable; existing
// instances are unaffected.
func (service Service) WithdrawCatalogEntry(ctx context.Context, slug CatalogSlug) CatalogMutationResult {
	return service.store.WithdrawCatalogEntry(ctx, slug)
}

// ReleaseCatalogEntry returns a withdrawn entry to the available state so it
// can be awarded again. Platform-admin authorization is the calling
// boundary's responsibility.
func (service Service) ReleaseCatalogEntry(ctx context.Context, slug CatalogSlug) CatalogMutationResult {
	return service.store.ReleaseCatalogEntry(ctx, slug)
}

// DeleteCatalogEntry removes a withdrawn entry no instance (live or
// withdrawn) references.
func (service Service) DeleteCatalogEntry(ctx context.Context, slug CatalogSlug) CatalogMutationResult {
	return service.store.DeleteCatalogEntry(ctx, slug)
}

// AwardFromCatalog mints a fresh instance of an available catalog entry for
// the recipient. Withdrawn entries are refused; the entry's kind semantics
// (uniqueness, edition caps) are enforced inside the store transaction.
// Platform-admin authorization is the calling boundary's responsibility.
func (service Service) AwardFromCatalog(ctx context.Context, issuer core.UserID, rawSlug string, ownerKind string, ownerID string, organizationID string) MintResult {
	slugResult := NewCatalogSlug(rawSlug)
	slug, slugMatched := slugResult.(CatalogSlugAccepted)
	if !slugMatched {
		return MintRejected{Reason: slugResult.(CatalogSlugRejected).Reason}
	}
	idResult := core.NewCollectibleID()
	idCreated, idMatched := idResult.(core.CollectibleIDCreated)
	if !idMatched {
		return MintRejected{Reason: idResult.(core.CollectibleIDRejected).Reason}
	}
	scopeID := strings.TrimSpace(organizationID)
	if ownerKind == CollectibleOwnerKindOrganization && scopeID == "" {
		scopeID = ownerID
	}
	result := service.store.AwardFromCatalog(ctx, AwardFromCatalogStoreCommand{
		NewID:          idCreated.Value,
		Slug:           slug.Value,
		OwnerKind:      ownerKind,
		OwnerID:        ownerID,
		OrganizationID: scopeID,
		Issuer:         issuer,
	})
	created, createdMatched := result.(CreateStoreAccepted)
	if !createdMatched {
		return MintRejected{Reason: result.(CreateStoreRejected).Reason}
	}
	return CollectibleMinted{Value: created.Value}
}

// WithdrawCollectible removes a catalog-minted instance from its holder.
// The holder learns through the collectible_withdrawn event, recorded inside
// the withdraw transaction. Platform-admin authorization is the calling
// boundary's responsibility; actor is the acting admin.
func (service Service) WithdrawCollectible(ctx context.Context, actor core.UserID, collectibleID core.CollectibleID) WithdrawResult {
	subject := event.NoSubjectRefs()
	subject.Collectible = event.CollectibleSubject{ID: collectibleID}
	draftResult := event.NewDraft(event.KindCollectibleWithdrawn, event.ActorUser{ID: actor}, subject,
		event.EmptyMetadata(), event.NewRecipients(actor))
	draftCreated, draftMatched := draftResult.(event.DraftCreated)
	if !draftMatched {
		return WithdrawRejected{Reason: draftResult.(event.DraftRejected).Reason}
	}
	result := service.store.WithdrawCollectible(ctx, WithdrawCollectibleStoreCommand{
		CollectibleID: collectibleID,
		Draft:         draftCreated.Value,
	})
	if withdrawn, matched := result.(CollectibleWithdrawn); matched {
		service.recorder.Dispatch(ctx, withdrawn.RecordedEvents...)
	}
	return result
}

// ReleaseCollectible returns a withdrawn instance to its holder's inventory
// (state -> minted, owner unchanged). The holder learns through the
// collectible_released event, recorded inside the release transaction. The
// store re-validates the entry's kind semantics under the entry row lock.
// Platform-admin authorization is the calling boundary's responsibility;
// actor is the acting admin.
func (service Service) ReleaseCollectible(ctx context.Context, actor core.UserID, collectibleID core.CollectibleID) ReleaseResult {
	subject := event.NoSubjectRefs()
	subject.Collectible = event.CollectibleSubject{ID: collectibleID}
	draftResult := event.NewDraft(event.KindCollectibleReleased, event.ActorUser{ID: actor}, subject,
		event.EmptyMetadata(), event.NewRecipients(actor))
	draftCreated, draftMatched := draftResult.(event.DraftCreated)
	if !draftMatched {
		return ReleaseRejected{Reason: draftResult.(event.DraftRejected).Reason}
	}
	result := service.store.ReleaseCollectible(ctx, ReleaseCollectibleStoreCommand{
		CollectibleID: collectibleID,
		Draft:         draftCreated.Value,
	})
	if released, matched := result.(CollectibleReleased); matched {
		service.recorder.Dispatch(ctx, released.RecordedEvents...)
	}
	return result
}

// DeleteWithdrawnCollectible hard-deletes a withdrawn instance. Refused for
// every other state and for instances a task escrow still references.
// Platform-admin authorization is the calling boundary's responsibility.
func (service Service) DeleteWithdrawnCollectible(ctx context.Context, collectibleID core.CollectibleID) DeleteCollectibleResult {
	return service.store.DeleteWithdrawnCollectible(ctx, collectibleID)
}

// TransferCollectibleToOrganization donates one of the actor's collectibles
// to an organization. Ownership, availability, and transfer policy are
// enforced in the store transaction.
func (service Service) TransferCollectibleToOrganization(ctx context.Context, from core.UserID, organizationID core.OrganizationID, collectibleID core.CollectibleID) GiftResult {
	subject := event.NoSubjectRefs()
	subject.Collectible = event.CollectibleSubject{ID: collectibleID}
	subject.Organization = event.OrganizationSubject{ID: organizationID}
	draft, draftProblem := collectibleAwardedDraft(event.ActorUser{ID: from}, subject, event.NewRecipients(from))
	if draftProblem != nil {
		return GiftRejected{Reason: *draftProblem}
	}
	result := service.store.TransferCollectibleToOrganization(ctx, TransferToOrganizationStoreCommand{
		FromUserID:     from,
		OrganizationID: organizationID,
		CollectibleID:  collectibleID,
		Draft:          draft,
	})
	if gifted, matched := result.(CollectibleGifted); matched {
		service.recorder.Dispatch(ctx, gifted.RecordedEvents...)
	}
	return result
}

// TransferCollectibleFromOrganization moves an organization-owned
// collectible to a user. The acting member's manage_collectibles permission
// is verified inside the store transaction.
func (service Service) TransferCollectibleFromOrganization(ctx context.Context, actor core.UserID, organizationID core.OrganizationID, to core.UserID, collectibleID core.CollectibleID) GiftResult {
	subject := event.NoSubjectRefs()
	subject.Collectible = event.CollectibleSubject{ID: collectibleID}
	subject.Organization = event.OrganizationSubject{ID: organizationID}
	draft, draftProblem := collectibleAwardedDraft(event.ActorUser{ID: actor}, subject, event.NewRecipients(to, actor))
	if draftProblem != nil {
		return GiftRejected{Reason: *draftProblem}
	}
	result := service.store.TransferCollectibleFromOrganization(ctx, TransferFromOrganizationStoreCommand{
		OrganizationID: organizationID,
		ActingUserID:   actor,
		ToUserID:       to,
		CollectibleID:  collectibleID,
		Draft:          draft,
	})
	if gifted, matched := result.(CollectibleGifted); matched {
		service.recorder.Dispatch(ctx, gifted.RecordedEvents...)
	}
	return result
}
