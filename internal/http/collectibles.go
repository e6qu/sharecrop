package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
)

type mintCollectibleRequest struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	// TransferPolicy is optional; absent mints freely tradeable.
	TransferPolicy string `json:"transfer_policy,omitempty"`
	// Art is optional free-form art for custom mints (catalog entries, by
	// contrast, must name a sprite from the fixed registry).
	Art string `json:"art,omitempty"`
}

type collectibleResponse struct {
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
	// from; empty for custom mints.
	CatalogSlug string `json:"catalog_slug"`
	// EditionNumber is the mint sequence number of an edition instance; 0 for
	// unnumbered instances (badges, uniques).
	EditionNumber int64 `json:"edition_number"`
	// IssuerDisplayName names the user who minted or awarded the instance.
	// Resolved on the list read paths; mutation responses and pre-provenance
	// rows leave it empty.
	IssuerDisplayName string `json:"issuer_display_name"`
}

type collectiblesResponse struct {
	Collectibles []collectibleResponse `json:"collectibles"`
	NextOffset   int                   `json:"next_offset"`
}

type catalogEntryResponse struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	TransferPolicy string `json:"transfer_policy"`
	Art            string `json:"art"`
	// State is the entry lifecycle: available (awardable) or withdrawn.
	State string `json:"state"`
	// MaxEditions bounds how many instances may ever be minted: 1 for
	// uniques, the declared run size for editions, 0 for uncapped badges.
	MaxEditions int64 `json:"max_editions"`
	// MintedCount counts the entry's live (non-withdrawn) instances.
	MintedCount int64 `json:"minted_count"`
}

type collectibleCatalogResponse struct {
	Entries []catalogEntryResponse `json:"entries"`
}

type collectibleRewardRequest struct {
	CollectibleID string `json:"collectible_id"`
}

type awardCollectibleRequest struct {
	Slug           string `json:"slug"`
	RecipientKind  string `json:"recipient_kind,omitempty"`
	RecipientID    string `json:"recipient_id"`
	OrganizationID string `json:"organization_id,omitempty"`
}

// catalogEntryRequest adds one collectible template to the catalog.
// max_editions is required to be 1 for kind unique, a positive run size for
// kind edition, and must be absent (0) for uncapped badges.
type catalogEntryRequest struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	TransferPolicy string `json:"transfer_policy"`
	Art            string `json:"art"`
	MaxEditions    int64  `json:"max_editions,omitempty"`
}

type transferCollectibleRequest struct {
	// TargetKind says who receives the collectible: "user" (the default when
	// absent) or "organization" (a donation to an organization's trophy
	// case). RecipientID is the matching user or organization id.
	TargetKind  string `json:"target_kind,omitempty"`
	RecipientID string `json:"recipient_id"`
}

type awardOrganizationCollectibleRequest struct {
	RecipientID string `json:"recipient_id"`
}

func (collectibleResponse) writableResponse() {}

func (collectiblesResponse) writableResponse() {}

func (collectibleCatalogResponse) writableResponse() {}

func (server Server) mintCollectible(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	var request mintCollectibleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}

	nameResult := assets.NewCollectibleName(request.Name)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		writeDomainError(w, nameResult.(assets.CollectibleNameRejected).Reason)
		return
	}
	kindResult := assets.ParseCollectibleKind(request.Kind)
	kind, kindMatched := kindResult.(assets.CollectibleKindAccepted)
	if !kindMatched {
		writeDomainError(w, kindResult.(assets.CollectibleKindRejected).Reason)
		return
	}
	// New mints default to freely tradeable unless the minter explicitly
	// restricts them.
	rawPolicy := request.TransferPolicy
	if strings.TrimSpace(rawPolicy) == "" {
		rawPolicy = assets.TransferPolicyTransferableBetweenUsers.String()
	}
	policyResult := assets.ParseTransferPolicy(rawPolicy)
	policy, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		writeDomainError(w, policyResult.(assets.TransferPolicyRejected).Reason)
		return
	}

	result := server.assetService.Mint(r.Context(), actor.subject.ID, assets.CollectibleOwnerKindUser, actor.subject.ID.String(), "", name.Value, kind.Value, policy.Value, request.Art)
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		writeDomainError(w, result.(assets.MintRejected).Reason)
		return
	}

	writeJSON(w, http.StatusCreated, collectibleToResponse(minted.Value))
}

