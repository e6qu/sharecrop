package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/auth"
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
		writeDomainError(w, rejected.Reason)
		return
	}

	result := server.organizationService.CreateOrganization(r.Context(), actor.subject, nameAccepted.Value)
	created, matched := result.(org.OrganizationCreated)
	if !matched {
		rejected := result.(org.CreateOrganizationRejected)
		writeDomainError(w, rejected.Reason)
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
		writeDomainError(w, rejected.Reason)
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
		writeDomainError(w, result.(org.ListMembersRejected).Reason)
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
		writeDomainError(w, rejected.Reason)
		return
	}

	writeOrganizationMemberResponse(w, http.StatusCreated, memberToResponse(provisioned.Value))
}

func (server Server) deactivateOrganizationMember(w http.ResponseWriter, r *http.Request) {
	target, ok := server.organizationMemberTarget(w, r)
	if !ok {
		return
	}

	result := server.organizationService.DeactivateMember(r.Context(), target.actor, target.organizationID, target.userID)
	if _, matched := result.(org.MemberDeactivationAccepted); !matched {
		rejected := result.(org.DeactivateMemberRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "deactivated"})
}

func (server Server) updateOrganizationMemberRoles(w http.ResponseWriter, r *http.Request) {
	target, ok := server.organizationMemberTarget(w, r)
	if !ok {
		return
	}

	var request updateMemberRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	rolesResult := parseOrganizationRoles(request.Roles)
	rolesAccepted, rolesMatched := rolesResult.(organizationRolesAccepted)
	if !rolesMatched {
		writeError(w, http.StatusBadRequest, rolesResult.(organizationRolesRejected).reason)
		return
	}

	result := server.organizationService.UpdateMemberRoles(r.Context(), target.actor, target.organizationID, target.userID, rolesAccepted.values)
	updated, matched := result.(org.MemberRolesUpdatedResult)
	if !matched {
		writeDomainError(w, result.(org.UpdateMemberRolesRejected).Reason)
		return
	}

	writeOrganizationMemberResponse(w, http.StatusOK, memberToResponse(updated.Value))
}

type organizationMemberTarget struct {
	actor          auth.UserSubject
	organizationID core.OrganizationID
	userID         core.UserID
}

func (server Server) organizationMemberTarget(w http.ResponseWriter, r *http.Request) (organizationMemberTarget, bool) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return organizationMemberTarget{}, false
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return organizationMemberTarget{}, false
	}

	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userIDAccepted, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		rejected := userIDResult.(core.UserIDRejected)
		writeDomainError(w, rejected.Reason)
		return organizationMemberTarget{}, false
	}

	return organizationMemberTarget{actor: actor.subject, organizationID: organizationIDAccepted.value, userID: userIDAccepted.Value}, true
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
		writeDomainError(w, rejected.Reason)
		return
	}

	result := server.organizationService.CreateOrganizationTeam(r.Context(), actor.subject, organizationIDAccepted.value, nameAccepted.Value)
	created, matched := result.(org.TeamCreated)
	if !matched {
		rejected := result.(org.CreateTeamRejected)
		writeDomainError(w, rejected.Reason)
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
		writeDomainError(w, rejected.Reason)
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
		writeDomainError(w, rejected.Reason)
		return
	}

	result := server.organizationService.CreateStandaloneTeam(r.Context(), actor.subject, nameAccepted.Value)
	created, matched := result.(org.TeamCreated)
	if !matched {
		rejected := result.(org.CreateTeamRejected)
		writeDomainError(w, rejected.Reason)
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
		writeDomainError(w, rejected.Reason)
		return
	}

	response := teamsResponse{Teams: make([]teamResponse, 0, len(listed.Values))}
	for _, team := range listed.Values {
		response.Teams = append(response.Teams, teamToResponse(team))
	}
	writeTeamsResponse(w, http.StatusOK, response)
}
