package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/schema"
	"github.com/e6qu/sharecrop/internal/task"
)

func (server Server) createTask(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	requestResult := decodeTaskRequest(r, actor.subject)
	requestAccepted, requestMatched := requestResult.(taskRequestAccepted)
	if !requestMatched {
		rejected := requestResult.(taskRequestRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.taskService.Create(r.Context(), requestAccepted.command)
	created, matched := result.(task.TaskCreated)
	if !matched {
		rejected := result.(task.CreateRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeTaskResponse(w, http.StatusCreated, taskToResponse(created.Value))
}
func (server Server) listTasks(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	scopeResult := parseTaskListScope(r, actor.subject)
	scopeAccepted, scopeMatched := scopeResult.(taskListScopeAccepted)
	if !scopeMatched {
		rejected := scopeResult.(taskListScopeRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	filtersResult := parseTaskListFilters(r)
	filtersAccepted, filtersMatched := filtersResult.(taskListFiltersAccepted)
	if !filtersMatched {
		writeDomainError(w, filtersResult.(taskListFiltersRejected).reason)
		return
	}

	result := server.taskService.List(r.Context(), actor.subject, scopeAccepted.value, filtersAccepted.value, parsePage(r))
	switch listed := result.(type) {
	case task.ListRejected:
		writeDomainError(w, listed.Reason)
	case task.TasksListed:
		writeTasksResponse(w, http.StatusOK, tasksToResponse(listed.Values))
	}
}
func tasksToResponse(values []task.ListItem) tasksResponse {
	response := tasksResponse{Tasks: make([]taskListItemResponse, 0, len(values))}
	for valueIndex := range values {
		response.Tasks = append(response.Tasks, taskListItemToResponse(values[valueIndex]))
	}
	return response
}
func (server Server) openTask(w http.ResponseWriter, r *http.Request) {
	server.changeTaskState(w, r, server.taskService.Open)
}
func (server Server) cancelTask(w http.ResponseWriter, r *http.Request) {
	server.changeTaskState(w, r, server.taskService.Cancel)
}
func (server Server) reserveTask(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	taskIDResult := parseTaskPathValue(r)
	if rejected, matched := taskIDResult.(taskIDRejected); matched {
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}
	taskIDAccepted := taskIDResult.(taskIDAccepted)

	result := server.taskService.Reserve(r.Context(), actor.subject, taskIDAccepted.value)
	created, matched := result.(task.ReservationCreated)
	if !matched {
		writeDomainError(w, result.(task.ReservationRejected).Reason)
		return
	}
	writeJSON(w, http.StatusCreated, reservationToResponse(created.Value))
}
func (server Server) listTaskReservations(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}

	result := server.taskService.ListReservations(r.Context(), actor.subject, taskIDAccepted.value)
	listed, matched := result.(task.ReservationsListed)
	if !matched {
		writeDomainError(w, result.(task.ReservationsListRejected).Reason)
		return
	}
	response := reservationsResponse{Reservations: make([]reservationResponse, 0, len(listed.Values))}
	for _, value := range listed.Values {
		response.Reservations = append(response.Reservations, reservationToResponse(value))
	}
	writeJSON(w, http.StatusOK, response)
}
func (server Server) approveTaskReservation(w http.ResponseWriter, r *http.Request) {
	server.changeTaskReservation(w, r, server.taskService.ApproveReservation)
}
func (server Server) declineTaskReservation(w http.ResponseWriter, r *http.Request) {
	server.changeTaskReservation(w, r, server.taskService.DeclineReservation)
}
func (server Server) cancelTaskReservation(w http.ResponseWriter, r *http.Request) {
	server.changeTaskReservation(w, r, server.taskService.CancelReservation)
}
func (server Server) changeTaskReservation(w http.ResponseWriter, r *http.Request, changer taskReservationChanger) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}
	reservationIDResult := parseReservationPathValue(r)
	reservationIDAccepted, reservationIDMatched := reservationIDResult.(reservationIDAccepted)
	if !reservationIDMatched {
		writeError(w, http.StatusBadRequest, reservationIDResult.(reservationIDRejected).reason)
		return
	}

	result := changer(r.Context(), actor.subject, taskIDAccepted.value, reservationIDAccepted.value)
	changed, matched := result.(task.ReservationStateChanged)
	if !matched {
		writeDomainError(w, result.(task.ReservationStateChangeRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, reservationToResponse(changed.Value))
}
func (server Server) changeTaskState(w http.ResponseWriter, r *http.Request, changer taskStateChanger) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		rejected := taskIDResult.(taskIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := changer(r.Context(), actor.subject, taskIDAccepted.value)
	changed, matched := result.(task.TaskStateChanged)
	if !matched {
		rejected := result.(task.ChangeStateRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeTaskResponse(w, http.StatusOK, taskToResponse(changed.Value))
}
func (server Server) createTaskCapabilityToken(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		rejected := taskIDResult.(taskIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.taskService.CreateCapabilityToken(r.Context(), actor.subject, taskIDAccepted.value)
	created, matched := result.(task.CapabilityTokenCreated)
	if !matched {
		rejected := result.(task.CreateCapabilityTokenRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeTaskCapabilityTokenResponse(w, http.StatusCreated, taskCapabilityTokenResponse{
		ID:     created.Value.ID.String(),
		TaskID: created.Value.TaskID.String(),
		State:  created.Value.State.String(),
		Token:  created.Plain.String(),
	})
}
func decodeTaskRequest(r *http.Request, actor auth.UserSubject) taskRequestResult {
	var request taskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return taskRequestRejected{reason: "request body is invalid"}
	}

	ownerResult := parseTaskOwnerRequest(request.Owner)
	ownerAccepted, ownerMatched := ownerResult.(taskOwnerAccepted)
	if !ownerMatched {
		rejected := ownerResult.(taskOwnerRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	titleResult := task.NewTitle(request.Title)
	titleAccepted, titleMatched := titleResult.(task.TitleAccepted)
	if !titleMatched {
		rejected := titleResult.(task.TitleRejected)
		return taskRequestRejected{reason: rejected.Reason.Description()}
	}

	descriptionResult := task.NewDescription(request.Description)
	descriptionAccepted, descriptionMatched := descriptionResult.(task.DescriptionAccepted)
	if !descriptionMatched {
		rejected := descriptionResult.(task.DescriptionRejected)
		return taskRequestRejected{reason: rejected.Reason.Description()}
	}

	rewardResult := parseTaskRewardRequest(request.Reward)
	rewardAccepted, rewardMatched := rewardResult.(taskRewardAccepted)
	if !rewardMatched {
		rejected := rewardResult.(taskRewardRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	participationResult := parseTaskParticipationRequest(request.Participation)
	participationAccepted, participationMatched := participationResult.(taskParticipationAccepted)
	if !participationMatched {
		rejected := participationResult.(taskParticipationRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	visibilityResult := parseTaskVisibilityRequest(request.Visibility, ownerAccepted.value)
	visibilityAccepted, visibilityMatched := visibilityResult.(taskVisibilityAccepted)
	if !visibilityMatched {
		rejected := visibilityResult.(taskVisibilityRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	placementResult := parseTaskPlacementRequest(request.Placement)
	placementAccepted, placementMatched := placementResult.(taskPlacementAccepted)
	if !placementMatched {
		rejected := placementResult.(taskPlacementRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	schemaResult := schema.ParseSchemaJSON([]byte(request.ResponseSchemaJSON))
	if _, schemaMatched := schemaResult.(schema.SchemaParsed); !schemaMatched {
		rejected := schemaResult.(schema.SchemaParseRejected)
		return taskRequestRejected{reason: rejected.Reason.Description()}
	}

	schemaSourceResult := task.NewResponseSchemaSource(request.ResponseSchemaJSON)
	schemaSourceAccepted, schemaSourceMatched := schemaSourceResult.(task.ResponseSchemaSourceAccepted)
	if !schemaSourceMatched {
		rejected := schemaSourceResult.(task.ResponseSchemaSourceRejected)
		return taskRequestRejected{reason: rejected.Reason.Description()}
	}

	payloadResult := parseTaskPayloadRequest(request.Payload)
	payloadAccepted, payloadMatched := payloadResult.(taskPayloadAccepted)
	if !payloadMatched {
		rejected := payloadResult.(taskPayloadRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	return taskRequestAccepted{command: task.CreateCommand{
		Actor:          actor,
		Owner:          ownerAccepted.value,
		Title:          titleAccepted.Value,
		Description:    descriptionAccepted.Value,
		Reward:         rewardAccepted.value,
		Participation:  participationAccepted.policy,
		AssigneeScope:  participationAccepted.assigneeScope,
		ReservationTTL: participationAccepted.ttl,
		Visibility:     visibilityAccepted.value,
		Placement:      placementAccepted.value,
		ResponseSchema: schemaSourceAccepted.Value,
		Payload:        payloadAccepted.value,
	}}
}
func parseTaskParticipationRequest(request taskParticipationRequest) taskParticipationResult {
	rawPolicy := request.Policy
	if rawPolicy == "" {
		rawPolicy = task.ParticipationPolicyOpen.String()
	}
	policyResult := task.ParseParticipationPolicy(rawPolicy)
	policyAccepted, policyMatched := policyResult.(task.ParticipationPolicyAccepted)
	if !policyMatched {
		rejected := policyResult.(task.ParticipationPolicyRejected)
		return taskParticipationRejected{reason: rejected.Reason.Description()}
	}

	rawAssigneeScope := request.AssigneeScope
	if rawAssigneeScope == "" {
		rawAssigneeScope = task.AssigneeScopeUser.String()
	}
	assigneeScopeResult := task.ParseAssigneeScope(rawAssigneeScope)
	assigneeScopeAccepted, assigneeScopeMatched := assigneeScopeResult.(task.AssigneeScopeAccepted)
	if !assigneeScopeMatched {
		rejected := assigneeScopeResult.(task.AssigneeScopeRejected)
		return taskParticipationRejected{reason: rejected.Reason.Description()}
	}

	ttl := task.DefaultReservationTTL()
	if request.ReservationExpiryHours != 0 {
		ttlResult := task.NewReservationTTL(request.ReservationExpiryHours)
		ttlAccepted, ttlMatched := ttlResult.(task.ReservationTTLAccepted)
		if !ttlMatched {
			rejected := ttlResult.(task.ReservationTTLRejected)
			return taskParticipationRejected{reason: rejected.Reason.Description()}
		}
		ttl = ttlAccepted.Value
	}

	return taskParticipationAccepted{policy: policyAccepted.Value, assigneeScope: assigneeScopeAccepted.Value, ttl: ttl}
}
func parseTaskRewardRequest(request taskRewardRequest) taskRewardResult {
	switch request.Kind {
	case task.RewardKindNone.String():
		return taskRewardAccepted{value: task.NoRewardSpec{}}
	case task.RewardKindCredit.String():
		amountResult := task.NewCreditRewardAmount(request.CreditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			rejected := amountResult.(task.CreditRewardAmountRejected)
			return taskRewardRejected{reason: rejected.Reason.Description()}
		}
		return taskRewardAccepted{value: task.CreditRewardSpec{Amount: amount.Value}}
	case task.RewardKindCollectible.String():
		countResult := task.NewCollectibleRewardCount(1)
		count := countResult.(task.CollectibleRewardCountAccepted)
		return taskRewardAccepted{value: task.CollectibleRewardSpec{Count: count.Value}}
	case task.RewardKindBundle.String():
		amountResult := task.NewCreditRewardAmount(request.CreditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			rejected := amountResult.(task.CreditRewardAmountRejected)
			return taskRewardRejected{reason: rejected.Reason.Description()}
		}
		countResult := task.NewCollectibleRewardCount(1)
		count := countResult.(task.CollectibleRewardCountAccepted)
		return taskRewardAccepted{value: task.BundleRewardSpec{Credit: amount.Value, Collectible: count.Value}}
	default:
		return taskRewardRejected{reason: "task reward kind is invalid"}
	}
}
func parseTaskOwnerRequest(request taskOwnerRequest) taskOwnerResult {
	switch request.Kind {
	case task.OwnerKindUser.String():
		userIDResult := core.ParseUserID(request.UserID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			rejected := userIDResult.(core.UserIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		return taskOwnerAccepted{value: task.UserOwner{UserID: userID.Value}}
	case task.OwnerKindTeam.String():
		teamIDResult := core.ParseTeamID(request.TeamID)
		teamID, matched := teamIDResult.(core.TeamIDCreated)
		if !matched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		return taskOwnerAccepted{value: task.TeamOwner{TeamID: teamID.Value}}
	case task.OwnerKindOrganization.String():
		organizationIDResult := core.ParseOrganizationID(request.OrganizationID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		return taskOwnerAccepted{value: task.OrganizationOwner{OrganizationID: organizationID.Value}}
	case task.OwnerKindOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(request.OrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		teamIDResult := core.ParseTeamID(request.TeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		return taskOwnerAccepted{value: task.OrganizationTeamOwner{OrganizationID: organizationID.Value, TeamID: teamID.Value}}
	default:
		return taskOwnerRejected{reason: "task owner kind is invalid"}
	}
}
func parseTaskVisibilityRequest(request taskVisibilityRequest, owner task.Owner) taskVisibilityResult {
	if request.Kind == "default" {
		return defaultVisibilityForOwner(owner)
	}
	switch request.Kind {
	case task.VisibilityKindPublic.String():
		return taskVisibilityAccepted{value: task.PublicVisibility{}}
	case task.VisibilityKindUser.String():
		userIDResult := core.ParseUserID(request.UserID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			rejected := userIDResult.(core.UserIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		return taskVisibilityAccepted{value: task.UserVisibility{UserID: userID.Value}}
	case task.VisibilityKindTeam.String():
		teamIDResult := core.ParseTeamID(request.TeamID)
		teamID, matched := teamIDResult.(core.TeamIDCreated)
		if !matched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		return taskVisibilityAccepted{value: task.TeamVisibility{TeamID: teamID.Value}}
	case task.VisibilityKindOrganization.String():
		organizationIDResult := core.ParseOrganizationID(request.OrganizationID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		return taskVisibilityAccepted{value: task.OrganizationVisibility{OrganizationID: organizationID.Value}}
	case task.VisibilityKindOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(request.OrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		teamIDResult := core.ParseTeamID(request.TeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		return taskVisibilityAccepted{value: task.OrganizationTeamVisibility{OrganizationID: organizationID.Value, TeamID: teamID.Value}}
	default:
		return taskVisibilityRejected{reason: "task visibility kind is invalid"}
	}
}
func defaultVisibilityForOwner(owner task.Owner) taskVisibilityResult {
	switch typed := owner.(type) {
	case task.UserOwner:
		return taskVisibilityAccepted{value: task.UserVisibility{UserID: typed.UserID}}
	case task.TeamOwner:
		return taskVisibilityAccepted{value: task.TeamVisibility{TeamID: typed.TeamID}}
	case task.OrganizationOwner:
		return taskVisibilityAccepted{value: task.OrganizationVisibility{OrganizationID: typed.OrganizationID}}
	case task.OrganizationTeamOwner:
		return taskVisibilityAccepted{value: task.OrganizationTeamVisibility{OrganizationID: typed.OrganizationID, TeamID: typed.TeamID}}
	default:
		return taskVisibilityRejected{reason: "task owner is invalid"}
	}
}
func parseTaskPlacementRequest(request taskPlacementRequest) taskPlacementResult {
	switch request.Kind {
	case "standalone":
		return taskPlacementAccepted{value: task.StandalonePlacement{}}
	case "new_series":
		titleResult := task.NewSeriesTitle(request.SeriesTitle)
		title, titleMatched := titleResult.(task.SeriesTitleAccepted)
		if !titleMatched {
			rejected := titleResult.(task.SeriesTitleRejected)
			return taskPlacementRejected{reason: rejected.Reason.Description()}
		}
		positionResult := task.NewSeriesPosition(request.SeriesPosition)
		position, positionMatched := positionResult.(task.SeriesPositionAccepted)
		if !positionMatched {
			rejected := positionResult.(task.SeriesPositionRejected)
			return taskPlacementRejected{reason: rejected.Reason.Description()}
		}
		return taskPlacementAccepted{value: task.NewSeriesPlacement{Title: title.Value, Position: position.Value}}
	case "existing_series":
		seriesIDResult := core.ParseTaskSeriesID(request.SeriesID)
		seriesID, seriesMatched := seriesIDResult.(core.TaskSeriesIDCreated)
		if !seriesMatched {
			rejected := seriesIDResult.(core.TaskSeriesIDRejected)
			return taskPlacementRejected{reason: rejected.Reason.Description()}
		}
		positionResult := task.NewSeriesPosition(request.SeriesPosition)
		position, positionMatched := positionResult.(task.SeriesPositionAccepted)
		if !positionMatched {
			rejected := positionResult.(task.SeriesPositionRejected)
			return taskPlacementRejected{reason: rejected.Reason.Description()}
		}
		return taskPlacementAccepted{value: task.ExistingSeriesPlacement{SeriesID: seriesID.Value, Position: position.Value}}
	default:
		return taskPlacementRejected{reason: "task series placement kind is invalid"}
	}
}
func parseTaskPayloadRequest(request taskPayloadRequest) taskPayloadResult {
	switch request.Kind {
	case "none":
		return taskPayloadAccepted{value: task.NoDataPayload{}}
	case "json":
		if !json.Valid([]byte(request.JSON)) {
			return taskPayloadRejected{reason: "task payload JSON is invalid"}
		}
		sourceResult := task.NewPayloadSource(request.JSON)
		source, matched := sourceResult.(task.PayloadSourceAccepted)
		if !matched {
			rejected := sourceResult.(task.PayloadSourceRejected)
			return taskPayloadRejected{reason: rejected.Reason.Description()}
		}
		return taskPayloadAccepted{value: task.JSONDataPayload{Source: source.Value}}
	default:
		return taskPayloadRejected{reason: "task payload kind is invalid"}
	}
}
func parseTaskPathValue(r *http.Request) taskIDResult {
	result := core.ParseTaskID(r.PathValue("task_id"))
	accepted, matched := result.(core.TaskIDCreated)
	if !matched {
		rejected := result.(core.TaskIDRejected)
		return taskIDRejected{reason: rejected.Reason.Description()}
	}
	return taskIDAccepted{value: accepted.Value}
}
func parseReservationPathValue(r *http.Request) reservationIDResult {
	result := core.ParseTaskReservationID(r.PathValue("reservation_id"))
	accepted, matched := result.(core.TaskReservationIDCreated)
	if !matched {
		rejected := result.(core.TaskReservationIDRejected)
		return reservationIDRejected{reason: rejected.Reason.Description()}
	}
	return reservationIDAccepted{value: accepted.Value}
}
func parseTaskListScope(r *http.Request, actor auth.UserSubject) taskListScopeResult {
	scope := r.URL.Query().Get("scope")
	includeReserved := r.URL.Query().Get("include_reserved") == "true"
	switch scope {
	case "public":
		return taskListScopeAccepted{value: task.PublicListScope{ViewerID: actor.ID, IncludeReserved: includeReserved}}
	case "user":
		return taskListScopeAccepted{value: task.UserListScope{UserID: actor.ID, IncludeReserved: includeReserved}}
	case "organization":
		organizationIDResult := core.ParseOrganizationID(r.URL.Query().Get("organization_id"))
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskListScopeRejected{reason: rejected.Reason.Description()}
		}
		return taskListScopeAccepted{value: task.OrganizationListScope{OrganizationID: organizationID.Value, UserID: actor.ID, IncludeReserved: includeReserved}}
	default:
		return taskListScopeRejected{reason: "task list scope is invalid"}
	}
}
func parseTaskListFilters(r *http.Request) taskListFiltersResult {
	query := r.URL.Query()
	filters := task.NoListFilters()

	if rawState := query.Get("state"); rawState != "" {
		stateResult := task.ParseState(rawState)
		stateAccepted, matched := stateResult.(task.StateAccepted)
		if !matched {
			return taskListFiltersRejected{reason: stateResult.(task.StateRejected).Reason}
		}
		filters.State = task.StateEquals{Value: stateAccepted.Value}
	}

	if rawPolicy := query.Get("participation_policy"); rawPolicy != "" {
		policyResult := task.ParseParticipationPolicy(rawPolicy)
		policyAccepted, matched := policyResult.(task.ParticipationPolicyAccepted)
		if !matched {
			return taskListFiltersRejected{reason: policyResult.(task.ParticipationPolicyRejected).Reason}
		}
		filters.Participation = task.ParticipationPolicyEquals{Value: policyAccepted.Value}
	}

	return taskListFiltersAccepted{value: filters}
}
func taskListItemToResponse(item task.ListItem) taskListItemResponse {
	value := item.Task
	owner := taskOwnerResponseParts(value.Owner)
	visibility := taskVisibilityResponseParts(value.Visibility)
	reward := taskRewardResponseParts(value.Reward)
	active := activeAssigneeResponseParts(item.ActiveAssignee)
	return taskListItemResponse{
		ID:                     value.ID.String(),
		OwnerKind:              owner.kind,
		Title:                  value.Title.String(),
		RewardKind:             reward.kind,
		RewardCreditAmount:     reward.amount,
		RewardCollectibleCount: reward.collectibleCount,
		ParticipationPolicy:    value.Participation.String(),
		AssigneeScope:          value.AssigneeScope.String(),
		ReservationExpiryHours: value.ReservationTTL.Hours(),
		State:                  value.State.String(),
		VisibilityKind:         visibility.kind,
		AvailabilityKind:       taskAvailabilityKind(value).String(),
		ViewerAction:           taskViewerAction(value).String(),
		CreatedBy:              value.CreatedBy.String(),
		ActiveAssigneeKind:     active.kind,
		ActiveAssigneeID:       active.id,
	}
}
func activeAssigneeResponseParts(active task.ActiveAssignee) activeAssigneeParts {
	switch typed := active.(type) {
	case task.ActiveUserAssignee:
		return activeAssigneeParts{kind: task.AssigneeScopeUser.String(), id: typed.UserID.String()}
	case task.ActiveOrganizationTeamAssignee:
		return activeAssigneeParts{kind: task.AssigneeScopeOrganizationTeam.String(), id: typed.TeamID.String()}
	default:
		return activeAssigneeParts{kind: "", id: ""}
	}
}
func taskToResponse(value task.Task) taskResponse {
	owner := taskOwnerResponseParts(value.Owner)
	visibility := taskVisibilityResponseParts(value.Visibility)
	placement := taskPlacementResponseParts(value.Placement)
	payload := taskPayloadResponseParts(value.Payload)
	reward := taskRewardResponseParts(value.Reward)
	return taskResponse{
		ID:                     value.ID.String(),
		OwnerKind:              owner.kind,
		OwnerID:                owner.id,
		Title:                  value.Title.String(),
		Description:            value.Description.String(),
		RewardKind:             reward.kind,
		RewardCreditAmount:     reward.amount,
		RewardCollectibleCount: reward.collectibleCount,
		ParticipationPolicy:    value.Participation.String(),
		AssigneeScope:          value.AssigneeScope.String(),
		ReservationExpiryHours: value.ReservationTTL.Hours(),
		State:                  value.State.String(),
		VisibilityKind:         visibility.kind,
		VisibilityID:           visibility.id,
		SeriesKind:             placement.kind,
		SeriesID:               placement.id,
		SeriesPosition:         placement.position,
		ResponseSchemaJSON:     value.ResponseSchema.String(),
		PayloadKind:            payload.kind,
		PayloadJSON:            payload.source,
		CreatedBy:              value.CreatedBy.String(),
		AvailabilityKind:       taskAvailabilityKind(value).String(),
		ViewerAction:           taskViewerAction(value).String(),
	}
}
func taskAvailabilityKind(value task.Task) task.AvailabilityKind {
	if value.State != task.StateOpen {
		return task.AvailabilityClosed
	}
	if value.Participation == task.ParticipationPolicyApprovalRequired {
		return task.AvailabilityAwaitingApproval
	}
	return task.AvailabilityAvailable
}
func taskViewerAction(value task.Task) task.ViewerAction {
	if value.State != task.StateOpen {
		return task.ViewerActionNone
	}
	switch value.Participation {
	case task.ParticipationPolicyOpen:
		return task.ViewerActionSubmit
	case task.ParticipationPolicyReservationRequired:
		return task.ViewerActionReserve
	case task.ParticipationPolicyApprovalRequired:
		return task.ViewerActionRequestApproval
	default:
		return task.ViewerActionNone
	}
}
func reservationToResponse(value task.Reservation) reservationResponse {
	assignee := reservationAssigneeResponseParts(value.Assignee)
	return reservationResponse{
		ID:           value.ID.String(),
		TaskID:       value.TaskID.String(),
		AssigneeKind: assignee.kind,
		AssigneeID:   assignee.id,
		State:        value.State.String(),
		RequestedBy:  value.RequestedBy.String(),
	}
}
func reservationAssigneeResponseParts(assignee task.Assignee) responseParts {
	switch typed := assignee.(type) {
	case task.UserAssignee:
		return responseParts{kind: task.AssigneeScopeUser.String(), id: typed.UserID.String()}
	case task.OrganizationTeamAssignee:
		return responseParts{kind: task.AssigneeScopeOrganizationTeam.String(), id: typed.TeamID.String()}
	default:
		return responseParts{}
	}
}
func taskRewardResponseParts(reward task.RewardSpec) rewardResponseParts {
	switch typed := reward.(type) {
	case task.NoRewardSpec:
		return rewardResponseParts{kind: task.RewardKindNone.String()}
	case task.CreditRewardSpec:
		return rewardResponseParts{kind: task.RewardKindCredit.String(), amount: typed.Amount.Int64()}
	case task.CollectibleRewardSpec:
		return rewardResponseParts{kind: task.RewardKindCollectible.String(), collectibleCount: typed.Count.Int()}
	case task.BundleRewardSpec:
		return rewardResponseParts{kind: task.RewardKindBundle.String(), amount: typed.Credit.Int64(), collectibleCount: typed.Collectible.Int()}
	default:
		return rewardResponseParts{}
	}
}
func taskOwnerResponseParts(owner task.Owner) responseParts {
	switch typed := owner.(type) {
	case task.UserOwner:
		return responseParts{kind: task.OwnerKindUser.String(), id: typed.UserID.String()}
	case task.TeamOwner:
		return responseParts{kind: task.OwnerKindTeam.String(), id: typed.TeamID.String()}
	case task.OrganizationOwner:
		return responseParts{kind: task.OwnerKindOrganization.String(), id: typed.OrganizationID.String()}
	case task.OrganizationTeamOwner:
		return responseParts{kind: task.OwnerKindOrganizationTeam.String(), id: typed.OrganizationID.String() + ":" + typed.TeamID.String()}
	default:
		return responseParts{}
	}
}
func taskVisibilityResponseParts(visibility task.Visibility) responseParts {
	switch typed := visibility.(type) {
	case task.PublicVisibility:
		return responseParts{kind: task.VisibilityKindPublic.String()}
	case task.UserVisibility:
		return responseParts{kind: task.VisibilityKindUser.String(), id: typed.UserID.String()}
	case task.TeamVisibility:
		return responseParts{kind: task.VisibilityKindTeam.String(), id: typed.TeamID.String()}
	case task.OrganizationVisibility:
		return responseParts{kind: task.VisibilityKindOrganization.String(), id: typed.OrganizationID.String()}
	case task.OrganizationTeamVisibility:
		return responseParts{kind: task.VisibilityKindOrganizationTeam.String(), id: typed.OrganizationID.String() + ":" + typed.TeamID.String()}
	default:
		return responseParts{}
	}
}
func taskPlacementResponseParts(placement task.SeriesPlacement) responseParts {
	switch typed := placement.(type) {
	case task.StandalonePlacement:
		return responseParts{kind: "standalone"}
	case task.NewSeriesPlacement:
		return responseParts{kind: "new_series", position: typed.Position.Int()}
	case task.ExistingSeriesPlacement:
		return responseParts{kind: "existing_series", id: typed.SeriesID.String(), position: typed.Position.Int()}
	default:
		return responseParts{}
	}
}
func taskPayloadResponseParts(payload task.DataPayload) responseParts {
	switch typed := payload.(type) {
	case task.NoDataPayload:
		return responseParts{kind: "none"}
	case task.JSONDataPayload:
		return responseParts{kind: "json", source: typed.Source.String()}
	default:
		return responseParts{}
	}
}
