package wasmdemo

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

// TaskBrowserStore implements task.Store against BrowserStorage. It covers
// the core task lifecycle (create/find/state transitions), core listing
// scopes (public/user/creator/assignee), and reservations -
// CheckSubmissionEligibility included. It shares storedTaskRecord with
// LedgerBrowserStore (browserstore_task_shared.go / browserstore_ledger.go)
// since the real Postgres stores for both domains read/write the same
// `tasks` table.
//
// Task series (ListSeries/FindSeries/CreateSeries/UpdateSeries/
// UpdateSeriesState/AddTaskToSeries/RemoveTaskFromSeries/ReorderSeries) and
// task/series comments (CreateSeriesComment/ListSeriesComments/
// CreateTaskComment/ListTaskComments) are not yet implemented - they
// return a clear "not yet implemented" rejection rather than silently
// doing the wrong thing.
type TaskBrowserStore struct {
	storage BrowserStorage
	ids     InteractionIDSource
}

func NewTaskBrowserStore(storage BrowserStorage, ids InteractionIDSource) TaskBrowserStore {
	return TaskBrowserStore{storage: storage, ids: ids}
}

func (store TaskBrowserStore) CreateTask(_ context.Context, _ core.TaskSeriesID, taskID core.TaskID, command task.CreateCommand) task.CreateTaskStoreResult {
	ownerKind, ownerUserID, ownerTeamID, ownerOrganizationID := ownerSQLColumnsBrowser(command.Owner)
	visibilityKind, visibilityUserID, visibilityTeamID, visibilityOrgID := visibilitySQLColumnsBrowser(command.Visibility)
	rewardKind, rewardCreditAmount, rewardCollectibleCount := rewardSQLColumnsBrowser(command.Reward)
	payloadKind, payloadSource := payloadSQLColumnsBrowser(command.Payload)

	record := storedTaskRecord{
		ID: taskID.String(), OwnerKind: ownerKind, OwnerUserID: ownerUserID, OwnerTeamID: ownerTeamID, OwnerOrganizationID: ownerOrganizationID,
		Title: command.Title.String(), Description: command.Description.String(), TaskType: command.Type.String(), ReferenceURL: command.Reference.String(),
		RewardKind: rewardKind, RewardCreditAmount: rewardCreditAmount, RewardCollectibleCount: rewardCollectibleCount,
		Participation: command.Participation.String(), AssigneeScope: command.AssigneeScope.String(), ReservationTTLHours: command.ReservationTTL.Hours(),
		State: task.StateDraft.String(), VisibilityKind: visibilityKind, VisibilityUserID: visibilityUserID, VisibilityTeamID: visibilityTeamID, VisibilityOrgID: visibilityOrgID,
		ResponseSchemaJSON: command.ResponseSchema.String(), PayloadKind: payloadKind, PayloadJSON: payloadSource,
		CreatedBy: command.Actor.ID.String(),
	}
	if !saveStoredTaskRecord(store.storage, record) {
		return task.CreateTaskStoreRejected{Reason: invalidState("insert task failed")}
	}
	if _, matched := appendStringIndex(store.storage, taskUserIndexKey(record.CreatedBy), record.ID, "task").(stringIndexStored); !matched {
		return task.CreateTaskStoreRejected{Reason: invalidState("update task index failed")}
	}
	if visibilityKind == task.VisibilityKindPublic.String() {
		if _, matched := appendStringIndex(store.storage, taskPublicIndexKey(), record.ID, "task").(stringIndexStored); !matched {
			return task.CreateTaskStoreRejected{Reason: invalidState("update public task index failed")}
		}
	}

	value, parseErr := parseStoredTaskRecord(record)
	if parseErr != nil {
		return task.CreateTaskStoreRejected{Reason: *parseErr}
	}
	return task.CreateTaskStoreAccepted{Value: value}
}

func (store TaskBrowserStore) FindTask(_ context.Context, taskID core.TaskID) task.FindTaskStoreResult {
	record, found, err := loadStoredTaskRecord(store.storage, taskID.String())
	if err != nil {
		return task.FindTaskStoreRejected{Reason: *err}
	}
	if !found {
		return task.FindTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task was not found")}
	}
	value, parseErr := parseStoredTaskRecord(record)
	if parseErr != nil {
		return task.FindTaskStoreRejected{Reason: *parseErr}
	}
	return task.FindTaskStoreAccepted{Value: value}
}

