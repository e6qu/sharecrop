package wasmdemo

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

// storedSeries is TaskBrowserStore's persistence format for a task series,
// analogous to storedTaskRecord - matches internal/db's task_series table.
type storedSeries struct {
	ID                  string `json:"id"`
	OwnerKind           string `json:"owner_kind"`
	OwnerUserID         string `json:"owner_user_id,omitempty"`
	OwnerTeamID         string `json:"owner_team_id,omitempty"`
	OwnerOrganizationID string `json:"owner_organization_id,omitempty"`
	Title               string `json:"title"`
	Description         string `json:"description"`
	State               string `json:"state"`
	CreatedBy           string `json:"created_by"`
}

func seriesRecordKey(id string) string { return "task:series:record:" + id }
func seriesCreatorIndexKey(creatorID string) string {
	return "task:series:creator_index:" + creatorID
}

func putStoredSeriesJSON(storage BrowserStorage, rawKey string, record storedSeries) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredSeriesJSON(storage BrowserStorage, rawKey string) (storedSeries, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedSeries{}, found, ok
	}
	var record storedSeries
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedSeries{}, false, false
	}
	return record, true, true
}

// parseStoredSeriesPlacement mirrors internal/db's parseSeriesPlacement: a
// task with no series_id is standalone; otherwise it's represented as an
// existing-series placement regardless of how it originally joined the
// series (matching what a fresh read from Postgres would also produce).
func parseStoredSeriesPlacement(rawSeriesID string, rawPosition int) (task.SeriesPlacement, *core.DomainError) {
	if rawSeriesID == "" {
		return task.StandalonePlacement{}, nil
	}
	seriesIDResult := core.ParseTaskSeriesID(rawSeriesID)
	seriesID, seriesMatched := seriesIDResult.(core.TaskSeriesIDCreated)
	if !seriesMatched {
		reason := seriesIDResult.(core.TaskSeriesIDRejected).Reason
		return nil, &reason
	}
	positionResult := task.NewSeriesPosition(rawPosition)
	position, positionMatched := positionResult.(task.SeriesPositionAccepted)
	if !positionMatched {
		reason := positionResult.(task.SeriesPositionRejected).Reason
		return nil, &reason
	}
	return task.ExistingSeriesPlacement{SeriesID: seriesID.Value, Position: position.Value}, nil
}

func parseStoredSeries(record storedSeries) (task.Series, *core.DomainError) {
	idResult := core.ParseTaskSeriesID(record.ID)
	id, idMatched := idResult.(core.TaskSeriesIDCreated)
	if !idMatched {
		reason := idResult.(core.TaskSeriesIDRejected).Reason
		return task.Series{}, &reason
	}
	owner, ownerErr := parseStoredOwnerColumns(record.OwnerKind, record.OwnerUserID, record.OwnerTeamID, record.OwnerOrganizationID)
	if ownerErr != nil {
		return task.Series{}, ownerErr
	}
	titleResult := task.NewSeriesTitle(record.Title)
	title, titleMatched := titleResult.(task.SeriesTitleAccepted)
	if !titleMatched {
		reason := titleResult.(task.SeriesTitleRejected).Reason
		return task.Series{}, &reason
	}
	descriptionResult := task.NewSeriesDescription(record.Description)
	description, descriptionMatched := descriptionResult.(task.SeriesDescriptionAccepted)
	if !descriptionMatched {
		reason := descriptionResult.(task.SeriesDescriptionRejected).Reason
		return task.Series{}, &reason
	}
	stateResult := task.ParseSeriesState(record.State)
	state, stateMatched := stateResult.(task.SeriesStateAccepted)
	if !stateMatched {
		reason := stateResult.(task.SeriesStateRejected).Reason
		return task.Series{}, &reason
	}
	createdByResult := core.ParseUserID(record.CreatedBy)
	createdBy, createdByMatched := createdByResult.(core.UserIDCreated)
	if !createdByMatched {
		reason := createdByResult.(core.UserIDRejected).Reason
		return task.Series{}, &reason
	}
	return task.Series{ID: id.Value, Owner: owner, Title: title.Value, Description: description.Value, State: state.Value, CreatedBy: createdBy.Value}, nil
}

// parseStoredOwnerColumns is parseStoredOwner's logic lifted to take raw
// columns directly, since storedSeries carries the same owner columns as
// storedTaskRecord but isn't itself a storedTaskRecord.
func parseStoredOwnerColumns(ownerKind string, ownerUserID string, ownerTeamID string, ownerOrganizationID string) (task.Owner, *core.DomainError) {
	return parseStoredOwner(storedTaskRecord{OwnerKind: ownerKind, OwnerUserID: ownerUserID, OwnerTeamID: ownerTeamID, OwnerOrganizationID: ownerOrganizationID})
}

