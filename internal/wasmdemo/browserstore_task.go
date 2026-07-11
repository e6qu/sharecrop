package wasmdemo

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
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
	clock   HandlerClock
}

func NewTaskBrowserStore(storage BrowserStorage, ids InteractionIDSource, clock HandlerClock) TaskBrowserStore {
	return TaskBrowserStore{storage: storage, ids: ids, clock: clock}
}

func (store TaskBrowserStore) CreateTask(_ context.Context, seriesID core.TaskSeriesID, taskID core.TaskID, command task.CreateCommand) task.CreateTaskStoreResult {
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
		CreatedBy:   command.Actor.ID.String(),
		Attachments: storedTaskAttachmentsFrom(command.Attachments),
	}

	switch placement := command.Placement.(type) {
	case task.StandalonePlacement:
		// No series linkage.
	case task.NewSeriesPlacement:
		seriesRecord := storedSeries{
			ID: seriesID.String(), OwnerKind: ownerKind, OwnerUserID: ownerUserID, OwnerTeamID: ownerTeamID, OwnerOrganizationID: ownerOrganizationID,
			Title: placement.Title.String(), Description: "", State: task.SeriesStateDraft.String(), CreatedBy: command.Actor.ID.String(),
		}
		if !putStoredSeriesJSON(store.storage, seriesRecordKey(seriesRecord.ID), seriesRecord) {
			return task.CreateTaskStoreRejected{Reason: invalidState("insert task series failed")}
		}
		if _, matched := appendStringIndex(store.storage, seriesCreatorIndexKey(seriesRecord.CreatedBy), seriesRecord.ID, "task series").(stringIndexStored); !matched {
			return task.CreateTaskStoreRejected{Reason: invalidState("update task series index failed")}
		}
		record.SeriesID = seriesID.String()
		record.SeriesPosition = placement.Position.Int()
	case task.ExistingSeriesPlacement:
		record.SeriesID = placement.SeriesID.String()
		record.SeriesPosition = placement.Position.Int()
	default:
		return task.CreateTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series placement is invalid")}
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
	if visibilityOrgID != "" {
		if _, matched := appendStringIndex(store.storage, taskOrganizationVisibilityIndexKey(visibilityOrgID), record.ID, "task").(stringIndexStored); !matched {
			return task.CreateTaskStoreRejected{Reason: invalidState("update organization task index failed")}
		}
	}
	if visibilityTeamID != "" {
		if _, matched := appendStringIndex(store.storage, taskTeamVisibilityIndexKey(visibilityTeamID), record.ID, "task").(stringIndexStored); !matched {
			return task.CreateTaskStoreRejected{Reason: invalidState("update team task index failed")}
		}
	}
	if visibilityUserID != "" {
		if _, matched := appendStringIndex(store.storage, taskUserVisibilityIndexKey(visibilityUserID), record.ID, "task").(stringIndexStored); !matched {
			return task.CreateTaskStoreRejected{Reason: invalidState("update user visibility task index failed")}
		}
	}
	if record.SeriesID != "" {
		if _, matched := appendStringIndex(store.storage, taskSeriesTaskIndexKey(record.SeriesID), record.ID, "task series task").(stringIndexStored); !matched {
			return task.CreateTaskStoreRejected{Reason: invalidState("update task series task index failed")}
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
		return task.FindTaskStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	value, parseErr := parseStoredTaskRecord(record)
	if parseErr != nil {
		return task.FindTaskStoreRejected{Reason: *parseErr}
	}
	return task.FindTaskStoreAccepted{Value: value}
}

// requireOpenableReward mirrors internal/db's requireOpenableReward: a
// credit/bundle reward needs a task_funds record whose allocated credits match
// the declared amount, and a collectible/bundle reward needs at least one held
// collectible-reward record (funded via AssetBrowserStore.FundCollectibleReward).
func (store TaskBrowserStore) requireOpenableReward(record storedTaskRecord) *core.DomainError {
	if record.RewardKind == "credit" || record.RewardKind == "bundle" {
		fund, found, err := (LedgerBrowserStore{storage: store.storage}).loadFund(record.ID)
		if err != nil {
			return err
		}
		if !found {
			reason := core.NewDomainError(core.ErrorCodeConflict, "credit reward must be funded before opening")
			return &reason
		}
		if fund.CreditAmount != record.RewardCreditAmount {
			reason := core.NewDomainError(core.ErrorCodeConflict, "allocated credits must match the declared credit reward")
			return &reason
		}
	}
	if record.RewardKind == "collectible" || record.RewardKind == "bundle" {
		heldCollectibles, heldErr := countHeldCollectibleRewards(store.storage, record.ID)
		if heldErr != nil {
			return heldErr
		}
		if heldCollectibles < 1 {
			reason := core.NewDomainError(core.ErrorCodeConflict, "collectible reward must be funded before opening")
			return &reason
		}
	}
	return nil
}

// settleFundedTaskOnCancel mirrors internal/db's helper of the same name:
// cancelling a funded task that has not been awarded returns its allocated
// credits and every held collectible to their funder before the task is
// cancelled, so nothing is orphaned. An unfunded task settles to nothing.
func (store TaskBrowserStore) settleFundedTaskOnCancel(taskID string) *core.DomainError {
	ledgerStore := LedgerBrowserStore{storage: store.storage, ids: store.ids}
	fund, found, err := ledgerStore.loadFund(taskID)
	if err != nil {
		return err
	}
	if found {
		entryResult := core.NewLedgerEntryID()
		entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
		if !matched {
			reason := entryResult.(core.LedgerEntryIDRejected).Reason
			return &reason
		}
		saveResult := SaveLedgerEntry(store.storage, StoredLedgerEntry{
			ID: entryCreated.Value.String(), OwnerKind: fund.FunderOwnerKind, OwnerID: fund.FunderOwnerID,
			Kind: ledger.EntryKindTaskRefund.String(), Amount: fund.CreditAmount, TaskID: taskID,
		})
		if _, matched := saveResult.(LedgerEntryStored); !matched {
			reason := invalidState("insert cancel refund ledger entry failed")
			return &reason
		}
		if !ledgerStore.clearFund(fund) {
			reason := invalidState("clear task fund failed")
			return &reason
		}
	}
	if reason := refundHeldCollectibleRewardBrowser(store.storage, store.ids, taskID); reason != nil {
		return reason
	}
	return nil
}

// releaseReservationsOnCancel mirrors internal/db's helper of the same name:
// every non-terminal reservation on a cancelled task (requested/active/
// submitted) moves to cancelled_by_requester, the same terminal state the
// owner-driven submission rejection uses. Terminal reservations are left
// untouched. It is a package-level function taking BrowserStorage (rather than
// a TaskBrowserStore method) so the refund path in browserstore_ledger.go,
// which cancels a task from a LedgerBrowserStore, can reuse it without
// reconstructing a TaskBrowserStore.
func releaseReservationsOnCancel(storage BrowserStorage, taskID string) *core.DomainError {
	indexResult := loadStringIndex(storage, reservationTaskIndexKey(taskID), "reservation")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return &reason
	}
	for _, id := range loaded.values {
		record, found, ok := getStoredReservationJSON(storage, reservationRecordKey(id))
		if !ok {
			reason := invalidState("read reservation failed")
			return &reason
		}
		if !found {
			continue
		}
		if record.State != task.ReservationStateRequested.String() &&
			record.State != task.ReservationStateActive.String() &&
			record.State != task.ReservationStateSubmitted.String() {
			continue
		}
		record.State = task.ReservationStateCancelledByRequester.String()
		if !putStoredReservationJSON(storage, reservationRecordKey(id), record) {
			reason := invalidState("release reservations on cancel failed")
			return &reason
		}
	}
	return nil
}

func (store TaskBrowserStore) ChangeTaskState(_ context.Context, taskID core.TaskID, state task.State) task.ChangeTaskStateStoreResult {
	record, found, err := loadStoredTaskRecord(store.storage, taskID.String())
	if err != nil {
		return task.ChangeTaskStateStoreRejected{Reason: *err}
	}
	if !found {
		return task.ChangeTaskStateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if state == task.StateOpen {
		if rejectReason := store.requireOpenableReward(record); rejectReason != nil {
			return task.ChangeTaskStateStoreRejected{Reason: *rejectReason}
		}
	}
	if state == task.StateCancelled {
		if rejectReason := store.settleFundedTaskOnCancel(record.ID); rejectReason != nil {
			return task.ChangeTaskStateStoreRejected{Reason: *rejectReason}
		}
		// A cancelled task can never be reserved, submitted to, or reviewed
		// again, so every reservation still held on it must be released.
		// Otherwise it dangles forever: the expiry sweep ignores submitted
		// reservations and no other path clears them.
		if rejectReason := releaseReservationsOnCancel(store.storage, record.ID); rejectReason != nil {
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
	if sweepErr := store.releaseExpiredReservations(); sweepErr != nil {
		return task.ListTasksStoreRejected{Reason: *sweepErr}
	}

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
		// Matches internal/db: visibility.user_id = @user_id or created_by =
		// @user_id - a task shared directly with this user, not just one
		// they created, belongs in "their" tasks.
		createdResult := loadStringIndex(store.storage, taskUserIndexKey(typed.UserID.String()), "task")
		created, createdMatched := createdResult.(stringIndexLoaded)
		if !createdMatched {
			return task.ListTasksStoreRejected{Reason: invalidState(createdResult.(stringIndexRejected).reason)}
		}
		sharedResult := loadStringIndex(store.storage, taskUserVisibilityIndexKey(typed.UserID.String()), "task")
		shared, sharedMatched := sharedResult.(stringIndexLoaded)
		if !sharedMatched {
			return task.ListTasksStoreRejected{Reason: invalidState(sharedResult.(stringIndexRejected).reason)}
		}
		candidateIDs = unionStringSlices(created.values, shared.values)
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
	case task.OrganizationListScope:
		indexResult := loadStringIndex(store.storage, taskOrganizationVisibilityIndexKey(typed.OrganizationID.String()), "task")
		loaded, matched := indexResult.(stringIndexLoaded)
		if !matched {
			return task.ListTasksStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
		}
		candidateIDs = loaded.values
	case task.TeamListScope:
		indexResult := loadStringIndex(store.storage, taskTeamVisibilityIndexKey(typed.TeamID.String()), "task")
		loaded, matched := indexResult.(stringIndexLoaded)
		if !matched {
			return task.ListTasksStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
		}
		// Matches internal/db: the team's queue includes tasks the team is
		// actively working on (its active reservation), not only tasks shared
		// with it via visibility.
		reservedIDs, reservedErr := store.taskIDsActivelyReservedByTeam(typed.TeamID.String())
		if reservedErr != nil {
			return task.ListTasksStoreRejected{Reason: *reservedErr}
		}
		candidateIDs = unionStringSlices(loaded.values, reservedIDs)
	default:
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
		activeAssignee, assigneeErr := store.activeAssigneeForTask(id)
		if assigneeErr != nil {
			return task.ListTasksStoreRejected{Reason: *assigneeErr}
		}
		// Matches internal/db: an actively-reserved task is hidden from the
		// public/team queue unless the viewer is exempt (already reserved it,
		// created it, or the caller explicitly asked to include reserved
		// tasks) - otherwise every reserved task would clutter the discovery
		// queue for everyone else.
		if typed, isPublic := scope.(task.PublicListScope); isPublic && !typed.IncludeReserved {
			if _, isActive := activeAssignee.(task.NoActiveAssignee); !isActive {
				userAssignee, isUserAssignee := activeAssignee.(task.ActiveUserAssignee)
				reservedByViewer := isUserAssignee && userAssignee.UserID == typed.ViewerID
				if !reservedByViewer && record.CreatedBy != typed.ViewerID.String() {
					continue
				}
			}
		}
		if typed, isTeam := scope.(task.TeamListScope); isTeam && !typed.IncludeReserved {
			if _, isActive := activeAssignee.(task.NoActiveAssignee); !isActive {
				reservedByThisTeam := false
				switch assignee := activeAssignee.(type) {
				case task.ActiveTeamAssignee:
					reservedByThisTeam = assignee.TeamID == typed.TeamID
				case task.ActiveOrganizationTeamAssignee:
					reservedByThisTeam = assignee.TeamID == typed.TeamID
				}
				if !reservedByThisTeam {
					continue
				}
			}
		}
		value, parseErr := parseStoredTaskRecord(record)
		if parseErr != nil {
			return task.ListTasksStoreRejected{Reason: *parseErr}
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

// taskIDsActivelyReservedByTeam scans the global reservation index for
// active reservations held by teamID and returns their task IDs, in
// reservation-creation order.
func (store TaskBrowserStore) taskIDsActivelyReservedByTeam(teamID string) ([]string, *core.DomainError) {
	indexResult := loadStringIndex(store.storage, reservationAllIndexKey(), "reservation")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return nil, &reason
	}
	taskIDs := make([]string, 0)
	for _, id := range loaded.values {
		record, found, ok := getStoredReservationJSON(store.storage, reservationRecordKey(id))
		if !ok {
			reason := invalidState("read reservation failed")
			return nil, &reason
		}
		if !found {
			continue
		}
		if record.State == task.ReservationStateActive.String() && record.AssigneeTeamID == teamID {
			taskIDs = append(taskIDs, record.TaskID)
		}
	}
	return taskIDs, nil
}

// unionStringSlices merges two id slices, preserving first-seen order and
// dropping duplicates - used where a task can qualify for a list scope via
// more than one index (e.g. UserListScope: created by this user or visible
// to them).
func unionStringSlices(first []string, second []string) []string {
	seen := make(map[string]bool, len(first)+len(second))
	merged := make([]string, 0, len(first)+len(second))
	for _, values := range [][]string{first, second} {
		for _, value := range values {
			if seen[value] {
				continue
			}
			seen[value] = true
			merged = append(merged, value)
		}
	}
	return merged
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
	ExpiresAt       int64  `json:"expires_at_unix,omitempty"`
}

func reservationRecordKey(id string) string { return "task:reservation:" + id }
func reservationTaskIndexKey(taskID string) string {
	return "task:reservation_index:" + taskID
}

// reservationAllIndexKey indexes every reservation across all tasks, so the
// expiry sweep (mirroring internal/db's expireReservationsSQL, which scans the
// whole task_reservations table) can visit each of them, and so the team task
// list can find tasks actively reserved by a team.
func reservationAllIndexKey() string { return "task:reservation_index_all" }

// releaseExpiredReservations mirrors internal/db's expireReservationsSQL:
// every requested/active reservation whose expiry has passed flips to expired. It
// runs at the start of every reservation-observing store method, the same
// call sites the real store sweeps at.
func (store TaskBrowserStore) releaseExpiredReservations() *core.DomainError {
	indexResult := loadStringIndex(store.storage, reservationAllIndexKey(), "reservation")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return &reason
	}
	nowUnixNano := store.clock.Now().UnixNano()
	for _, id := range loaded.values {
		record, found, ok := getStoredReservationJSON(store.storage, reservationRecordKey(id))
		if !ok {
			reason := invalidState("release expired reservations failed")
			return &reason
		}
		if !found {
			continue
		}
		if record.State != task.ReservationStateRequested.String() && record.State != task.ReservationStateActive.String() {
			continue
		}
		if record.ExpiresAt > nowUnixNano {
			continue
		}
		record.State = task.ReservationStateExpired.String()
		if !putStoredReservationJSON(store.storage, reservationRecordKey(id), record) {
			reason := invalidState("release expired reservations failed")
			return &reason
		}
	}
	return nil
}

// storedImplementorBan mirrors internal/db's task_implementor_bans row: a
// reviewer rejecting a submission may ban the implementor from the task, and
// the ban blocks both new reservations and submission eligibility.
type storedImplementorBan struct {
	TaskID       string `json:"task_id"`
	AssigneeKind string `json:"assignee_kind"`
	UserID       string `json:"user_id,omitempty"`
	TeamID       string `json:"team_id,omitempty"`
	OrgID        string `json:"organization_id,omitempty"`
	BannedBy     string `json:"banned_by_user_id"`
}

// implementorBanKey mirrors the (task_id, assignee_kind, assignee_key)
// primary key of task_implementor_bans.
func implementorBanKey(taskID string, assigneeKind string, assigneeKey string) string {
	return "task:implementor_ban:" + taskID + ":" + assigneeKind + ":" + assigneeKey
}

func implementorBanAssigneeKey(kind string, userID string, teamID string, orgID string) string {
	switch kind {
	case task.AssigneeScopeUser.String():
		return userID
	case task.AssigneeScopeOrganizationTeam.String():
		return orgID + ":" + teamID
	default:
		return teamID
	}
}

func saveImplementorBan(storage BrowserStorage, ban storedImplementorBan) bool {
	encoded, err := json.Marshal(ban)
	if err != nil {
		return false
	}
	assigneeKey := implementorBanAssigneeKey(ban.AssigneeKind, ban.UserID, ban.TeamID, ban.OrgID)
	return putStorageString(storage, implementorBanKey(ban.TaskID, ban.AssigneeKind, assigneeKey), string(encoded))
}

func isImplementorBanned(storage BrowserStorage, taskID string, assigneeKind string, userID string, teamID string, orgID string) (bool, *core.DomainError) {
	assigneeKey := implementorBanAssigneeKey(assigneeKind, userID, teamID, orgID)
	_, found, ok := getStorageString(storage, implementorBanKey(taskID, assigneeKind, assigneeKey))
	if !ok {
		reason := invalidState("check task implementor ban failed")
		return false, &reason
	}
	return found, nil
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

// taskSeriesBlocksExecution mirrors internal/db's method of the same name:
// a task belonging to a series that is not currently published cannot be
// reserved or submitted to. A standalone task (no series) is never blocked.
func taskSeriesBlocksExecution(storage BrowserStorage, seriesID string) *core.DomainError {
	if seriesID == "" {
		return nil
	}
	series, found, ok := getStoredSeriesJSON(storage, seriesRecordKey(seriesID))
	if !ok {
		reason := invalidState("check task series state failed")
		return &reason
	}
	if !found || series.State != task.SeriesStatePublished.String() {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "the task's series is not published")
		return &reason
	}
	return nil
}

func (store TaskBrowserStore) CreateReservation(_ context.Context, reservationID core.TaskReservationID, command task.ReservationCommand) task.CreateReservationStoreResult {
	if sweepErr := store.releaseExpiredReservations(); sweepErr != nil {
		return task.CreateReservationStoreRejected{Reason: *sweepErr}
	}

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
	if blocked := taskSeriesBlocksExecution(store.storage, record.SeriesID); blocked != nil {
		return task.CreateReservationStoreRejected{Reason: *blocked}
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

	assigneeKind, assigneeUserID, assigneeTeamID, assigneeOrgID := assigneeSQLColumnsBrowser(command.Assignee)
	banned, banErr := isImplementorBanned(store.storage, command.TaskID.String(), assigneeKind, assigneeUserID, assigneeTeamID, assigneeOrgID)
	if banErr != nil {
		return task.CreateReservationStoreRejected{Reason: *banErr}
	}
	if banned {
		return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "implementor is banned from the task")}
	}

	existing, existingErr := store.loadReservations(command.TaskID.String())
	if existingErr != nil {
		return task.CreateReservationStoreRejected{Reason: *existingErr}
	}
	for _, reservation := range existing {
		if reservation.State != task.ReservationStateRequested && reservation.State != task.ReservationStateActive {
			continue
		}
		existingKind, existingUserID, existingTeamID, existingOrgID := assigneeSQLColumnsBrowser(reservation.Assignee)
		if existingKind == assigneeKind && existingUserID == assigneeUserID && existingTeamID == assigneeTeamID && existingOrgID == assigneeOrgID {
			return task.CreateReservationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "assignee already has an active or pending reservation")}
		}
	}
	// Mirror internal/db's task_reservations_one_active_idx partial unique
	// index: at most one active reservation per task, whoever holds it.
	if initialState == task.ReservationStateActive {
		if activeErr := store.requireNoActiveReservation(command.TaskID.String(), "task already has an active reservation"); activeErr != nil {
			return task.CreateReservationStoreRejected{Reason: *activeErr}
		}
	}

	stored := storedReservation{
		ID: reservationID.String(), TaskID: command.TaskID.String(), AssigneeKind: assigneeKind,
		AssigneeUserID: assigneeUserID, AssigneeTeamID: assigneeTeamID, AssigneeOrgID: assigneeOrgID,
		State: initialState.String(), RequestedByUser: command.RequestedBy.String(),
		ExpiresAt: store.clock.Now().Add(time.Duration(record.ReservationTTLHours) * time.Hour).UnixNano(),
	}
	if !putStoredReservationJSON(store.storage, reservationRecordKey(stored.ID), stored) {
		return task.CreateReservationStoreRejected{Reason: invalidState("insert reservation failed")}
	}
	if _, matched := appendStringIndex(store.storage, reservationTaskIndexKey(command.TaskID.String()), stored.ID, "reservation").(stringIndexStored); !matched {
		return task.CreateReservationStoreRejected{Reason: invalidState("update reservation index failed")}
	}
	if _, matched := appendStringIndex(store.storage, reservationAllIndexKey(), stored.ID, "reservation").(stringIndexStored); !matched {
		return task.CreateReservationStoreRejected{Reason: invalidState("update reservation index failed")}
	}

	value, parseErr := parseStoredReservation(stored)
	if parseErr != nil {
		return task.CreateReservationStoreRejected{Reason: *parseErr}
	}
	return task.CreateReservationStoreAccepted{Value: value}
}

// requireNoActiveReservation rejects with a conflict carrying message when
// the task already has an active reservation, mirroring the partial unique
// index the real store relies on.
func (store TaskBrowserStore) requireNoActiveReservation(taskID string, message string) *core.DomainError {
	existing, existingErr := store.loadReservations(taskID)
	if existingErr != nil {
		return existingErr
	}
	for _, reservation := range existing {
		if reservation.State == task.ReservationStateActive {
			reason := core.NewDomainError(core.ErrorCodeConflict, message)
			return &reason
		}
	}
	return nil
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
	if sweepErr := store.releaseExpiredReservations(); sweepErr != nil {
		return task.ChangeReservationStateStoreRejected{Reason: *sweepErr}
	}

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
	// Approving a reservation while another one is already active would break
	// the one-active-reservation-per-task rule; the real store's partial
	// unique index surfaces this as "reservation state was not changed".
	if state == task.ReservationStateActive && record.State != task.ReservationStateActive.String() {
		if activeErr := store.requireNoActiveReservation(taskID.String(), "reservation state was not changed"); activeErr != nil {
			return task.ChangeReservationStateStoreRejected{Reason: *activeErr}
		}
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
	if sweepErr := store.releaseExpiredReservations(); sweepErr != nil {
		return task.ListReservationsStoreRejected{Reason: *sweepErr}
	}

	values, err := store.loadReservations(taskID.String())
	if err != nil {
		return task.ListReservationsStoreRejected{Reason: *err}
	}
	return task.ListReservationsStoreAccepted{Values: values}
}

func (store TaskBrowserStore) CheckSubmissionEligibility(_ context.Context, taskID core.TaskID, submitterID core.UserID) task.SubmissionEligibilityStoreResult {
	if sweepErr := store.releaseExpiredReservations(); sweepErr != nil {
		return task.SubmissionEligibilityRejected{Reason: *sweepErr}
	}

	record, found, err := loadStoredTaskRecord(store.storage, taskID.String())
	if err != nil {
		return task.SubmissionEligibilityRejected{Reason: *err}
	}
	if !found {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if blocked := taskSeriesBlocksExecution(store.storage, record.SeriesID); blocked != nil {
		return task.SubmissionEligibilityRejected{Reason: *blocked}
	}
	banned, banErr := isImplementorBanned(store.storage, taskID.String(), task.AssigneeScopeUser.String(), submitterID.String(), "", "")
	if banErr != nil {
		return task.SubmissionEligibilityRejected{Reason: *banErr}
	}
	if banned {
		return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "implementor is banned from the task")}
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
		switch assignee := reservation.Assignee.(type) {
		case task.UserAssignee:
			if assignee.UserID == submitterID {
				return task.SubmissionEligible{}
			}
		case task.OrganizationTeamAssignee:
			if isTeamMember(store.storage, assignee.TeamID.String(), submitterID.String()) {
				return task.SubmissionEligible{}
			}
		case task.TeamAssignee:
			if isTeamMember(store.storage, assignee.TeamID.String(), submitterID.String()) {
				return task.SubmissionEligible{}
			}
		}
	}
	return task.SubmissionEligibilityRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task requires an active reservation for the submitter")}
}

// isTeamMember reports whether userID is recorded as a member of teamID,
// mirroring internal/db's team_members join used by
// CheckSubmissionEligibility for a team/organization-team reservation - a
// submission is eligible from every member of the team the reservation is
// held by, not just the literal requester.
func isTeamMember(storage BrowserStorage, teamID string, userID string) bool {
	indexResult := loadStringIndex(storage, teamMembersKey(teamID), "team member")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return false
	}
	for _, memberID := range loaded.values {
		if memberID == userID {
			return true
		}
	}
	return false
}