// requireOpenableRewardBrowser mirrors internal/db's requireOpenableReward:
// a credit/bundle reward needs a held escrow matching the declared amount;
// a collectible/bundle reward needs at least one held collectible reward
// (not yet implemented - see AssetBrowserStore - so this always rejects for
// a collectible-bearing task, matching "not funded yet" rather than
// silently allowing it).
func (store TaskBrowserStore) requireOpenableReward(record storedTaskRecord) *core.DomainError {
	if record.RewardKind == "credit" || record.RewardKind == "bundle" {
		escrow, found, err := (LedgerBrowserStore{storage: store.storage}).loadEscrow(record.ID)
		if err != nil {
			return err
		}
		if !found || escrow.State != "held" || escrow.Amount != record.RewardCreditAmount {
			reason := core.NewDomainError(core.ErrorCodeConflict, "credit reward must be funded before opening")
			return &reason
		}
	}
	if record.RewardKind == "collectible" || record.RewardKind == "bundle" {
		reason := core.NewDomainError(core.ErrorCodeConflict, "collectible reward must be funded before opening")
		return &reason
	}
	return nil
}

func (store TaskBrowserStore) requireNoHeldEscrow(record storedTaskRecord) *core.DomainError {
	escrow, found, err := (LedgerBrowserStore{storage: store.storage}).loadEscrow(record.ID)
	if err != nil {
		return err
	}
	if found && escrow.State == "held" {
		reason := core.NewDomainError(core.ErrorCodeConflict, "refund the task's held escrow before cancelling")
		return &reason
	}
	return nil
}

func (store TaskBrowserStore) ChangeTaskState(_ context.Context, taskID core.TaskID, state task.State) task.ChangeTaskStateStoreResult {
	record, found, err := loadStoredTaskRecord(store.storage, taskID.String())
	if err != nil {
		return task.ChangeTaskStateStoreRejected{Reason: *err}
	}
	if !found {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task was not found")}
	}
	if state == task.StateOpen {
		if rejectReason := store.requireOpenableReward(record); rejectReason != nil {
			return task.ChangeTaskStateStoreRejected{Reason: *rejectReason}
		}
	}
	if state == task.StateCancelled {
		if rejectReason := store.requireNoHeldEscrow(record); rejectReason != nil {
			return task.ChangeTaskStateStoreRejected{Reason: *rejectReason}
		}
	}
	record.State = state.String()
	if !saveStoredTaskRecord(store.storage, record) {
		return task.ChangeTaskStateStoreRejected{Reason: invalidState("change task state failed")}
	}
	value, parseErr := parseStoredTaskRecord(record)
	if parseErr != nil {
		return task.ChangeTaskStateStoreRejected{Reason: *parseErr}
	}
	return task.ChangeTaskStateStoreAccepted{Value: value}
}

// activeAssigneeForTask returns the ActiveAssignee for a task's currently
// active reservation, or NoActiveAssignee if none.
func (store TaskBrowserStore) activeAssigneeForTask(taskID string) (task.ActiveAssignee, *core.DomainError) {
	reservations, err := store.loadReservations(taskID)
	if err != nil {
		return nil, err
	}
	for _, reservation := range reservations {
		if reservation.State != task.ReservationStateActive {
			continue
		}
		switch assignee := reservation.Assignee.(type) {
		case task.UserAssignee:
			return task.ActiveUserAssignee{UserID: assignee.UserID}, nil
		case task.OrganizationTeamAssignee:
			return task.ActiveOrganizationTeamAssignee{OrganizationID: assignee.OrganizationID, TeamID: assignee.TeamID}, nil
		case task.TeamAssignee:
			return task.ActiveTeamAssignee{TeamID: assignee.TeamID}, nil
		}
	}
	return task.NoActiveAssignee{}, nil
}