// collectibleCatalog lists every collectible catalog entry (the templates an
// admin can award) with its lifecycle state, edition cap, and live-instance
// count. Withdrawn entries stay listed (their instances remain in circulation)
// but are no longer awardable.
func (server Server) collectibleCatalog(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	if _, matched := actorResult.(userSubjectAccepted); !matched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	result := server.assetService.ListCatalog(r.Context())
	listed, matched := result.(assets.CatalogListed)
	if !matched {
		writeDomainError(w, result.(assets.CatalogListRejected).Reason)
		return
	}
	response := collectibleCatalogResponse{Entries: make([]catalogEntryResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		entry := catalogEntryToResponse(listed.Values[index].Entry)
		entry.MintedCount = listed.Values[index].LiveInstanceCount
		response.Entries = append(response.Entries, entry)
	}
	writeJSON(w, http.StatusOK, response)
}

func catalogEntryToResponse(entry assets.CatalogEntry) catalogEntryResponse {
	maxEditions := int64(0)
	if capped, matched := entry.Cap.(assets.EditionCapOf); matched {
		maxEditions = capped.Limit
	}
	return catalogEntryResponse{
		Slug:           entry.Slug.String(),
		Name:           entry.Name.String(),
		Kind:           entry.Kind.String(),
		TransferPolicy: entry.Policy.String(),
		Art:            entry.Art,
		State:          entry.State.String(),
		MaxEditions:    maxEditions,
	}
}

func (catalogEntryResponse) writableResponse() {}

// addCatalogEntry creates a new awardable catalog entry (platform admin).
func (server Server) addCatalogEntry(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}

	var request catalogEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	slugResult := assets.NewCatalogSlug(request.Slug)
	slug, slugMatched := slugResult.(assets.CatalogSlugAccepted)
	if !slugMatched {
		writeDomainError(w, slugResult.(assets.CatalogSlugRejected).Reason)
		return
	}
	nameResult := assets.NewCollectibleName(request.Name)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		writeDomainError(w, nameResult.(assets.CollectibleNameRejected).Reason)
		return
	}
	kindResult := assets.ParseCollectibleKind(request.Kind)
	kind, kindMatched := kindResult.(assets.CollectibleKindAccepted)
	if !kindMatched {
		writeDomainError(w, kindResult.(assets.CollectibleKindRejected).Reason)
		return
	}
	policyResult := assets.ParseTransferPolicy(request.TransferPolicy)
	policy, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		writeDomainError(w, policyResult.(assets.TransferPolicyRejected).Reason)
		return
	}
	if request.MaxEditions < 0 {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "max_editions cannot be negative")
		return
	}
	var cap assets.EditionCap = assets.NoEditionCap{}
	if request.MaxEditions > 0 {
		cap = assets.EditionCapOf{Limit: request.MaxEditions}
	}

	result := server.assetService.AddCatalogEntry(r.Context(), assets.CatalogEntry{
		Slug:   slug.Value,
		Name:   name.Value,
		Kind:   kind.Value,
		Policy: policy.Value,
		Art:    request.Art,
		State:  assets.CatalogEntryStateAvailable,
		Cap:    cap,
	})
	mutated, matched := result.(assets.CatalogMutated)
	if !matched {
		writeDomainError(w, result.(assets.CatalogMutationRejected).Reason)
		return
	}
	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionAdminCatalogEntryAdded, audit.Subject{Kind: "collectible_catalog_entry", ID: mutated.Value.Slug.String()}, audit.EmptyMetadata())
	writeJSON(w, http.StatusCreated, catalogEntryToResponse(mutated.Value))
}

// withdrawCatalogEntry marks an entry no longer awardable (platform admin);
// existing instances are unaffected.
func (server Server) withdrawCatalogEntry(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	slugResult := assets.NewCatalogSlug(r.PathValue("slug"))
	slug, slugMatched := slugResult.(assets.CatalogSlugAccepted)
	if !slugMatched {
		writeDomainError(w, slugResult.(assets.CatalogSlugRejected).Reason)
		return
	}
	result := server.assetService.WithdrawCatalogEntry(r.Context(), slug.Value)
	mutated, matched := result.(assets.CatalogMutated)
	if !matched {
		writeDomainError(w, result.(assets.CatalogMutationRejected).Reason)
		return
	}
	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionAdminCatalogEntryWithdrawn, audit.Subject{Kind: "collectible_catalog_entry", ID: mutated.Value.Slug.String()}, audit.EmptyMetadata())
	writeJSON(w, http.StatusOK, catalogEntryToResponse(mutated.Value))
}

