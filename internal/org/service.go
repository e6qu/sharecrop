package org

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/orgactor"
)

type Store interface {
	CreateOrganization(context.Context, core.OrganizationID, OrganizationName, core.UserID, core.OrganizationMembershipID) CreateOrganizationStoreResult
	ListOrganizationsForUser(context.Context, core.UserID, string, core.Page) ListOrganizationsResult
	FindMemberRoles(context.Context, core.OrganizationID, core.UserID) MemberRolesResult
	ListMembers(context.Context, core.OrganizationID, core.Page) ListMembersResult
	ProvisionMember(context.Context, core.OrganizationMembershipID, core.OrganizationID, auth.EmailAddress, []Role) ProvisionMemberStoreResult
	DeactivateMember(context.Context, core.OrganizationID, core.UserID) DeactivateMemberStoreResult
	UpdateMemberRoles(context.Context, core.OrganizationID, core.UserID, []Role) UpdateMemberRolesStoreResult
	CreateOrganizationTeam(context.Context, core.TeamID, core.OrganizationID, TeamName, core.UserID) CreateTeamStoreResult
	CreateStandaloneTeam(context.Context, core.TeamID, core.UserID, TeamName) CreateTeamStoreResult
	AddTeamMember(context.Context, core.TeamID, core.UserID) AddTeamMemberStoreResult
	AddTeamMemberByEmail(context.Context, core.TeamID, auth.EmailAddress) AddTeamMemberStoreResult
	ListOrganizationTeams(context.Context, core.OrganizationID, core.UserID, string, core.Page) TeamListResult
	ListStandaloneTeams(context.Context, core.UserID, string, core.Page) TeamListResult
	FindTeam(context.Context, core.TeamID) FindTeamResult
	ListTeamMembers(context.Context, core.TeamID) TeamMembersResult
}

type Service struct {
	store Store
}

func NewService(store Store) Service {
	return Service{store: store}
}

type CreateOrganizationResult interface {
	createOrganizationResult()
}

type OrganizationCreated struct {
	Value Organization
}

type CreateOrganizationRejected struct {
	Reason core.DomainError
}

func (OrganizationCreated) createOrganizationResult() {}

func (CreateOrganizationRejected) createOrganizationResult() {}

