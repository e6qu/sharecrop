package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/task"
)

type organizationSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type organizationsPayload struct {
	Organizations []organizationSummary `json:"organizations"`
}

type memberSummary struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Roles          []string `json:"roles"`
}

type membersPayload struct {
	Members []memberSummary `json:"members"`
}

type teamSummary struct {
	ID             string `json:"id"`
	OwnerKind      string `json:"owner_kind"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	CreatedBy      string `json:"created_by"`
}

type teamsPayload struct {
	Teams []teamSummary `json:"teams"`
}

type teamDetailPayload struct {
	Team    teamSummary `json:"team"`
	Members []string    `json:"members"`
}

type orgCredentialSummary struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	Label          string   `json:"label"`
	Scopes         []string `json:"scopes"`
	State          string   `json:"state"`
	ExpiresAt      string   `json:"expires_at"`
}

type orgCredentialCreatedPayload struct {
	Credential orgCredentialSummary `json:"credential"`
	Secret     string               `json:"secret"`
}

type orgCredentialsPayload struct {
	Credentials []orgCredentialSummary `json:"credentials"`
}

func (organizationsPayload) payloadValue() {}

func (membersPayload) payloadValue() {}

func (teamsPayload) payloadValue() {}

func (teamDetailPayload) payloadValue() {}

func (orgCredentialCreatedPayload) payloadValue() {}

func (orgCredentialsPayload) payloadValue() {}

func (orgCredentialSummary) payloadValue() {}

func organizationToSummary(value org.Organization) organizationSummary {
	return organizationSummary{ID: value.ID.String(), Name: value.Name.String()}
}

func memberToSummary(value org.OrganizationMember) memberSummary {
	rawRoles := make([]string, 0, len(value.Roles))
	for index := range value.Roles {
		rawRoles = append(rawRoles, value.Roles[index].String())
	}
	return memberSummary{
		ID:             value.ID.String(),
		OrganizationID: value.OrganizationID.String(),
		UserID:         value.UserID.String(),
		Status:         value.Status.String(),
		Roles:          rawRoles,
	}
}

func teamToSummary(value org.Team) teamSummary {
	summary := teamSummary{ID: value.ID.String(), Name: value.Name.String(), CreatedBy: value.CreatedBy.String()}
	switch typed := value.Owner.(type) {
	case org.OrganizationOwnedTeam:
		summary.OwnerKind = "organization"
		summary.OrganizationID = typed.OrganizationID.String()
	case org.UserOwnedTeam:
		summary.OwnerKind = "user"
	}
	return summary
}

func orgCredentialToSummary(value orgcred.Credential) orgCredentialSummary {
	scopes := value.Scopes.Values()
	rawScopes := make([]string, 0, len(scopes))
	for index := range scopes {
		rawScopes = append(rawScopes, scopes[index].String())
	}
	var rawExpiresAt string
	if value.ExpiresAt != nil {
		rawExpiresAt = value.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return orgCredentialSummary{
		ID:             value.ID.String(),
		OrganizationID: value.OrganizationID.String(),
		Label:          value.Label.String(),
		Scopes:         rawScopes,
		State:          value.State.String(),
		ExpiresAt:      rawExpiresAt,
	}
}

func parseOrganizationID(arguments json.RawMessage) (core.OrganizationID, toolResult) {
	var args struct {
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return core.OrganizationID{}, invalidArguments()
	}
	result := core.ParseOrganizationID(args.OrganizationID)
	organizationID, matched := result.(core.OrganizationIDCreated)
	if !matched {
		return core.OrganizationID{}, toolProtocolError{code: codeInvalidParams, message: result.(core.OrganizationIDRejected).Reason.Description()}
	}
	return organizationID.Value, nil
}

func parseTeamID(arguments json.RawMessage) (core.TeamID, toolResult) {
	var args struct {
		TeamID string `json:"team_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return core.TeamID{}, invalidArguments()
	}
	result := core.ParseTeamID(args.TeamID)
	teamID, matched := result.(core.TeamIDCreated)
	if !matched {
		return core.TeamID{}, toolProtocolError{code: codeInvalidParams, message: result.(core.TeamIDRejected).Reason.Description()}
	}
	return teamID.Value, nil
}

func parseOrganizationRoles(raw []string) ([]org.Role, toolResult) {
	roles := make([]org.Role, 0, len(raw))
	for _, rawRole := range raw {
		roleResult := org.ParseRole(rawRole)
		roleAccepted, roleMatched := roleResult.(org.RoleAccepted)
		if !roleMatched {
			return nil, toolProtocolError{code: codeInvalidParams, message: roleResult.(org.RoleRejected).Reason.Description()}
		}
		roles = append(roles, roleAccepted.Value)
	}
	return roles, nil
}

