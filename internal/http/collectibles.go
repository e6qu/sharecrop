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
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	TransferPolicy string `json:"transfer_policy"`
	Art            string `json:"art"`
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
}

type collectiblesResponse struct {
	Collectibles []collectibleResponse `json:"collectibles"`
}

type catalogEntryResponse struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	TransferPolicy string `json:"transfer_policy"`
	Art            string `json:"art"`
}

type collectibleCatalogResponse struct {
	Entries []catalogEntryResponse `json:"entries"`
}

type collectibleRewardRequest struct {
	CollectibleID string `json:"collectible_id"`
}

type awardCollectibleRequest struct {
	Slug           string `json:"slug"`
	RecipientKind  string `json:"recipient_kind"`
	RecipientID    string `json:"recipient_id"`
	OrganizationID string `json:"organization_id"`
}

type transferCollectibleRequest struct {
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
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	var request mintCollectibleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
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

	result := server.assetService.Mint(r.Context(), assets.CollectibleOwnerKindUser, actor.subject.ID.String(), "", name.Value, kind.Value, policy.Value, request.Art)
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		writeDomainError(w, result.(assets.MintRejected).Reason)
		return
	}

	writeJSON(w, http.StatusCreated, collectibleToResponse(minted.Value))
}

// collectibleCatalog lists the platform's default collectibles (the templates an
// admin can award). It is a fixed, code-defined set, so no store is consulted.
func (server Server) collectibleCatalog(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	if _, matched := actorResult.(userSubjectAccepted); !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	entries := assets.Catalog()
	response := collectibleCatalogResponse{Entries: make([]catalogEntryResponse, 0, len(entries))}
	for index := range entries {
		response.Entries = append(response.Entries, catalogEntryResponse{
			Slug:           entries[index].Slug,
			Name:           entries[index].Name,
			Kind:           entries[index].Kind.String(),
			TransferPolicy: entries[index].Policy.String(),
			Art:            entries[index].Art,
		})
	}
	writeJSON(w, http.StatusOK, response)
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
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	entry, found := assets.CatalogBySlug(request.Slug)
	if !found {
		writeError(w, http.StatusBadRequest, "unknown default collectible")
		return
	}
	recipientKind := request.RecipientKind
	if recipientKind == "" {
		recipientKind = assets.CollectibleOwnerKindUser
	}
	if !assets.ValidCollectibleOwnerKind(recipientKind) {
		writeError(w, http.StatusBadRequest, "recipient kind must be user, team, or organization")
		return
	}
	if strings.TrimSpace(request.RecipientID) == "" {
		writeError(w, http.StatusBadRequest, "recipient id is required")
		return
	}
	nameResult := assets.NewCollectibleName(entry.Name)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		writeDomainError(w, nameResult.(assets.CollectibleNameRejected).Reason)
		return
	}

	result := server.assetService.Mint(r.Context(), recipientKind, strings.TrimSpace(request.RecipientID), strings.TrimSpace(request.OrganizationID), name.Value, entry.Kind, entry.Policy, entry.Art)
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		writeDomainError(w, result.(assets.MintRejected).Reason)
		return
	}
	if !server.recordAudit(w, r.Context(), actor.subject.ID, audit.ActionAdminCollectibleAwarded, audit.Subject{Kind: "collectible", ID: minted.Value.ID.String()}, audit.EmptyMetadata()) {
		return
	}
	writeJSON(w, http.StatusCreated, collectibleToResponse(minted.Value))
}

// transferCollectible moves an owned, transferable collectible to another user,
// enforcing the transfer policy in the store transaction.
func (server Server) transferCollectible(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	collectibleIDResult := core.ParseCollectibleID(r.PathValue("id"))
	collectibleID, idMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !idMatched {
		writeDomainError(w, collectibleIDResult.(core.CollectibleIDRejected).Reason)
		return
	}
	var request transferCollectibleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	recipientResult := core.ParseUserID(request.RecipientID)
	recipient, recipientMatched := recipientResult.(core.UserIDCreated)
	if !recipientMatched {
		writeDomainError(w, recipientResult.(core.UserIDRejected).Reason)
		return
	}

	result := server.assetService.GiftCollectible(r.Context(), actor.subject.ID, recipient.Value, collectibleID.Value)
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
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	result := server.assetService.ListCollectibles(r.Context(), actor.subject.ID, parsePage(r))
	listed, matched := result.(assets.CollectiblesListed)
	if !matched {
		writeDomainError(w, result.(assets.ListRejected).Reason)
		return
	}

	response := collectiblesResponse{Collectibles: make([]collectibleResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		response.Collectibles = append(response.Collectibles, collectibleToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) fundCollectibleReward(w http.ResponseWriter, r *http.Request) {
	actor, taskID, ok := server.collectibleRewardActor(w, r)
	if !ok {
		return
	}

	var request collectibleRewardRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
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
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return actorSubject{}, core.TaskID{}, false
	}
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return actorSubject{}, core.TaskID{}, false
	}
	return actorSubject{ID: actor.subject.ID}, taskIDAccepted.value, true
}

type actorSubject struct {
	ID core.UserID
}

// listOrganizationCollectibles and listTeamCollectibles expose the collectibles
// held by an organization or team (e.g. defaults an admin awarded to them).
// This is a public "trophy case" view (any authenticated user, not just
// members, can see what an org/team holds) - deliberately not membership-
// gated, matching awardOrganizationCollectible's comment about defaults an
// admin awarded being visible more broadly than the award action itself.
func (server Server) listOrganizationCollectibles(w http.ResponseWriter, r *http.Request) {
	organizationIDResult := parseOrganizationPathValue(r)
	organizationID, organizationMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationMatched {
		writeError(w, http.StatusBadRequest, organizationIDResult.(organizationIDRejected).reason)
		return
	}
	server.listOwnerCollectibles(w, r, assets.CollectibleOwnerKindOrganization, organizationID.value.String())
}

func (server Server) listTeamCollectibles(w http.ResponseWriter, r *http.Request) {
	teamIDResult := core.ParseTeamID(r.PathValue("id"))
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
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	organizationIDResult := parseOrganizationPathValue(r)
	organizationID, organizationMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationMatched {
		writeError(w, http.StatusBadRequest, organizationIDResult.(organizationIDRejected).reason)
		return
	}
	collectibleIDResult := core.ParseCollectibleID(r.PathValue("id"))
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		writeDomainError(w, collectibleIDResult.(core.CollectibleIDRejected).Reason)
		return
	}

	check := server.organizationService.CheckOrganizationPermission(r.Context(), organizationID.value, actor.subject.ID, org.PermissionManageMembers)
	if rejected, denied := check.(org.PermissionDenied); denied {
		writeDomainError(w, rejected.Reason)
		return
	}

	var request awardOrganizationCollectibleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
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
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	result := server.assetService.ListByOwner(r.Context(), ownerKind, ownerID, parsePage(r))
	listed, matched := result.(assets.CollectiblesListed)
	if !matched {
		writeDomainError(w, result.(assets.ListRejected).Reason)
		return
	}
	response := collectiblesResponse{Collectibles: make([]collectibleResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		response.Collectibles = append(response.Collectibles, collectibleToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func collectibleToResponse(value assets.Collectible) collectibleResponse {
	return collectibleResponse{
		ID:             value.ID.String(),
		Name:           value.Name.String(),
		Kind:           value.Kind.String(),
		State:          value.State.String(),
		TransferPolicy: value.Policy.String(),
		OwnerID:        value.OwnerID,
		OwnerKind:      value.OwnerKind,
		OrganizationID: value.OrganizationID,
		Art:            value.Art,
	}
}