// Task series live in browserstore_task_series.go. Series comments below -
// turns out every series mutation response (create/update/publish/...)
// embeds the series' comment thread (internal/http's writeSeriesDetailStatus
// always calls ListSeriesComments), so this is not an optional, deferred
// feature the way it first looked: without it, every series operation fails.

type storedSeriesComment struct {
	ID       string `json:"id"`
	SeriesID string `json:"series_id"`
	AuthorID string `json:"author_id"`
	Body     string `json:"body"`
}

func seriesCommentRecordKey(id string) string { return "task:series_comment:" + id }
func seriesCommentIndexKey(seriesID string) string {
	return "task:series_comment_index:" + seriesID
}

func putStoredSeriesCommentJSON(storage BrowserStorage, rawKey string, record storedSeriesComment) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredSeriesCommentJSON(storage BrowserStorage, rawKey string) (storedSeriesComment, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedSeriesComment{}, found, ok
	}
	var record storedSeriesComment
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedSeriesComment{}, false, false
	}
	return record, true, true
}

func parseStoredSeriesComment(record storedSeriesComment) (task.SeriesComment, *core.DomainError) {
	idResult := core.ParseSeriesCommentID(record.ID)
	id, idMatched := idResult.(core.SeriesCommentIDCreated)
	if !idMatched {
		reason := idResult.(core.SeriesCommentIDRejected).Reason
		return task.SeriesComment{}, &reason
	}
	seriesIDResult := core.ParseTaskSeriesID(record.SeriesID)
	seriesID, seriesIDMatched := seriesIDResult.(core.TaskSeriesIDCreated)
	if !seriesIDMatched {
		reason := seriesIDResult.(core.TaskSeriesIDRejected).Reason
		return task.SeriesComment{}, &reason
	}
	authorResult := core.ParseUserID(record.AuthorID)
	author, authorMatched := authorResult.(core.UserIDCreated)
	if !authorMatched {
		reason := authorResult.(core.UserIDRejected).Reason
		return task.SeriesComment{}, &reason
	}
	bodyResult := task.NewCommentBody(record.Body)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		reason := bodyResult.(task.CommentBodyRejected).Reason
		return task.SeriesComment{}, &reason
	}
	return task.SeriesComment{ID: id.Value, SeriesID: seriesID.Value, AuthorID: author.Value, Body: body.Value}, nil
}

