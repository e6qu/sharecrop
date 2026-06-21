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
	ID             core.TeamID
	OrganizationID core.OrganizationID
	Name           TeamName
	CreatedBy      core.UserID
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
