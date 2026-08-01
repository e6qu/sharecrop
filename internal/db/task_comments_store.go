package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/task"
)

func (store TaskStore) CreateTaskComment(ctx context.Context, comment task.TaskComment, draft event.Draft) task.CreateTaskCommentStoreResult {
	tx, txErr := store.db.Begin(ctx)
	if txErr != nil {
		return task.CreateTaskCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin create task comment transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		insert into task_comments (id, task_id, author_user_id, body)
		values ($1, $2, $3, $4)
		returning created_at
	`, comment.ID.String(), comment.TaskID.String(), comment.AuthorID.String(), comment.Body.String()).Scan(&createdAt)
	if err != nil {
		return task.CreateTaskCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create task comment failed")}
	}
	if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
		return task.CreateTaskCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record task comment event failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return task.CreateTaskCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create task comment transaction failed")}
	}
	comment.CreatedAt = createdAt
	authorName, nameReason := fetchUserDisplayName(ctx, store.db, comment.AuthorID)
	if nameReason != nil {
		return task.CreateTaskCommentStoreRejected{Reason: *nameReason}
	}
	comment.AuthorDisplayName = authorName
	return task.CreateTaskCommentStoreAccepted{Value: comment}
}

func (store TaskStore) ListTaskComments(ctx context.Context, taskID core.TaskID, page core.Page) task.ListTaskCommentsStoreResult {
	rows, err := store.db.Query(ctx, `
		select task_comments.id::text, task_comments.task_id::text, task_comments.author_user_id::text,
			`+displayNameSQL("users")+`, task_comments.body, task_comments.created_at
		from task_comments
		join users on users.id = task_comments.author_user_id
		where task_comments.task_id = $1
		order by task_comments.created_at desc, task_comments.id desc
		limit $2 offset $3
	`, taskID.String(), page.Limit(), page.Offset())
	if err != nil {
		return task.ListTaskCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list task comments failed")}
	}
	defer rows.Close()

	values := make([]task.TaskComment, 0)
	for rows.Next() {
		var rawID, rawTaskID, rawAuthor, rawAuthorName, rawBody string
		var createdAt time.Time
		if scanErr := rows.Scan(&rawID, &rawTaskID, &rawAuthor, &rawAuthorName, &rawBody, &createdAt); scanErr != nil {
			return task.ListTaskCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan task comment failed")}
		}
		parsed := parseTaskComment(rawID, rawTaskID, rawAuthor, rawAuthorName, rawBody, createdAt)
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

func parseTaskComment(rawID string, rawTaskID string, rawAuthor string, rawAuthorName string, rawBody string, createdAt time.Time) task.CreateTaskCommentStoreResult {
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
	authorNameResult := auth.NewDisplayName(rawAuthorName)
	authorName, authorNameMatched := authorNameResult.(auth.DisplayNameAccepted)
	if !authorNameMatched {
		return task.CreateTaskCommentStoreRejected{Reason: authorNameResult.(auth.DisplayNameRejected).Reason}
	}
	return task.CreateTaskCommentStoreAccepted{Value: task.TaskComment{
		ID:                commentID.Value,
		TaskID:            taskID.Value,
		AuthorID:          author.Value,
		AuthorDisplayName: authorName.Value,
		Body:              body.Value,
		CreatedAt:         createdAt,
	}}
}
