package org

import (
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

type OrganizationName struct {
	value string
}

type TeamName struct {
	value string
}

type OrganizationNameResult interface {
	organizationNameResult()
}

type OrganizationNameAccepted struct {
	Value OrganizationName
}

type OrganizationNameRejected struct {
	Reason core.DomainError
}

func (OrganizationNameAccepted) organizationNameResult() {}

func (OrganizationNameRejected) organizationNameResult() {}

func NewOrganizationName(raw string) OrganizationNameResult {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 2 {
		return OrganizationNameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "organization name must contain at least 2 characters")}
	}
	if len(trimmed) > 120 {
		return OrganizationNameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "organization name must contain at most 120 characters")}
	}
	return OrganizationNameAccepted{Value: OrganizationName{value: trimmed}}
}

func (name OrganizationName) String() string {
	return name.value
}

type TeamNameResult interface {
	teamNameResult()
}

type TeamNameAccepted struct {
	Value TeamName
}

type TeamNameRejected struct {
	Reason core.DomainError
}

func (TeamNameAccepted) teamNameResult() {}

func (TeamNameRejected) teamNameResult() {}

func NewTeamName(raw string) TeamNameResult {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 2 {
		return TeamNameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "team name must contain at least 2 characters")}
	}
	if len(trimmed) > 120 {
		return TeamNameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "team name must contain at most 120 characters")}
	}
	return TeamNameAccepted{Value: TeamName{value: trimmed}}
}

func (name TeamName) String() string {
	return name.value
}