func (service Service) CreateOrganization(ctx context.Context, actor auth.UserSubject, name OrganizationName) CreateOrganizationResult {
	organizationIDResult := core.NewOrganizationID()
	organizationIDCreated, organizationIDMatched := organizationIDResult.(core.OrganizationIDCreated)
	if !organizationIDMatched {
		rejected := organizationIDResult.(core.OrganizationIDRejected)
		return CreateOrganizationRejected{Reason: rejected.Reason}
	}

	membershipIDResult := core.NewOrganizationMembershipID()
	membershipIDCreated, membershipIDMatched := membershipIDResult.(core.OrganizationMembershipIDCreated)
	if !membershipIDMatched {
		rejected := membershipIDResult.(core.OrganizationMembershipIDRejected)
		return CreateOrganizationRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.CreateOrganization(ctx, organizationIDCreated.Value, name, actor.ID, membershipIDCreated.Value)
	if rejected, matched := storeResult.(CreateOrganizationStoreRejected); matched {
		return CreateOrganizationRejected{Reason: rejected.Reason}
	}

	return OrganizationCreated{
		Value: Organization{
			ID:        organizationIDCreated.Value,
			Name:      name,
			CreatedBy: actor.ID,
		},
	}
}

type ListOrganizationsResult interface {
	listOrganizationsResult()
}

type OrganizationsListed struct {
	Values []Organization
}

type ListOrganizationsRejected struct {
	Reason core.DomainError
}

func (OrganizationsListed) listOrganizationsResult() {}

func (ListOrganizationsRejected) listOrganizationsResult() {}

func (service Service) ListOrganizations(ctx context.Context, actor auth.UserSubject, query string, page core.Page) ListOrganizationsResult {
	result := service.store.ListOrganizationsForUser(ctx, actor.ID, query, page)
	listed, matched := result.(OrganizationsListed)
	if !matched {
		rejected := result.(ListOrganizationsRejected)
		return ListOrganizationsRejected{Reason: rejected.Reason}
	}
	return listed
}

type ProvisionMemberResult interface {
	provisionMemberResult()
}

type MemberProvisioned struct {
	Value OrganizationMember
}

type ProvisionMemberRejected struct {
	Reason core.DomainError
}

func (MemberProvisioned) provisionMemberResult() {}

func (ProvisionMemberRejected) provisionMemberResult() {}

func (service Service) ProvisionMember(ctx context.Context, actor auth.Subject, organizationID core.OrganizationID, email auth.EmailAddress, roles []Role) ProvisionMemberResult {
	permissionResult := service.requirePermissionForActor(ctx, actor, organizationID, PermissionManageMembers)
	if rejected, matched := permissionResult.(PermissionDenied); matched {
		return ProvisionMemberRejected{Reason: rejected.Reason}
	}

	membershipIDResult := core.NewOrganizationMembershipID()
	membershipIDCreated, membershipIDMatched := membershipIDResult.(core.OrganizationMembershipIDCreated)
	if !membershipIDMatched {
		rejected := membershipIDResult.(core.OrganizationMembershipIDRejected)
		return ProvisionMemberRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.ProvisionMember(ctx, membershipIDCreated.Value, organizationID, email, roles)
	provisioned, matched := storeResult.(MemberProvisioned)
	if !matched {
		rejected := storeResult.(ProvisionMemberStoreRejected)
		return ProvisionMemberRejected{Reason: rejected.Reason}
	}

	return provisioned
}

type ListMembersResult interface {
	listMembersResult()
}

type MembersListed struct {
	Values []OrganizationMember
}

type ListMembersRejected struct {
	Reason core.DomainError
}

func (MembersListed) listMembersResult() {}

func (ListMembersRejected) listMembersResult() {}

// ListMembers returns an organization's members. Only an active member of the
// organization may view the roster.
func (service Service) ListMembers(ctx context.Context, actor auth.UserSubject, organizationID core.OrganizationID, page core.Page) ListMembersResult {
	rolesResult := service.store.FindMemberRoles(ctx, organizationID, actor.ID)
	if _, matched := rolesResult.(MemberRolesFound); !matched {
		return ListMembersRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "organization member access denied")}
	}

	result := service.store.ListMembers(ctx, organizationID, page)
	listed, matched := result.(MembersListed)
	if !matched {
		return ListMembersRejected{Reason: result.(ListMembersRejected).Reason}
	}
	return listed
}

type CreateTeamResult interface {
	createTeamResult()
}

type TeamCreated struct {
	Value Team
}

type CreateTeamRejected struct {
	Reason core.DomainError
}

func (TeamCreated) createTeamResult() {}

func (CreateTeamRejected) createTeamResult() {}