func (store TaskBrowserStore) ListTasks(_ context.Context, scope task.ListScope, filters task.ListFilters, page core.Page) task.ListTasksStoreResult {
	var candidateIDs []string
	switch typed := scope.(type) {
	case task.PublicListScope:
		indexResult := loadStringIndex(store.storage, taskPublicIndexKey(), "task")
		loaded, matched := indexResult.(stringIndexLoaded)
		if !matched {
			return task.ListTasksStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
		}
		candidateIDs = loaded.values
	case task.UserListScope:
		indexResult := loadStringIndex(store.storage, taskUserIndexKey(typed.UserID.String()), "task")
		loaded, matched := indexResult.(stringIndexLoaded)
		if !matched {
			return task.ListTasksStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
		}
		candidateIDs = loaded.values
	case task.CreatorListScope:
		indexResult := loadStringIndex(store.storage, taskUserIndexKey(typed.CreatorID.String()), "task")
		loaded, matched := indexResult.(stringIndexLoaded)
		if !matched {
			return task.ListTasksStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
		}
		candidateIDs = loaded.values
	case task.AssigneeListScope:
		indexResult := loadStringIndex(store.storage, taskPublicIndexKey(), "task")
		loaded, matched := indexResult.(stringIndexLoaded)
		if !matched {
			return task.ListTasksStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
		}
		candidateIDs = loaded.values
	default:
		// OrganizationListScope/TeamListScope are not yet implemented.
		return task.ListTasksStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "this task list scope is not yet implemented in the browser demo")}
	}

	values := make([]task.ListItem, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		record, found, err := loadStoredTaskRecord(store.storage, id)
		if err != nil {
			return task.ListTasksStoreRejected{Reason: *err}
		}
		if !found {
			continue
		}
		if typed, isCreator := scope.(task.CreatorListScope); isCreator && (record.VisibilityKind != task.VisibilityKindPublic.String() || record.CreatedBy != typed.CreatorID.String()) {
			continue
		}
		if !passesTaskFilters(record, filters) {
			continue
		}
		value, parseErr := parseStoredTaskRecord(record)
		if parseErr != nil {
			return task.ListTasksStoreRejected{Reason: *parseErr}
		}
		activeAssignee, assigneeErr := store.activeAssigneeForTask(id)
		if assigneeErr != nil {
			return task.ListTasksStoreRejected{Reason: *assigneeErr}
		}
		if typed, isAssignee := scope.(task.AssigneeListScope); isAssignee {
			userAssignee, isUserAssignee := activeAssignee.(task.ActiveUserAssignee)
			if !isUserAssignee || userAssignee.UserID != typed.AssigneeID {
				continue
			}
		}
		values = append(values, task.ListItem{Task: value, ActiveAssignee: activeAssignee})
	}

	sortTaskListItems(values, filters.Sort)

	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	return task.ListTasksStoreAccepted{Values: values[start:end]}
}

func passesTaskFilters(record storedTaskRecord, filters task.ListFilters) bool {
	switch typed := filters.State.(type) {
	case task.StateEquals:
		if record.State != typed.Value.String() {
			return false
		}
	case task.StateIn:
		matched := false
		for _, value := range typed.Values {
			if record.State == value.String() {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if typed, matched := filters.Participation.(task.ParticipationPolicyEquals); matched && record.Participation != typed.Value.String() {
		return false
	}
	if typed, matched := filters.Type.(task.TypeEquals); matched && record.TaskType != typed.Value.String() {
		return false
	}
	if typed, matched := filters.Search.(task.SearchContains); matched {
		query := strings.ToLower(typed.Value.String())
		if !strings.Contains(strings.ToLower(record.Title), query) && !strings.Contains(strings.ToLower(record.ID), query) {
			return false
		}
	}
	return true
}

func sortTaskListItems(values []task.ListItem, order task.SortOrder) {
	switch order {
	case task.SortOldest:
		// Insertion order (index append order) is already oldest-first.
	case task.SortTitleAsc:
		sort.SliceStable(values, func(i, j int) bool { return values[i].Task.Title.String() < values[j].Task.Title.String() })
	case task.SortTitleDesc:
		sort.SliceStable(values, func(i, j int) bool { return values[i].Task.Title.String() > values[j].Task.Title.String() })
	case task.SortRewardDesc:
		sort.SliceStable(values, func(i, j int) bool { return rewardSortAmount(values[i].Task) > rewardSortAmount(values[j].Task) })
	case task.SortRewardAsc:
		sort.SliceStable(values, func(i, j int) bool { return rewardSortAmount(values[i].Task) < rewardSortAmount(values[j].Task) })
	default:
		// SortNewest (the default): reverse insertion order.
		reverseTaskListItems(values)
	}
}

func rewardSortAmount(value task.Task) int64 {
	switch typed := value.Reward.(type) {
	case task.CreditRewardSpec:
		return typed.Amount.Int64()
	case task.BundleRewardSpec:
		return typed.Credit.Int64()
	default:
		return 0
	}
}

func reverseTaskListItems(values []task.ListItem) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}

type storedReservation struct {
	ID              string `json:"id"`
	TaskID          string `json:"task_id"`
	AssigneeKind    string `json:"assignee_kind"`
	AssigneeUserID  string `json:"assignee_user_id,omitempty"`
	AssigneeTeamID  string `json:"assignee_team_id,omitempty"`
	AssigneeOrgID   string `json:"assignee_organization_id,omitempty"`
	State           string `json:"state"`
	RequestedByUser string `json:"requested_by_user_id"`
}

func reservationRecordKey(id string) string { return "task:reservation:" + id }
func reservationTaskIndexKey(taskID string) string {
	return "task:reservation_index:" + taskID
}

func putStoredReservationJSON(storage BrowserStorage, rawKey string, record storedReservation) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredReservationJSON(storage BrowserStorage, rawKey string) (storedReservation, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedReservation{}, found, ok
	}
	var record storedReservation
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedReservation{}, false, false
	}
	return record, true, true
}

