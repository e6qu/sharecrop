package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

type collectibleSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	State          string `json:"state"`
	TransferPolicy string `json:"transfer_policy"`
	OwnerID        string `json:"owner_id"`
	OwnerKind      string `json:"owner_kind"`
	OrganizationID string `json:"organization_id"`
	Art            string `json:"art"`
	// CatalogSlug names the catalog entry a catalog-awarded instance came
	// from; empty for custom mints (mirroring REST's collectible response).
	CatalogSlug string `json:"catalog_slug"`
	// EditionNumber is the mint sequence number of an edition instance; 0
	// for unnumbered instances (badges, uniques).
	EditionNumber int64 `json:"edition_number"`
	// IssuerDisplayName names the user who minted or awarded the instance.
	// Resolved on the list read paths; mutation results leave it empty.
	IssuerDisplayName string `json:"issuer_display_name"`
	// OwnerDisplayName labels the current owner (user display name,
	// organization name, or team name, per owner_kind). Resolved on the list
	// read paths; mutation results leave it empty.
	OwnerDisplayName string `json:"owner_display_name"`
}

type collectiblesListPayload struct {
	Collectibles []collectibleSummary `json:"collectibles"`
	NextOffset   int                  `json:"next_offset"`
}

type catalogEntryPayload struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	TransferPolicy string `json:"transfer_policy"`
	Art            string `json:"art"`
	// State is the entry lifecycle: available (awardable) or withdrawn (its
	// instances stay in circulation, but awarding is refused).
	State string `json:"state"`
	// MaxEditions bounds how many instances may ever be minted: 1 for
	// uniques, the declared run size for editions, 0 for uncapped badges.
	MaxEditions int64 `json:"max_editions"`
	// MintedCount counts the entry's live (non-withdrawn) instances.
	MintedCount int64 `json:"minted_count"`
	// LiveOwnerCount counts the distinct owners holding live instances.
	LiveOwnerCount int64 `json:"live_owner_count"`
	// OwnerDisplayName labels the holder of a unique entry's live instance;
	// empty for non-unique entries and unminted slots.
	OwnerDisplayName string `json:"owner_display_name"`
}

type catalogPayload struct {
	Entries []catalogEntryPayload `json:"entries"`
}

func (collectibleSummary) payloadValue() {}

func (collectiblesListPayload) payloadValue() {}

func (catalogPayload) payloadValue() {}

func collectibleToSummary(value assets.Collectible) collectibleSummary {
	catalogSlug := ""
	if fromCatalog, matched := value.Catalog.(assets.FromCatalog); matched {
		catalogSlug = fromCatalog.Slug.String()
	}
	editionNumber := int64(0)
	if numbered, matched := value.Edition.(assets.EditionNumbered); matched {
		editionNumber = numbered.Number
	}
	return collectibleSummary{
		ID:                value.ID.String(),
		Name:              value.Name.String(),
		Kind:              value.Kind.String(),
		State:             value.State.String(),
		TransferPolicy:    value.Policy.String(),
		OwnerID:           value.OwnerID,
		OwnerKind:         value.OwnerKind,
		OrganizationID:    value.OrganizationID,
		Art:               value.Art,
		CatalogSlug:       catalogSlug,
		EditionNumber:     editionNumber,
		IssuerDisplayName: value.IssuerDisplayName.String(),
		OwnerDisplayName:  value.OwnerDisplayName.String(),
	}
}

// parsedCollectibleTemplate is the validated (name, kind, transfer policy)
// triple shared by mint_collectible and add_catalog_entry.
type parsedCollectibleTemplate struct {
	name   assets.CollectibleName
	kind   assets.CollectibleKind
	policy assets.TransferPolicy
}

func parseMCPCollectibleTemplate(rawName string, rawKind string, rawPolicy string) (parsedCollectibleTemplate, toolResult) {
	nameResult := assets.NewCollectibleName(rawName)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		return parsedCollectibleTemplate{}, toolProtocolError{code: codeInvalidParams, message: nameResult.(assets.CollectibleNameRejected).Reason.Description()}
	}
	kindResult := assets.ParseCollectibleKind(rawKind)
	kind, kindMatched := kindResult.(assets.CollectibleKindAccepted)
	if !kindMatched {
		return parsedCollectibleTemplate{}, toolProtocolError{code: codeInvalidParams, message: kindResult.(assets.CollectibleKindRejected).Reason.Description()}
	}
	policyResult := assets.ParseTransferPolicy(rawPolicy)
	policy, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		return parsedCollectibleTemplate{}, toolProtocolError{code: codeInvalidParams, message: policyResult.(assets.TransferPolicyRejected).Reason.Description()}
	}
	return parsedCollectibleTemplate{name: name.Value, kind: kind.Value, policy: policy.Value}, nil
}

