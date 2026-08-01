// Package assetsbridge is the WASI bridge for internal/assets's Store
// (collectibles): hand-written per-type codecs (this file) plus a generated
// dispatcher and guest client (bridge_gen.go). Shared core types (ids, page) are
// serialized by internal/wasibridge/corewire.
package assetsbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
	"github.com/e6qu/sharecrop/internal/wasibridge/eventbridge"
)

// ---- collectible value types (string wrappers) ----

func encodeName(name assets.CollectibleName) string { return name.String() }

func decodeName(raw string) (assets.CollectibleName, error) {
	accepted, matched := assets.NewCollectibleName(raw).(assets.CollectibleNameAccepted)
	if !matched {
		return assets.CollectibleName{}, fmt.Errorf("invalid collectible name %q", raw)
	}
	return accepted.Value, nil
}

func encodeKind(kind assets.CollectibleKind) string { return kind.String() }

func decodeKind(raw string) (assets.CollectibleKind, error) {
	accepted, matched := assets.ParseCollectibleKind(raw).(assets.CollectibleKindAccepted)
	if !matched {
		return assets.CollectibleKind{}, fmt.Errorf("invalid collectible kind %q", raw)
	}
	return accepted.Value, nil
}

func encodeState(state assets.CollectibleState) string { return state.String() }

func decodeState(raw string) (assets.CollectibleState, error) {
	accepted, matched := assets.ParseCollectibleState(raw).(assets.CollectibleStateAccepted)
	if !matched {
		return assets.CollectibleState{}, fmt.Errorf("invalid collectible state %q", raw)
	}
	return accepted.Value, nil
}

func encodePolicy(policy assets.TransferPolicy) string { return policy.String() }

func decodePolicy(raw string) (assets.TransferPolicy, error) {
	accepted, matched := assets.ParseTransferPolicy(raw).(assets.TransferPolicyAccepted)
	if !matched {
		return assets.TransferPolicy{}, fmt.Errorf("invalid transfer policy %q", raw)
	}
	return accepted.Value, nil
}

// ---- assets.Collectible ----

type collectibleWire struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	State          string `json:"state"`
	Policy         string `json:"policy"`
	OwnerKind      string `json:"owner_kind"`
	OwnerID        string `json:"owner_id"`
	OrganizationID string `json:"organization_id"`
	Art            string `json:"art"`
	// CatalogSlug is empty for custom mints; EditionNumber is nil for
	// non-edition instances; Issuer is empty for pre-provenance rows.
	CatalogSlug   string `json:"catalog_slug"`
	EditionNumber *int64 `json:"edition_number,omitempty"`
	Issuer        string `json:"issuer"`
	// IssuerName is the resolved issuer display name on list reads; empty
	// everywhere else.
	IssuerName string `json:"issuer_name,omitempty"`
}

func encodeCollectible(collectible assets.Collectible) collectibleWire {
	catalogSlug := ""
	if fromCatalog, matched := collectible.Catalog.(assets.FromCatalog); matched {
		catalogSlug = fromCatalog.Slug.String()
	}
	var editionNumber *int64
	if numbered, matched := collectible.Edition.(assets.EditionNumbered); matched {
		number := numbered.Number
		editionNumber = &number
	}
	issuer := ""
	if issuedBy, matched := collectible.Issuer.(assets.IssuedBy); matched {
		issuer = issuedBy.ID.String()
	}
	return collectibleWire{
		ID:             corewire.EncodeCollectibleID(collectible.ID),
		Name:           encodeName(collectible.Name),
		Kind:           encodeKind(collectible.Kind),
		State:          encodeState(collectible.State),
		Policy:         encodePolicy(collectible.Policy),
		OwnerKind:      collectible.OwnerKind,
		OwnerID:        collectible.OwnerID,
		OrganizationID: collectible.OrganizationID,
		Art:            collectible.Art,
		CatalogSlug:    catalogSlug,
		EditionNumber:  editionNumber,
		Issuer:         issuer,
		IssuerName:     collectible.IssuerDisplayName.String(),
	}
}