func (server Server) callCreateOrganization(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	nameResult := org.NewOrganizationName(args.Name)
	name, nameMatched := nameResult.(org.OrganizationNameAccepted)
	if !nameMatched {
		return toolProtocolError{code: codeInvalidParams, message: nameResult.(org.OrganizationNameRejected).Reason.Description()}
	}
	result := server.services.CreateOrganization(ctx, subject, name.Value)
	created, matched := result.(org.OrganizationCreated)
	if !matched {
		return toolFailed{message: result.(org.CreateOrganizationRejected).Reason.Description()}
	}
	return marshalPayload(organizationToSummary(created.Value))
}

func (organizationSummary) payloadValue() {}

func (server Server) callListOrganizations(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	result := server.services.ListOrganizations(ctx, subject, args.Query, core.DefaultPage())
	listed, matched := result.(org.OrganizationsListed)
	if !matched {
		return toolFailed{message: result.(org.ListOrganizationsRejected).Reason.Description()}
	}
	summaries := make([]organizationSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, organizationToSummary(listed.Values[index]))
	}
	return marshalPayload(organizationsPayload{Organizations: summaries})
}

func (server Server) callListOrgMembers(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	organizationID, problem := parseOrganizationID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.ListOrganizationMembers(ctx, subject, organizationID, core.DefaultPage())
	listed, matched := result.(org.MembersListed)
	if !matched {
		return toolFailed{message: result.(org.ListMembersRejected).Reason.Description()}
	}
	summaries := make([]memberSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, memberToSummary(listed.Values[index]))
	}
	return marshalPayload(membersPayload{Members: summaries})
}

func (server Server) callProvisionOrgMember(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		OrganizationID string   `json:"organization_id"`
		Email          string   `json:"email"`
		Roles          []string `json:"roles"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	organizationIDResult := core.ParseOrganizationID(args.OrganizationID)
	organizationID, organizationIDMatched := organizationIDResult.(core.OrganizationIDCreated)
	if !organizationIDMatched {
		return toolProtocolError{code: codeInvalidParams, message: organizationIDResult.(core.OrganizationIDRejected).Reason.Description()}
	}
	emailResult := auth.NewEmailAddress(args.Email)
	email, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		return toolProtocolError{code: codeInvalidParams, message: emailResult.(auth.EmailAddressRejected).Reason.Description()}
	}
	roles, problem := parseOrganizationRoles(args.Roles)
	if problem != nil {
		return problem
	}
	result := server.services.ProvisionOrganizationMember(ctx, subject, organizationID.Value, email.Value, roles)
	provisioned, matched := result.(org.MemberProvisioned)
	if !matched {
		return toolFailed{message: result.(org.ProvisionMemberRejected).Reason.Description()}
	}
	return marshalPayload(memberToSummary(provisioned.Value))
}

func (memberSummary) payloadValue() {}

func (server Server) callDeactivateOrgMember(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		OrganizationID string `json:"organization_id"`
		UserID         string `json:"user_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	organizationIDResult := core.ParseOrganizationID(args.OrganizationID)
	organizationID, organizationIDMatched := organizationIDResult.(core.OrganizationIDCreated)
	if !organizationIDMatched {
		return toolProtocolError{code: codeInvalidParams, message: organizationIDResult.(core.OrganizationIDRejected).Reason.Description()}
	}
	userIDResult := core.ParseUserID(args.UserID)
	userID, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		return toolProtocolError{code: codeInvalidParams, message: userIDResult.(core.UserIDRejected).Reason.Description()}
	}
	result := server.services.DeactivateOrganizationMember(ctx, subject, organizationID.Value, userID.Value)
	if _, matched := result.(org.MemberDeactivationAccepted); !matched {
		return toolFailed{message: result.(org.DeactivateMemberRejected).Reason.Description()}
	}
	return toolSucceeded{payload: json.RawMessage(`{"status":"deactivated"}`)}
}

