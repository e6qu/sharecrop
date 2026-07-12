// Package taskbridge is the WASI bridge for internal/task's Store (tasks,
// series, comments, and reservations - the widest store): hand-written per-type
// codecs (this file, models_codec.go, results_codec.go) plus a generated
// dispatcher and guest client (bridge_gen.go). Shared core types (ids, page) are
// serialized by internal/wasibridge/corewire; the domain error by
// internal/wasibridge/domainwire.
//
// Value objects round-trip through their existing validating constructors: the
// store only ever emits values those constructors accept (non-empty titles,
// positive amounts, known enum strings), so re-validation on decode never
// rejects a legitimately-stored value and doubles as a wire-corruption guard.
package taskbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
)

// ---- enums (string wrappers with Parse* constructors) ----

func encodeState(state task.State) string { return state.String() }

func decodeState(raw string) (task.State, error) {
	accepted, matched := task.ParseState(raw).(task.StateAccepted)
	if !matched {
		return task.State{}, fmt.Errorf("invalid task state %q", raw)
	}
	return accepted.Value, nil
}

func encodeReservationState(state task.ReservationState) string { return state.String() }

func decodeReservationState(raw string) (task.ReservationState, error) {
	accepted, matched := task.ParseReservationState(raw).(task.ReservationStateAccepted)
	if !matched {
		return task.ReservationState{}, fmt.Errorf("invalid reservation state %q", raw)
	}
	return accepted.Value, nil
}

func encodeSeriesState(state task.SeriesState) string { return state.String() }

func decodeSeriesState(raw string) (task.SeriesState, error) {
	accepted, matched := task.ParseSeriesState(raw).(task.SeriesStateAccepted)
	if !matched {
		return task.SeriesState{}, fmt.Errorf("invalid series state %q", raw)
	}
	return accepted.Value, nil
}

func encodeParticipationPolicy(policy task.ParticipationPolicy) string { return policy.String() }

func decodeParticipationPolicy(raw string) (task.ParticipationPolicy, error) {
	accepted, matched := task.ParseParticipationPolicy(raw).(task.ParticipationPolicyAccepted)
	if !matched {
		return task.ParticipationPolicy{}, fmt.Errorf("invalid participation policy %q", raw)
	}
	return accepted.Value, nil
}

func encodeAssigneeScope(scope task.AssigneeScope) string { return scope.String() }

func decodeAssigneeScope(raw string) (task.AssigneeScope, error) {
	accepted, matched := task.ParseAssigneeScope(raw).(task.AssigneeScopeAccepted)
	if !matched {
		return task.AssigneeScope{}, fmt.Errorf("invalid assignee scope %q", raw)
	}
	return accepted.Value, nil
}

func encodeTaskType(taskType task.TaskType) string { return taskType.String() }

func decodeTaskType(raw string) (task.TaskType, error) {
	accepted, matched := task.ParseTaskType(raw).(task.TaskTypeAccepted)
	if !matched {
		return task.TaskType{}, fmt.Errorf("invalid task type %q", raw)
	}
	return accepted.Value, nil
}

func encodeSortOrder(order task.SortOrder) string { return order.String() }

func decodeSortOrder(raw string) (task.SortOrder, error) {
	accepted, matched := task.ParseSortOrder(raw).(task.SortOrderAccepted)
	if !matched {
		return task.SortOrder{}, fmt.Errorf("invalid sort order %q", raw)
	}
	return accepted.Value, nil
}

// ---- string value objects (New* constructors) ----

func encodeTitle(title task.Title) string { return title.String() }

func decodeTitle(raw string) (task.Title, error) {
	accepted, matched := task.NewTitle(raw).(task.TitleAccepted)
	if !matched {
		return task.Title{}, fmt.Errorf("invalid task title")
	}
	return accepted.Value, nil
}

func encodeDescription(description task.Description) string { return description.String() }

func decodeDescription(raw string) (task.Description, error) {
	accepted, matched := task.NewDescription(raw).(task.DescriptionAccepted)
	if !matched {
		return task.Description{}, fmt.Errorf("invalid task description")
	}
	return accepted.Value, nil
}