func (store TaskBrowserStore) CreateSeriesComment(_ context.Context, comment task.SeriesComment) task.CreateSeriesCommentStoreResult {
	record := storedSeriesComment{ID: comment.ID.String(), SeriesID: comment.SeriesID.String(), AuthorID: comment.AuthorID.String(), Body: comment.Body.String()}
	if !putStoredSeriesCommentJSON(store.storage, seriesCommentRecordKey(record.ID), record) {
		return task.CreateSeriesCommentStoreRejected{Reason: invalidState("insert series comment failed")}
	}
	if _, matched := appendStringIndex(store.storage, seriesCommentIndexKey(record.SeriesID), record.ID, "series comment").(stringIndexStored); !matched {
		return task.CreateSeriesCommentStoreRejected{Reason: invalidState("update series comment index failed")}
	}
	value, parseErr := parseStoredSeriesComment(record)
	if parseErr != nil {
		return task.CreateSeriesCommentStoreRejected{Reason: *parseErr}
	}
	return task.CreateSeriesCommentStoreAccepted{Value: value}
}

func (store TaskBrowserStore) ListSeriesComments(_ context.Context, seriesID core.TaskSeriesID) task.ListSeriesCommentsStoreResult {
	indexResult := loadStringIndex(store.storage, seriesCommentIndexKey(seriesID.String()), "series comment")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return task.ListSeriesCommentsStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}
	values := make([]task.SeriesComment, 0, len(loaded.values))
	for _, id := range loaded.values {
		record, found, ok := getStoredSeriesCommentJSON(store.storage, seriesCommentRecordKey(id))
		if !ok {
			return task.ListSeriesCommentsStoreRejected{Reason: invalidState("read series comment failed")}
		}
		if !found {
			continue
		}
		value, parseErr := parseStoredSeriesComment(record)
		if parseErr != nil {
			return task.ListSeriesCommentsStoreRejected{Reason: *parseErr}
		}
		values = append(values, value)
	}
	return task.ListSeriesCommentsStoreAccepted{Values: values}
}

