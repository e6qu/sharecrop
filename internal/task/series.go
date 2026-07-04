package task

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/authz"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
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

// requireSeriesViewPermission grants a series's creator view access
// outright (bypassing the draft guard below — a creator can always see
// their own draft), then blocks drafts from everyone else, including an
// org token: a draft series is creator-private even to other members of
// the owning org, and an org token has no individual creator identity to
// match. Organization/OrganizationTeam ownership routes through
// authz.RequireOrganizationAccess, matching human org-admin members
// exactly for an org token (unconditional org-id match, no fallback
// permission check) and the acting user's organization permission
// otherwise.
func (service Service) requireSeriesViewPermission(ctx context.Context, actor auth.Subject, series Series) viewPermissionResult {
	_, isOrg := actor.(auth.OrgSubject)
	userActor, isUser := actor.(auth.UserSubject)
	if !isOrg && !isUser {
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	}
	if isUser && series.CreatedBy == userActor.ID {
		return viewPermissionAccepted{}
	}
	if series.State == SeriesStateDraft {
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	}
	switch typed := series.Owner.(type) {
	case UserOwner:
		if isUser && typed.UserID == userActor.ID {
			return viewPermissionAccepted{}
		}
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	case OrganizationOwner:
		return viewPermissionResultFromDecision(authz.RequireOrganizationAccess(ctx, actor, typed.OrganizationID, service.organizationPermissions, org.PermissionCreateOrganizationTask, core.ErrorCodeInvalidState, "task series view access denied"))
	case OrganizationTeamOwner:
		return viewPermissionResultFromDecision(authz.RequireOrganizationAccess(ctx, actor, typed.OrganizationID, service.organizationPermissions, org.PermissionCreateOrganizationTask, core.ErrorCodeInvalidState, "task series view access denied"))
	default:
		return viewPermissionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "task series view access denied")}
	}
}
