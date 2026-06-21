package db

import (
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

type taskOwnerResult interface {
	taskOwnerResult()
}

type taskOwnerAccepted struct {
	value task.Owner
}

type taskOwnerRejected struct {
	reason core.DomainError
}

func (taskOwnerAccepted) taskOwnerResult() {}

func (taskOwnerRejected) taskOwnerResult() {}

func parseTaskOwner(kind string, rawUserID string, rawTeamID string, rawOrganizationID string) taskOwnerResult {
	switch kind {
	case task.OwnerKindUser.String():
		userIDResult := core.ParseUserID(rawUserID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			rejected := userIDResult.(core.UserIDRejected)
			return taskOwnerRejected{reason: rejected.Reason}
		}
		return taskOwnerAccepted{value: task.UserOwner{UserID: userID.Value}}
	case task.OwnerKindTeam.String():
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamID, matched := teamIDResult.(core.TeamIDCreated)
		if !matched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskOwnerRejected{reason: rejected.Reason}
		}
		return taskOwnerAccepted{value: task.TeamOwner{TeamID: teamID.Value}}
	case task.OwnerKindOrganization.String():
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskOwnerRejected{reason: rejected.Reason}
		}
		return taskOwnerAccepted{value: task.OrganizationOwner{OrganizationID: organizationID.Value}}
	case task.OwnerKindOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskOwnerRejected{reason: rejected.Reason}
		}
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskOwnerRejected{reason: rejected.Reason}
		}
		return taskOwnerAccepted{value: task.OrganizationTeamOwner{OrganizationID: organizationID.Value, TeamID: teamID.Value}}
	default:
		return taskOwnerRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task owner kind is invalid")}
	}
}

type taskVisibilityResult interface {
	taskVisibilityResult()
}

type taskVisibilityAccepted struct {
	value task.Visibility
}

type taskVisibilityRejected struct {
	reason core.DomainError
}

func (taskVisibilityAccepted) taskVisibilityResult() {}

func (taskVisibilityRejected) taskVisibilityResult() {}

func parseTaskVisibility(kind string, rawUserID string, rawTeamID string, rawOrganizationID string) taskVisibilityResult {
	switch kind {
	case task.VisibilityKindPublic.String():
		return taskVisibilityAccepted{value: task.PublicVisibility{}}
	case task.VisibilityKindUser.String():
		userIDResult := core.ParseUserID(rawUserID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			rejected := userIDResult.(core.UserIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason}
		}
		return taskVisibilityAccepted{value: task.UserVisibility{UserID: userID.Value}}
	case task.VisibilityKindTeam.String():
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamID, matched := teamIDResult.(core.TeamIDCreated)
		if !matched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason}
		}
		return taskVisibilityAccepted{value: task.TeamVisibility{TeamID: teamID.Value}}
	case task.VisibilityKindOrganization.String():
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason}
		}
		return taskVisibilityAccepted{value: task.OrganizationVisibility{OrganizationID: organizationID.Value}}
	case task.VisibilityKindOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason}
		}
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason}
		}
		return taskVisibilityAccepted{value: task.OrganizationTeamVisibility{OrganizationID: organizationID.Value, TeamID: teamID.Value}}
	default:
		return taskVisibilityRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task visibility kind is invalid")}
	}
}

type seriesPlacementResult interface {
	seriesPlacementResult()
}

type seriesPlacementAccepted struct {
	value task.SeriesPlacement
}

type seriesPlacementRejected struct {
	reason core.DomainError
}

func (seriesPlacementAccepted) seriesPlacementResult() {}

func (seriesPlacementRejected) seriesPlacementResult() {}

func parseSeriesPlacement(rawSeriesID string, rawPosition int) seriesPlacementResult {
	if rawSeriesID == "" {
		return seriesPlacementAccepted{value: task.StandalonePlacement{}}
	}
	seriesIDResult := core.ParseTaskSeriesID(rawSeriesID)
	seriesID, seriesMatched := seriesIDResult.(core.TaskSeriesIDCreated)
	if !seriesMatched {
		rejected := seriesIDResult.(core.TaskSeriesIDRejected)
		return seriesPlacementRejected{reason: rejected.Reason}
	}
	positionResult := task.NewSeriesPosition(rawPosition)
	position, positionMatched := positionResult.(task.SeriesPositionAccepted)
	if !positionMatched {
		rejected := positionResult.(task.SeriesPositionRejected)
		return seriesPlacementRejected{reason: rejected.Reason}
	}
	return seriesPlacementAccepted{value: task.ExistingSeriesPlacement{SeriesID: seriesID.Value, Position: position.Value}}
}

type dataPayloadResult interface {
	dataPayloadResult()
}

type dataPayloadAccepted struct {
	value task.DataPayload
}

type dataPayloadRejected struct {
	reason core.DomainError
}

func (dataPayloadAccepted) dataPayloadResult() {}

func (dataPayloadRejected) dataPayloadResult() {}

func parseDataPayload(kind string, rawPayload string) dataPayloadResult {
	switch kind {
	case "none":
		return dataPayloadAccepted{value: task.NoDataPayload{}}
	case "json":
		payloadResult := task.NewPayloadSource(rawPayload)
		payload, matched := payloadResult.(task.PayloadSourceAccepted)
		if !matched {
			rejected := payloadResult.(task.PayloadSourceRejected)
			return dataPayloadRejected{reason: rejected.Reason}
		}
		return dataPayloadAccepted{value: task.JSONDataPayload{Source: payload.Value}}
	default:
		return dataPayloadRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task payload kind is invalid")}
	}
}