type storedTaskComment struct {
	ID       string `json:"id"`
	TaskID   string `json:"task_id"`
	AuthorID string `json:"author_id"`
	Body     string `json:"body"`
}

func taskCommentRecordKey(id string) string { return "task:comment:" + id }
func taskCommentIndexKey(taskID string) string {
	return "task:comment_index:" + taskID
}

func putStoredTaskCommentJSON(storage BrowserStorage, rawKey string, record storedTaskComment) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredTaskCommentJSON(storage BrowserStorage, rawKey string) (storedTaskComment, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedTaskComment{}, found, ok
	}
	var record storedTaskComment
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedTaskComment{}, false, false
	}
	return record, true, true
}

func parseStoredTaskComment(record storedTaskComment) (task.TaskComment, *core.DomainError) {
	idResult := core.ParseTaskCommentID(record.ID)
	id, idMatched := idResult.(core.TaskCommentIDCreated)
	if !idMatched {
		reason := idResult.(core.TaskCommentIDRejected).Reason
		return task.TaskComment{}, &reason
	}
	taskIDResult := core.ParseTaskID(record.TaskID)
	taskID, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		reason := taskIDResult.(core.TaskIDRejected).Reason
		return task.TaskComment{}, &reason
	}
	authorResult := core.ParseUserID(record.AuthorID)
	author, authorMatched := authorResult.(core.UserIDCreated)
	if !authorMatched {
		reason := authorResult.(core.UserIDRejected).Reason
		return task.TaskComment{}, &reason
	}
	bodyResult := task.NewCommentBody(record.Body)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		reason := bodyResult.(task.CommentBodyRejected).Reason
		return task.TaskComment{}, &reason
	}
	return task.TaskComment{ID: id.Value, TaskID: taskID.Value, AuthorID: author.Value, Body: body.Value}, nil
}