func (server Server) callUpdateOrgMemberRoles(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		OrganizationID string   `json:"organization_id"`
		UserID         string   `json:"user_id"`
		Roles          []string `json:"roles"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	organizationIDResult := core.ParseOrganizationID(args.OrganizationID)
	organizationID, organizationIDMatched := organizationIDResult.(core.OrganizationIDCreated)
	if !organizationIDMatched {
		return toolProtocolError{code: codeInvalidParams, message: organizationIDResult.(core.OrganizationIDRejected).Reason.Description()}
	}
	userIDResult := core.ParseUserID(args.UserID)
	userID, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		return toolProtocolError{code: codeInvalidParams, message: userIDResult.(core.UserIDRejected).Reason.Description()}
	}
	roles, problem := parseOrganizationRoles(args.Roles)
	if problem != nil {
		return problem
	}
	result := server.services.UpdateOrganizationMemberRoles(ctx, subject, organizationID.Value, userID.Value, roles)
	updated, matched := result.(org.MemberRolesUpdatedResult)
	if !matched {
		return toolFailed{message: result.(org.UpdateMemberRolesRejected).Reason.Description()}
	}
	return marshalPayload(memberToSummary(updated.Value))
}

func (server Server) callCreateOrganizationTeam(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	organizationID, problem := parseOrganizationID(arguments)
	if problem != nil {
		return problem
	}
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	nameResult := org.NewTeamName(args.Name)
	name, nameMatched := nameResult.(org.TeamNameAccepted)
	if !nameMatched {
		return toolProtocolError{code: codeInvalidParams, message: nameResult.(org.TeamNameRejected).Reason.Description()}
	}
	result := server.services.CreateOrganizationTeam(ctx, subject, organizationID, name.Value)
	created, matched := result.(org.TeamCreated)
	if !matched {
		return toolFailed{message: result.(org.CreateTeamRejected).Reason.Description()}
	}
	return marshalPayload(teamToSummary(created.Value))
}

func (teamSummary) payloadValue() {}

func (server Server) callListOrganizationTeams(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	organizationID, problem := parseOrganizationID(arguments)
	if problem != nil {
		return problem
	}
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	result := server.services.ListOrganizationTeams(ctx, subject, organizationID, args.Query, core.DefaultPage())
	listed, matched := result.(org.OrganizationTeamsListed)
	if !matched {
		return toolFailed{message: result.(org.ListTeamsRejected).Reason.Description()}
	}
	summaries := make([]teamSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, teamToSummary(listed.Values[index]))
	}
	return marshalPayload(teamsPayload{Teams: summaries})
}

func (server Server) callCreateStandaloneTeam(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	nameResult := org.NewTeamName(args.Name)
	name, nameMatched := nameResult.(org.TeamNameAccepted)
	if !nameMatched {
		return toolProtocolError{code: codeInvalidParams, message: nameResult.(org.TeamNameRejected).Reason.Description()}
	}
	result := server.services.CreateStandaloneTeam(ctx, subject, name.Value)
	created, matched := result.(org.TeamCreated)
	if !matched {
		return toolFailed{message: result.(org.CreateTeamRejected).Reason.Description()}
	}
	return marshalPayload(teamToSummary(created.Value))
}

func (server Server) callListStandaloneTeams(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	result := server.services.ListStandaloneTeams(ctx, subject, args.Query, core.DefaultPage())
	listed, matched := result.(org.OrganizationTeamsListed)
	if !matched {
		return toolFailed{message: result.(org.ListTeamsRejected).Reason.Description()}
	}
	summaries := make([]teamSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, teamToSummary(listed.Values[index]))
	}
	return marshalPayload(teamsPayload{Teams: summaries})
}

func (server Server) callGetTeam(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	teamID, problem := parseTeamID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.GetTeam(ctx, subject, teamID)
	got, matched := result.(org.TeamGot)
	if !matched {
		return toolFailed{message: result.(org.GetTeamRejected).Reason.Description()}
	}
	members := make([]string, 0, len(got.Members))
	for index := range got.Members {
		members = append(members, got.Members[index].String())
	}
	return marshalPayload(teamDetailPayload{Team: teamToSummary(got.Team), Members: members})
}

func (server Server) callGetTeamWork(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	teamID, problem := parseTeamID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.GetTeamWork(ctx, subject, teamID, task.NoListFilters(), core.DefaultPage())
	listed, matched := result.(task.TasksListed)
	if !matched {
		return toolFailed{message: result.(task.ListRejected).Reason.Description()}
	}
	summaries := make([]taskSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, taskToSummary(listed.Values[index].Task))
	}
	return marshalPayload(tasksPayload{Tasks: summaries})
}

func (server Server) callAddTeamMember(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	teamID, problem := parseTeamID(arguments)
	if problem != nil {
		return problem
	}
	var args struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	emailResult := auth.NewEmailAddress(args.Email)
	email, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		return toolProtocolError{code: codeInvalidParams, message: emailResult.(auth.EmailAddressRejected).Reason.Description()}
	}
	result := server.services.AddTeamMember(ctx, subject, teamID, email.Value)
	if _, matched := result.(org.TeamMemberAddedResult); !matched {
		return toolFailed{message: result.(org.AddTeamMemberRejected).Reason.Description()}
	}
	return toolSucceeded{payload: json.RawMessage(`{"status":"added"}`)}
}

func (server Server) callCreateOrgCredential(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	organizationID, problem := parseOrganizationID(arguments)
	if problem != nil {
		return problem
	}
	var args struct {
		Label     string   `json:"label"`
		Scopes    []string `json:"scopes"`
		ExpiresAt string   `json:"expires_at"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	permissionCheck := server.services.CheckOrganizationPermission(ctx, organizationID, subject.ID, org.PermissionManageMembers)
	if _, denied := permissionCheck.(org.PermissionDenied); denied {
		return toolFailed{message: "organization credential management access denied"}
	}
	labelResult := agent.NewLabel(args.Label)
	label, labelMatched := labelResult.(agent.LabelAccepted)
	if !labelMatched {
		return toolProtocolError{code: codeInvalidParams, message: labelResult.(agent.LabelRejected).Reason.Description()}
	}
	scopes, problem := parseOrgCredentialScopes(args.Scopes)
	if problem != nil {
		return problem
	}
	var expiresAt *time.Time
	if args.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, args.ExpiresAt)
		if err != nil {
			return toolProtocolError{code: codeInvalidParams, message: "expires_at must be an RFC3339 timestamp"}
		}
		expiresAt = &parsed
	}
	result := server.services.CreateOrgCredential(ctx, organizationID, label.Value, scopes, expiresAt)
	created, matched := result.(orgcred.CredentialCreated)
	if !matched {
		return toolFailed{message: result.(orgcred.CreateRejected).Reason.Description()}
	}
	return marshalPayload(orgCredentialCreatedPayload{
		Credential: orgCredentialToSummary(created.Value),
		Secret:     created.Secret.String(),
	})
}

