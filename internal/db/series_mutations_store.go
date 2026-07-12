package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

func (store TaskStore) CreateSeries(ctx context.Context, series task.Series) task.SeriesMutationStoreResult {
	ownerColumns := ownerSQLColumns(series.Owner)
	_, err := store.db.Exec(ctx, `
		insert into task_series (id, owner_kind, user_id, team_id, organization_id, title, description, state, created_by_user_id)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, series.ID.String(), ownerColumns.kind, ownerColumns.userID, ownerColumns.teamID, ownerColumns.organizationID,
		series.Title.String(), series.Description.String(), series.State.String(), series.CreatedBy.String())
	if err != nil {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create task series failed")}
	}
	return store.seriesMutationDetail(ctx, series.ID)
}

func (store TaskStore) UpdateSeries(ctx context.Context, seriesID core.TaskSeriesID, title task.SeriesTitle, description task.SeriesDescription) task.SeriesMutationStoreResult {
	_, err := store.db.Exec(ctx, `
		update task_series set title = $2, description = $3, updated_at = now() where id = $1
	`, seriesID.String(), title.String(), description.String())
	if err != nil {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update task series failed")}
	}
	return store.seriesMutationDetail(ctx, seriesID)
}

func (store TaskStore) UpdateSeriesState(ctx context.Context, seriesID core.TaskSeriesID, state task.SeriesState) task.SeriesMutationStoreResult {
	_, err := store.db.Exec(ctx, `
		update task_series set state = $2, updated_at = now() where id = $1
	`, seriesID.String(), state.String())
	if err != nil {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update task series state failed")}
	}
	return store.seriesMutationDetail(ctx, seriesID)
}

func (store TaskStore) AddTaskToSeries(ctx context.Context, seriesID core.TaskSeriesID, taskID core.TaskID) task.SeriesMutationStoreResult {
	tag, err := store.db.Exec(ctx, `
		update tasks
		set series_id = $1,
			series_position = (select coalesce(max(series_position), 0) + 1 from tasks where series_id = $1)
		where id = $2
	`, seriesID.String(), taskID.String())
	if err != nil {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "add task to series failed")}
	}
	if tag == 0 {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task was not found")}
	}
	return store.seriesMutationDetail(ctx, seriesID)
}

func (store TaskStore) RemoveTaskFromSeries(ctx context.Context, seriesID core.TaskSeriesID, taskID core.TaskID) task.SeriesMutationStoreResult {
	tag, err := store.db.Exec(ctx, `
		update tasks set series_id = null, series_position = null where id = $1 and series_id = $2
	`, taskID.String(), seriesID.String())
	if err != nil {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "remove task from series failed")}
	}
	if tag == 0 {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task is not in this series")}
	}
	return store.seriesMutationDetail(ctx, seriesID)
}

func (store TaskStore) ReorderSeries(ctx context.Context, seriesID core.TaskSeriesID, order []core.TaskID) task.SeriesMutationStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "reorder task series failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for index := range order {
		tag, execErr := tx.Exec(ctx, `
			update tasks set series_position = $3 where id = $1 and series_id = $2
		`, order[index].String(), seriesID.String(), index+1)
		if execErr != nil {
			return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "reorder task series failed")}
		}
		if tag == 0 {
			return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task is not in this series")}
		}
	}
	if commitErr := tx.Commit(ctx); commitErr != nil {
		return task.SeriesMutationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "reorder task series failed")}
	}
	return store.seriesMutationDetail(ctx, seriesID)
}

// taskSeriesBlocksExecution returns a rejection reason when the task belongs to
// a series that is not published, so its tasks cannot be reserved or submitted
// to. A standalone task (no series) is never blocked.
func (store TaskStore) taskSeriesBlocksExecution(ctx context.Context, taskID core.TaskID) *core.DomainError {
	var blocked bool
	err := store.db.QueryRow(ctx, `
		select exists(
			select 1 from tasks t
			join task_series s on s.id = t.series_id
			where t.id = $1 and s.state <> 'published'
		)
	`, taskID.String()).Scan(&blocked)
	if err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "check task series state failed")
		return &reason
	}
	if blocked {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "the task's series is not published")
		return &reason
	}
	return nil
}

func (store TaskStore) seriesMutationDetail(ctx context.Context, seriesID core.TaskSeriesID) task.SeriesMutationStoreResult {
	found := store.FindSeries(ctx, seriesID)
	accepted, matched := found.(task.FindSeriesStoreAccepted)
	if !matched {
		return task.SeriesMutationStoreRejected{Reason: found.(task.FindSeriesStoreRejected).Reason}
	}
	return task.SeriesMutationStoreAccepted{Value: accepted.Value}
}

func (store TaskStore) CreateSeriesComment(ctx context.Context, comment task.SeriesComment) task.CreateSeriesCommentStoreResult {
	var createdAt time.Time
	err := store.db.QueryRow(ctx, `
		insert into series_comments (id, series_id, author_user_id, body)
		values ($1, $2, $3, $4)
		returning created_at
	`, comment.ID.String(), comment.SeriesID.String(), comment.AuthorID.String(), comment.Body.String()).Scan(&createdAt)
	if err != nil {
		return task.CreateSeriesCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create series comment failed")}
	}
	comment.CreatedAt = createdAt
	return task.CreateSeriesCommentStoreAccepted{Value: comment}
}

func (store TaskStore) ListSeriesComments(ctx context.Context, seriesID core.TaskSeriesID) task.ListSeriesCommentsStoreResult {
	rows, err := store.db.Query(ctx, `
		select id::text, series_id::text, author_user_id::text, body, created_at
		from series_comments
		where series_id = $1
		order by created_at, id
	`, seriesID.String())
	if err != nil {
		return task.ListSeriesCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list series comments failed")}
	}
	defer rows.Close()

	values := make([]task.SeriesComment, 0)
	for rows.Next() {
		var rawID, rawSeriesID, rawAuthor, rawBody string
		var createdAt time.Time
		if scanErr := rows.Scan(&rawID, &rawSeriesID, &rawAuthor, &rawBody, &createdAt); scanErr != nil {
			return task.ListSeriesCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan series comment failed")}
		}
		parsed := parseSeriesComment(rawID, rawSeriesID, rawAuthor, rawBody, createdAt)
		accepted, matched := parsed.(task.CreateSeriesCommentStoreAccepted)
		if !matched {
			return task.ListSeriesCommentsStoreRejected{Reason: parsed.(task.CreateSeriesCommentStoreRejected).Reason}
		}
		values = append(values, accepted.Value)
	}
	if err := rows.Err(); err != nil {
		if errors.Is(err, ErrNoRows) {
			return task.ListSeriesCommentsStoreAccepted{Values: values}
		}
		return task.ListSeriesCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read series comments failed")}
	}
	return task.ListSeriesCommentsStoreAccepted{Values: values}
}

func parseSeriesComment(rawID string, rawSeriesID string, rawAuthor string, rawBody string, createdAt time.Time) task.CreateSeriesCommentStoreResult {
	idResult := core.ParseSeriesCommentID(rawID)
	commentID, idMatched := idResult.(core.SeriesCommentIDCreated)
	if !idMatched {
		return task.CreateSeriesCommentStoreRejected{Reason: idResult.(core.SeriesCommentIDRejected).Reason}
	}
	seriesResult := core.ParseTaskSeriesID(rawSeriesID)
	seriesID, seriesMatched := seriesResult.(core.TaskSeriesIDCreated)
	if !seriesMatched {
		return task.CreateSeriesCommentStoreRejected{Reason: seriesResult.(core.TaskSeriesIDRejected).Reason}
	}
	authorResult := core.ParseUserID(rawAuthor)
	author, authorMatched := authorResult.(core.UserIDCreated)
	if !authorMatched {
		return task.CreateSeriesCommentStoreRejected{Reason: authorResult.(core.UserIDRejected).Reason}
	}
	bodyResult := task.NewCommentBody(rawBody)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		return task.CreateSeriesCommentStoreRejected{Reason: bodyResult.(task.CommentBodyRejected).Reason}
	}
	return task.CreateSeriesCommentStoreAccepted{Value: task.SeriesComment{
		ID:        commentID.Value,
		SeriesID:  seriesID.Value,
		AuthorID:  author.Value,
		Body:      body.Value,
		CreatedAt: createdAt,
	}}
}