func (server Server) callMintCollectible(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Name           string `json:"name"`
		Kind           string `json:"kind"`
		TransferPolicy string `json:"transfer_policy"`
		Art            string `json:"art"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	template, templateProblem := parseMCPCollectibleTemplate(args.Name, args.Kind, args.TransferPolicy)
	if templateProblem != nil {
		return templateProblem
	}
	result := server.services.MintCollectible(ctx, subject.ID, assets.CollectibleOwnerKindUser, subject.ID.String(), "", template.name, template.kind, template.policy, args.Art)
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		return toolFailed{code: result.(assets.MintRejected).Reason.Code(), message: result.(assets.MintRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(minted.Value))
}

func (server Server) callCollectibleCatalog(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	result := server.services.ListCollectibleCatalog(ctx)
	listed, matched := result.(assets.CatalogListed)
	if !matched {
		rejected := result.(assets.CatalogListRejected)
		return toolFailed{code: rejected.Reason.Code(), message: rejected.Reason.Description()}
	}
	// Every entry is listed — including withdrawn ones, whose instances
	// remain in circulation — with an explicit state marker, matching REST:
	// hiding a withdrawn entry would make existing instances unexplainable,
	// and awarding from one is refused at award time anyway.
	payload := catalogPayload{Entries: make([]catalogEntryPayload, 0, len(listed.Values))}
	for index := range listed.Values {
		entry := catalogEntryToPayload(listed.Values[index].Entry)
		entry.MintedCount = listed.Values[index].LiveInstanceCount
		entry.LiveOwnerCount = listed.Values[index].LiveOwnerCount
		entry.OwnerDisplayName = listed.Values[index].OwnerDisplayName.String()
		payload.Entries = append(payload.Entries, entry)
	}
	return marshalPayload(payload)
}

// callTransferCollectible moves an owned collectible to another user (the
// default) or donates it to an organization's trophy case, mirroring REST's
// transfer body: target_kind is "user" (default when absent) or
// "organization", and recipient_id is the matching user or organization id.
func (server Server) callTransferCollectible(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		CollectibleID string `json:"collectible_id"`
		TargetKind    string `json:"target_kind"`
		RecipientID   string `json:"recipient_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	collectibleIDResult := core.ParseCollectibleID(args.CollectibleID)
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return invalidIDArgument("collectible_id")
	}
	var result assets.GiftResult
	switch args.TargetKind {
	case "", "user":
		recipientResult := core.ParseUserID(args.RecipientID)
		recipient, recipientMatched := recipientResult.(core.UserIDCreated)
		if !recipientMatched {
			return invalidIDArgument("recipient_id")
		}
		result = server.services.TransferCollectible(ctx, subject.ID, recipient.Value, collectibleID.Value)
	case "organization":
		organizationIDResult := core.ParseOrganizationID(args.RecipientID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			return invalidIDArgument("recipient_id")
		}
		result = server.services.TransferCollectibleToOrganization(ctx, subject.ID, organizationID.Value, collectibleID.Value)
	default:
		return toolProtocolError{code: codeInvalidParams, message: "target_kind must be user or organization"}
	}
	gifted, matched := result.(assets.CollectibleGifted)
	if !matched {
		return toolFailed{code: result.(assets.GiftRejected).Reason.Code(), message: result.(assets.GiftRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(gifted.Value))
}

// callTransferOrgCollectible moves an organization-owned collectible to a
// user. A personal agent credential acts as its user, whose
// manage_collectibles permission is verified inside the store transaction
// (mirroring REST's org-transfer route); an organization credential acts for
// its own organization only, with the permission inherent (org-wide
// credentials act with org-admin parity), through the member-award path — so
// the recipient must then be an active member of the organization.
func (server Server) callTransferOrgCollectible(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	var args struct {
		OrganizationID string `json:"organization_id"`
		CollectibleID  string `json:"collectible_id"`
		RecipientID    string `json:"recipient_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	collectibleIDResult := core.ParseCollectibleID(args.CollectibleID)
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return invalidIDArgument("collectible_id")
	}
	recipientResult := core.ParseUserID(args.RecipientID)
	recipient, recipientMatched := recipientResult.(core.UserIDCreated)
	if !recipientMatched {
		return invalidIDArgument("recipient_id")
	}
	switch typed := subject.(type) {
	case auth.UserSubject:
		organizationIDResult := core.ParseOrganizationID(args.OrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			return invalidIDArgument("organization_id")
		}
		result := server.services.TransferCollectibleFromOrganization(ctx, typed.ID, organizationID.Value, recipient.Value, collectibleID.Value)
		gifted, matched := result.(assets.CollectibleGifted)
		if !matched {
			return toolFailed{code: result.(assets.GiftRejected).Reason.Code(), message: result.(assets.GiftRejected).Reason.Description()}
		}
		return marshalPayload(collectibleToSummary(gifted.Value))
	case auth.OrgSubject:
		if args.OrganizationID != "" && args.OrganizationID != typed.ID.String() {
			return toolFailed{code: core.ErrorCodePermissionDenied, message: "an organization credential may only transfer its own organization's collectibles"}
		}
		result := server.services.AwardOrganizationCollectible(ctx, typed.ID, collectibleID.Value, recipient.Value)
		gifted, matched := result.(assets.CollectibleGifted)
		if !matched {
			return toolFailed{code: result.(assets.GiftRejected).Reason.Code(), message: result.(assets.GiftRejected).Reason.Description()}
		}
		return marshalPayload(collectibleToSummary(gifted.Value))
	default:
		return toolFailed{code: core.ErrorCodePermissionDenied, message: "a personal agent credential or an organization credential is required"}
	}
}

func (server Server) callListCollectibles(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListCollectibles(ctx, subject.ID, page.Probe())
	listed, matched := result.(assets.CollectiblesListed)
	if !matched {
		return toolFailed{code: result.(assets.ListRejected).Reason.Code(), message: result.(assets.ListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	summaries := make([]collectibleSummary, 0, visible)
	for index := range listed.Values[:visible] {
		summaries = append(summaries, collectibleToSummary(listed.Values[index]))
	}
	return marshalPayload(collectiblesListPayload{Collectibles: summaries, NextOffset: nextOffset})
}

func (server Server) callFundCollectibleReward(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	var args struct {
		CollectibleID string `json:"collectible_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	collectibleIDResult := core.ParseCollectibleID(args.CollectibleID)
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return invalidIDArgument("collectible_id")
	}
	result := server.services.FundCollectibleReward(ctx, subject.ID, taskID, collectibleID.Value)
	funded, matched := result.(assets.RewardFunded)
	if !matched {
		return toolFailed{code: result.(assets.FundRewardRejected).Reason.Code(), message: result.(assets.FundRewardRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(funded.Value))
}

func (server Server) callRefundCollectibleReward(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.RefundCollectibleReward(ctx, subject.ID, taskID)
	refunded, matched := result.(assets.RewardRefunded)
	if !matched {
		return toolFailed{code: result.(assets.RefundRewardRejected).Reason.Code(), message: result.(assets.RefundRewardRejected).Reason.Description()}
	}
	summaries := make([]collectibleSummary, 0, len(refunded.Values))
	for index := range refunded.Values {
		summaries = append(summaries, collectibleToSummary(refunded.Values[index]))
	}
	return marshalPayload(collectiblesListPayload{Collectibles: summaries})
}

func (server Server) callListOrganizationCollectibles(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	organizationID, problem := parseOrganizationID(arguments)
	if problem != nil {
		return problem
	}
	return server.listCollectiblesByOwner(ctx, assets.CollectibleOwnerKindOrganization, organizationID.String(), arguments)
}

func (server Server) callListTeamCollectibles(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	teamID, problem := parseTeamID(arguments)
	if problem != nil {
		return problem
	}
	return server.listCollectiblesByOwner(ctx, assets.CollectibleOwnerKindTeam, teamID.String(), arguments)
}

func (server Server) listCollectiblesByOwner(ctx context.Context, ownerKind string, ownerID string, arguments json.RawMessage) toolResult {
	if strings.TrimSpace(ownerID) == "" {
		return toolProtocolError{code: codeInvalidParams, message: "owner id is required"}
	}
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListCollectiblesByOwner(ctx, ownerKind, ownerID, page.Probe())
	listed, matched := result.(assets.CollectiblesListed)
	if !matched {
		return toolFailed{code: result.(assets.ListRejected).Reason.Code(), message: result.(assets.ListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	summaries := make([]collectibleSummary, 0, visible)
	for index := range listed.Values[:visible] {
		summaries = append(summaries, collectibleToSummary(listed.Values[index]))
	}
	return marshalPayload(collectiblesListPayload{Collectibles: summaries, NextOffset: nextOffset})
}

func (server Server) callAwardCollectible(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Slug           string `json:"slug"`
		RecipientKind  string `json:"recipient_kind"`
		RecipientID    string `json:"recipient_id"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	recipientKind := strings.TrimSpace(args.RecipientKind)
	if recipientKind == "" {
		recipientKind = assets.CollectibleOwnerKindUser
	}
	if !assets.ValidCollectibleOwnerKind(recipientKind) {
		return toolProtocolError{code: codeInvalidParams, message: "recipient kind must be user, team, or organization"}
	}
	if strings.TrimSpace(args.RecipientID) == "" {
		return toolProtocolError{code: codeInvalidParams, message: "recipient id is required"}
	}
	result := server.services.AwardCollectible(ctx, subject.ID, strings.TrimSpace(args.Slug), recipientKind, strings.TrimSpace(args.RecipientID), strings.TrimSpace(args.OrganizationID))
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		return toolFailed{code: result.(assets.MintRejected).Reason.Code(), message: result.(assets.MintRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(minted.Value))
}

// deletedPayload is the wire shape of the two hard-delete admin tools,
// mirroring REST's empty "deleted" response.
type deletedPayload struct {
	Status string `json:"status"`
}

func (deletedPayload) payloadValue() {}

func (catalogEntryPayload) payloadValue() {}

func catalogEntryToPayload(entry assets.CatalogEntry) catalogEntryPayload {
	maxEditions := int64(0)
	if capped, matched := entry.Cap.(assets.EditionCapOf); matched {
		maxEditions = capped.Limit
	}
	return catalogEntryPayload{
		Slug:           entry.Slug.String(),
		Name:           entry.Name.String(),
		Kind:           entry.Kind.String(),
		TransferPolicy: entry.Policy.String(),
		Art:            entry.Art,
		State:          entry.State.String(),
		MaxEditions:    maxEditions,
	}
}

// callAddCatalogEntry creates a new awardable catalog entry, mirroring
// REST's POST /api/admin/collectibles/catalog field-for-field. The admin
// re-check happened in dispatch (requireAdminSubjectForTool).
func (server Server) callAddCatalogEntry(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Slug           string `json:"slug"`
		Name           string `json:"name"`
		Kind           string `json:"kind"`
		TransferPolicy string `json:"transfer_policy"`
		Art            string `json:"art"`
		MaxEditions    int64  `json:"max_editions"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	slugResult := assets.NewCatalogSlug(args.Slug)
	slug, slugMatched := slugResult.(assets.CatalogSlugAccepted)
	if !slugMatched {
		return toolProtocolError{code: codeInvalidParams, message: slugResult.(assets.CatalogSlugRejected).Reason.Description()}
	}
	template, templateProblem := parseMCPCollectibleTemplate(args.Name, args.Kind, args.TransferPolicy)
	if templateProblem != nil {
		return templateProblem
	}
	if args.MaxEditions < 0 {
		return toolProtocolError{code: codeInvalidParams, message: "max_editions cannot be negative"}
	}
	var editionCap assets.EditionCap = assets.NoEditionCap{}
	if args.MaxEditions > 0 {
		editionCap = assets.EditionCapOf{Limit: args.MaxEditions}
	}
	result := server.services.AddCatalogEntry(ctx, assets.CatalogEntry{
		Slug:   slug.Value,
		Name:   template.name,
		Kind:   template.kind,
		Policy: template.policy,
		Art:    args.Art,
		State:  assets.CatalogEntryStateAvailable,
		Cap:    editionCap,
	})
	mutated, matched := result.(assets.CatalogMutated)
	if !matched {
		return toolFailed{code: result.(assets.CatalogMutationRejected).Reason.Code(), message: result.(assets.CatalogMutationRejected).Reason.Description()}
	}
	return marshalPayload(catalogEntryToPayload(mutated.Value))
}

func parseMCPCatalogSlug(arguments json.RawMessage) (assets.CatalogSlug, toolResult) {
	var args struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return assets.CatalogSlug{}, invalidArguments()
	}
	slugResult := assets.NewCatalogSlug(args.Slug)
	slug, matched := slugResult.(assets.CatalogSlugAccepted)
	if !matched {
		return assets.CatalogSlug{}, toolProtocolError{code: codeInvalidParams, message: slugResult.(assets.CatalogSlugRejected).Reason.Description()}
	}
	return slug.Value, nil
}

// callWithdrawCatalogEntry marks an entry no longer awardable; existing
// instances are unaffected. Admin-gated in dispatch.
func (server Server) callWithdrawCatalogEntry(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	slug, problem := parseMCPCatalogSlug(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.WithdrawCatalogEntry(ctx, slug)
	mutated, matched := result.(assets.CatalogMutated)
	if !matched {
		return toolFailed{code: result.(assets.CatalogMutationRejected).Reason.Code(), message: result.(assets.CatalogMutationRejected).Reason.Description()}
	}
	return marshalPayload(catalogEntryToPayload(mutated.Value))
}

// callReleaseCatalogEntry returns a withdrawn entry to the available state
// so it can be awarded again; an entry that is not withdrawn is refused with
// a conflict. Admin-gated in dispatch.
func (server Server) callReleaseCatalogEntry(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	slug, problem := parseMCPCatalogSlug(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.ReleaseCatalogEntry(ctx, slug)
	mutated, matched := result.(assets.CatalogMutated)
	if !matched {
		return toolFailed{code: result.(assets.CatalogMutationRejected).Reason.Code(), message: result.(assets.CatalogMutationRejected).Reason.Description()}
	}
	return marshalPayload(catalogEntryToPayload(mutated.Value))
}

// callDeleteCatalogEntry removes a withdrawn entry that no instance — live
// or withdrawn — references; a not-yet-withdrawn entry or one with remaining
// instances is refused with a conflict. Admin-gated in dispatch.
func (server Server) callDeleteCatalogEntry(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	slug, problem := parseMCPCatalogSlug(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.DeleteCatalogEntry(ctx, slug)
	if _, matched := result.(assets.CatalogMutated); !matched {
		return toolFailed{code: result.(assets.CatalogMutationRejected).Reason.Code(), message: result.(assets.CatalogMutationRejected).Reason.Description()}
	}
	return marshalPayload(deletedPayload{Status: "deleted"})
}

// callWithdrawCollectible removes a catalog-minted instance from its holder;
// the former holder learns through the collectible_withdrawn event.
// Admin-gated in dispatch.
func (server Server) callWithdrawCollectible(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		CollectibleID string `json:"collectible_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	collectibleIDResult := core.ParseCollectibleID(args.CollectibleID)
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return invalidIDArgument("collectible_id")
	}
	result := server.services.WithdrawCollectible(ctx, subject.ID, collectibleID.Value)
	withdrawn, matched := result.(assets.CollectibleWithdrawn)
	if !matched {
		return toolFailed{code: result.(assets.WithdrawRejected).Reason.Code(), message: result.(assets.WithdrawRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(withdrawn.Value))
}

// callReleaseCollectible returns a withdrawn instance to its holder's
// inventory; the holder learns through the collectible_released event. A
// non-withdrawn instance, or a unique whose live slot was re-minted, is
// refused with a conflict. Admin-gated in dispatch.
func (server Server) callReleaseCollectible(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		CollectibleID string `json:"collectible_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	collectibleIDResult := core.ParseCollectibleID(args.CollectibleID)
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return invalidIDArgument("collectible_id")
	}
	result := server.services.ReleaseCollectible(ctx, subject.ID, collectibleID.Value)
	released, matched := result.(assets.CollectibleReleased)
	if !matched {
		return toolFailed{code: result.(assets.ReleaseRejected).Reason.Code(), message: result.(assets.ReleaseRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(released.Value))
}

// callDeleteWithdrawnCollectible hard-deletes a withdrawn instance; every
// other state is refused with a conflict. Admin-gated in dispatch.
func (server Server) callDeleteWithdrawnCollectible(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		CollectibleID string `json:"collectible_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	collectibleIDResult := core.ParseCollectibleID(args.CollectibleID)
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return invalidIDArgument("collectible_id")
	}
	result := server.services.DeleteWithdrawnCollectible(ctx, collectibleID.Value)
	if _, matched := result.(assets.CollectibleDeleted); !matched {
		return toolFailed{code: result.(assets.DeleteCollectibleRejected).Reason.Code(), message: result.(assets.DeleteCollectibleRejected).Reason.Description()}
	}
	return marshalPayload(deletedPayload{Status: "deleted"})
}
