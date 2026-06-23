package org

import "github.com/e6qu/sharecrop/internal/core"

type Organization struct {
	ID        core.OrganizationID
	Name      OrganizationName
	CreatedBy core.UserID
}

type OrganizationMember struct {
	ID             core.OrganizationMembershipID
	OrganizationID core.OrganizationID
	UserID         core.UserID
	Status         MembershipStatus
	Roles          []Role
}

type Team struct {
	ID        core.TeamID
	Owner     TeamOwner
	Name      TeamName
	CreatedBy core.UserID
}

// TeamOwner is a tagged union over the entities that can own a team. A team is
// owned either by an organization or directly by a user (a standalone team).
// Ownership is an explicit variant rather than a nullable organization column
// used as a flag.
type TeamOwner interface {
	teamOwner()
	Kind() TeamOwnerKind
}

type OrganizationOwnedTeam struct {
	OrganizationID core.OrganizationID
}

type UserOwnedTeam struct {
	OwnerUserID core.UserID
}

func (OrganizationOwnedTeam) teamOwner() {}

func (UserOwnedTeam) teamOwner() {}

func (OrganizationOwnedTeam) Kind() TeamOwnerKind { return TeamOwnerKindOrganization }

func (UserOwnedTeam) Kind() TeamOwnerKind { return TeamOwnerKindUser }

type TeamOwnerKind struct {
	value string
}

var (
	TeamOwnerKindOrganization = TeamOwnerKind{value: "organization"}
	TeamOwnerKindUser         = TeamOwnerKind{value: "user"}
)

func (kind TeamOwnerKind) String() string {
	return kind.value
}

type TeamMember struct {
	TeamID core.TeamID
	UserID core.UserID
}

type MembershipStatus struct {
	value string
}

var (
	MembershipStatusActive      = MembershipStatus{value: "active"}
	MembershipStatusDeactivated = MembershipStatus{value: "deactivated"}
	MembershipStatusRemoved     = MembershipStatus{value: "removed"}
)

type MembershipStatusResult interface {
	membershipStatusResult()
}

type MembershipStatusAccepted struct {
	Value MembershipStatus
}

type MembershipStatusRejected struct {
	Reason core.DomainError
}

func (MembershipStatusAccepted) membershipStatusResult() {}

func (MembershipStatusRejected) membershipStatusResult() {}

func ParseMembershipStatus(raw string) MembershipStatusResult {
	switch raw {
	case MembershipStatusActive.value:
		return MembershipStatusAccepted{Value: MembershipStatusActive}
	case MembershipStatusDeactivated.value:
		return MembershipStatusAccepted{Value: MembershipStatusDeactivated}
	case MembershipStatusRemoved.value:
		return MembershipStatusAccepted{Value: MembershipStatusRemoved}
	default:
		return MembershipStatusRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "membership status is invalid")}
	}
}

func (status MembershipStatus) String() string {
	return status.value
}
