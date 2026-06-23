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
}

type collectibleResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	State          string `json:"state"`
	TransferPolicy string `json:"transfer_policy"`
	OwnerID        string `json:"owner_id"`
}

type collectiblesResponse struct {
	Collectibles []collectibleResponse `json:"collectibles"`
}

type collectibleRewardRequest struct {
	CollectibleID string `json:"collectible_id"`
}

func (collectibleResponse) writableResponse() {}

func (collectiblesResponse) writableResponse() {}

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

	result := server.assetService.Mint(r.Context(), actor.subject.ID, name.Value, kind.Value, policy.Value)
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(assets.MintRejected).Reason.Description())
		return
	}

	writeJSON(w, http.StatusCreated, collectibleToResponse(minted.Value))
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
	}
}
