package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5"
)

func (store TaskStore) ListSeries(ctx context.Context, owner core.UserID, page core.Page) task.ListSeriesStoreResult {
	rows, err := store.pool.Query(ctx, seriesSelectSQL()+`
		where task_series.created_by_user_id = $1
		order by task_series.created_at desc, task_series.id
		limit $2 offset $3
	`, owner.String(), page.Limit(), page.Offset())
	if err != nil {
		return task.ListSeriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list task series failed")}
	}
	defer rows.Close()

	valuesResult := scanSeriesRows(rows)
	values, matched := valuesResult.(seriesRowsAccepted)
	if !matched {
		return task.ListSeriesStoreRejected{Reason: valuesResult.(seriesRowsRejected).reason}
	}
	return task.ListSeriesStoreAccepted{Values: values.values}
}

func (store TaskStore) FindSeries(ctx context.Context, seriesID core.TaskSeriesID) task.FindSeriesStoreResult {
	row, err := store.pool.Query(ctx, seriesSelectSQL()+" where task_series.id = $1", seriesID.String())
	if err != nil {
		return task.FindSeriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "find task series failed")}
	}
	seriesResult := scanSeriesRows(row)
	row.Close()
	values, matched := seriesResult.(seriesRowsAccepted)
	if !matched {
		return task.FindSeriesStoreRejected{Reason: seriesResult.(seriesRowsRejected).reason}
	}
	if len(values.values) != 1 {
		return task.FindSeriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task series was not found")}
	}

	taskRows, err := store.pool.Query(ctx, taskSelectSQL()+`
		where tasks.series_id = $1
		order by tasks.series_position, tasks.created_at
	`, seriesID.String())
	if err != nil {
		return task.FindSeriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list task series tasks failed")}
	}
	defer taskRows.Close()

	taskValuesResult := scanTaskRows(taskRows)
	taskValues, taskMatched := taskValuesResult.(taskRowsAccepted)
	if !taskMatched {
		return task.FindSeriesStoreRejected{Reason: taskValuesResult.(taskRowsRejected).reason}
	}

	return task.FindSeriesStoreAccepted{Value: task.SeriesDetail{Series: values.values[0], Tasks: taskValues.values}}
}

func seriesSelectSQL() string {
	return `
		select task_series.id::text, task_series.owner_kind, coalesce(task_series.user_id::text, ''),
			coalesce(task_series.team_id::text, ''), coalesce(task_series.organization_id::text, ''),
			task_series.title, task_series.description, task_series.state, task_series.created_by_user_id::text
		from task_series
	`
}

type seriesRowsResult interface {
	seriesRowsResult()
}

type seriesRowsAccepted struct {
	values []task.Series
}

type seriesRowsRejected struct {
	reason core.DomainError
}

func (seriesRowsAccepted) seriesRowsResult() {}

func (seriesRowsRejected) seriesRowsResult() {}

func scanSeriesRows(rows pgx.Rows) seriesRowsResult {
	values := make([]task.Series, 0)
	for rows.Next() {
		var rawID string
		var rawOwnerKind string
		var rawOwnerUserID string
		var rawOwnerTeamID string
		var rawOwnerOrganizationID string
		var rawTitle string
		var rawDescription string
		var rawState string
		var rawCreatedBy string
		if err := rows.Scan(&rawID, &rawOwnerKind, &rawOwnerUserID, &rawOwnerTeamID, &rawOwnerOrganizationID, &rawTitle, &rawDescription, &rawState, &rawCreatedBy); err != nil {
			return seriesRowsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan task series failed")}
		}
		parsed := parseSeriesRow(rawID, rawOwnerKind, rawOwnerUserID, rawOwnerTeamID, rawOwnerOrganizationID, rawTitle, rawDescription, rawState, rawCreatedBy)
		accepted, matched := parsed.(seriesRowAccepted)
		if !matched {
			return seriesRowsRejected{reason: parsed.(seriesRowRejected).reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return seriesRowsAccepted{values: values}
		}
		return seriesRowsRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task series failed")}
	}
	return seriesRowsAccepted{values: values}
}

type seriesRowResult interface {
	seriesRowResult()
}

type seriesRowAccepted struct {
	value task.Series
}

type seriesRowRejected struct {
	reason core.DomainError
}

func (seriesRowAccepted) seriesRowResult() {}

func (seriesRowRejected) seriesRowResult() {}

func parseSeriesRow(rawID string, rawOwnerKind string, rawOwnerUserID string, rawOwnerTeamID string, rawOwnerOrganizationID string, rawTitle string, rawDescription string, rawState string, rawCreatedBy string) seriesRowResult {
	idResult := core.ParseTaskSeriesID(rawID)
	seriesID, idMatched := idResult.(core.TaskSeriesIDCreated)
	if !idMatched {
		return seriesRowRejected{reason: idResult.(core.TaskSeriesIDRejected).Reason}
	}
	ownerResult := parseTaskOwner(rawOwnerKind, rawOwnerUserID, rawOwnerTeamID, rawOwnerOrganizationID)
	owner, ownerMatched := ownerResult.(taskOwnerAccepted)
	if !ownerMatched {
		return seriesRowRejected{reason: ownerResult.(taskOwnerRejected).reason}
	}
	titleResult := task.NewSeriesTitle(rawTitle)
	title, titleMatched := titleResult.(task.SeriesTitleAccepted)
	if !titleMatched {
		return seriesRowRejected{reason: titleResult.(task.SeriesTitleRejected).Reason}
	}
	createdByResult := core.ParseUserID(rawCreatedBy)
	createdBy, createdByMatched := createdByResult.(core.UserIDCreated)
	if !createdByMatched {
		return seriesRowRejected{reason: createdByResult.(core.UserIDRejected).Reason}
	}
	descriptionResult := task.NewSeriesDescription(rawDescription)
	description, descriptionMatched := descriptionResult.(task.SeriesDescriptionAccepted)
	if !descriptionMatched {
		return seriesRowRejected{reason: descriptionResult.(task.SeriesDescriptionRejected).Reason}
	}
	stateResult := task.ParseSeriesState(rawState)
	state, stateMatched := stateResult.(task.SeriesStateAccepted)
	if !stateMatched {
		return seriesRowRejected{reason: stateResult.(task.SeriesStateRejected).Reason}
	}
	return seriesRowAccepted{value: task.Series{
		ID:          seriesID.Value,
		Owner:       owner.value,
		Title:       title.Value,
		Description: description.Value,
		State:       state.Value,
		CreatedBy:   createdBy.Value,
	}}
}