func (service Service) CreateOrganizationTeam(ctx context.Context, actor auth.UserSubject, organizationID core.OrganizationID, name TeamName) CreateTeamResult {
	permissionResult := service.requirePermission(ctx, organizationID, actor.ID, PermissionManageTeams)
	if rejected, matched := permissionResult.(PermissionDenied); matched {
		return CreateTeamRejected{Reason: rejected.Reason}
	}

	teamIDResult := core.NewTeamID()
	teamIDCreated, teamIDMatched := teamIDResult.(core.TeamIDCreated)
	if !teamIDMatched {
		rejected := teamIDResult.(core.TeamIDRejected)
		return CreateTeamRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.CreateOrganizationTeam(ctx, teamIDCreated.Value, organizationID, name, actor.ID)
	if rejected, matched := storeResult.(CreateTeamStoreRejected); matched {
		return CreateTeamRejected{Reason: rejected.Reason}
	}

	return TeamCreated{Value: Team{ID: teamIDCreated.Value, Owner: OrganizationOwnedTeam{OrganizationID: organizationID}, Name: name, CreatedBy: actor.ID}}
}

func (service Service) CreateStandaloneTeam(ctx context.Context, actor auth.UserSubject, name TeamName) CreateTeamResult {
	teamIDResult := core.NewTeamID()
	teamIDCreated, teamIDMatched := teamIDResult.(core.TeamIDCreated)
	if !teamIDMatched {
		rejected := teamIDResult.(core.TeamIDRejected)
		return CreateTeamRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.CreateStandaloneTeam(ctx, teamIDCreated.Value, actor.ID, name)
	if rejected, matched := storeResult.(CreateTeamStoreRejected); matched {
		return CreateTeamRejected{Reason: rejected.Reason}
	}

	return TeamCreated{Value: Team{ID: teamIDCreated.Value, Owner: UserOwnedTeam{OwnerUserID: actor.ID}, Name: name, CreatedBy: actor.ID}}
}

func (service Service) ListStandaloneTeams(ctx context.Context, actor auth.UserSubject, query string, page core.Page) ListTeamsResult {
	result := service.store.ListStandaloneTeams(ctx, actor.ID, query, page)
	listed, matched := result.(TeamsListed)
	if !matched {
		rejected := result.(TeamListRejected)
		return ListTeamsRejected{Reason: rejected.Reason}
	}
	return OrganizationTeamsListed{Values: listed.Values}
}

type ListTeamsResult interface {
	listTeamsResult()
}

type OrganizationTeamsListed struct {
	Values []Team
}

type ListTeamsRejected struct {
	Reason core.DomainError
}

func (OrganizationTeamsListed) listTeamsResult() {}

func (ListTeamsRejected) listTeamsResult() {}

func (service Service) ListOrganizationTeams(ctx context.Context, actor auth.UserSubject, organizationID core.OrganizationID, query string, page core.Page) ListTeamsResult {
	result := service.store.ListOrganizationTeams(ctx, organizationID, actor.ID, query, page)
	listed, matched := result.(TeamsListed)
	if !matched {
		rejected := result.(TeamListRejected)
		return ListTeamsRejected{Reason: rejected.Reason}
	}
	return OrganizationTeamsListed{Values: listed.Values}
}

type FindTeamResult interface {
	findTeamResult()
}

type TeamFound struct {
	Value Team
}

type TeamMissing struct {
	Reason core.DomainError
}

func (TeamFound) findTeamResult() {}

func (TeamMissing) findTeamResult() {}

type TeamMembersResult interface {
	teamMembersResult()
}

type TeamMembersListed struct {
	Values []core.UserID
}

type TeamMembersRejected struct {
	Reason core.DomainError
}

func (TeamMembersListed) teamMembersResult() {}

func (TeamMembersRejected) teamMembersResult() {}

type GetTeamResult interface {
	getTeamResult()
}

type TeamGot struct {
	Team    Team
	Members []core.UserID
}

type GetTeamRejected struct {
	Reason core.DomainError
}

func (TeamGot) getTeamResult() {}

func (GetTeamRejected) getTeamResult() {}

// GetTeam returns a team and its roster. A viewer may see a team only when they
// own it, belong to it, or (for an organization team) are a member of the owning
// organization, so a team roster never leaks to unrelated users.
func (service Service) GetTeam(ctx context.Context, actor auth.Subject, teamID core.TeamID) GetTeamResult {
	findResult := service.store.FindTeam(ctx, teamID)
	found, matched := findResult.(TeamFound)
	if !matched {
		return GetTeamRejected{Reason: findResult.(TeamMissing).Reason}
	}

	membersResult := service.store.ListTeamMembers(ctx, teamID)
	membersListed, membersMatched := membersResult.(TeamMembersListed)
	if !membersMatched {
		return GetTeamRejected{Reason: membersResult.(TeamMembersRejected).Reason}
	}

	if !service.canViewTeam(ctx, actor, found.Value, membersListed.Values) {
		return GetTeamRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "team access denied")}
	}
	return TeamGot{Team: found.Value, Members: membersListed.Values}
}