// deleteCatalogEntry removes a withdrawn entry that no live instance
// references (platform admin). A not-yet-withdrawn entry or one with live
// instances is refused with a conflict.
func (server Server) deleteCatalogEntry(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	slugResult := assets.NewCatalogSlug(r.PathValue("slug"))
	slug, slugMatched := slugResult.(assets.CatalogSlugAccepted)
	if !slugMatched {
		writeDomainError(w, slugResult.(assets.CatalogSlugRejected).Reason)
		return
	}
	result := server.assetService.DeleteCatalogEntry(r.Context(), slug.Value)
	mutated, matched := result.(assets.CatalogMutated)
	if !matched {
		writeDomainError(w, result.(assets.CatalogMutationRejected).Reason)
		return
	}
	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionAdminCatalogEntryDeleted, audit.Subject{Kind: "collectible_catalog_entry", ID: mutated.Value.Slug.String()}, audit.EmptyMetadata())
	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "deleted"})
}

// withdrawCollectible removes a catalog-minted instance from its holder
// (platform admin); the former holder is notified through the
// collectible_withdrawn event.
func (server Server) withdrawCollectible(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	collectibleIDResult := core.ParseCollectibleID(r.PathValue("collectible_id"))
	collectibleID, idMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !idMatched {
		writeDomainError(w, collectibleIDResult.(core.CollectibleIDRejected).Reason)
		return
	}
	result := server.assetService.WithdrawCollectible(r.Context(), actor.subject.ID, collectibleID.Value)
	withdrawn, matched := result.(assets.CollectibleWithdrawn)
	if !matched {
		writeDomainError(w, result.(assets.WithdrawRejected).Reason)
		return
	}
	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionAdminCollectibleWithdrawn, audit.Subject{Kind: "collectible", ID: withdrawn.Value.ID.String()}, audit.EmptyMetadata())
	writeJSON(w, http.StatusOK, collectibleToResponse(withdrawn.Value))
}

// deleteCollectible hard-deletes a withdrawn instance (platform admin). Every
// other state is refused with a conflict.
func (server Server) deleteCollectible(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	collectibleIDResult := core.ParseCollectibleID(r.PathValue("collectible_id"))
	collectibleID, idMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !idMatched {
		writeDomainError(w, collectibleIDResult.(core.CollectibleIDRejected).Reason)
		return
	}
	result := server.assetService.DeleteWithdrawnCollectible(r.Context(), collectibleID.Value)
	if _, matched := result.(assets.CollectibleDeleted); !matched {
		writeDomainError(w, result.(assets.DeleteCollectibleRejected).Reason)
		return
	}
	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionAdminCollectibleDeleted, audit.Subject{Kind: "collectible", ID: collectibleID.Value.String()}, audit.EmptyMetadata())
	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "deleted"})
}

// awardCollectible mints a fresh copy of a default-catalog collectible owned by
// the recipient. Trading then lets the recipient move it on.
func (server Server) awardCollectible(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}

	var request awardCollectibleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	recipientKind := request.RecipientKind
	if recipientKind == "" {
		recipientKind = assets.CollectibleOwnerKindUser
	}
	if !assets.ValidCollectibleOwnerKind(recipientKind) {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "recipient kind must be user, team, or organization")
		return
	}
	if strings.TrimSpace(request.RecipientID) == "" {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "recipient id is required")
		return
	}

	result := server.assetService.AwardFromCatalog(r.Context(), actor.subject.ID, strings.TrimSpace(request.Slug), recipientKind, strings.TrimSpace(request.RecipientID), strings.TrimSpace(request.OrganizationID))
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		writeDomainError(w, result.(assets.MintRejected).Reason)
		return
	}
	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionAdminCollectibleAwarded, audit.Subject{Kind: "collectible", ID: minted.Value.ID.String()}, audit.EmptyMetadata())
	writeJSON(w, http.StatusCreated, collectibleToResponse(minted.Value))
}

