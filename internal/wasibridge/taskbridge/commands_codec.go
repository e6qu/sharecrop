package taskbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasibridge/attachmentwire"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
)

// ---- task.CreateCommand ----
//
// The store reads only the actor's id from CreateCommand.Actor, so the actor
// crosses the wire as a plain user id and is rebuilt as an auth.UserSubject.

type createCommandWire struct {
	ActorID        string                `json:"actor_id"`
	Owner          ownerWire             `json:"owner"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Type           string                `json:"type"`
	Reference      string                `json:"reference"`
	Reward         rewardSpecWire        `json:"reward"`
	Participation  string                `json:"participation"`
	AssigneeScope  string                `json:"assignee_scope"`
	ReservationTTL int                   `json:"reservation_ttl"`
	Visibility     visibilityWire        `json:"visibility"`
	Placement      seriesPlacementWire   `json:"placement"`
	ResponseSchema string                `json:"response_schema"`
	Payload        dataPayloadWire       `json:"payload"`
	Attachments    []attachmentwire.Wire `json:"attachments,omitempty"`
}

func encodeCreateCommand(command task.CreateCommand) createCommandWire {
	return createCommandWire{
		ActorID:        corewire.EncodeUserID(command.Actor.ID),
		Owner:          encodeOwner(command.Owner),
		Title:          encodeTitle(command.Title),
		Description:    encodeDescription(command.Description),
		Type:           encodeTaskType(command.Type),
		Reference:      encodeReferenceURL(command.Reference),
		Reward:         encodeRewardSpec(command.Reward),
		Participation:  encodeParticipationPolicy(command.Participation),
		AssigneeScope:  encodeAssigneeScope(command.AssigneeScope),
		ReservationTTL: encodeReservationTTL(command.ReservationTTL),
		Visibility:     encodeVisibility(command.Visibility),
		Placement:      encodeSeriesPlacement(command.Placement),
		ResponseSchema: encodeResponseSchema(command.ResponseSchema),
		Payload:        encodeDataPayload(command.Payload),
		Attachments:    attachmentwire.EncodeSlice(command.Attachments),
	}
}

func decodeCreateCommand(wire createCommandWire) (task.CreateCommand, error) {
	actorID, err := corewire.DecodeUserID(wire.ActorID)
	if err != nil {
		return task.CreateCommand{}, err
	}
	owner, err := decodeOwner(wire.Owner)
	if err != nil {
		return task.CreateCommand{}, err
	}
	title, err := decodeTitle(wire.Title)
	if err != nil {
		return task.CreateCommand{}, err
	}
	description, err := decodeDescription(wire.Description)
	if err != nil {
		return task.CreateCommand{}, err
	}
	taskType, err := decodeTaskType(wire.Type)
	if err != nil {
		return task.CreateCommand{}, err
	}
	reference, err := decodeReferenceURL(wire.Reference)
	if err != nil {
		return task.CreateCommand{}, err
	}
	reward, err := decodeRewardSpec(wire.Reward)
	if err != nil {
		return task.CreateCommand{}, err
	}
	participation, err := decodeParticipationPolicy(wire.Participation)
	if err != nil {
		return task.CreateCommand{}, err
	}
	assigneeScope, err := decodeAssigneeScope(wire.AssigneeScope)
	if err != nil {
		return task.CreateCommand{}, err
	}
	reservationTTL, err := decodeReservationTTL(wire.ReservationTTL)
	if err != nil {
		return task.CreateCommand{}, err
	}
	visibility, err := decodeVisibility(wire.Visibility)
	if err != nil {
		return task.CreateCommand{}, err
	}
	placement, err := decodeSeriesPlacement(wire.Placement)
	if err != nil {
		return task.CreateCommand{}, err
	}
	responseSchema, err := decodeResponseSchema(wire.ResponseSchema)
	if err != nil {
		return task.CreateCommand{}, err
	}
	payload, err := decodeDataPayload(wire.Payload)
	if err != nil {
		return task.CreateCommand{}, err
	}
	attachments, err := attachmentwire.DecodeSlice(wire.Attachments)
	if err != nil {
		return task.CreateCommand{}, err
	}
	return task.CreateCommand{
		Actor:          auth.UserSubject{ID: actorID},
		Owner:          owner,
		Title:          title,
		Description:    description,
		Type:           taskType,
		Reference:      reference,
		Reward:         reward,
		Participation:  participation,
		AssigneeScope:  assigneeScope,
		ReservationTTL: reservationTTL,
		Visibility:     visibility,
		Placement:      placement,
		ResponseSchema: responseSchema,
		Payload:        payload,
		Attachments:    attachments,
	}, nil
}

// ---- task.ReservationCommand ----

type reservationCommandWire struct {
	TaskID      string       `json:"task_id"`
	Assignee    assigneeWire `json:"assignee"`
	RequestedBy string       `json:"requested_by"`
}

func encodeReservationCommand(command task.ReservationCommand) reservationCommandWire {
	return reservationCommandWire{
		TaskID:      corewire.EncodeTaskID(command.TaskID),
		Assignee:    encodeAssignee(command.Assignee),
		RequestedBy: corewire.EncodeUserID(command.RequestedBy),
	}
}

func decodeReservationCommand(wire reservationCommandWire) (task.ReservationCommand, error) {
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return task.ReservationCommand{}, err
	}
	assignee, err := decodeAssignee(wire.Assignee)
	if err != nil {
		return task.ReservationCommand{}, err
	}
	requestedBy, err := corewire.DecodeUserID(wire.RequestedBy)
	if err != nil {
		return task.ReservationCommand{}, err
	}
	return task.ReservationCommand{TaskID: taskID, Assignee: assignee, RequestedBy: requestedBy}, nil
}

// ---- task.ListScope union ----

type listScopeWire struct {
	Kind            string `json:"kind"`
	UserID          string `json:"user_id,omitempty"`
	OrganizationID  string `json:"organization_id,omitempty"`
	TeamID          string `json:"team_id,omitempty"`
	IncludeReserved bool   `json:"include_reserved,omitempty"`
}

func encodeListScope(scope task.ListScope) listScopeWire {
	switch typed := scope.(type) {
	case task.PublicListScope:
		return listScopeWire{Kind: "public", UserID: corewire.EncodeUserID(typed.ViewerID), IncludeReserved: typed.IncludeReserved}
	case task.UserListScope:
		return listScopeWire{Kind: "user", UserID: corewire.EncodeUserID(typed.UserID), IncludeReserved: typed.IncludeReserved}
	case task.OrganizationListScope:
		return listScopeWire{Kind: "organization", OrganizationID: corewire.EncodeOrganizationID(typed.OrganizationID), UserID: corewire.EncodeUserID(typed.UserID), IncludeReserved: typed.IncludeReserved}
	case task.TeamListScope:
		return listScopeWire{Kind: "team", TeamID: corewire.EncodeTeamID(typed.TeamID), IncludeReserved: typed.IncludeReserved}
	case task.CreatorListScope:
		return listScopeWire{Kind: "creator", UserID: corewire.EncodeUserID(typed.CreatorID)}
	case task.AssigneeListScope:
		return listScopeWire{Kind: "assignee", UserID: corewire.EncodeUserID(typed.AssigneeID)}
	default:
		return listScopeWire{Kind: "public"}
	}
}

func decodeListScope(wire listScopeWire) (task.ListScope, error) {
	switch wire.Kind {
	case "public":
		id, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return task.PublicListScope{ViewerID: id, IncludeReserved: wire.IncludeReserved}, nil
	case "user":
		id, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return task.UserListScope{UserID: id, IncludeReserved: wire.IncludeReserved}, nil
	case "organization":
		organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
		if err != nil {
			return nil, err
		}
		userID, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return task.OrganizationListScope{OrganizationID: organizationID, UserID: userID, IncludeReserved: wire.IncludeReserved}, nil
	case "team":
		id, err := corewire.DecodeTeamID(wire.TeamID)
		if err != nil {
			return nil, err
		}
		return task.TeamListScope{TeamID: id, IncludeReserved: wire.IncludeReserved}, nil
	case "creator":
		id, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return task.CreatorListScope{CreatorID: id}, nil
	case "assignee":
		id, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return task.AssigneeListScope{AssigneeID: id}, nil
	default:
		return nil, fmt.Errorf("unknown list scope kind %q", wire.Kind)
	}
}

// ---- task.ListFilters + its sub-filter unions ----

type listFiltersWire struct {
	State         stateFilterWire         `json:"state"`
	Participation participationFilterWire `json:"participation"`
	Search        searchFilterWire        `json:"search"`
	Type          typeFilterWire          `json:"type"`
	Sort          string                  `json:"sort"`
}

type stateFilterWire struct {
	Kind   string   `json:"kind"`
	Value  string   `json:"value,omitempty"`
	Values []string `json:"values,omitempty"`
}

type participationFilterWire struct {
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
}

type searchFilterWire struct {
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
}

type typeFilterWire struct {
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
}

func encodeListFilters(filters task.ListFilters) listFiltersWire {
	return listFiltersWire{
		State:         encodeStateFilter(filters.State),
		Participation: encodeParticipationFilter(filters.Participation),
		Search:        encodeSearchFilter(filters.Search),
		Type:          encodeTypeFilter(filters.Type),
		Sort:          encodeSortOrder(filters.Sort),
	}
}

func decodeListFilters(wire listFiltersWire) (task.ListFilters, error) {
	stateFilter, err := decodeStateFilter(wire.State)
	if err != nil {
		return task.ListFilters{}, err
	}
	participationFilter, err := decodeParticipationFilter(wire.Participation)
	if err != nil {
		return task.ListFilters{}, err
	}
	searchFilter, err := decodeSearchFilter(wire.Search)
	if err != nil {
		return task.ListFilters{}, err
	}
	typeFilter, err := decodeTypeFilter(wire.Type)
	if err != nil {
		return task.ListFilters{}, err
	}
	sort, err := decodeSortOrder(wire.Sort)
	if err != nil {
		return task.ListFilters{}, err
	}
	return task.ListFilters{State: stateFilter, Participation: participationFilter, Search: searchFilter, Type: typeFilter, Sort: sort}, nil
}

func encodeStateFilter(filter task.StateFilter) stateFilterWire {
	switch typed := filter.(type) {
	case task.StateEquals:
		return stateFilterWire{Kind: "equals", Value: encodeState(typed.Value)}
	case task.StateIn:
		values := make([]string, 0, len(typed.Values))
		for index := range typed.Values {
			values = append(values, encodeState(typed.Values[index]))
		}
		return stateFilterWire{Kind: "in", Values: values}
	default:
		return stateFilterWire{Kind: "unfiltered"}
	}
}

func decodeStateFilter(wire stateFilterWire) (task.StateFilter, error) {
	switch wire.Kind {
	case "unfiltered":
		return task.AnyStateFilter{}, nil
	case "equals":
		state, err := decodeState(wire.Value)
		if err != nil {
			return nil, err
		}
		return task.StateEquals{Value: state}, nil
	case "in":
		states := make([]task.State, 0, len(wire.Values))
		for index := range wire.Values {
			state, err := decodeState(wire.Values[index])
			if err != nil {
				return nil, err
			}
			states = append(states, state)
		}
		return task.StateIn{Values: states}, nil
	default:
		return nil, fmt.Errorf("unknown state filter kind %q", wire.Kind)
	}
}

func encodeParticipationFilter(filter task.ParticipationPolicyFilter) participationFilterWire {
	equals, matched := filter.(task.ParticipationPolicyEquals)
	if !matched {
		return participationFilterWire{Kind: "unfiltered"}
	}
	return participationFilterWire{Kind: "equals", Value: encodeParticipationPolicy(equals.Value)}
}

func decodeParticipationFilter(wire participationFilterWire) (task.ParticipationPolicyFilter, error) {
	switch wire.Kind {
	case "unfiltered":
		return task.AnyParticipationPolicyFilter{}, nil
	case "equals":
		policy, err := decodeParticipationPolicy(wire.Value)
		if err != nil {
			return nil, err
		}
		return task.ParticipationPolicyEquals{Value: policy}, nil
	default:
		return nil, fmt.Errorf("unknown participation filter kind %q", wire.Kind)
	}
}

func encodeSearchFilter(filter task.SearchFilter) searchFilterWire {
	contains, matched := filter.(task.SearchContains)
	if !matched {
		return searchFilterWire{Kind: "none"}
	}
	return searchFilterWire{Kind: "contains", Value: encodeSearchText(contains.Value)}
}

func decodeSearchFilter(wire searchFilterWire) (task.SearchFilter, error) {
	switch wire.Kind {
	case "none":
		return task.NoSearchFilter{}, nil
	case "contains":
		text, err := decodeSearchText(wire.Value)
		if err != nil {
			return nil, err
		}
		return task.SearchContains{Value: text}, nil
	default:
		return nil, fmt.Errorf("unknown search filter kind %q", wire.Kind)
	}
}

func encodeTypeFilter(filter task.TypeFilter) typeFilterWire {
	equals, matched := filter.(task.TypeEquals)
	if !matched {
		return typeFilterWire{Kind: "unfiltered"}
	}
	return typeFilterWire{Kind: "equals", Value: encodeTaskType(equals.Value)}
}

func decodeTypeFilter(wire typeFilterWire) (task.TypeFilter, error) {
	switch wire.Kind {
	case "unfiltered":
		return task.AnyTypeFilter{}, nil
	case "equals":
		taskType, err := decodeTaskType(wire.Value)
		if err != nil {
			return nil, err
		}
		return task.TypeEquals{Value: taskType}, nil
	default:
		return nil, fmt.Errorf("unknown type filter kind %q", wire.Kind)
	}
}
