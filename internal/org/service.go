package org

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

type Store interface {
	CreateOrganization(context.Context, core.OrganizationID, OrganizationName, core.UserID, core.OrganizationMembershipID) CreateOrganizationStoreResult
	ListOrganizationsForUser(context.Context, core.UserID) ListOrganizationsResult
	FindMemberRoles(context.Context, core.OrganizationID, core.UserID) MemberRolesResult
	ProvisionMember(context.Context, core.OrganizationMembershipID, core.OrganizationID, auth.EmailAddress, []Role) ProvisionMemberStoreResult
	DeactivateMember(context.Context, core.OrganizationID, core.UserID) DeactivateMemberStoreResult
	CreateOrganizationTeam(context.Context, core.TeamID, core.OrganizationID, TeamName, core.UserID) CreateTeamStoreResult
	AddTeamMember(context.Context, core.TeamID, core.UserID) AddTeamMemberStoreResult
	ListOrganizationTeams(context.Context, core.OrganizationID, core.UserID) TeamListResult
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

func (service Service) ListOrganizations(ctx context.Context, actor auth.UserSubject) ListOrganizationsResult {
	result := service.store.ListOrganizationsForUser(ctx, actor.ID)
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

func (service Service) ProvisionMember(ctx context.Context, actor auth.UserSubject, organizationID core.OrganizationID, email auth.EmailAddress, roles []Role) ProvisionMemberResult {
	permissionResult := service.requirePermission(ctx, organizationID, actor.ID, PermissionManageMembers)
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

	return TeamCreated{Value: Team{ID: teamIDCreated.Value, OrganizationID: organizationID, Name: name, CreatedBy: actor.ID}}
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

func (service Service) ListOrganizationTeams(ctx context.Context, actor auth.UserSubject, organizationID core.OrganizationID) ListTeamsResult {
	result := service.store.ListOrganizationTeams(ctx, organizationID, actor.ID)
	listed, matched := result.(TeamsListed)
	if !matched {
		rejected := result.(TeamListRejected)
		return ListTeamsRejected{Reason: rejected.Reason}
	}
	return OrganizationTeamsListed{Values: listed.Values}
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

func (service Service) DeactivateMember(ctx context.Context, actor auth.UserSubject, organizationID core.OrganizationID, userID core.UserID) DeactivateMemberResult {
	permissionResult := service.requirePermission(ctx, organizationID, actor.ID, PermissionManageMembers)
	if rejected, matched := permissionResult.(PermissionDenied); matched {
		return DeactivateMemberRejected{Reason: rejected.Reason}
	}

	storeResult := service.store.DeactivateMember(ctx, organizationID, userID)
	if rejected, matched := storeResult.(DeactivateMemberStoreRejected); matched {
		return DeactivateMemberRejected{Reason: rejected.Reason}
	}

	return MemberDeactivationAccepted{}
}

func (service Service) requirePermission(ctx context.Context, organizationID core.OrganizationID, userID core.UserID, permission Permission) PermissionCheck {
	rolesResult := service.store.FindMemberRoles(ctx, organizationID, userID)
	rolesFound, matched := rolesResult.(MemberRolesFound)
	if !matched {
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "organization member roles were not found")}
	}
	return CheckPermission(rolesFound.Roles, permission)
}

func (service Service) CheckOrganizationPermission(ctx context.Context, organizationID core.OrganizationID, userID core.UserID, permission Permission) PermissionCheck {
	return service.requirePermission(ctx, organizationID, userID, permission)
}