// transferCollectible moves an owned, transferable collectible to another
// user (the default) or donates it to an organization, enforcing ownership,
// availability, and the transfer policy in the store transaction.
func (server Server) transferCollectible(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}
	collectibleIDResult := core.ParseCollectibleID(r.PathValue("collectible_id"))
	collectibleID, idMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !idMatched {
		writeDomainError(w, collectibleIDResult.(core.CollectibleIDRejected).Reason)
		return
	}
	var request transferCollectibleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}

	var result assets.GiftResult
	switch request.TargetKind {
	case "", "user":
		recipientResult := core.ParseUserID(request.RecipientID)
		recipient, recipientMatched := recipientResult.(core.UserIDCreated)
		if !recipientMatched {
			writeDomainError(w, recipientResult.(core.UserIDRejected).Reason)
			return
		}
		result = server.assetService.GiftCollectible(r.Context(), actor.subject.ID, recipient.Value, collectibleID.Value)
	case "organization":
		organizationIDResult := core.ParseOrganizationID(request.RecipientID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			writeDomainError(w, organizationIDResult.(core.OrganizationIDRejected).Reason)
			return
		}
		result = server.assetService.TransferCollectibleToOrganization(r.Context(), actor.subject.ID, organizationID.Value, collectibleID.Value)
	default:
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidEnum, "collectible transfer target kind must be user or organization")
		return
	}

	gifted, matched := result.(assets.CollectibleGifted)
	if !matched {
		writeDomainError(w, result.(assets.GiftRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, collectibleToResponse(gifted.Value))
}

// transferOrganizationCollectible moves an organization-owned collectible to
// a user. The acting member's manage_collectibles permission is verified
// inside the store transaction, mirroring the org-award route shape.
func (server Server) transferOrganizationCollectible(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}
	organizationIDResult := parseOrganizationPathValue(r)
	organizationID, organizationMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, organizationIDResult.(organizationIDRejected).reason)
		return
	}
	collectibleIDResult := core.ParseCollectibleID(r.PathValue("collectible_id"))
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		writeDomainError(w, collectibleIDResult.(core.CollectibleIDRejected).Reason)
		return
	}
	var request awardOrganizationCollectibleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	recipientResult := core.ParseUserID(request.RecipientID)
	recipient, recipientMatched := recipientResult.(core.UserIDCreated)
	if !recipientMatched {
		writeDomainError(w, recipientResult.(core.UserIDRejected).Reason)
		return
	}

	result := server.assetService.TransferCollectibleFromOrganization(r.Context(), actor.subject.ID, organizationID.value, recipient.Value, collectibleID.Value)
	gifted, matched := result.(assets.CollectibleGifted)
	if !matched {
		writeDomainError(w, result.(assets.GiftRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, collectibleToResponse(gifted.Value))
}

func (server Server) listCollectibles(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.assetService.ListCollectibles(r.Context(), actor.subject.ID, page.Probe())
	listed, matched := result.(assets.CollectiblesListed)
	if !matched {
		writeDomainError(w, result.(assets.ListRejected).Reason)
		return
	}

	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := collectiblesResponse{Collectibles: make([]collectibleResponse, 0, visible), NextOffset: nextOffset}
	for index := range listed.Values[:visible] {
		response.Collectibles = append(response.Collectibles, collectibleToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) fundCollectibleReward(w http.ResponseWriter, r *http.Request) {
	actor, taskID, ok := server.collectibleRewardActor(w, r)
	if !ok {
		return
	}
	if !server.allowBySubject(w, actor.ID.String()) {
		return
	}

	var request collectibleRewardRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	collectibleIDResult := core.ParseCollectibleID(request.CollectibleID)
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		writeDomainError(w, collectibleIDResult.(core.CollectibleIDRejected).Reason)
		return
	}

	result := server.assetService.FundReward(r.Context(), actor.ID, taskID, collectibleID.Value)
	funded, matched := result.(assets.RewardFunded)
	if !matched {
		writeDomainError(w, result.(assets.FundRewardRejected).Reason)
		return
	}
	writeJSON(w, http.StatusCreated, collectibleToResponse(funded.Value))
}

func (server Server) refundCollectibleReward(w http.ResponseWriter, r *http.Request) {
	actor, taskID, ok := server.collectibleRewardActor(w, r)
	if !ok {
		return
	}

	result := server.assetService.RefundReward(r.Context(), actor.ID, taskID)
	refunded, matched := result.(assets.RewardRefunded)
	if !matched {
		writeDomainError(w, result.(assets.RefundRewardRejected).Reason)
		return
	}
	response := collectiblesResponse{Collectibles: make([]collectibleResponse, 0, len(refunded.Values))}
	for index := range refunded.Values {
		response.Collectibles = append(response.Collectibles, collectibleToResponse(refunded.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) collectibleRewardActor(w http.ResponseWriter, r *http.Request) (actorSubject, core.TaskID, bool) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return actorSubject{}, core.TaskID{}, false
	}
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, taskIDResult.(taskIDRejected).reason)
		return actorSubject{}, core.TaskID{}, false
	}
	return actorSubject{ID: actor.subject.ID}, taskIDAccepted.value, true
}

type actorSubject struct {
	ID core.UserID
}

// listOrganizationCollectibles and listTeamCollectibles expose the collectibles
// held by an organization or team (e.g. defaults an admin awarded to them).
// This is a public "trophy case" view (every authenticated user, not just
// members, can see what an org/team holds) - deliberately not membership-
// gated, matching awardOrganizationCollectible's comment about defaults an
// admin awarded being visible more broadly than the award action itself.
func (server Server) listOrganizationCollectibles(w http.ResponseWriter, r *http.Request) {
	organizationIDResult := parseOrganizationPathValue(r)
	organizationID, organizationMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, organizationIDResult.(organizationIDRejected).reason)
		return
	}
	server.listOwnerCollectibles(w, r, assets.CollectibleOwnerKindOrganization, organizationID.value.String())
}