func (store TaskBrowserStore) ListSeries(_ context.Context, owner core.UserID, page core.Page) task.ListSeriesStoreResult {
	indexResult := loadStringIndex(store.storage, seriesCreatorIndexKey(owner.String()), "task series")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return task.ListSeriesStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}
	values := make([]task.Series, 0, len(loaded.values))
	for index := len(loaded.values) - 1; index >= 0; index-- {
		record, found, ok := getStoredSeriesJSON(store.storage, seriesRecordKey(loaded.values[index]))
		if !ok {
			return task.ListSeriesStoreRejected{Reason: invalidState("read task series failed")}
		}
		if !found {
			continue
		}
		value, parseErr := parseStoredSeries(record)
		if parseErr != nil {
			return task.ListSeriesStoreRejected{Reason: *parseErr}
		}
		values = append(values, value)
	}
	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	return task.ListSeriesStoreAccepted{Values: values[start:end]}
}

func (store TaskBrowserStore) findSeriesDetail(seriesID string) task.FindSeriesStoreResult {
	record, found, ok := getStoredSeriesJSON(store.storage, seriesRecordKey(seriesID))
	if !ok {
		return task.FindSeriesStoreRejected{Reason: invalidState("find task series failed")}
	}
	if !found {
		return task.FindSeriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task series was not found")}
	}
	series, parseErr := parseStoredSeries(record)
	if parseErr != nil {
		return task.FindSeriesStoreRejected{Reason: *parseErr}
	}

	indexResult := loadStringIndex(store.storage, taskSeriesTaskIndexKey(seriesID), "task series task")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return task.FindSeriesStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}
	tasks := make([]task.Task, 0, len(loaded.values))
	for _, taskID := range loaded.values {
		taskRecord, taskFound, taskErr := loadStoredTaskRecord(store.storage, taskID)
		if taskErr != nil {
			return task.FindSeriesStoreRejected{Reason: *taskErr}
		}
		if !taskFound {
			continue
		}
		value, parseErr := parseStoredTaskRecord(taskRecord)
		if parseErr != nil {
			return task.FindSeriesStoreRejected{Reason: *parseErr}
		}
		tasks = append(tasks, value)
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		return seriesPositionOf(tasks[i]) < seriesPositionOf(tasks[j])
	})

	return task.FindSeriesStoreAccepted{Value: task.SeriesDetail{Series: series, Tasks: tasks}}
}

func seriesPositionOf(value task.Task) int {
	if placement, matched := value.Placement.(task.ExistingSeriesPlacement); matched {
		return placement.Position.Int()
	}
	return 0
}

func (store TaskBrowserStore) FindSeries(_ context.Context, seriesID core.TaskSeriesID) task.FindSeriesStoreResult {
	return store.findSeriesDetail(seriesID.String())
}

func (store TaskBrowserStore) CreateSeries(_ context.Context, series task.Series) task.SeriesMutationStoreResult {
	ownerKind, ownerUserID, ownerTeamID, ownerOrganizationID := ownerSQLColumnsBrowser(series.Owner)
	record := storedSeries{
		ID: series.ID.String(), OwnerKind: ownerKind, OwnerUserID: ownerUserID, OwnerTeamID: ownerTeamID, OwnerOrganizationID: ownerOrganizationID,
		Title: series.Title.String(), Description: series.Description.String(), State: series.State.String(), CreatedBy: series.CreatedBy.String(),
	}
	if !putStoredSeriesJSON(store.storage, seriesRecordKey(record.ID), record) {
		return task.SeriesMutationStoreRejected{Reason: invalidState("create task series failed")}
	}
	if _, matched := appendStringIndex(store.storage, seriesCreatorIndexKey(record.CreatedBy), record.ID, "task series").(stringIndexStored); !matched {
		return task.SeriesMutationStoreRejected{Reason: invalidState("update task series index failed")}
	}
	return store.seriesMutationDetail(record.ID)
}

func (store TaskBrowserStore) seriesMutationDetail(seriesID string) task.SeriesMutationStoreResult {
	found := store.findSeriesDetail(seriesID)
	accepted, matched := found.(task.FindSeriesStoreAccepted)
	if !matched {
		return task.SeriesMutationStoreRejected{Reason: found.(task.FindSeriesStoreRejected).Reason}
	}
	return task.SeriesMutationStoreAccepted{Value: accepted.Value}
}

func (store TaskBrowserStore) UpdateSeries(_ context.Context, seriesID core.TaskSeriesID, title task.SeriesTitle, description task.SeriesDescription) task.SeriesMutationStoreResult {
	record, found, ok := getStoredSeriesJSON(store.storage, seriesRecordKey(seriesID.String()))
	if !ok || !found {
		return task.SeriesMutationStoreRejected{Reason: invalidState("update task series failed")}
	}
	record.Title = title.String()
	record.Description = description.String()
	if !putStoredSeriesJSON(store.storage, seriesRecordKey(seriesID.String()), record) {
		return task.SeriesMutationStoreRejected{Reason: invalidState("update task series failed")}
	}
	return store.seriesMutationDetail(seriesID.String())
}

