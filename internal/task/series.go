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

func (service Service) ListSeries(ctx context.Context, actor auth.UserSubject, page core.Page) ListSeriesResult {
	storeResult := service.store.ListSeries(ctx, actor.ID, page)
	listed, matched := storeResult.(ListSeriesStoreAccepted)
	if !matched {
		return ListSeriesRejected{Reason: storeResult.(ListSeriesStoreRejected).Reason}
	}
	return SeriesListed{Values: listed.Values}
}

func (service Service) GetSeries(ctx context.Context, actor auth.Subject, seriesID core.TaskSeriesID) GetSeriesResult {
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

func (service Service) requireSeriesViewPermission(ctx context.Context, actor auth.Subject, series Series) viewPermissionResult {
	if orgActor, isOrg := actor.(auth.OrgSubject); isOrg {
		// Matches human org-admin members exactly, not more: a draft series
		// is creator-private even to other members of the owning org, so an
		// org token (which has no individual creator identity to match) is
		// blocked from drafts the same way.
		if series.State == SeriesStateDraft {
			return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
		}
		switch typed := series.Owner.(type) {
		case OrganizationOwner:
			if typed.OrganizationID == orgActor.ID {
				return viewPermissionAccepted{}
			}
		case OrganizationTeamOwner:
			if typed.OrganizationID == orgActor.ID {
				return viewPermissionAccepted{}
			}
		}
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	}
	userActor, isUser := actor.(auth.UserSubject)
	if !isUser {
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	}
	if series.CreatedBy == userActor.ID {
		return viewPermissionAccepted{}
	}
	// A draft series is private to its creator until published.
	if series.State == SeriesStateDraft {
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	}
	switch typed := series.Owner.(type) {
	case UserOwner:
		if typed.UserID == userActor.ID {
			return viewPermissionAccepted{}
		}
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	case OrganizationOwner:
		return service.requireOrganizationViewPermission(ctx, typed.OrganizationID, userActor.ID)
	case OrganizationTeamOwner:
		return service.requireOrganizationViewPermission(ctx, typed.OrganizationID, userActor.ID)
	default:
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	}
}
