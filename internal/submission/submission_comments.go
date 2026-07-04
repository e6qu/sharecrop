package submission

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

// SubmissionComment is one message on a private thread attached to a single
// submission, so the submission's author and the owner of the submission's task
// can exchange clarifying messages while the submission is under review.
type SubmissionComment struct {
	ID           core.SubmissionCommentID
	SubmissionID core.SubmissionID
	AuthorID     core.UserID
	Body         task.CommentBody
	CreatedAt    time.Time
}

type CreateSubmissionCommentStoreResult interface {
	createSubmissionCommentStoreResult()
}

type CreateSubmissionCommentStoreAccepted struct {
	Value SubmissionComment
}

type CreateSubmissionCommentStoreRejected struct {
	Reason core.DomainError
}

func (CreateSubmissionCommentStoreAccepted) createSubmissionCommentStoreResult() {}

func (CreateSubmissionCommentStoreRejected) createSubmissionCommentStoreResult() {}

type ListSubmissionCommentsStoreResult interface {
	listSubmissionCommentsStoreResult()
}

type ListSubmissionCommentsStoreAccepted struct {
	Values []SubmissionComment
}

type ListSubmissionCommentsStoreRejected struct {
	Reason core.DomainError
}

func (ListSubmissionCommentsStoreAccepted) listSubmissionCommentsStoreResult() {}

func (ListSubmissionCommentsStoreRejected) listSubmissionCommentsStoreResult() {}

type SubmissionCommentResult interface {
	submissionCommentResult()
}

type SubmissionCommentAdded struct {
	Value         SubmissionComment
	TaskID        core.TaskID
	SubmitterID   core.UserID
	TaskCreatorID core.UserID
}

type SubmissionCommentRejected struct {
	Reason core.DomainError
}

func (SubmissionCommentAdded) submissionCommentResult() {}

func (SubmissionCommentRejected) submissionCommentResult() {}

type SubmissionCommentsResult interface {
	submissionCommentsResult()
}

type SubmissionCommentsListed struct {
	Values []SubmissionComment
}

type SubmissionCommentsListRejected struct {
	Reason core.DomainError
}

func (SubmissionCommentsListed) submissionCommentsResult() {}

func (SubmissionCommentsListRejected) submissionCommentsResult() {}

func (service Service) AddSubmissionComment(ctx context.Context, actor auth.UserSubject, submissionID core.SubmissionID, body task.CommentBody) SubmissionCommentResult {
	value, taskValue, problem := service.loadCommentableSubmissionWithTask(ctx, actor, submissionID)
	if problem != nil {
		return SubmissionCommentRejected{Reason: *problem}
	}
	idResult := core.NewSubmissionCommentID()
	created, matched := idResult.(core.SubmissionCommentIDCreated)
	if !matched {
		return SubmissionCommentRejected{Reason: idResult.(core.SubmissionCommentIDRejected).Reason}
	}
	comment := SubmissionComment{ID: created.Value, SubmissionID: value.ID, AuthorID: actor.ID, Body: body}
	storeResult := service.store.CreateSubmissionComment(ctx, comment)
	accepted, storedMatched := storeResult.(CreateSubmissionCommentStoreAccepted)
	if !storedMatched {
		return SubmissionCommentRejected{Reason: storeResult.(CreateSubmissionCommentStoreRejected).Reason}
	}
	return SubmissionCommentAdded{Value: accepted.Value, TaskID: value.TaskID, SubmitterID: value.SubmitterID, TaskCreatorID: taskValue.CreatedBy}
}

func (service Service) ListSubmissionComments(ctx context.Context, actor auth.Subject, submissionID core.SubmissionID) SubmissionCommentsResult {
	_, problem := service.loadCommentableSubmission(ctx, actor, submissionID)
	if problem != nil {
		return SubmissionCommentsListRejected{Reason: *problem}
	}
	storeResult := service.store.ListSubmissionComments(ctx, submissionID)
	listed, matched := storeResult.(ListSubmissionCommentsStoreAccepted)
	if !matched {
		return SubmissionCommentsListRejected{Reason: storeResult.(ListSubmissionCommentsStoreRejected).Reason}
	}
	return SubmissionCommentsListed{Values: listed.Values}
}

// loadCommentableSubmission finds the submission and permits commenting only for
// the submission's author or the owner of the submission's task. Other actors
// are denied so the thread stays private to those two parties.
func (service Service) loadCommentableSubmission(ctx context.Context, actor auth.Subject, submissionID core.SubmissionID) (Submission, *core.DomainError) {
	value, _, problem := service.loadCommentableSubmissionWithTask(ctx, actor, submissionID)
	return value, problem
}

func (service Service) loadCommentableSubmissionWithTask(ctx context.Context, actor auth.Subject, submissionID core.SubmissionID) (Submission, task.Task, *core.DomainError) {
	submissionResult := service.store.FindSubmission(ctx, submissionID)
	found, matched := submissionResult.(FindSubmissionStoreAccepted)
	if !matched {
		reason := submissionResult.(FindSubmissionStoreRejected).Reason
		return Submission{}, task.Task{}, &reason
	}
	taskResult := service.taskStore.FindTask(ctx, found.Value.TaskID)
	taskFound, taskMatched := taskResult.(task.FindTaskStoreAccepted)
	if !taskMatched {
		reason := taskResult.(task.FindTaskStoreRejected).Reason
		return Submission{}, task.Task{}, &reason
	}
	// An org token is never "the submitter" (submitting is an individual
	// act), so it always goes through the review-permission check below.
	if userActor, isUser := actor.(auth.UserSubject); isUser && found.Value.SubmitterID == userActor.ID {
		return found.Value, taskFound.Value, nil
	}
	if rejected, denied := service.requireReviewPermissionForActor(ctx, actor, taskFound.Value).(reviewPermissionRejected); denied {
		return Submission{}, task.Task{}, &rejected.reason
	}
	return found.Value, taskFound.Value, nil
}