func (store TaskBrowserStore) loadReservations(taskID string) ([]task.Reservation, *core.DomainError) {
	indexResult := loadStringIndex(store.storage, reservationTaskIndexKey(taskID), "reservation")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return nil, &reason
	}
	values := make([]task.Reservation, 0, len(loaded.values))
	for _, id := range loaded.values {
		record, found, ok := getStoredReservationJSON(store.storage, reservationRecordKey(id))
		if !ok {
			reason := invalidState("read reservation failed")
			return nil, &reason
		}
		if !found {
			continue
		}
		value, err := parseStoredReservation(record)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func parseStoredReservation(record storedReservation) (task.Reservation, *core.DomainError) {
	idResult := core.ParseTaskReservationID(record.ID)
	id, idMatched := idResult.(core.TaskReservationIDCreated)
	if !idMatched {
		reason := idResult.(core.TaskReservationIDRejected).Reason
		return task.Reservation{}, &reason
	}
	taskIDResult := core.ParseTaskID(record.TaskID)
	taskID, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		reason := taskIDResult.(core.TaskIDRejected).Reason
		return task.Reservation{}, &reason
	}
	stateResult := task.ParseReservationState(record.State)
	state, stateMatched := stateResult.(task.ReservationStateAccepted)
	if !stateMatched {
		reason := stateResult.(task.ReservationStateRejected).Reason
		return task.Reservation{}, &reason
	}
	requestedByResult := core.ParseUserID(record.RequestedByUser)
	requestedBy, requestedByMatched := requestedByResult.(core.UserIDCreated)
	if !requestedByMatched {
		reason := requestedByResult.(core.UserIDRejected).Reason
		return task.Reservation{}, &reason
	}

	var assignee task.Assignee
	switch record.AssigneeKind {
	case task.AssigneeScopeUser.String():
		result := core.ParseUserID(record.AssigneeUserID)
		created, matched := result.(core.UserIDCreated)
		if !matched {
			reason := result.(core.UserIDRejected).Reason
			return task.Reservation{}, &reason
		}
		assignee = task.UserAssignee{UserID: created.Value}
	case task.AssigneeScopeOrganizationTeam.String():
		orgResult := core.ParseOrganizationID(record.AssigneeOrgID)
		orgCreated, orgMatched := orgResult.(core.OrganizationIDCreated)
		if !orgMatched {
			reason := orgResult.(core.OrganizationIDRejected).Reason
			return task.Reservation{}, &reason
		}
		teamResult := core.ParseTeamID(record.AssigneeTeamID)
		teamCreated, teamMatched := teamResult.(core.TeamIDCreated)
		if !teamMatched {
			reason := teamResult.(core.TeamIDRejected).Reason
			return task.Reservation{}, &reason
		}
		assignee = task.OrganizationTeamAssignee{OrganizationID: orgCreated.Value, TeamID: teamCreated.Value}
	case task.AssigneeScopeTeam.String():
		result := core.ParseTeamID(record.AssigneeTeamID)
		created, matched := result.(core.TeamIDCreated)
		if !matched {
			reason := result.(core.TeamIDRejected).Reason
			return task.Reservation{}, &reason
		}
		assignee = task.TeamAssignee{TeamID: created.Value}
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "reservation assignee kind is invalid")
		return task.Reservation{}, &reason
	}

	return task.Reservation{ID: id.Value, TaskID: taskID.Value, Assignee: assignee, State: state.Value, RequestedBy: requestedBy.Value}, nil
}