func (service Service) canViewTeam(ctx context.Context, actor auth.Subject, team Team, members []core.UserID) bool {
	if _, isOrg := actor.(auth.OrgSubject); isOrg {
		owner, matched := team.Owner.(OrganizationOwnedTeam)
		return matched && orgactor.Check(actor, owner.OrganizationID) == orgactor.Match
	}
	userActor, isUser := actor.(auth.UserSubject)
	if !isUser {
		return false
	}
	for _, member := range members {
		if member == userActor.ID {
			return true
		}
	}
	switch owner := team.Owner.(type) {
	case UserOwnedTeam:
		return owner.OwnerUserID == userActor.ID
	case OrganizationOwnedTeam:
		_, matched := service.store.FindMemberRoles(ctx, owner.OrganizationID, userActor.ID).(MemberRolesFound)
		return matched
	default:
		return false
	}
}

type AddTeamMemberResult interface {
	addTeamMemberResult()
}

type TeamMemberAddedResult struct{}

type AddTeamMemberRejected struct {
	Reason core.DomainError
}

func (TeamMemberAddedResult) addTeamMemberResult() {}

func (AddTeamMemberRejected) addTeamMemberResult() {}

// AddTeamMember adds a member (by email) to a team. Only the owner of a
// user-owned team, or a member with team-management permission in the owning
// organization, may add members.
func (service Service) AddTeamMember(ctx context.Context, actor auth.Subject, teamID core.TeamID, email auth.EmailAddress) AddTeamMemberResult {
	findResult := service.store.FindTeam(ctx, teamID)
	found, matched := findResult.(TeamFound)
	if !matched {
		return AddTeamMemberRejected{Reason: findResult.(TeamMissing).Reason}
	}
	if !service.canManageTeam(ctx, actor, found.Value) {
		return AddTeamMemberRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "team member management denied")}
	}

	result := service.store.AddTeamMemberByEmail(ctx, teamID, email)
	if _, added := result.(TeamMemberAdded); !added {
		return AddTeamMemberRejected{Reason: result.(AddTeamMemberStoreRejected).Reason}
	}
	return TeamMemberAddedResult{}
}

func (service Service) canManageTeam(ctx context.Context, actor auth.Subject, team Team) bool {
	owner, isOrgOwned := team.Owner.(OrganizationOwnedTeam)
	if !isOrgOwned {
		userActor, isUser := actor.(auth.UserSubject)
		if !isUser {
			return false
		}
		userOwner, matched := team.Owner.(UserOwnedTeam)
		return matched && userOwner.OwnerUserID == userActor.ID
	}
	_, denied := service.requirePermissionForActor(ctx, actor, owner.OrganizationID, PermissionManageTeams).(PermissionDenied)
	return !denied
}

type DeactivateMemberResult interface {
	deactivateMemberResult()
}

type MemberDeactivationAccepted struct{}

type DeactivateMemberRejected struct {
	Reason core.DomainError
}

func (MemberDeactivationAccepted) deactivateMemberResult() {}

func (DeactivateMemberRejected) deactivateMemberResult() {}

func (service Service) DeactivateMember(ctx context.Context, actor auth.Subject, organizationID core.OrganizationID, userID core.UserID) DeactivateMemberResult {
	permissionResult := service.requirePermissionForActor(ctx, actor, organizationID, PermissionManageMembers)
	if rejected, matched := permissionResult.(PermissionDenied); matched {
		return DeactivateMemberRejected{Reason: rejected.Reason}
	}

	targetRolesResult := service.store.FindMemberRoles(ctx, organizationID, userID)
	targetRolesFound, targetFound := targetRolesResult.(MemberRolesFound)
	if targetFound && containsRole(targetRolesFound.Roles, RoleOwner) {
		if reason := service.requireOwnerTierActor(ctx, actor, organizationID); reason != nil {
			return DeactivateMemberRejected{Reason: *reason}
		}
		if reason := service.requireNotLastActiveOwner(ctx, organizationID, userID); reason != nil {
			return DeactivateMemberRejected{Reason: *reason}
		}
	}

	storeResult := service.store.DeactivateMember(ctx, organizationID, userID)
	if rejected, matched := storeResult.(DeactivateMemberStoreRejected); matched {
		return DeactivateMemberRejected{Reason: rejected.Reason}
	}

	return MemberDeactivationAccepted{}
}

