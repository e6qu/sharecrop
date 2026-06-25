package task

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// Store result types for series mutations. A mutation returns the refreshed
// series detail (series plus its ordered tasks).

type SeriesMutationStoreResult interface {
	seriesMutationStoreResult()
}

type SeriesMutationStoreAccepted struct {
	Value SeriesDetail
}

type SeriesMutationStoreRejected struct {
	Reason core.DomainError
}

func (SeriesMutationStoreAccepted) seriesMutationStoreResult() {}

func (SeriesMutationStoreRejected) seriesMutationStoreResult() {}

type CreateSeriesCommentStoreResult interface {
	createSeriesCommentStoreResult()
}

type CreateSeriesCommentStoreAccepted struct {
	Value SeriesComment
}

type CreateSeriesCommentStoreRejected struct {
	Reason core.DomainError
}

func (CreateSeriesCommentStoreAccepted) createSeriesCommentStoreResult() {}

func (CreateSeriesCommentStoreRejected) createSeriesCommentStoreResult() {}

type ListSeriesCommentsStoreResult interface {
	listSeriesCommentsStoreResult()
}

type ListSeriesCommentsStoreAccepted struct {
	Values []SeriesComment
}

type ListSeriesCommentsStoreRejected struct {
	Reason core.DomainError
}

func (ListSeriesCommentsStoreAccepted) listSeriesCommentsStoreResult() {}

func (ListSeriesCommentsStoreRejected) listSeriesCommentsStoreResult() {}

// Service result types.

type SeriesMutationResult interface {
	seriesMutationResult()
}

type SeriesMutated struct {
	Value SeriesDetail
}

type SeriesMutationRejected struct {
	Reason core.DomainError
}

func (SeriesMutated) seriesMutationResult() {}

func (SeriesMutationRejected) seriesMutationResult() {}

type SeriesCommentResult interface {
	seriesCommentResult()
}

type SeriesCommentAdded struct {
	Value SeriesComment
}

type SeriesCommentRejected struct {
	Reason core.DomainError
}

func (SeriesCommentAdded) seriesCommentResult() {}

func (SeriesCommentRejected) seriesCommentResult() {}

type SeriesCommentsResult interface {
	seriesCommentsResult()
}

type SeriesCommentsListed struct {
	Values []SeriesComment
}

type SeriesCommentsListRejected struct {
	Reason core.DomainError
}

func (SeriesCommentsListed) seriesCommentsResult() {}

func (SeriesCommentsListRejected) seriesCommentsResult() {}

func (service Service) requireSeriesEditPermission(actor auth.UserSubject, series Series) viewPermissionResult {
	if series.CreatedBy == actor.ID {
		return viewPermissionAccepted{}
	}
	return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "only the series creator can edit it")}
}

func (service Service) CreateSeries(ctx context.Context, actor auth.UserSubject, title SeriesTitle, description SeriesDescription) SeriesMutationResult {
	idResult := core.NewTaskSeriesID()
	created, matched := idResult.(core.TaskSeriesIDCreated)
	if !matched {
		return SeriesMutationRejected{Reason: idResult.(core.TaskSeriesIDRejected).Reason}
	}
	series := Series{
		ID:          created.Value,
		Owner:       UserOwner{UserID: actor.ID},
		Title:       title,
		Description: description,
		State:       SeriesStateDraft,
		CreatedBy:   actor.ID,
	}
	return service.toSeriesMutationResult(service.store.CreateSeries(ctx, series))
}

func (service Service) UpdateSeries(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID, title SeriesTitle, description SeriesDescription) SeriesMutationResult {
	series, problem := service.loadEditableSeries(ctx, actor, seriesID)
	if problem != nil {
		return SeriesMutationRejected{Reason: *problem}
	}
	_ = series
	return service.toSeriesMutationResult(service.store.UpdateSeries(ctx, seriesID, title, description))
}

func (service Service) ChangeSeriesState(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID, transition SeriesStateTransition) SeriesMutationResult {
	series, problem := service.loadEditableSeries(ctx, actor, seriesID)
	if problem != nil {
		return SeriesMutationRejected{Reason: *problem}
	}
	transitionResult := transition(series.State)
	accepted, matched := transitionResult.(SeriesStateTransitionAccepted)
	if !matched {
		return SeriesMutationRejected{Reason: transitionResult.(SeriesStateTransitionRejected).Reason}
	}
	return service.toSeriesMutationResult(service.store.UpdateSeriesState(ctx, seriesID, accepted.Value))
}