func decodeCollectible(wire collectibleWire) (assets.Collectible, error) {
	id, err := corewire.DecodeCollectibleID(wire.ID)
	if err != nil {
		return assets.Collectible{}, err
	}
	name, err := decodeName(wire.Name)
	if err != nil {
		return assets.Collectible{}, err
	}
	kind, err := decodeKind(wire.Kind)
	if err != nil {
		return assets.Collectible{}, err
	}
	state, err := decodeState(wire.State)
	if err != nil {
		return assets.Collectible{}, err
	}
	policy, err := decodePolicy(wire.Policy)
	if err != nil {
		return assets.Collectible{}, err
	}
	var catalogRef assets.CatalogRef = assets.NoCatalogRef{}
	if wire.CatalogSlug != "" {
		slug, err := decodeCatalogSlug(wire.CatalogSlug)
		if err != nil {
			return assets.Collectible{}, err
		}
		catalogRef = assets.FromCatalog{Slug: slug}
	}
	var editionRef assets.EditionRef = assets.NoEditionNumber{}
	if wire.EditionNumber != nil {
		editionRef = assets.EditionNumbered{Number: *wire.EditionNumber}
	}
	var issuerRef assets.IssuerRef = assets.NoIssuer{}
	if wire.Issuer != "" {
		issuer, err := corewire.DecodeUserID(wire.Issuer)
		if err != nil {
			return assets.Collectible{}, err
		}
		issuerRef = assets.IssuedBy{ID: issuer}
	}
	var issuerName auth.DisplayName
	if wire.IssuerName != "" {
		nameResult, nameMatched := auth.NewDisplayName(wire.IssuerName).(auth.DisplayNameAccepted)
		if !nameMatched {
			return assets.Collectible{}, fmt.Errorf("collectible issuer display name is invalid")
		}
		issuerName = nameResult.Value
	}
	return assets.Collectible{
		ID:                id,
		Name:              name,
		Kind:              kind,
		State:             state,
		Policy:            policy,
		OwnerKind:         wire.OwnerKind,
		OwnerID:           wire.OwnerID,
		OrganizationID:    wire.OrganizationID,
		Art:               wire.Art,
		Catalog:           catalogRef,
		Edition:           editionRef,
		Issuer:            issuerRef,
		IssuerDisplayName: issuerName,
	}, nil
}

func encodeCollectibles(values []assets.Collectible) []collectibleWire {
	encoded := make([]collectibleWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeCollectible(values[index]))
	}
	return encoded
}

