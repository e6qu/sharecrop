package task

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
)

// TaskComment is one message on a task discussion thread, so requester and
// worker can exchange clarifying questions on a detailed task. It carries the
// author's display name, resolved by the store, so threads can name the
// author without a per-row user fetch.
type TaskComment struct {
	ID                core.TaskCommentID
	TaskID            core.TaskID
	AuthorID          core.UserID
	AuthorDisplayName auth.DisplayName
	Body              CommentBody
	CreatedAt         time.Time
}

type CreateTaskCommentStoreResult interface {
	createTaskCommentStoreResult()
}

type CreateTaskCommentStoreAccepted struct {
	Value TaskComment
}

type CreateTaskCommentStoreRejected struct {
	Reason core.DomainError
}

func (CreateTaskCommentStoreAccepted) createTaskCommentStoreResult() {}

func (CreateTaskCommentStoreRejected) createTaskCommentStoreResult() {}

type ListTaskCommentsStoreResult interface {
	listTaskCommentsStoreResult()
}

type ListTaskCommentsStoreAccepted struct {
	Values []TaskComment
}

type ListTaskCommentsStoreRejected struct {
	Reason core.DomainError
}

func (ListTaskCommentsStoreAccepted) listTaskCommentsStoreResult() {}

func (ListTaskCommentsStoreRejected) listTaskCommentsStoreResult() {}

type TaskCommentResult interface {
	taskCommentResult()
}

type TaskCommentAdded struct {
	Value TaskComment
}

type TaskCommentRejected struct {
	Reason core.DomainError
}

func (TaskCommentAdded) taskCommentResult() {}

func (TaskCommentRejected) taskCommentResult() {}

type TaskCommentsResult interface {
	taskCommentsResult()
}

type TaskCommentsListed struct {
	Values []TaskComment
}

type TaskCommentsListRejected struct {
	Reason core.DomainError
}

func (TaskCommentsListed) taskCommentsResult() {}

func (TaskCommentsListRejected) taskCommentsResult() {}

func (service Service) AddTaskComment(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, body CommentBody) TaskCommentResult {
	value, problem := service.loadViewableTask(ctx, actor, taskID)
	if problem != nil {
		return TaskCommentRejected{Reason: *problem}
	}
	idResult := core.NewTaskCommentID()
	created, matched := idResult.(core.TaskCommentIDCreated)
	if !matched {
		return TaskCommentRejected{Reason: idResult.(core.TaskCommentIDRejected).Reason}
	}
	comment := TaskComment{ID: created.Value, TaskID: value.ID, AuthorID: actor.ID, Body: body}
	// The active assignee (if one holds the task) is read via the existing
	// reservation listing so both discussion parties hear about the comment;
	// the event is recorded inside the comment's insert transaction.
	recipients := append([]core.UserID{value.CreatedBy, actor.ID}, service.activeReservationHolders(ctx, value.ID)...)
	draft, draftProblem := taskEventDraft(event.KindTaskCommented, event.ActorUser{ID: actor.ID}, value.ID,
		event.TaskMetadata(value.ID), event.NewRecipients(recipients...))
	if draftProblem != nil {
		return TaskCommentRejected{Reason: *draftProblem}
	}
	storeResult := service.store.CreateTaskComment(ctx, comment, draft)
	accepted, accepted_ := storeResult.(CreateTaskCommentStoreAccepted)
	if !accepted_ {
		return TaskCommentRejected{Reason: storeResult.(CreateTaskCommentStoreRejected).Reason}
	}
	service.recorder.Dispatch(ctx, draft)
	return TaskCommentAdded{Value: accepted.Value}
}

func (service Service) ListTaskComments(ctx context.Context, actor auth.UserSubject, taskID core.TaskID, page core.Page) TaskCommentsResult {
	_, problem := service.loadViewableTask(ctx, actor, taskID)
	if problem != nil {
		return TaskCommentsListRejected{Reason: *problem}
	}
	storeResult := service.store.ListTaskComments(ctx, taskID, page)
	listed, matched := storeResult.(ListTaskCommentsStoreAccepted)
	if !matched {
		return TaskCommentsListRejected{Reason: storeResult.(ListTaskCommentsStoreRejected).Reason}
	}
	return TaskCommentsListed{Values: listed.Values}
}

func (service Service) loadViewableTask(ctx context.Context, actor auth.UserSubject, taskID core.TaskID) (Task, *core.DomainError) {
	taskResult := service.store.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(FindTaskStoreAccepted)
	if !taskMatched {
		reason := taskResult.(FindTaskStoreRejected).Reason
		return Task{}, &reason
	}
	if rejected, denied := service.requireViewPermission(ctx, actor, taskFound.Value).(viewPermissionRejected); denied {
		return Task{}, &rejected.reason
	}
	return taskFound.Value, nil
}
