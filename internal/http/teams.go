package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
)

type teamDetailResponse struct {
	Team    teamResponse `json:"team"`
	Members []string     `json:"members"`
}

func (server Server) getTeam(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	teamIDResult := core.ParseTeamID(r.PathValue("team_id"))
	teamIDCreated, teamIDMatched := teamIDResult.(core.TeamIDCreated)
	if !teamIDMatched {
		writeError(w, http.StatusBadRequest, teamIDResult.(core.TeamIDRejected).Reason.Description())
		return
	}

	result := server.organizationService.GetTeam(r.Context(), actor.subject, teamIDCreated.Value)
	got, matched := result.(org.TeamGot)
	if !matched {
		writeDomainError(w, result.(org.GetTeamRejected).Reason)
		return
	}

	response := teamDetailResponse{Team: teamToResponse(got.Team), Members: make([]string, 0, len(got.Members))}
	for _, member := range got.Members {
		response.Members = append(response.Members, member.String())
	}
	writeTeamDetailResponse(w, http.StatusOK, response)
}

func writeTeamDetailResponse(w http.ResponseWriter, status int, response teamDetailResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