// requireOwnerTierActor gates an action that touches the owner role itself
// (granting it, revoking it, or deactivating a current owner): PermissionManageMembers
// alone (which RoleAdmin already holds) is not enough, since that would let an
// admin promote themselves to owner or remove the org's actual owners. An org-wide
// credential is deliberately not treated as owner-tier here even though it gets
// full admin parity elsewhere (requirePermissionForActor) - owner-level trust is
// reserved for a real member holding the role, not an API credential.
func (service Service) requireOwnerTierActor(ctx context.Context, actor auth.Subject, organizationID core.OrganizationID) *core.DomainError {
	userActor, isUser := actor.(auth.UserSubject)
	if !isUser {
		reason := core.NewDomainError(core.ErrorCodePermissionDenied, "only an organization owner can change ownership")
		return &reason
	}
	rolesResult := service.store.FindMemberRoles(ctx, organizationID, userActor.ID)
	rolesFound, matched := rolesResult.(MemberRolesFound)
	if !matched || !containsRole(rolesFound.Roles, RoleOwner) {
		reason := core.NewDomainError(core.ErrorCodePermissionDenied, "only an organization owner can change ownership")
		return &reason
	}
	return nil
}

// requireNotLastActiveOwner rejects removing owner status from userID if they
// are the organization's only remaining active owner, so an organization can
// never be left without one. Limited to the first page of members (the
// pagination maximum, 200) - sufficient in practice since owners are a small
// subset of an organization's roster.
func (service Service) requireNotLastActiveOwner(ctx context.Context, organizationID core.OrganizationID, excludeUserID core.UserID) *core.DomainError {
	page := core.NewPage(200, 0).(core.PageAccepted).Value
	membersResult := service.store.ListMembers(ctx, organizationID, page)
	listed, matched := membersResult.(MembersListed)
	if !matched {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "list organization members failed")
		return &reason
	}
	for _, member := range listed.Values {
		if member.UserID == excludeUserID || member.Status != MembershipStatusActive {
			continue
		}
		if containsRole(member.Roles, RoleOwner) {
			return nil
		}
	}
	reason := core.NewDomainError(core.ErrorCodeConflict, "organization must have at least one owner")
	return &reason
}

func containsRole(roles []Role, target Role) bool {
	for _, role := range roles {
		if role == target {
			return true
		}
	}
	return false
}

type UpdateMemberRolesResult interface {
	updateMemberRolesResult()
}

type MemberRolesUpdatedResult struct {
	Value OrganizationMember
}

type UpdateMemberRolesRejected struct {
	Reason core.DomainError
}

func (MemberRolesUpdatedResult) updateMemberRolesResult() {}

func (UpdateMemberRolesRejected) updateMemberRolesResult() {}

