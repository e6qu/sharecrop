package task

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// SeriesDetail is a task series together with its ordered tasks.
type SeriesDetail struct {
	Series Series
	Tasks  []Task
}

type ListSeriesStoreResult interface {
	listSeriesStoreResult()
}

type ListSeriesStoreAccepted struct {
	Values []Series
}

type ListSeriesStoreRejected struct {
	Reason core.DomainError
}

func (ListSeriesStoreAccepted) listSeriesStoreResult() {}

func (ListSeriesStoreRejected) listSeriesStoreResult() {}

type FindSeriesStoreResult interface {
	findSeriesStoreResult()
}

type FindSeriesStoreAccepted struct {
	Value SeriesDetail
}

type FindSeriesStoreRejected struct {
	Reason core.DomainError
}

func (FindSeriesStoreAccepted) findSeriesStoreResult() {}

func (FindSeriesStoreRejected) findSeriesStoreResult() {}

type ListSeriesResult interface {
	listSeriesResult()
}

type SeriesListed struct {
	Values []Series
}

type ListSeriesRejected struct {
	Reason core.DomainError
}

func (SeriesListed) listSeriesResult() {}

func (ListSeriesRejected) listSeriesResult() {}

type GetSeriesResult interface {
	getSeriesResult()
}

type SeriesGot struct {
	Value SeriesDetail
}

type GetSeriesRejected struct {
	Reason core.DomainError
}

func (SeriesGot) getSeriesResult() {}

func (GetSeriesRejected) getSeriesResult() {}

func (service Service) ListSeries(ctx context.Context, actor auth.UserSubject) ListSeriesResult {
	storeResult := service.store.ListSeries(ctx, actor.ID)
	listed, matched := storeResult.(ListSeriesStoreAccepted)
	if !matched {
		return ListSeriesRejected{Reason: storeResult.(ListSeriesStoreRejected).Reason}
	}
	return SeriesListed{Values: listed.Values}
}

func (service Service) GetSeries(ctx context.Context, actor auth.UserSubject, seriesID core.TaskSeriesID) GetSeriesResult {
	storeResult := service.store.FindSeries(ctx, seriesID)
	found, matched := storeResult.(FindSeriesStoreAccepted)
	if !matched {
		return GetSeriesRejected{Reason: storeResult.(FindSeriesStoreRejected).Reason}
	}

	permission := service.requireSeriesViewPermission(ctx, actor, found.Value.Series)
	if rejected, denied := permission.(viewPermissionRejected); denied {
		return GetSeriesRejected{Reason: rejected.reason}
	}
	return SeriesGot{Value: found.Value}
}

func (service Service) requireSeriesViewPermission(ctx context.Context, actor auth.UserSubject, series Series) viewPermissionResult {
	if series.CreatedBy == actor.ID {
		return viewPermissionAccepted{}
	}
	switch typed := series.Owner.(type) {
	case UserOwner:
		if typed.UserID == actor.ID {
			return viewPermissionAccepted{}
		}
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	case OrganizationOwner:
		return service.requireOrganizationViewPermission(ctx, typed.OrganizationID, actor.ID)
	case OrganizationTeamOwner:
		return service.requireOrganizationViewPermission(ctx, typed.OrganizationID, actor.ID)
	default:
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	}
}