func (store TaskBrowserStore) CreateReservation(_ context.Context, reservationID core.TaskReservationID, command task.ReservationCommand) task.CreateReservationStoreResult {
	record, found, err := loadStoredTaskRecord(store.storage, command.TaskID.String())
	if err != nil {
		return task.CreateReservationStoreRejected{Reason: *err}
	}
	if !found {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if record.State != task.StateOpen.String() {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can be reserved")}
	}

	existing, existingErr := store.loadReservations(command.TaskID.String())
	if existingErr != nil {
		return task.CreateReservationStoreRejected{Reason: *existingErr}
	}
	assigneeKind, assigneeUserID, assigneeTeamID, assigneeOrgID := assigneeSQLColumnsBrowser(command.Assignee)
	for _, reservation := range existing {
		if reservation.State != task.ReservationStateRequested && reservation.State != task.ReservationStateActive {
			continue
		}
		existingKind, existingUserID, existingTeamID, existingOrgID := assigneeSQLColumnsBrowser(reservation.Assignee)
		if existingKind == assigneeKind && existingUserID == assigneeUserID && existingTeamID == assigneeTeamID && existingOrgID == assigneeOrgID {
			return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "assignee already has an active or pending reservation")}
		}
	}

	var initialState task.ReservationState
	switch record.Participation {
	case task.ParticipationPolicyReservationRequired.String():
		initialState = task.ReservationStateActive
	case task.ParticipationPolicyApprovalRequired.String():
		initialState = task.ReservationStateRequested
	default:
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "task does not require reservation")}
	}

	stored := storedReservation{
		ID: reservationID.String(), TaskID: command.TaskID.String(), AssigneeKind: assigneeKind,
		AssigneeUserID: assigneeUserID, AssigneeTeamID: assigneeTeamID, AssigneeOrgID: assigneeOrgID,
		State: initialState.String(), RequestedByUser: command.RequestedBy.String(),
	}
	if !putStoredReservationJSON(store.storage, reservationRecordKey(stored.ID), stored) {
		return task.CreateReservationStoreRejected{Reason: invalidState("insert reservation failed")}
	}
	if _, matched := appendStringIndex(store.storage, reservationTaskIndexKey(command.TaskID.String()), stored.ID, "reservation").(stringIndexStored); !matched {
		return task.CreateReservationStoreRejected{Reason: invalidState("update reservation index failed")}
	}

	value, parseErr := parseStoredReservation(stored)
	if parseErr != nil {
		return task.CreateReservationStoreRejected{Reason: *parseErr}
	}
	return task.CreateReservationStoreAccepted{Value: value}
}

func assigneeSQLColumnsBrowser(assignee task.Assignee) (kind string, userID string, teamID string, organizationID string) {
	switch typed := assignee.(type) {
	case task.UserAssignee:
		return task.AssigneeScopeUser.String(), typed.UserID.String(), "", ""
	case task.OrganizationTeamAssignee:
		return task.AssigneeScopeOrganizationTeam.String(), "", typed.TeamID.String(), typed.OrganizationID.String()
	case task.TeamAssignee:
		return task.AssigneeScopeTeam.String(), "", typed.TeamID.String(), ""
	default:
		return "", "", "", ""
	}
}