func (service Service) UpdateMemberRoles(ctx context.Context, actor auth.Subject, organizationID core.OrganizationID, userID core.UserID, roles []Role) UpdateMemberRolesResult {
	permissionResult := service.requirePermissionForActor(ctx, actor, organizationID, PermissionManageMembers)
	if rejected, matched := permissionResult.(PermissionDenied); matched {
		return UpdateMemberRolesRejected{Reason: rejected.Reason}
	}
	if len(roles) == 0 {
		return UpdateMemberRolesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "organization member roles are required")}
	}

	targetRolesResult := service.store.FindMemberRoles(ctx, organizationID, userID)
	targetRolesFound, targetFound := targetRolesResult.(MemberRolesFound)
	targetIsCurrentlyOwner := targetFound && containsRole(targetRolesFound.Roles, RoleOwner)
	grantingOwner := containsRole(roles, RoleOwner)
	if targetIsCurrentlyOwner || grantingOwner {
		if reason := service.requireOwnerTierActor(ctx, actor, organizationID); reason != nil {
			return UpdateMemberRolesRejected{Reason: *reason}
		}
	}
	if targetIsCurrentlyOwner && !grantingOwner {
		if reason := service.requireNotLastActiveOwner(ctx, organizationID, userID); reason != nil {
			return UpdateMemberRolesRejected{Reason: *reason}
		}
	}

	storeResult := service.store.UpdateMemberRoles(ctx, organizationID, userID, roles)
	updated, matched := storeResult.(MemberRolesUpdated)
	if !matched {
		return UpdateMemberRolesRejected{Reason: storeResult.(UpdateMemberRolesStoreRejected).Reason}
	}
	return MemberRolesUpdatedResult{Value: updated.Value}
}

func (service Service) requirePermission(ctx context.Context, organizationID core.OrganizationID, userID core.UserID, permission Permission) PermissionCheck {
	rolesResult := service.store.FindMemberRoles(ctx, organizationID, userID)
	rolesFound, matched := rolesResult.(MemberRolesFound)
	if !matched {
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "organization member roles were not found")}
	}
	return CheckPermission(rolesFound.Roles, permission)
}

// requirePermissionForActor adds org-token support in front of
// requirePermission: an org token is granted unconditionally for its own
// organization (full parity with an org-admin member), with no per-member
// role table to consult, since the token itself represents the org. A
// UserSubject actor delegates to the existing per-member check unchanged.
func (service Service) requirePermissionForActor(ctx context.Context, actor auth.Subject, organizationID core.OrganizationID, permission Permission) PermissionCheck {
	switch orgactor.Check(actor, organizationID) {
	case orgactor.Match:
		return PermissionGranted{}
	case orgactor.Mismatch:
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "organization access denied")}
	}
	userActor, isUser := actor.(auth.UserSubject)
	if !isUser {
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "organization access denied")}
	}
	return service.requirePermission(ctx, organizationID, userActor.ID, permission)
}

func (service Service) CheckOrganizationPermission(ctx context.Context, organizationID core.OrganizationID, userID core.UserID, permission Permission) PermissionCheck {
	return service.requirePermission(ctx, organizationID, userID, permission)
}

func (service Service) CheckOrganizationTeamMembership(ctx context.Context, organizationID core.OrganizationID, teamID core.TeamID, userID core.UserID) PermissionCheck {
	findResult := service.store.FindTeam(ctx, teamID)
	found, matched := findResult.(TeamFound)
	if !matched {
		return PermissionDenied{Reason: findResult.(TeamMissing).Reason}
	}
	owner, ownerMatched := found.Value.Owner.(OrganizationOwnedTeam)
	if !ownerMatched || owner.OrganizationID != organizationID {
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "organization team access denied")}
	}

	membersResult := service.store.ListTeamMembers(ctx, teamID)
	membersListed, membersMatched := membersResult.(TeamMembersListed)
	if !membersMatched {
		return PermissionDenied{Reason: membersResult.(TeamMembersRejected).Reason}
	}
	for _, memberID := range membersListed.Values {
		if memberID == userID {
			return PermissionGranted{}
		}
	}
	return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "organization team membership denied")}
}

func (service Service) CheckTeamMembership(ctx context.Context, teamID core.TeamID, userID core.UserID) PermissionCheck {
	membersResult := service.store.ListTeamMembers(ctx, teamID)
	members, matched := membersResult.(TeamMembersListed)
	if !matched {
		return PermissionDenied{Reason: membersResult.(TeamMembersRejected).Reason}
	}
	for _, memberID := range members.Values {
		if memberID == userID {
			return PermissionGranted{}
		}
	}
	return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "team membership denied")}
}