func encodeReferenceURL(reference task.ReferenceURL) string { return reference.String() }

func decodeReferenceURL(raw string) (task.ReferenceURL, error) {
	accepted, matched := task.NewReferenceURL(raw).(task.ReferenceURLAccepted)
	if !matched {
		return task.ReferenceURL{}, fmt.Errorf("invalid reference URL")
	}
	return accepted.Value, nil
}

func encodeResponseSchema(source task.ResponseSchemaSource) string { return source.String() }

func decodeResponseSchema(raw string) (task.ResponseSchemaSource, error) {
	accepted, matched := task.NewResponseSchemaSource(raw).(task.ResponseSchemaSourceAccepted)
	if !matched {
		return task.ResponseSchemaSource{}, fmt.Errorf("invalid response schema")
	}
	return accepted.Value, nil
}

func encodeSeriesTitle(title task.SeriesTitle) string { return title.String() }

func decodeSeriesTitle(raw string) (task.SeriesTitle, error) {
	accepted, matched := task.NewSeriesTitle(raw).(task.SeriesTitleAccepted)
	if !matched {
		return task.SeriesTitle{}, fmt.Errorf("invalid series title")
	}
	return accepted.Value, nil
}

func encodeSeriesDescription(description task.SeriesDescription) string { return description.String() }

func decodeSeriesDescription(raw string) (task.SeriesDescription, error) {
	accepted, matched := task.NewSeriesDescription(raw).(task.SeriesDescriptionAccepted)
	if !matched {
		return task.SeriesDescription{}, fmt.Errorf("invalid series description")
	}
	return accepted.Value, nil
}

func encodeSearchText(text task.SearchText) string { return text.String() }

func decodeSearchText(raw string) (task.SearchText, error) {
	accepted, matched := task.NewSearchText(raw).(task.SearchTextAccepted)
	if !matched {
		return task.SearchText{}, fmt.Errorf("invalid search text")
	}
	return accepted.Value, nil
}

func encodeCommentBody(body task.CommentBody) string { return body.String() }

func decodeCommentBody(raw string) (task.CommentBody, error) {
	accepted, matched := task.NewCommentBody(raw).(task.CommentBodyAccepted)
	if !matched {
		return task.CommentBody{}, fmt.Errorf("invalid comment body")
	}
	return accepted.Value, nil
}

// ---- task id slices ----

func encodeTaskIDs(ids []core.TaskID) []string {
	encoded := make([]string, 0, len(ids))
	for index := range ids {
		encoded = append(encoded, corewire.EncodeTaskID(ids[index]))
	}
	return encoded
}

func decodeTaskIDs(raw []string) ([]core.TaskID, error) {
	ids := make([]core.TaskID, 0, len(raw))
	for index := range raw {
		id, err := corewire.DecodeTaskID(raw[index])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ---- Owner union ----

type ownerWire struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
}

func encodeOwner(owner task.Owner) ownerWire {
	switch typed := owner.(type) {
	case task.UserOwner:
		return ownerWire{Kind: "user", UserID: corewire.EncodeUserID(typed.UserID)}
	case task.TeamOwner:
		return ownerWire{Kind: "team", TeamID: corewire.EncodeTeamID(typed.TeamID)}
	case task.OrganizationOwner:
		return ownerWire{Kind: "organization", OrganizationID: corewire.EncodeOrganizationID(typed.OrganizationID)}
	case task.OrganizationTeamOwner:
		return ownerWire{Kind: "organization_team", OrganizationID: corewire.EncodeOrganizationID(typed.OrganizationID), TeamID: corewire.EncodeTeamID(typed.TeamID)}
	default:
		return ownerWire{Kind: "user"}
	}
}

func decodeOwner(wire ownerWire) (task.Owner, error) {
	switch wire.Kind {
	case "user":
		id, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return task.UserOwner{UserID: id}, nil
	case "team":
		id, err := corewire.DecodeTeamID(wire.TeamID)
		if err != nil {
			return nil, err
		}
		return task.TeamOwner{TeamID: id}, nil
	case "organization":
		id, err := corewire.DecodeOrganizationID(wire.OrganizationID)
		if err != nil {
			return nil, err
		}
		return task.OrganizationOwner{OrganizationID: id}, nil
	case "organization_team":
		organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
		if err != nil {
			return nil, err
		}
		teamID, err := corewire.DecodeTeamID(wire.TeamID)
		if err != nil {
			return nil, err
		}
		return task.OrganizationTeamOwner{OrganizationID: organizationID, TeamID: teamID}, nil
	default:
		return nil, fmt.Errorf("unknown owner kind %q", wire.Kind)
	}
}

// ---- Visibility union ----

type visibilityWire struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
}

