package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

func (store SubmissionStore) CreateSubmissionComment(ctx context.Context, comment submission.SubmissionComment, draft event.Draft) submission.CreateSubmissionCommentStoreResult {
	tx, txErr := store.db.Begin(ctx)
	if txErr != nil {
		return submission.CreateSubmissionCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin create submission comment transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var createdAt time.Time
	err := tx.QueryRow(ctx, `
		insert into submission_comments (id, submission_id, author_user_id, body)
		values ($1, $2, $3, $4)
		returning created_at
	`, comment.ID.String(), comment.SubmissionID.String(), comment.AuthorID.String(), comment.Body.String()).Scan(&createdAt)
	if err != nil {
		return submission.CreateSubmissionCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create submission comment failed")}
	}
	if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
		return submission.CreateSubmissionCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record submission comment event failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return submission.CreateSubmissionCommentStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create submission comment transaction failed")}
	}
	comment.CreatedAt = createdAt
	authorName, nameReason := fetchUserDisplayName(ctx, store.db, comment.AuthorID)
	if nameReason != nil {
		return submission.CreateSubmissionCommentStoreRejected{Reason: *nameReason}
	}
	comment.AuthorDisplayName = authorName
	return submission.CreateSubmissionCommentStoreAccepted{Value: comment}
}

func (store SubmissionStore) ListSubmissionComments(ctx context.Context, submissionID core.SubmissionID, page core.Page) submission.ListSubmissionCommentsStoreResult {
	rows, err := store.db.Query(ctx, `
		select submission_comments.id::text, submission_comments.submission_id::text, submission_comments.author_user_id::text,
			`+displayNameSQL("users")+`, submission_comments.body, submission_comments.created_at
		from submission_comments
		join users on users.id = submission_comments.author_user_id
		where submission_comments.submission_id = $1
		order by submission_comments.created_at desc, submission_comments.id desc
		limit $2 offset $3
	`, submissionID.String(), page.Limit(), page.Offset())
	if err != nil {
		return submission.ListSubmissionCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list submission comments failed")}
	}
	defer rows.Close()

	values := make([]submission.SubmissionComment, 0)
	for rows.Next() {
		var rawID, rawSubmissionID, rawAuthor, rawAuthorName, rawBody string
		var createdAt time.Time
		if scanErr := rows.Scan(&rawID, &rawSubmissionID, &rawAuthor, &rawAuthorName, &rawBody, &createdAt); scanErr != nil {
			return submission.ListSubmissionCommentsStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan submission comment failed")}
		}
		parsed := parseSubmissionComment(rawID, rawSubmissionID, rawAuthor, rawAuthorName, rawBody, createdAt)
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

func parseSubmissionComment(rawID string, rawSubmissionID string, rawAuthor string, rawAuthorName string, rawBody string, createdAt time.Time) submission.CreateSubmissionCommentStoreResult {
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
	authorNameResult := auth.NewDisplayName(rawAuthorName)
	authorName, authorNameMatched := authorNameResult.(auth.DisplayNameAccepted)
	if !authorNameMatched {
		return submission.CreateSubmissionCommentStoreRejected{Reason: authorNameResult.(auth.DisplayNameRejected).Reason}
	}
	return submission.CreateSubmissionCommentStoreAccepted{Value: submission.SubmissionComment{
		ID:                commentID.Value,
		SubmissionID:      submissionID.Value,
		AuthorID:          author.Value,
		AuthorDisplayName: authorName.Value,
		Body:              body.Value,
		CreatedAt:         createdAt,
	}}
}