func (store TaskBrowserStore) UpdateSeriesState(_ context.Context, seriesID core.TaskSeriesID, state task.SeriesState) task.SeriesMutationStoreResult {
	record, found, ok := getStoredSeriesJSON(store.storage, seriesRecordKey(seriesID.String()))
	if !ok || !found {
		return task.SeriesMutationStoreRejected{Reason: invalidState("update task series state failed")}
	}
	record.State = state.String()
	if !putStoredSeriesJSON(store.storage, seriesRecordKey(seriesID.String()), record) {
		return task.SeriesMutationStoreRejected{Reason: invalidState("update task series state failed")}
	}
	return store.seriesMutationDetail(seriesID.String())
}

func (store TaskBrowserStore) AddTaskToSeries(_ context.Context, seriesID core.TaskSeriesID, taskID core.TaskID) task.SeriesMutationStoreResult {
	record, found, taskErr := loadStoredTaskRecord(store.storage, taskID.String())
	if taskErr != nil {
		return task.SeriesMutationStoreRejected{Reason: *taskErr}
	}
	if !found {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task was not found")}
	}
	// Mirror internal/db's clean overwrite (update tasks set series_id = ...):
	// a task moving from series A to B must leave A's task index, or A keeps
	// listing a task that no longer belongs to it.
	if record.SeriesID != "" && record.SeriesID != seriesID.String() {
		if !removeFromStringIndex(store.storage, taskSeriesTaskIndexKey(record.SeriesID), taskID.String()) {
			return task.SeriesMutationStoreRejected{Reason: invalidState("update task series task index failed")}
		}
	}
	nextPosition, positionErr := store.nextSeriesPosition(seriesID.String())
	if positionErr != nil {
		return task.SeriesMutationStoreRejected{Reason: *positionErr}
	}
	record.SeriesID = seriesID.String()
	record.SeriesPosition = nextPosition
	if !saveStoredTaskRecord(store.storage, record) {
		return task.SeriesMutationStoreRejected{Reason: invalidState("add task to series failed")}
	}
	if _, matched := appendStringIndex(store.storage, taskSeriesTaskIndexKey(seriesID.String()), taskID.String(), "task series task").(stringIndexStored); !matched {
		return task.SeriesMutationStoreRejected{Reason: invalidState("update task series task index failed")}
	}
	return store.seriesMutationDetail(seriesID.String())
}

// nextSeriesPosition mirrors internal/db's `coalesce(max(series_position), 0)
// + 1` over the target series' current tasks: index length would collide with
// existing positions after a removal, since positions are not compacted.
func (store TaskBrowserStore) nextSeriesPosition(seriesID string) (int, *core.DomainError) {
	indexResult := loadStringIndex(store.storage, taskSeriesTaskIndexKey(seriesID), "task series task")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		reason := invalidState(indexResult.(stringIndexRejected).reason)
		return 0, &reason
	}
	maxPosition := 0
	for _, memberTaskID := range loaded.values {
		member, memberFound, memberErr := loadStoredTaskRecord(store.storage, memberTaskID)
		if memberErr != nil {
			return 0, memberErr
		}
		if !memberFound || member.SeriesID != seriesID {
			continue
		}
		if member.SeriesPosition > maxPosition {
			maxPosition = member.SeriesPosition
		}
	}
	return maxPosition + 1, nil
}

func (store TaskBrowserStore) RemoveTaskFromSeries(_ context.Context, seriesID core.TaskSeriesID, taskID core.TaskID) task.SeriesMutationStoreResult {
	record, found, taskErr := loadStoredTaskRecord(store.storage, taskID.String())
	if taskErr != nil {
		return task.SeriesMutationStoreRejected{Reason: *taskErr}
	}
	if !found || record.SeriesID != seriesID.String() {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task is not in this series")}
	}
	record.SeriesID = ""
	record.SeriesPosition = 0
	if !saveStoredTaskRecord(store.storage, record) {
		return task.SeriesMutationStoreRejected{Reason: invalidState("remove task from series failed")}
	}
	if !removeFromStringIndex(store.storage, taskSeriesTaskIndexKey(seriesID.String()), taskID.String()) {
		return task.SeriesMutationStoreRejected{Reason: invalidState("update task series task index failed")}
	}
	return store.seriesMutationDetail(seriesID.String())
}

func (store TaskBrowserStore) ReorderSeries(_ context.Context, seriesID core.TaskSeriesID, order []core.TaskID) task.SeriesMutationStoreResult {
	for index, taskID := range order {
		record, found, taskErr := loadStoredTaskRecord(store.storage, taskID.String())
		if taskErr != nil {
			return task.SeriesMutationStoreRejected{Reason: *taskErr}
		}
		if !found || record.SeriesID != seriesID.String() {
			return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task is not in this series")}
		}
		record.SeriesPosition = index + 1
		if !saveStoredTaskRecord(store.storage, record) {
			return task.SeriesMutationStoreRejected{Reason: invalidState("reorder task series failed")}
		}
	}
	return store.seriesMutationDetail(seriesID.String())
}
