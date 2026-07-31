package httpserver

import (
	"encoding/json"
	"github.com/e6qu/sharecrop/internal/agent"
	"net/http"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/task"
)

type teamDetailResponse struct {
	Team    teamResponse `json:"team"`
	Members []string     `json:"members"`
}

type teamMemberRequest struct {
	Email string `json:"email"`
}

func teamDetailFrom(got org.TeamGot) teamDetailResponse {
	response := teamDetailResponse{Team: teamToResponse(got.Team), Members: make([]string, 0, len(got.Members))}
	for _, member := range got.Members {
		response.Members = append(response.Members, member.String())
	}
	return response
}

func (server Server) getTeam(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserOrOrgSubject(r, agent.ScopeOrgRead)
	actor, actorMatched := actorResult.(actorAccepted)
	if !actorMatched {
		writeActorRejection(w, actorResult)
		return
	}

	teamIDResult := core.ParseTeamID(r.PathValue("team_id"))
	teamIDCreated, teamIDMatched := teamIDResult.(core.TeamIDCreated)
	if !teamIDMatched {
		writeDomainError(w, teamIDResult.(core.TeamIDRejected).Reason)
		return
	}

	result := server.organizationService.GetTeam(r.Context(), actor.actor, teamIDCreated.Value)
	got, matched := result.(org.TeamGot)
	if !matched {
		writeDomainError(w, result.(org.GetTeamRejected).Reason)
		return
	}

	writeTeamDetailResponse(w, http.StatusOK, teamDetailFrom(got))
}

func (server Server) getTeamWork(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	teamIDResult := core.ParseTeamID(r.PathValue("team_id"))
	teamIDCreated, teamIDMatched := teamIDResult.(core.TeamIDCreated)
	if !teamIDMatched {
		writeDomainError(w, teamIDResult.(core.TeamIDRejected).Reason)
		return
	}

	teamResult := server.organizationService.GetTeam(r.Context(), actor.subject, teamIDCreated.Value)
	if _, matched := teamResult.(org.TeamGot); !matched {
		writeDomainError(w, teamResult.(org.GetTeamRejected).Reason)
		return
	}

	filtersResult := parseTaskListFilters(r)
	filtersAccepted, filtersMatched := filtersResult.(taskListFiltersAccepted)
	if !filtersMatched {
		writeDomainError(w, filtersResult.(taskListFiltersRejected).reason)
		return
	}

	pageResult := parsePageStrict(r)
	pageAccepted, pageMatched := pageResult.(pageParseAccepted)
	if !pageMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, pageResult.(pageParseRejected).reason)
		return
	}

	result := server.taskService.List(r.Context(), actor.subject, task.TeamListScope{TeamID: teamIDCreated.Value, IncludeReserved: true}, filtersAccepted.value, pageAccepted.value.Probe())
	listed, matched := result.(task.TasksListed)
	if !matched {
		writeDomainError(w, result.(task.ListRejected).Reason)
		return
	}
	visible, nextOffset := probeListWindow(len(listed.Values), pageAccepted.value)
	response := tasksToResponse(listed.Values[:visible])
	response.NextOffset = nextOffset
	writeTasksResponse(w, http.StatusOK, response)
}

func (server Server) addTeamMember(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserOrOrgSubject(r, agent.ScopeOrgManage)
	actor, actorMatched := actorResult.(actorAccepted)
	if !actorMatched {
		writeActorRejection(w, actorResult)
		return
	}

	teamIDResult := core.ParseTeamID(r.PathValue("team_id"))
	teamIDCreated, teamIDMatched := teamIDResult.(core.TeamIDCreated)
	if !teamIDMatched {
		writeDomainError(w, teamIDResult.(core.TeamIDRejected).Reason)
		return
	}

	var request teamMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	emailResult := auth.NewEmailAddress(request.Email)
	emailAccepted, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		writeDomainError(w, emailResult.(auth.EmailAddressRejected).Reason)
		return
	}

	result := server.organizationService.AddTeamMember(r.Context(), actor.actor, teamIDCreated.Value, emailAccepted.Value)
	if _, added := result.(org.TeamMemberAddedResult); !added {
		writeDomainError(w, result.(org.AddTeamMemberRejected).Reason)
		return
	}

	gotResult := server.organizationService.GetTeam(r.Context(), actor.actor, teamIDCreated.Value)
	got, gotMatched := gotResult.(org.TeamGot)
	if !gotMatched {
		writeDomainError(w, gotResult.(org.GetTeamRejected).Reason)
		return
	}
	writeTeamDetailResponse(w, http.StatusCreated, teamDetailFrom(got))
}

func writeTeamDetailResponse(w http.ResponseWriter, status int, response teamDetailResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