func (server Server) listTeamCollectibles(w http.ResponseWriter, r *http.Request) {
	teamIDResult := core.ParseTeamID(r.PathValue("team_id"))
	teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
	if !teamMatched {
		writeDomainError(w, teamIDResult.(core.TeamIDRejected).Reason)
		return
	}
	server.listOwnerCollectibles(w, r, assets.CollectibleOwnerKindTeam, teamID.Value.String())
}

func (server Server) awardOrganizationCollectible(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}
	organizationIDResult := parseOrganizationPathValue(r)
	organizationID, organizationMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, organizationIDResult.(organizationIDRejected).reason)
		return
	}
	collectibleIDResult := core.ParseCollectibleID(r.PathValue("collectible_id"))
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		writeDomainError(w, collectibleIDResult.(core.CollectibleIDRejected).Reason)
		return
	}

	// Awarding organization collectibles is a collectible operation, gated by
	// the dedicated manage_collectibles permission (owner and admin roles
	// grant it) rather than piggybacking on member management.
	check := server.organizationService.CheckOrganizationPermission(r.Context(), organizationID.value, actor.subject.ID, org.PermissionManageCollectibles)
	if rejected, denied := check.(org.PermissionDenied); denied {
		writeDomainError(w, rejected.Reason)
		return
	}

	var request awardOrganizationCollectibleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	recipientResult := core.ParseUserID(request.RecipientID)
	recipient, recipientMatched := recipientResult.(core.UserIDCreated)
	if !recipientMatched {
		writeDomainError(w, recipientResult.(core.UserIDRejected).Reason)
		return
	}

	result := server.assetService.AwardOrganizationCollectible(r.Context(), organizationID.value, collectibleID.Value, recipient.Value)
	awarded, matched := result.(assets.CollectibleGifted)
	if !matched {
		writeDomainError(w, result.(assets.GiftRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, collectibleToResponse(awarded.Value))
}

func (server Server) listOwnerCollectibles(w http.ResponseWriter, r *http.Request, ownerKind string, ownerID string) {
	actorResult := server.requireUserSubject(r)
	if _, matched := actorResult.(userSubjectAccepted); !matched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.assetService.ListByOwner(r.Context(), ownerKind, ownerID, page.Probe())
	listed, matched := result.(assets.CollectiblesListed)
	if !matched {
		writeDomainError(w, result.(assets.ListRejected).Reason)
		return
	}
	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := collectiblesResponse{Collectibles: make([]collectibleResponse, 0, visible), NextOffset: nextOffset}
	for index := range listed.Values[:visible] {
		response.Collectibles = append(response.Collectibles, collectibleToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func collectibleToResponse(value assets.Collectible) collectibleResponse {
	catalogSlug := ""
	if fromCatalog, matched := value.Catalog.(assets.FromCatalog); matched {
		catalogSlug = fromCatalog.Slug.String()
	}
	editionNumber := int64(0)
	if numbered, matched := value.Edition.(assets.EditionNumbered); matched {
		editionNumber = numbered.Number
	}
	return collectibleResponse{
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
	}
}
