package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
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
	Slug          string `json:"slug"`
	RecipientKind string `json:"recipient_kind"`
	RecipientID   string `json:"recipient_id"`
}

type transferCollectibleRequest struct {
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
		writeError(w, http.StatusBadRequest, nameResult.(assets.CollectibleNameRejected).Reason.Description())
		return
	}
	kindResult := assets.ParseCollectibleKind(request.Kind)
	kind, kindMatched := kindResult.(assets.CollectibleKindAccepted)
	if !kindMatched {
		writeError(w, http.StatusBadRequest, kindResult.(assets.CollectibleKindRejected).Reason.Description())
		return
	}
	policyResult := assets.ParseTransferPolicy(request.TransferPolicy)
	policy, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		writeError(w, http.StatusBadRequest, policyResult.(assets.TransferPolicyRejected).Reason.Description())
		return
	}

	result := server.assetService.Mint(r.Context(), actor.subject.ID, name.Value, kind.Value, policy.Value, request.Art)
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(assets.MintRejected).Reason.Description())
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
	actorResult := server.requireUserSubject(r)
	if _, matched := actorResult.(userSubjectAccepted); !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
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
	if request.RecipientKind != "" && request.RecipientKind != "user" {
		writeError(w, http.StatusBadRequest, "awarding to a team or organization is only available in the demo")
		return
	}
	recipientResult := core.ParseUserID(request.RecipientID)
	recipient, recipientMatched := recipientResult.(core.UserIDCreated)
	if !recipientMatched {
		writeError(w, http.StatusBadRequest, recipientResult.(core.UserIDRejected).Reason.Description())
		return
	}
	nameResult := assets.NewCollectibleName(entry.Name)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		writeError(w, http.StatusBadRequest, nameResult.(assets.CollectibleNameRejected).Reason.Description())
		return
	}

	result := server.assetService.Mint(r.Context(), recipient.Value, name.Value, entry.Kind, entry.Policy, entry.Art)
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(assets.MintRejected).Reason.Description())
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
		writeError(w, http.StatusBadRequest, collectibleIDResult.(core.CollectibleIDRejected).Reason.Description())
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
		writeError(w, http.StatusBadRequest, recipientResult.(core.UserIDRejected).Reason.Description())
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
		writeError(w, http.StatusBadRequest, result.(assets.ListRejected).Reason.Description())
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
		writeError(w, http.StatusBadRequest, collectibleIDResult.(core.CollectibleIDRejected).Reason.Description())
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

func collectibleToResponse(value assets.Collectible) collectibleResponse {
	return collectibleResponse{
		ID:             value.ID.String(),
		Name:           value.Name.String(),
		Kind:           value.Kind.String(),
		State:          value.State.String(),
		TransferPolicy: value.Policy.String(),
		OwnerID:        value.OwnerID.String(),
		Art:            value.Art,
	}
}