func (store TaskBrowserStore) CreateTaskComment(_ context.Context, comment task.TaskComment) task.CreateTaskCommentStoreResult {
	record := storedTaskComment{ID: comment.ID.String(), TaskID: comment.TaskID.String(), AuthorID: comment.AuthorID.String(), Body: comment.Body.String()}
	if !putStoredTaskCommentJSON(store.storage, taskCommentRecordKey(record.ID), record) {
		return task.CreateTaskCommentStoreRejected{Reason: invalidState("insert task comment failed")}
	}
	if _, matched := appendStringIndex(store.storage, taskCommentIndexKey(record.TaskID), record.ID, "task comment").(stringIndexStored); !matched {
		return task.CreateTaskCommentStoreRejected{Reason: invalidState("update task comment index failed")}
	}
	value, parseErr := parseStoredTaskComment(record)
	if parseErr != nil {
		return task.CreateTaskCommentStoreRejected{Reason: *parseErr}
	}
	return task.CreateTaskCommentStoreAccepted{Value: value}
}

func (store TaskBrowserStore) ListTaskComments(_ context.Context, taskID core.TaskID) task.ListTaskCommentsStoreResult {
	indexResult := loadStringIndex(store.storage, taskCommentIndexKey(taskID.String()), "task comment")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return task.ListTaskCommentsStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}
	values := make([]task.TaskComment, 0, len(loaded.values))
	for _, id := range loaded.values {
		record, found, ok := getStoredTaskCommentJSON(store.storage, taskCommentRecordKey(id))
		if !ok {
			return task.ListTaskCommentsStoreRejected{Reason: invalidState("read task comment failed")}
		}
		if !found {
			continue
		}
		value, parseErr := parseStoredTaskComment(record)
		if parseErr != nil {
			return task.ListTaskCommentsStoreRejected{Reason: *parseErr}
		}
		values = append(values, value)
	}
	return task.ListTaskCommentsStoreAccepted{Values: values}
}
