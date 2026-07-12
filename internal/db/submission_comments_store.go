package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

func (store SubmissionStore) CreateSubmissionComment(ctx context.Context, comment submission.SubmissionComment) submission.CreateSubmissionCommentStoreResult {
	var createdAt time.Time
	err := store.db.QueryRow(ctx, `
		insert into submission_comments (id, submission_id, author_user_id, body)
		values ($1, $2, $3, $4)
		returning created_at
	`, comment.ID.String(), comment.SubmissionID.String(), comment.AuthorID.String(), comment.Body.String()).Scan(&createdAt)
	if err != nil {
		return submission.CreateSubmissionCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create submission comment failed")}
	}
	comment.CreatedAt = createdAt
	return submission.CreateSubmissionCommentStoreAccepted{Value: comment}
}

func (store SubmissionStore) ListSubmissionComments(ctx context.Context, submissionID core.SubmissionID) submission.ListSubmissionCommentsStoreResult {
	rows, err := store.db.Query(ctx, `
		select id::text, submission_id::text, author_user_id::text, body, created_at
		from submission_comments
		where submission_id = $1
		order by created_at, id
	`, submissionID.String())
	if err != nil {
		return submission.ListSubmissionCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list submission comments failed")}
	}
	defer rows.Close()

	values := make([]submission.SubmissionComment, 0)
	for rows.Next() {
		var rawID, rawSubmissionID, rawAuthor, rawBody string
		var createdAt time.Time
		if scanErr := rows.Scan(&rawID, &rawSubmissionID, &rawAuthor, &rawBody, &createdAt); scanErr != nil {
			return submission.ListSubmissionCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan submission comment failed")}
		}
		parsed := parseSubmissionComment(rawID, rawSubmissionID, rawAuthor, rawBody, createdAt)
		accepted, matched := parsed.(submission.CreateSubmissionCommentStoreAccepted)
		if !matched {
			return submission.ListSubmissionCommentsStoreRejected{Reason: parsed.(submission.CreateSubmissionCommentStoreRejected).Reason}
		}
		values = append(values, accepted.Value)
	}
	if err := rows.Err(); err != nil {
		if errors.Is(err, ErrNoRows) {
			return submission.ListSubmissionCommentsStoreAccepted{Values: values}
		}
		return submission.ListSubmissionCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read submission comments failed")}
	}
	return submission.ListSubmissionCommentsStoreAccepted{Values: values}
}

func parseSubmissionComment(rawID string, rawSubmissionID string, rawAuthor string, rawBody string, createdAt time.Time) submission.CreateSubmissionCommentStoreResult {
	idResult := core.ParseSubmissionCommentID(rawID)
	commentID, idMatched := idResult.(core.SubmissionCommentIDCreated)
	if !idMatched {
		return submission.CreateSubmissionCommentStoreRejected{Reason: idResult.(core.SubmissionCommentIDRejected).Reason}
	}
	submissionResult := core.ParseSubmissionID(rawSubmissionID)
	submissionID, submissionMatched := submissionResult.(core.SubmissionIDCreated)
	if !submissionMatched {
		return submission.CreateSubmissionCommentStoreRejected{Reason: submissionResult.(core.SubmissionIDRejected).Reason}
	}
	authorResult := core.ParseUserID(rawAuthor)
	author, authorMatched := authorResult.(core.UserIDCreated)
	if !authorMatched {
		return submission.CreateSubmissionCommentStoreRejected{Reason: authorResult.(core.UserIDRejected).Reason}
	}
	bodyResult := task.NewCommentBody(rawBody)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		return submission.CreateSubmissionCommentStoreRejected{Reason: bodyResult.(task.CommentBodyRejected).Reason}
	}
	return submission.CreateSubmissionCommentStoreAccepted{Value: submission.SubmissionComment{
		ID:           commentID.Value,
		SubmissionID: submissionID.Value,
		AuthorID:     author.Value,
		Body:         body.Value,
		CreatedAt:    createdAt,
	}}
}
