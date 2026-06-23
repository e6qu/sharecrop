package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
)

func (server Server) createOrganization(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	var request organizationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	nameResult := org.NewOrganizationName(request.Name)
	nameAccepted, nameMatched := nameResult.(org.OrganizationNameAccepted)
	if !nameMatched {
		rejected := nameResult.(org.OrganizationNameRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.CreateOrganization(r.Context(), actor.subject, nameAccepted.Value)
	created, matched := result.(org.OrganizationCreated)
	if !matched {
		rejected := result.(org.CreateOrganizationRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	writeOrganizationResponse(w, http.StatusCreated, organizationToResponse(created.Value))
}

func (server Server) listOrganizations(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	result := server.organizationService.ListOrganizations(r.Context(), actor.subject, parsePage(r))
	listed, matched := result.(org.OrganizationsListed)
	if !matched {
		rejected := result.(org.ListOrganizationsRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	response := organizationsResponse{Organizations: make([]organizationResponse, 0, len(listed.Values))}
	for _, organization := range listed.Values {
		response.Organizations = append(response.Organizations, organizationToResponse(organization))
	}
	writeOrganizationsResponse(w, http.StatusOK, response)
}

func (server Server) listOrganizationMembers(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.organizationService.ListMembers(r.Context(), actor.subject, organizationIDAccepted.value, parsePage(r))
	listed, matched := result.(org.MembersListed)
	if !matched {
		writeError(w, http.StatusForbidden, result.(org.ListMembersRejected).Reason.Description())
		return
	}

	response := organizationMembersResponse{Members: make([]organizationMemberResponse, 0, len(listed.Values))}
	for _, member := range listed.Values {
		response.Members = append(response.Members, memberToResponse(member))
	}
	writeOrganizationMembersResponse(w, http.StatusOK, response)
}

func (server Server) provisionOrganizationMember(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	requestResult := decodeProvisionMemberRequest(r)
	requestAccepted, requestMatched := requestResult.(provisionMemberAccepted)
	if !requestMatched {
		rejected := requestResult.(provisionMemberRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.organizationService.ProvisionMember(r.Context(), actor.subject, organizationIDAccepted.value, requestAccepted.email, requestAccepted.roles)
	provisioned, matched := result.(org.MemberProvisioned)
	if !matched {
		rejected := result.(org.ProvisionMemberRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeOrganizationMemberResponse(w, http.StatusCreated, memberToResponse(provisioned.Value))
}

func (server Server) deactivateOrganizationMember(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userIDAccepted, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		rejected := userIDResult.(core.UserIDRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.DeactivateMember(r.Context(), actor.subject, organizationIDAccepted.value, userIDAccepted.Value)
	if _, matched := result.(org.MemberDeactivationAccepted); !matched {
		rejected := result.(org.DeactivateMemberRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "deactivated"})
}

func (server Server) createOrganizationTeam(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	var request teamRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	nameResult := org.NewTeamName(request.Name)
	nameAccepted, nameMatched := nameResult.(org.TeamNameAccepted)
	if !nameMatched {
		rejected := nameResult.(org.TeamNameRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.CreateOrganizationTeam(r.Context(), actor.subject, organizationIDAccepted.value, nameAccepted.Value)
	created, matched := result.(org.TeamCreated)
	if !matched {
		rejected := result.(org.CreateTeamRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeTeamResponse(w, http.StatusCreated, teamToResponse(created.Value))
}

func (server Server) listOrganizationTeams(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.organizationService.ListOrganizationTeams(r.Context(), actor.subject, organizationIDAccepted.value, parsePage(r))
	listed, matched := result.(org.OrganizationTeamsListed)
	if !matched {
		rejected := result.(org.ListTeamsRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	response := teamsResponse{Teams: make([]teamResponse, 0, len(listed.Values))}
	for _, team := range listed.Values {
		response.Teams = append(response.Teams, teamToResponse(team))
	}
	writeTeamsResponse(w, http.StatusOK, response)
}

func (server Server) createStandaloneTeam(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	var request teamRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	nameResult := org.NewTeamName(request.Name)
	nameAccepted, nameMatched := nameResult.(org.TeamNameAccepted)
	if !nameMatched {
		rejected := nameResult.(org.TeamNameRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.CreateStandaloneTeam(r.Context(), actor.subject, nameAccepted.Value)
	created, matched := result.(org.TeamCreated)
	if !matched {
		rejected := result.(org.CreateTeamRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeTeamResponse(w, http.StatusCreated, teamToResponse(created.Value))
}

func (server Server) listStandaloneTeams(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	result := server.organizationService.ListStandaloneTeams(r.Context(), actor.subject, parsePage(r))
	listed, matched := result.(org.OrganizationTeamsListed)
	if !matched {
		rejected := result.(org.ListTeamsRejected)
		writeError(w, http.StatusInternalServerError, rejected.Reason.Description())
		return
	}

	response := teamsResponse{Teams: make([]teamResponse, 0, len(listed.Values))}
	for _, team := range listed.Values {
		response.Teams = append(response.Teams, teamToResponse(team))
	}
	writeTeamsResponse(w, http.StatusOK, response)
}