func (store TaskBrowserStore) ChangeReservationState(_ context.Context, taskID core.TaskID, reservationID core.TaskReservationID, state task.ReservationState) task.ChangeReservationStateStoreResult {
	record, found, ok := getStoredReservationJSON(store.storage, reservationRecordKey(reservationID.String()))
	if !ok {
		return task.ChangeReservationStateStoreRejected{Reason: invalidState("read reservation failed")}
	}
	if !found || record.TaskID != taskID.String() {
		return task.ChangeReservationStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "reservation is not pending or active")}
	}
	if record.State != task.ReservationStateRequested.String() && record.State != task.ReservationStateActive.String() {
		return task.ChangeReservationStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "reservation is not pending or active")}
	}
	record.State = state.String()
	if !putStoredReservationJSON(store.storage, reservationRecordKey(reservationID.String()), record) {
		return task.ChangeReservationStateStoreRejected{Reason: invalidState("change reservation state failed")}
	}
	value, parseErr := parseStoredReservation(record)
	if parseErr != nil {
		return task.ChangeReservationStateStoreRejected{Reason: *parseErr}
	}
	return task.ChangeReservationStateStoreAccepted{Value: value}
}

func (store TaskBrowserStore) ListReservations(_ context.Context, taskID core.TaskID) task.ListReservationsStoreResult {
	values, err := store.loadReservations(taskID.String())
	if err != nil {
		return task.ListReservationsStoreRejected{Reason: *err}
	}
	return task.ListReservationsStoreAccepted{Values: values}
}

func (store TaskBrowserStore) CheckSubmissionEligibility(_ context.Context, taskID core.TaskID, submitterID core.UserID) task.SubmissionEligibilityStoreResult {
	record, found, err := loadStoredTaskRecord(store.storage, taskID.String())
	if err != nil {
		return task.SubmissionEligibilityRejected{Reason: *err}
	}
	if !found {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if record.Participation == task.ParticipationPolicyOpen.String() {
		return task.SubmissionEligible{}
	}
	reservations, reservationsErr := store.loadReservations(taskID.String())
	if reservationsErr != nil {
		return task.SubmissionEligibilityRejected{Reason: *reservationsErr}
	}
	for _, reservation := range reservations {
		if reservation.State != task.ReservationStateActive {
			continue
		}
		if userAssignee, matched := reservation.Assignee.(task.UserAssignee); matched && userAssignee.UserID == submitterID {
			return task.SubmissionEligible{}
		}
	}
	return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task requires an active reservation for the submitter")}
}

// Task series and comments are not yet implemented in the browser store.
func (store TaskBrowserStore) ListSeries(context.Context, core.UserID, core.Page) task.ListSeriesStoreResult {
	return task.ListSeriesStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) FindSeries(context.Context, core.TaskSeriesID) task.FindSeriesStoreResult {
	return task.FindSeriesStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) CreateSeries(context.Context, task.Series) task.SeriesMutationStoreResult {
	return task.SeriesMutationStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) UpdateSeries(context.Context, core.TaskSeriesID, task.SeriesTitle, task.SeriesDescription) task.SeriesMutationStoreResult {
	return task.SeriesMutationStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) UpdateSeriesState(context.Context, core.TaskSeriesID, task.SeriesState) task.SeriesMutationStoreResult {
	return task.SeriesMutationStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) AddTaskToSeries(context.Context, core.TaskSeriesID, core.TaskID) task.SeriesMutationStoreResult {
	return task.SeriesMutationStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) RemoveTaskFromSeries(context.Context, core.TaskSeriesID, core.TaskID) task.SeriesMutationStoreResult {
	return task.SeriesMutationStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) ReorderSeries(context.Context, core.TaskSeriesID, []core.TaskID) task.SeriesMutationStoreResult {
	return task.SeriesMutationStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) CreateSeriesComment(context.Context, task.SeriesComment) task.CreateSeriesCommentStoreResult {
	return task.CreateSeriesCommentStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) ListSeriesComments(context.Context, core.TaskSeriesID) task.ListSeriesCommentsStoreResult {
	return task.ListSeriesCommentsStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) CreateTaskComment(context.Context, task.TaskComment) task.CreateTaskCommentStoreResult {
	return task.CreateTaskCommentStoreRejected{Reason: notImplementedTaskSeries()}
}

func (store TaskBrowserStore) ListTaskComments(context.Context, core.TaskID) task.ListTaskCommentsStoreResult {
	return task.ListTaskCommentsStoreRejected{Reason: notImplementedTaskSeries()}
}

func notImplementedTaskSeries() core.DomainError {
	return core.NewDomainError(core.ErrorCodeInvalidState, "task series and comments are not yet implemented in the browser demo")
}