func encodeVisibility(visibility task.Visibility) visibilityWire {
	switch typed := visibility.(type) {
	case task.UserVisibility:
		return visibilityWire{Kind: "user", UserID: corewire.EncodeUserID(typed.UserID)}
	case task.TeamVisibility:
		return visibilityWire{Kind: "team", TeamID: corewire.EncodeTeamID(typed.TeamID)}
	case task.OrganizationVisibility:
		return visibilityWire{Kind: "organization", OrganizationID: corewire.EncodeOrganizationID(typed.OrganizationID)}
	case task.OrganizationTeamVisibility:
		return visibilityWire{Kind: "organization_team", OrganizationID: corewire.EncodeOrganizationID(typed.OrganizationID), TeamID: corewire.EncodeTeamID(typed.TeamID)}
	default:
		return visibilityWire{Kind: "public"}
	}
}

func decodeVisibility(wire visibilityWire) (task.Visibility, error) {
	switch wire.Kind {
	case "public":
		return task.PublicVisibility{}, nil
	case "user":
		id, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return task.UserVisibility{UserID: id}, nil
	case "team":
		id, err := corewire.DecodeTeamID(wire.TeamID)
		if err != nil {
			return nil, err
		}
		return task.TeamVisibility{TeamID: id}, nil
	case "organization":
		id, err := corewire.DecodeOrganizationID(wire.OrganizationID)
		if err != nil {
			return nil, err
		}
		return task.OrganizationVisibility{OrganizationID: id}, nil
	case "organization_team":
		organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
		if err != nil {
			return nil, err
		}
		teamID, err := corewire.DecodeTeamID(wire.TeamID)
		if err != nil {
			return nil, err
		}
		return task.OrganizationTeamVisibility{OrganizationID: organizationID, TeamID: teamID}, nil
	default:
		return nil, fmt.Errorf("unknown visibility kind %q", wire.Kind)
	}
}

// ---- Assignee union ----

type assigneeWire struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
}

func encodeAssignee(assignee task.Assignee) assigneeWire {
	switch typed := assignee.(type) {
	case task.UserAssignee:
		return assigneeWire{Kind: "user", UserID: corewire.EncodeUserID(typed.UserID)}
	case task.OrganizationTeamAssignee:
		return assigneeWire{Kind: "organization_team", OrganizationID: corewire.EncodeOrganizationID(typed.OrganizationID), TeamID: corewire.EncodeTeamID(typed.TeamID)}
	case task.TeamAssignee:
		return assigneeWire{Kind: "team", TeamID: corewire.EncodeTeamID(typed.TeamID)}
	default:
		return assigneeWire{Kind: "user"}
	}
}

func decodeAssignee(wire assigneeWire) (task.Assignee, error) {
	switch wire.Kind {
	case "user":
		id, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return task.UserAssignee{UserID: id}, nil
	case "team":
		id, err := corewire.DecodeTeamID(wire.TeamID)
		if err != nil {
			return nil, err
		}
		return task.TeamAssignee{TeamID: id}, nil
	case "organization_team":
		organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
		if err != nil {
			return nil, err
		}
		teamID, err := corewire.DecodeTeamID(wire.TeamID)
		if err != nil {
			return nil, err
		}
		return task.OrganizationTeamAssignee{OrganizationID: organizationID, TeamID: teamID}, nil
	default:
		return nil, fmt.Errorf("unknown assignee kind %q", wire.Kind)
	}
}

