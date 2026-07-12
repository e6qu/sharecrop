package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

func (store TaskStore) CreateTaskComment(ctx context.Context, comment task.TaskComment) task.CreateTaskCommentStoreResult {
	var createdAt time.Time
	err := store.db.QueryRow(ctx, `
		insert into task_comments (id, task_id, author_user_id, body)
		values ($1, $2, $3, $4)
		returning created_at
	`, comment.ID.String(), comment.TaskID.String(), comment.AuthorID.String(), comment.Body.String()).Scan(&createdAt)
	if err != nil {
		return task.CreateTaskCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create task comment failed")}
	}
	comment.CreatedAt = createdAt
	return task.CreateTaskCommentStoreAccepted{Value: comment}
}

func (store TaskStore) ListTaskComments(ctx context.Context, taskID core.TaskID) task.ListTaskCommentsStoreResult {
	rows, err := store.db.Query(ctx, `
		select id::text, task_id::text, author_user_id::text, body, created_at
		from task_comments
		where task_id = $1
		order by created_at, id
	`, taskID.String())
	if err != nil {
		return task.ListTaskCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list task comments failed")}
	}
	defer rows.Close()

	values := make([]task.TaskComment, 0)
	for rows.Next() {
		var rawID, rawTaskID, rawAuthor, rawBody string
		var createdAt time.Time
		if scanErr := rows.Scan(&rawID, &rawTaskID, &rawAuthor, &rawBody, &createdAt); scanErr != nil {
			return task.ListTaskCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan task comment failed")}
		}
		parsed := parseTaskComment(rawID, rawTaskID, rawAuthor, rawBody, createdAt)
		accepted, matched := parsed.(task.CreateTaskCommentStoreAccepted)
		if !matched {
			return task.ListTaskCommentsStoreRejected{Reason: parsed.(task.CreateTaskCommentStoreRejected).Reason}
		}
		values = append(values, accepted.Value)
	}
	if err := rows.Err(); err != nil {
		if errors.Is(err, ErrNoRows) {
			return task.ListTaskCommentsStoreAccepted{Values: values}
		}
		return task.ListTaskCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task comments failed")}
	}
	return task.ListTaskCommentsStoreAccepted{Values: values}
}

func parseTaskComment(rawID string, rawTaskID string, rawAuthor string, rawBody string, createdAt time.Time) task.CreateTaskCommentStoreResult {
	idResult := core.ParseTaskCommentID(rawID)
	commentID, idMatched := idResult.(core.TaskCommentIDCreated)
	if !idMatched {
		return task.CreateTaskCommentStoreRejected{Reason: idResult.(core.TaskCommentIDRejected).Reason}
	}
	taskResult := core.ParseTaskID(rawTaskID)
	taskID, taskMatched := taskResult.(core.TaskIDCreated)
	if !taskMatched {
		return task.CreateTaskCommentStoreRejected{Reason: taskResult.(core.TaskIDRejected).Reason}
	}
	authorResult := core.ParseUserID(rawAuthor)
	author, authorMatched := authorResult.(core.UserIDCreated)
	if !authorMatched {
		return task.CreateTaskCommentStoreRejected{Reason: authorResult.(core.UserIDRejected).Reason}
	}
	bodyResult := task.NewCommentBody(rawBody)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		return task.CreateTaskCommentStoreRejected{Reason: bodyResult.(task.CommentBodyRejected).Reason}
	}
	return task.CreateTaskCommentStoreAccepted{Value: task.TaskComment{
		ID:        commentID.Value,
		TaskID:    taskID.Value,
		AuthorID:  author.Value,
		Body:      body.Value,
		CreatedAt: createdAt,
	}}
}