func (service Service) AddTaskToSeries(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID, taskID core.TaskID) SeriesMutationResult {
	_, problem := service.loadEditableSeries(ctx, actor, seriesID)
	if problem != nil {
		return SeriesMutationRejected{Reason: *problem}
	}
	taskResult := service.store.FindTask(ctx, taskID)
	taskFound, taskMatched := taskResult.(FindTaskStoreAccepted)
	if !taskMatched {
		return SeriesMutationRejected{Reason: taskResult.(FindTaskStoreRejected).Reason}
	}
	if taskFound.Value.CreatedBy != actor.ID {
		return SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only tasks you created can be added to your series")}
	}
	return service.toSeriesMutationResult(service.store.AddTaskToSeries(ctx, seriesID, taskID))
}

func (service Service) RemoveTaskFromSeries(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID, taskID core.TaskID) SeriesMutationResult {
	_, problem := service.loadEditableSeries(ctx, actor, seriesID)
	if problem != nil {
		return SeriesMutationRejected{Reason: *problem}
	}
	return service.toSeriesMutationResult(service.store.RemoveTaskFromSeries(ctx, seriesID, taskID))
}

func (service Service) ReorderSeries(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID, order []core.TaskID) SeriesMutationResult {
	_, problem := service.loadEditableSeries(ctx, actor, seriesID)
	if problem != nil {
		return SeriesMutationRejected{Reason: *problem}
	}
	return service.toSeriesMutationResult(service.store.ReorderSeries(ctx, seriesID, order))
}

func (service Service) AddSeriesComment(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID, body CommentBody) SeriesCommentResult {
	series, problem := service.loadViewableSeries(ctx, actor, seriesID)
	if problem != nil {
		return SeriesCommentRejected{Reason: *problem}
	}
	idResult := core.NewSeriesCommentID()
	created, matched := idResult.(core.SeriesCommentIDCreated)
	if !matched {
		return SeriesCommentRejected{Reason: idResult.(core.SeriesCommentIDRejected).Reason}
	}
	comment := SeriesComment{
		ID:       created.Value,
		SeriesID: series.ID,
		AuthorID: actor.ID,
		Body:     body,
	}
	storeResult := service.store.CreateSeriesComment(ctx, comment)
	accepted, accepted_ := storeResult.(CreateSeriesCommentStoreAccepted)
	if !accepted_ {
		return SeriesCommentRejected{Reason: storeResult.(CreateSeriesCommentStoreRejected).Reason}
	}
	return SeriesCommentAdded{Value: accepted.Value}
}

func (service Service) ListSeriesComments(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID) SeriesCommentsResult {
	_, problem := service.loadViewableSeries(ctx, actor, seriesID)
	if problem != nil {
		return SeriesCommentsListRejected{Reason: *problem}
	}
	storeResult := service.store.ListSeriesComments(ctx, seriesID)
	listed, matched := storeResult.(ListSeriesCommentsStoreAccepted)
	if !matched {
		return SeriesCommentsListRejected{Reason: storeResult.(ListSeriesCommentsStoreRejected).Reason}
	}
	return SeriesCommentsListed{Values: listed.Values}
}

// loadEditableSeries finds a series and verifies the actor is its creator.
func (service Service) loadEditableSeries(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID) (Series, *core.DomainError) {
	storeResult := service.store.FindSeries(ctx, seriesID)
	found, matched := storeResult.(FindSeriesStoreAccepted)
	if !matched {
		reason := storeResult.(FindSeriesStoreRejected).Reason
		return Series{}, &reason
	}
	permission := service.requireSeriesEditPermission(actor, found.Value.Series)
	if rejected, denied := permission.(viewPermissionRejected); denied {
		return Series{}, &rejected.reason
	}
	return found.Value.Series, nil
}

// loadViewableSeries finds a series and verifies the actor may view it.
func (service Service) loadViewableSeries(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID) (Series, *core.DomainError) {
	storeResult := service.store.FindSeries(ctx, seriesID)
	found, matched := storeResult.(FindSeriesStoreAccepted)
	if !matched {
		reason := storeResult.(FindSeriesStoreRejected).Reason
		return Series{}, &reason
	}
	permission := service.requireSeriesViewPermission(ctx, actor, found.Value.Series)
	if rejected, denied := permission.(viewPermissionRejected); denied {
		return Series{}, &rejected.reason
	}
	return found.Value.Series, nil
}

func (service Service) toSeriesMutationResult(storeResult SeriesMutationStoreResult) SeriesMutationResult {
	accepted, matched := storeResult.(SeriesMutationStoreAccepted)
	if !matched {
		return SeriesMutationRejected{Reason: storeResult.(SeriesMutationStoreRejected).Reason}
	}
	return SeriesMutated{Value: accepted.Value}
}