func decodeCollectibles(wires []collectibleWire) ([]assets.Collectible, error) {
	values := make([]assets.Collectible, 0, len(wires))
	for index := range wires {
		value, err := decodeCollectible(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- command structs ----

type awardCommandWire struct {
	OrganizationID  string                `json:"organization_id"`
	CollectibleID   string                `json:"collectible_id"`
	RecipientUserID string                `json:"recipient_user_id"`
	Draft           eventbridge.DraftWire `json:"draft"`
}

func encodeAwardCommand(command assets.AwardOrganizationCollectibleStoreCommand) awardCommandWire {
	return awardCommandWire{
		OrganizationID:  corewire.EncodeOrganizationID(command.OrganizationID),
		CollectibleID:   corewire.EncodeCollectibleID(command.CollectibleID),
		RecipientUserID: corewire.EncodeUserID(command.RecipientUserID),
		Draft:           eventbridge.EncodeDraft(command.Draft),
	}
}

func decodeAwardCommand(wire awardCommandWire) (assets.AwardOrganizationCollectibleStoreCommand, error) {
	organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
	if err != nil {
		return assets.AwardOrganizationCollectibleStoreCommand{}, err
	}
	collectibleID, err := corewire.DecodeCollectibleID(wire.CollectibleID)
	if err != nil {
		return assets.AwardOrganizationCollectibleStoreCommand{}, err
	}
	recipientUserID, err := corewire.DecodeUserID(wire.RecipientUserID)
	if err != nil {
		return assets.AwardOrganizationCollectibleStoreCommand{}, err
	}
	draft, err := eventbridge.DecodeDraft(wire.Draft)
	if err != nil {
		return assets.AwardOrganizationCollectibleStoreCommand{}, err
	}
	return assets.AwardOrganizationCollectibleStoreCommand{
		OrganizationID:  organizationID,
		CollectibleID:   collectibleID,
		RecipientUserID: recipientUserID,
		Draft:           draft,
	}, nil
}

type giftCommandWire struct {
	FromUserID    string                `json:"from_user_id"`
	ToUserID      string                `json:"to_user_id"`
	CollectibleID string                `json:"collectible_id"`
	Draft         eventbridge.DraftWire `json:"draft"`
}

func encodeGiftCommand(command assets.GiftStoreCommand) giftCommandWire {
	return giftCommandWire{
		FromUserID:    corewire.EncodeUserID(command.FromUserID),
		ToUserID:      corewire.EncodeUserID(command.ToUserID),
		CollectibleID: corewire.EncodeCollectibleID(command.CollectibleID),
		Draft:         eventbridge.EncodeDraft(command.Draft),
	}
}

func decodeGiftCommand(wire giftCommandWire) (assets.GiftStoreCommand, error) {
	fromUserID, err := corewire.DecodeUserID(wire.FromUserID)
	if err != nil {
		return assets.GiftStoreCommand{}, err
	}
	toUserID, err := corewire.DecodeUserID(wire.ToUserID)
	if err != nil {
		return assets.GiftStoreCommand{}, err
	}
	collectibleID, err := corewire.DecodeCollectibleID(wire.CollectibleID)
	if err != nil {
		return assets.GiftStoreCommand{}, err
	}
	draft, err := eventbridge.DecodeDraft(wire.Draft)
	if err != nil {
		return assets.GiftStoreCommand{}, err
	}
	return assets.GiftStoreCommand{FromUserID: fromUserID, ToUserID: toUserID, CollectibleID: collectibleID, Draft: draft}, nil
}

type fundCommandWire struct {
	FunderUserID  string `json:"funder_user_id"`
	TaskID        string `json:"task_id"`
	CollectibleID string `json:"collectible_id"`
}

func encodeFundCommand(command assets.FundRewardStoreCommand) fundCommandWire {
	return fundCommandWire{
		FunderUserID:  corewire.EncodeUserID(command.FunderUserID),
		TaskID:        corewire.EncodeTaskID(command.TaskID),
		CollectibleID: corewire.EncodeCollectibleID(command.CollectibleID),
	}
}

func decodeFundCommand(wire fundCommandWire) (assets.FundRewardStoreCommand, error) {
	funderUserID, err := corewire.DecodeUserID(wire.FunderUserID)
	if err != nil {
		return assets.FundRewardStoreCommand{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return assets.FundRewardStoreCommand{}, err
	}
	collectibleID, err := corewire.DecodeCollectibleID(wire.CollectibleID)
	if err != nil {
		return assets.FundRewardStoreCommand{}, err
	}
	return assets.FundRewardStoreCommand{FunderUserID: funderUserID, TaskID: taskID, CollectibleID: collectibleID}, nil
}

type refundCommandWire struct {
	RequesterUserID string `json:"requester_user_id"`
	TaskID          string `json:"task_id"`
}

func encodeRefundCommand(command assets.RefundRewardStoreCommand) refundCommandWire {
	return refundCommandWire{
		RequesterUserID: corewire.EncodeUserID(command.RequesterUserID),
		TaskID:          corewire.EncodeTaskID(command.TaskID),
	}
}

func decodeRefundCommand(wire refundCommandWire) (assets.RefundRewardStoreCommand, error) {
	requesterUserID, err := corewire.DecodeUserID(wire.RequesterUserID)
	if err != nil {
		return assets.RefundRewardStoreCommand{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return assets.RefundRewardStoreCommand{}, err
	}
	return assets.RefundRewardStoreCommand{RequesterUserID: requesterUserID, TaskID: taskID}, nil
}

// ---- result unions ----

type acceptedRejectedWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateResult(result assets.CreateStoreResult) collectibleResultWire {
	switch typed := result.(type) {
	case assets.CreateStoreAccepted:
		collectible := encodeCollectible(typed.Value)
		return collectibleResultWire{Variant: "accepted", Collectible: &collectible}
	case assets.CreateStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return collectibleResultWire{Variant: "rejected", Error: &reason}
	default:
		return collectibleResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeCreateResult(wire collectibleResultWire) (assets.CreateStoreResult, error) {
	switch wire.Variant {
	case "accepted":
		if wire.Collectible == nil {
			return nil, fmt.Errorf("create result is missing its collectible")
		}
		collectible, err := decodeCollectible(*wire.Collectible)
		if err != nil {
			return nil, err
		}
		return assets.CreateStoreAccepted{Value: collectible}, nil
	case "rejected":
		return assets.CreateStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create result variant %q", wire.Variant)
	}
}

// collectiblesResultWire backs the list and refund results, which each carry a
// collectible slice on success.
type collectiblesResultWire struct {
	Variant      string                  `json:"variant"`
	Collectibles []collectibleWire       `json:"collectibles,omitempty"`
	Error        *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result assets.ListStoreResult) collectiblesResultWire {
	switch typed := result.(type) {
	case assets.ListStoreListed:
		return collectiblesResultWire{Variant: "listed", Collectibles: encodeCollectibles(typed.Values)}
	case assets.ListStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return collectiblesResultWire{Variant: "rejected", Error: &reason}
	default:
		return collectiblesResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeListResult(wire collectiblesResultWire) (assets.ListStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeCollectibles(wire.Collectibles)
		if err != nil {
			return nil, err
		}
		return assets.ListStoreListed{Values: values}, nil
	case "rejected":
		return assets.ListStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list result variant %q", wire.Variant)
	}
}

func encodeRefundRewardResult(result assets.RefundRewardResult) collectiblesResultWire {
	switch typed := result.(type) {
	case assets.RewardRefunded:
		return collectiblesResultWire{Variant: "refunded", Collectibles: encodeCollectibles(typed.Values)}
	case assets.RefundRewardRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return collectiblesResultWire{Variant: "rejected", Error: &reason}
	default:
		return collectiblesResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeRefundRewardResult(wire collectiblesResultWire) (assets.RefundRewardResult, error) {
	switch wire.Variant {
	case "refunded":
		values, err := decodeCollectibles(wire.Collectibles)
		if err != nil {
			return nil, err
		}
		return assets.RewardRefunded{Values: values}, nil
	case "rejected":
		return assets.RefundRewardRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown refund result variant %q", wire.Variant)
	}
}

// collectibleResultWire backs the fund and gift results, which each carry a
// single collectible on success.
type collectibleResultWire struct {
	Variant     string           `json:"variant"`
	Collectible *collectibleWire `json:"collectible,omitempty"`
	// RecordedEvents carries the drafts a gift/award recorded in its
	// transaction; the fund result leaves it empty.
	RecordedEvents []eventbridge.DraftWire `json:"recorded_events,omitempty"`
	Error          *domainwire.DomainError `json:"error,omitempty"`
}

func encodeFundRewardResult(result assets.FundRewardResult) collectibleResultWire {
	switch typed := result.(type) {
	case assets.RewardFunded:
		collectible := encodeCollectible(typed.Value)
		return collectibleResultWire{Variant: "funded", Collectible: &collectible}
	case assets.FundRewardRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return collectibleResultWire{Variant: "rejected", Error: &reason}
	default:
		return collectibleResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeFundRewardResult(wire collectibleResultWire) (assets.FundRewardResult, error) {
	switch wire.Variant {
	case "funded":
		collectible, err := decodeCollectiblePayload(wire.Collectible)
		if err != nil {
			return nil, err
		}
		return assets.RewardFunded{Value: collectible}, nil
	case "rejected":
		return assets.FundRewardRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown fund result variant %q", wire.Variant)
	}
}

func encodeGiftResult(result assets.GiftResult) collectibleResultWire {
	switch typed := result.(type) {
	case assets.CollectibleGifted:
		collectible := encodeCollectible(typed.Value)
		return collectibleResultWire{Variant: "gifted", Collectible: &collectible, RecordedEvents: eventbridge.EncodeDrafts(typed.RecordedEvents)}
	case assets.GiftRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return collectibleResultWire{Variant: "rejected", Error: &reason}
	default:
		return collectibleResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeGiftResult(wire collectibleResultWire) (assets.GiftResult, error) {
	switch wire.Variant {
	case "gifted":
		collectible, err := decodeCollectiblePayload(wire.Collectible)
		if err != nil {
			return nil, err
		}
		recorded, err := eventbridge.DecodeDrafts(wire.RecordedEvents)
		if err != nil {
			return nil, err
		}
		return assets.CollectibleGifted{Value: collectible, RecordedEvents: recorded}, nil
	case "rejected":
		return assets.GiftRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown gift result variant %q", wire.Variant)
	}
}

type taskHeldResultWire struct {
	Variant string                  `json:"variant"`
	IDs     []string                `json:"ids,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeTaskHeldResult(result assets.TaskHeldCollectiblesResult) taskHeldResultWire {
	switch typed := result.(type) {
	case assets.TaskHeldCollectiblesFound:
		ids := make([]string, 0, len(typed.IDs))
		for index := range typed.IDs {
			ids = append(ids, corewire.EncodeCollectibleID(typed.IDs[index]))
		}
		return taskHeldResultWire{Variant: "found", IDs: ids}
	case assets.TaskHeldCollectiblesRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return taskHeldResultWire{Variant: "rejected", Error: &reason}
	default:
		return taskHeldResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeTaskHeldResult(wire taskHeldResultWire) (assets.TaskHeldCollectiblesResult, error) {
	switch wire.Variant {
	case "found":
		ids := make([]core.CollectibleID, 0, len(wire.IDs))
		for index := range wire.IDs {
			id, err := corewire.DecodeCollectibleID(wire.IDs[index])
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return assets.TaskHeldCollectiblesFound{IDs: ids}, nil
	case "rejected":
		return assets.TaskHeldCollectiblesRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown task-held result variant %q", wire.Variant)
	}
}

func decodeCollectiblePayload(wire *collectibleWire) (assets.Collectible, error) {
	if wire == nil {
		return assets.Collectible{}, fmt.Errorf("result is missing its collectible")
	}
	return decodeCollectible(*wire)
}

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "assets bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}

// ---- catalog types ----

func decodeCatalogSlug(raw string) (assets.CatalogSlug, error) {
	accepted, matched := assets.NewCatalogSlug(raw).(assets.CatalogSlugAccepted)
	if !matched {
		return assets.CatalogSlug{}, fmt.Errorf("invalid catalog slug %q", raw)
	}
	return accepted.Value, nil
}

func encodeCatalogSlug(slug assets.CatalogSlug) string { return slug.String() }

type catalogEntryWire struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Policy string `json:"policy"`
	Art    string `json:"art"`
	State  string `json:"state"`
	// MaxEditions is nil for uncapped (badge) entries.
	MaxEditions *int64 `json:"max_editions,omitempty"`
}

func encodeCatalogEntry(entry assets.CatalogEntry) catalogEntryWire {
	var maxEditions *int64
	if capped, matched := entry.Cap.(assets.EditionCapOf); matched {
		limit := capped.Limit
		maxEditions = &limit
	}
	return catalogEntryWire{
		Slug:        entry.Slug.String(),
		Name:        encodeName(entry.Name),
		Kind:        encodeKind(entry.Kind),
		Policy:      encodePolicy(entry.Policy),
		Art:         entry.Art,
		State:       entry.State.String(),
		MaxEditions: maxEditions,
	}
}

func decodeCatalogEntry(wire catalogEntryWire) (assets.CatalogEntry, error) {
	slug, err := decodeCatalogSlug(wire.Slug)
	if err != nil {
		return assets.CatalogEntry{}, err
	}
	name, err := decodeName(wire.Name)
	if err != nil {
		return assets.CatalogEntry{}, err
	}
	kind, err := decodeKind(wire.Kind)
	if err != nil {
		return assets.CatalogEntry{}, err
	}
	policy, err := decodePolicy(wire.Policy)
	if err != nil {
		return assets.CatalogEntry{}, err
	}
	stateAccepted, stateMatched := assets.ParseCatalogEntryState(wire.State).(assets.CatalogEntryStateAccepted)
	if !stateMatched {
		return assets.CatalogEntry{}, fmt.Errorf("invalid catalog entry state %q", wire.State)
	}
	var cap assets.EditionCap = assets.NoEditionCap{}
	if wire.MaxEditions != nil {
		cap = assets.EditionCapOf{Limit: *wire.MaxEditions}
	}
	return assets.CatalogEntry{
		Slug:   slug,
		Name:   name,
		Kind:   kind,
		Policy: policy,
		Art:    wire.Art,
		State:  stateAccepted.Value,
		Cap:    cap,
	}, nil
}

type catalogListingWire struct {
	Entry catalogEntryWire `json:"entry"`
	// LiveInstances counts the entry's non-withdrawn instances.
	LiveInstances int64 `json:"live_instances"`
}

type catalogListResultWire struct {
	Variant string                  `json:"variant"`
	Entries []catalogListingWire    `json:"entries,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCatalogListResult(result assets.CatalogListResult) catalogListResultWire {
	switch typed := result.(type) {
	case assets.CatalogListed:
		entries := make([]catalogListingWire, 0, len(typed.Values))
		for _, listing := range typed.Values {
			entries = append(entries, catalogListingWire{Entry: encodeCatalogEntry(listing.Entry), LiveInstances: listing.LiveInstanceCount})
		}
		return catalogListResultWire{Variant: "listed", Entries: entries}
	case assets.CatalogListRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return catalogListResultWire{Variant: "rejected", Error: &reason}
	default:
		return catalogListResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeCatalogListResult(wire catalogListResultWire) (assets.CatalogListResult, error) {
	switch wire.Variant {
	case "listed":
		entries := make([]assets.CatalogListing, 0, len(wire.Entries))
		for _, listingWire := range wire.Entries {
			entry, err := decodeCatalogEntry(listingWire.Entry)
			if err != nil {
				return nil, err
			}
			entries = append(entries, assets.CatalogListing{Entry: entry, LiveInstanceCount: listingWire.LiveInstances})
		}
		return assets.CatalogListed{Values: entries}, nil
	case "rejected":
		return assets.CatalogListRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown catalog list result variant %q", wire.Variant)
	}
}

type catalogMutationResultWire struct {
	Variant string                  `json:"variant"`
	Entry   *catalogEntryWire       `json:"entry,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCatalogMutationResult(result assets.CatalogMutationResult) catalogMutationResultWire {
	switch typed := result.(type) {
	case assets.CatalogMutated:
		entry := encodeCatalogEntry(typed.Value)
		return catalogMutationResultWire{Variant: "mutated", Entry: &entry}
	case assets.CatalogMutationRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return catalogMutationResultWire{Variant: "rejected", Error: &reason}
	default:
		return catalogMutationResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeCatalogMutationResult(wire catalogMutationResultWire) (assets.CatalogMutationResult, error) {
	switch wire.Variant {
	case "mutated":
		if wire.Entry == nil {
			return nil, fmt.Errorf("catalog mutation result is missing its entry")
		}
		entry, err := decodeCatalogEntry(*wire.Entry)
		if err != nil {
			return nil, err
		}
		return assets.CatalogMutated{Value: entry}, nil
	case "rejected":
		return assets.CatalogMutationRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown catalog mutation result variant %q", wire.Variant)
	}
}

// ---- lifecycle commands ----

type awardFromCatalogCommandWire struct {
	NewID          string `json:"new_id"`
	Slug           string `json:"slug"`
	OwnerKind      string `json:"owner_kind"`
	OwnerID        string `json:"owner_id"`
	OrganizationID string `json:"organization_id"`
	Issuer         string `json:"issuer"`
}

func encodeAwardFromCatalogCommand(command assets.AwardFromCatalogStoreCommand) awardFromCatalogCommandWire {
	return awardFromCatalogCommandWire{
		NewID:          corewire.EncodeCollectibleID(command.NewID),
		Slug:           encodeCatalogSlug(command.Slug),
		OwnerKind:      command.OwnerKind,
		OwnerID:        command.OwnerID,
		OrganizationID: command.OrganizationID,
		Issuer:         corewire.EncodeUserID(command.Issuer),
	}
}

func decodeAwardFromCatalogCommand(wire awardFromCatalogCommandWire) (assets.AwardFromCatalogStoreCommand, error) {
	newID, err := corewire.DecodeCollectibleID(wire.NewID)
	if err != nil {
		return assets.AwardFromCatalogStoreCommand{}, err
	}
	slug, err := decodeCatalogSlug(wire.Slug)
	if err != nil {
		return assets.AwardFromCatalogStoreCommand{}, err
	}
	issuer, err := corewire.DecodeUserID(wire.Issuer)
	if err != nil {
		return assets.AwardFromCatalogStoreCommand{}, err
	}
	return assets.AwardFromCatalogStoreCommand{
		NewID:          newID,
		Slug:           slug,
		OwnerKind:      wire.OwnerKind,
		OwnerID:        wire.OwnerID,
		OrganizationID: wire.OrganizationID,
		Issuer:         issuer,
	}, nil
}

type withdrawCollectibleCommandWire struct {
	CollectibleID string                `json:"collectible_id"`
	Draft         eventbridge.DraftWire `json:"draft"`
}

func encodeWithdrawCollectibleCommand(command assets.WithdrawCollectibleStoreCommand) withdrawCollectibleCommandWire {
	return withdrawCollectibleCommandWire{
		CollectibleID: corewire.EncodeCollectibleID(command.CollectibleID),
		Draft:         eventbridge.EncodeDraft(command.Draft),
	}
}

func decodeWithdrawCollectibleCommand(wire withdrawCollectibleCommandWire) (assets.WithdrawCollectibleStoreCommand, error) {
	collectibleID, err := corewire.DecodeCollectibleID(wire.CollectibleID)
	if err != nil {
		return assets.WithdrawCollectibleStoreCommand{}, err
	}
	draft, err := eventbridge.DecodeDraft(wire.Draft)
	if err != nil {
		return assets.WithdrawCollectibleStoreCommand{}, err
	}
	return assets.WithdrawCollectibleStoreCommand{CollectibleID: collectibleID, Draft: draft}, nil
}

type transferToOrganizationCommandWire struct {
	FromUserID     string                `json:"from_user_id"`
	OrganizationID string                `json:"organization_id"`
	CollectibleID  string                `json:"collectible_id"`
	Draft          eventbridge.DraftWire `json:"draft"`
}

func encodeTransferToOrganizationCommand(command assets.TransferToOrganizationStoreCommand) transferToOrganizationCommandWire {
	return transferToOrganizationCommandWire{
		FromUserID:     corewire.EncodeUserID(command.FromUserID),
		OrganizationID: corewire.EncodeOrganizationID(command.OrganizationID),
		CollectibleID:  corewire.EncodeCollectibleID(command.CollectibleID),
		Draft:          eventbridge.EncodeDraft(command.Draft),
	}
}

func decodeTransferToOrganizationCommand(wire transferToOrganizationCommandWire) (assets.TransferToOrganizationStoreCommand, error) {
	fromUserID, err := corewire.DecodeUserID(wire.FromUserID)
	if err != nil {
		return assets.TransferToOrganizationStoreCommand{}, err
	}
	organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
	if err != nil {
		return assets.TransferToOrganizationStoreCommand{}, err
	}
	collectibleID, err := corewire.DecodeCollectibleID(wire.CollectibleID)
	if err != nil {
		return assets.TransferToOrganizationStoreCommand{}, err
	}
	draft, err := eventbridge.DecodeDraft(wire.Draft)
	if err != nil {
		return assets.TransferToOrganizationStoreCommand{}, err
	}
	return assets.TransferToOrganizationStoreCommand{
		FromUserID:     fromUserID,
		OrganizationID: organizationID,
		CollectibleID:  collectibleID,
		Draft:          draft,
	}, nil
}

type transferFromOrganizationCommandWire struct {
	OrganizationID string                `json:"organization_id"`
	ActingUserID   string                `json:"acting_user_id"`
	ToUserID       string                `json:"to_user_id"`
	CollectibleID  string                `json:"collectible_id"`
	Draft          eventbridge.DraftWire `json:"draft"`
}

func encodeTransferFromOrganizationCommand(command assets.TransferFromOrganizationStoreCommand) transferFromOrganizationCommandWire {
	return transferFromOrganizationCommandWire{
		OrganizationID: corewire.EncodeOrganizationID(command.OrganizationID),
		ActingUserID:   corewire.EncodeUserID(command.ActingUserID),
		ToUserID:       corewire.EncodeUserID(command.ToUserID),
		CollectibleID:  corewire.EncodeCollectibleID(command.CollectibleID),
		Draft:          eventbridge.EncodeDraft(command.Draft),
	}
}

func decodeTransferFromOrganizationCommand(wire transferFromOrganizationCommandWire) (assets.TransferFromOrganizationStoreCommand, error) {
	organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
	if err != nil {
		return assets.TransferFromOrganizationStoreCommand{}, err
	}
	actingUserID, err := corewire.DecodeUserID(wire.ActingUserID)
	if err != nil {
		return assets.TransferFromOrganizationStoreCommand{}, err
	}
	toUserID, err := corewire.DecodeUserID(wire.ToUserID)
	if err != nil {
		return assets.TransferFromOrganizationStoreCommand{}, err
	}
	collectibleID, err := corewire.DecodeCollectibleID(wire.CollectibleID)
	if err != nil {
		return assets.TransferFromOrganizationStoreCommand{}, err
	}
	draft, err := eventbridge.DecodeDraft(wire.Draft)
	if err != nil {
		return assets.TransferFromOrganizationStoreCommand{}, err
	}
	return assets.TransferFromOrganizationStoreCommand{
		OrganizationID: organizationID,
		ActingUserID:   actingUserID,
		ToUserID:       toUserID,
		CollectibleID:  collectibleID,
		Draft:          draft,
	}, nil
}

// ---- lifecycle results ----

type withdrawResultWire struct {
	Variant        string                  `json:"variant"`
	Collectible    *collectibleWire        `json:"collectible,omitempty"`
	RecordedEvents []eventbridge.DraftWire `json:"recorded_events,omitempty"`
	Error          *domainwire.DomainError `json:"error,omitempty"`
}

func encodeWithdrawResult(result assets.WithdrawResult) withdrawResultWire {
	switch typed := result.(type) {
	case assets.CollectibleWithdrawn:
		collectible := encodeCollectible(typed.Value)
		return withdrawResultWire{Variant: "withdrawn", Collectible: &collectible, RecordedEvents: eventbridge.EncodeDrafts(typed.RecordedEvents)}
	case assets.WithdrawRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return withdrawResultWire{Variant: "rejected", Error: &reason}
	default:
		return withdrawResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeWithdrawResult(wire withdrawResultWire) (assets.WithdrawResult, error) {
	switch wire.Variant {
	case "withdrawn":
		if wire.Collectible == nil {
			return nil, fmt.Errorf("withdraw result is missing its collectible")
		}
		collectible, err := decodeCollectible(*wire.Collectible)
		if err != nil {
			return nil, err
		}
		recorded, err := eventbridge.DecodeDrafts(wire.RecordedEvents)
		if err != nil {
			return nil, err
		}
		return assets.CollectibleWithdrawn{Value: collectible, RecordedEvents: recorded}, nil
	case "rejected":
		return assets.WithdrawRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown withdraw result variant %q", wire.Variant)
	}
}

func encodeDeleteCollectibleResult(result assets.DeleteCollectibleResult) acceptedRejectedWire {
	switch typed := result.(type) {
	case assets.CollectibleDeleted:
		return acceptedRejectedWire{Variant: "deleted"}
	case assets.DeleteCollectibleRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return acceptedRejectedWire{Variant: "rejected", Error: &reason}
	default:
		return acceptedRejectedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeDeleteCollectibleResult(wire acceptedRejectedWire) (assets.DeleteCollectibleResult, error) {
	switch wire.Variant {
	case "deleted":
		return assets.CollectibleDeleted{}, nil
	case "rejected":
		return assets.DeleteCollectibleRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown delete collectible result variant %q", wire.Variant)
	}
}
