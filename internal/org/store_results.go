package org

import "github.com/e6qu/sharecrop/internal/core"

type CreateOrganizationStoreResult interface {
	createOrganizationStoreResult()
}

type CreateOrganizationStoreAccepted struct{}

type CreateOrganizationStoreRejected struct {
	Reason core.DomainError
}

func (CreateOrganizationStoreAccepted) createOrganizationStoreResult() {}

func (CreateOrganizationStoreRejected) createOrganizationStoreResult() {}

type MemberRolesResult interface {
	memberRolesResult()
}

type MemberRolesFound struct {
	Roles []Role
}

type MemberRolesMissing struct{}

type MemberRolesRejected struct {
	Reason core.DomainError
}

func (MemberRolesFound) memberRolesResult() {}

func (MemberRolesMissing) memberRolesResult() {}

func (MemberRolesRejected) memberRolesResult() {}

type ProvisionMemberStoreResult interface {
	provisionMemberStoreResult()
}

type ProvisionMemberStoreRejected struct {
	Reason core.DomainError
}

func (MemberProvisioned) provisionMemberStoreResult() {}

func (ProvisionMemberStoreRejected) provisionMemberStoreResult() {}

type DeactivateMemberStoreResult interface {
	deactivateMemberStoreResult()
}

type MemberDeactivated struct{}

type DeactivateMemberStoreRejected struct {
	Reason core.DomainError
}

func (MemberDeactivated) deactivateMemberStoreResult() {}

func (DeactivateMemberStoreRejected) deactivateMemberStoreResult() {}

type CreateTeamStoreResult interface {
	createTeamStoreResult()
}

type CreateTeamStoreAccepted struct{}

type CreateTeamStoreRejected struct {
	Reason core.DomainError
}

func (CreateTeamStoreAccepted) createTeamStoreResult() {}

func (CreateTeamStoreRejected) createTeamStoreResult() {}

type AddTeamMemberStoreResult interface {
	addTeamMemberStoreResult()
}

type TeamMemberAdded struct{}

type AddTeamMemberStoreRejected struct {
	Reason core.DomainError
}

func (TeamMemberAdded) addTeamMemberStoreResult() {}

func (AddTeamMemberStoreRejected) addTeamMemberStoreResult() {}

type TeamListResult interface {
	teamListResult()
}

type TeamsListed struct {
	Values []Team
}

type TeamListRejected struct {
	Reason core.DomainError
}

func (TeamsListed) teamListResult() {}

func (TeamListRejected) teamListResult() {}