func parseOrgCredentialScopes(raw []string) (agent.ScopeSet, toolResult) {
	scopes := make([]agent.Scope, 0, len(raw))
	for _, rawScope := range raw {
		scopeResult := agent.ParseScope(rawScope)
		scope, matched := scopeResult.(agent.ScopeAccepted)
		if !matched {
			return agent.ScopeSet{}, toolProtocolError{code: codeInvalidParams, message: scopeResult.(agent.ScopeRejected).Reason.Description()}
		}
		scopes = append(scopes, scope.Value)
	}
	set := agent.NewScopeSet(scopes)
	if set.IsEmpty() {
		return agent.ScopeSet{}, toolProtocolError{code: codeInvalidParams, message: "at least one scope is required"}
	}
	return set, nil
}

func (server Server) callListOrgCredentials(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	organizationID, problem := parseOrganizationID(arguments)
	if problem != nil {
		return problem
	}
	permissionCheck := server.services.CheckOrganizationPermission(ctx, organizationID, subject.ID, org.PermissionManageMembers)
	if _, denied := permissionCheck.(org.PermissionDenied); denied {
		return toolFailed{message: "organization credential management access denied"}
	}
	result := server.services.ListOrgCredentials(ctx, organizationID, core.DefaultPage())
	listed, matched := result.(orgcred.CredentialsListed)
	if !matched {
		return toolFailed{message: result.(orgcred.ListRejected).Reason.Description()}
	}
	summaries := make([]orgCredentialSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, orgCredentialToSummary(listed.Values[index]))
	}
	return marshalPayload(orgCredentialsPayload{Credentials: summaries})
}

func (server Server) callRevokeOrgCredential(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	organizationID, problem := parseOrganizationID(arguments)
	if problem != nil {
		return problem
	}
	var args struct {
		CredentialID string `json:"credential_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	permissionCheck := server.services.CheckOrganizationPermission(ctx, organizationID, subject.ID, org.PermissionManageMembers)
	if _, denied := permissionCheck.(org.PermissionDenied); denied {
		return toolFailed{message: "organization credential management access denied"}
	}
	credentialIDResult := core.ParseOrgCredentialID(args.CredentialID)
	credentialID, credentialIDMatched := credentialIDResult.(core.OrgCredentialIDCreated)
	if !credentialIDMatched {
		return toolProtocolError{code: codeInvalidParams, message: credentialIDResult.(core.OrgCredentialIDRejected).Reason.Description()}
	}
	result := server.services.RevokeOrgCredential(ctx, organizationID, credentialID.Value)
	revoked, matched := result.(orgcred.CredentialRevoked)
	if !matched {
		return toolFailed{message: result.(orgcred.RevokeRejected).Reason.Description()}
	}
	return marshalPayload(orgCredentialToSummary(revoked.Value))
}