// ---- ActiveAssignee union ----

type activeAssigneeWire struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id,omitempty"`
	TeamID         string `json:"team_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
}

func encodeActiveAssignee(assignee task.ActiveAssignee) activeAssigneeWire {
	switch typed := assignee.(type) {
	case task.ActiveUserAssignee:
		return activeAssigneeWire{Kind: "user", UserID: corewire.EncodeUserID(typed.UserID)}
	case task.ActiveOrganizationTeamAssignee:
		return activeAssigneeWire{Kind: "organization_team", OrganizationID: corewire.EncodeOrganizationID(typed.OrganizationID), TeamID: corewire.EncodeTeamID(typed.TeamID)}
	case task.ActiveTeamAssignee:
		return activeAssigneeWire{Kind: "team", TeamID: corewire.EncodeTeamID(typed.TeamID)}
	default:
		return activeAssigneeWire{Kind: "none"}
	}
}

func decodeActiveAssignee(wire activeAssigneeWire) (task.ActiveAssignee, error) {
	switch wire.Kind {
	case "none":
		return task.NoActiveAssignee{}, nil
	case "user":
		id, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return task.ActiveUserAssignee{UserID: id}, nil
	case "team":
		id, err := corewire.DecodeTeamID(wire.TeamID)
		if err != nil {
			return nil, err
		}
		return task.ActiveTeamAssignee{TeamID: id}, nil
	case "organization_team":
		organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
		if err != nil {
			return nil, err
		}
		teamID, err := corewire.DecodeTeamID(wire.TeamID)
		if err != nil {
			return nil, err
		}
		return task.ActiveOrganizationTeamAssignee{OrganizationID: organizationID, TeamID: teamID}, nil
	default:
		return nil, fmt.Errorf("unknown active assignee kind %q", wire.Kind)
	}
}

// ---- RewardSpec union ----

type rewardSpecWire struct {
	Kind        string `json:"kind"`
	Credit      int64  `json:"credit,omitempty"`
	Collectible int    `json:"collectible,omitempty"`
}

func encodeRewardSpec(reward task.RewardSpec) rewardSpecWire {
	switch typed := reward.(type) {
	case task.CreditRewardSpec:
		return rewardSpecWire{Kind: "credit", Credit: typed.Amount.Int64()}
	case task.CollectibleRewardSpec:
		return rewardSpecWire{Kind: "collectible", Collectible: typed.Count.Int()}
	case task.BundleRewardSpec:
		return rewardSpecWire{Kind: "bundle", Credit: typed.Credit.Int64(), Collectible: typed.Collectible.Int()}
	default:
		return rewardSpecWire{Kind: "none"}
	}
}

func decodeRewardSpec(wire rewardSpecWire) (task.RewardSpec, error) {
	switch wire.Kind {
	case "none":
		return task.NoRewardSpec{}, nil
	case "credit":
		amount, err := decodeCreditRewardAmount(wire.Credit)
		if err != nil {
			return nil, err
		}
		return task.CreditRewardSpec{Amount: amount}, nil
	case "collectible":
		count, err := decodeCollectibleRewardCount(wire.Collectible)
		if err != nil {
			return nil, err
		}
		return task.CollectibleRewardSpec{Count: count}, nil
	case "bundle":
		amount, err := decodeCreditRewardAmount(wire.Credit)
		if err != nil {
			return nil, err
		}
		count, err := decodeCollectibleRewardCount(wire.Collectible)
		if err != nil {
			return nil, err
		}
		return task.BundleRewardSpec{Credit: amount, Collectible: count}, nil
	default:
		return nil, fmt.Errorf("unknown reward spec kind %q", wire.Kind)
	}
}

func decodeCreditRewardAmount(value int64) (task.CreditRewardAmount, error) {
	accepted, matched := task.NewCreditRewardAmount(value).(task.CreditRewardAmountAccepted)
	if !matched {
		return task.CreditRewardAmount{}, fmt.Errorf("invalid credit reward amount %d", value)
	}
	return accepted.Value, nil
}

func decodeCollectibleRewardCount(value int) (task.CollectibleRewardCount, error) {
	accepted, matched := task.NewCollectibleRewardCount(value).(task.CollectibleRewardCountAccepted)
	if !matched {
		return task.CollectibleRewardCount{}, fmt.Errorf("invalid collectible reward count %d", value)
	}
	return accepted.Value, nil
}

// ---- ReservationTTL ----

func encodeReservationTTL(ttl task.ReservationTTL) int { return ttl.Hours() }

func decodeReservationTTL(hours int) (task.ReservationTTL, error) {
	accepted, matched := task.NewReservationTTL(hours).(task.ReservationTTLAccepted)
	if !matched {
		return task.ReservationTTL{}, fmt.Errorf("invalid reservation ttl %d", hours)
	}
	return accepted.Value, nil
}

// ---- SeriesPlacement union ----

type seriesPlacementWire struct {
	Kind     string `json:"kind"`
	Title    string `json:"title,omitempty"`
	SeriesID string `json:"series_id,omitempty"`
	Position int    `json:"position,omitempty"`
}

func encodeSeriesPlacement(placement task.SeriesPlacement) seriesPlacementWire {
	switch typed := placement.(type) {
	case task.NewSeriesPlacement:
		return seriesPlacementWire{Kind: "new", Title: typed.Title.String(), Position: typed.Position.Int()}
	case task.ExistingSeriesPlacement:
		return seriesPlacementWire{Kind: "existing", SeriesID: corewire.EncodeTaskSeriesID(typed.SeriesID), Position: typed.Position.Int()}
	default:
		return seriesPlacementWire{Kind: "standalone"}
	}
}

func decodeSeriesPlacement(wire seriesPlacementWire) (task.SeriesPlacement, error) {
	switch wire.Kind {
	case "standalone":
		return task.StandalonePlacement{}, nil
	case "new":
		title, err := decodeSeriesTitle(wire.Title)
		if err != nil {
			return nil, err
		}
		position, err := decodeSeriesPosition(wire.Position)
		if err != nil {
			return nil, err
		}
		return task.NewSeriesPlacement{Title: title, Position: position}, nil
	case "existing":
		seriesID, err := corewire.DecodeTaskSeriesID(wire.SeriesID)
		if err != nil {
			return nil, err
		}
		position, err := decodeSeriesPosition(wire.Position)
		if err != nil {
			return nil, err
		}
		return task.ExistingSeriesPlacement{SeriesID: seriesID, Position: position}, nil
	default:
		return nil, fmt.Errorf("unknown series placement kind %q", wire.Kind)
	}
}

func decodeSeriesPosition(value int) (task.SeriesPosition, error) {
	accepted, matched := task.NewSeriesPosition(value).(task.SeriesPositionAccepted)
	if !matched {
		return task.SeriesPosition{}, fmt.Errorf("invalid series position %d", value)
	}
	return accepted.Value, nil
}

// ---- DataPayload union ----

type dataPayloadWire struct {
	Kind   string `json:"kind"`
	Source string `json:"source,omitempty"`
}

func encodeDataPayload(payload task.DataPayload) dataPayloadWire {
	json, matched := payload.(task.JSONDataPayload)
	if !matched {
		return dataPayloadWire{Kind: "none"}
	}
	return dataPayloadWire{Kind: "json", Source: json.Source.String()}
}

func decodeDataPayload(wire dataPayloadWire) (task.DataPayload, error) {
	switch wire.Kind {
	case "none":
		return task.NoDataPayload{}, nil
	case "json":
		accepted, matched := task.NewPayloadSource(wire.Source).(task.PayloadSourceAccepted)
		if !matched {
			return nil, fmt.Errorf("invalid data payload source")
		}
		return task.JSONDataPayload{Source: accepted.Value}, nil
	default:
		return nil, fmt.Errorf("unknown data payload kind %q", wire.Kind)
	}
}
