package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/e6qu/sharecrop/internal/core"
)

const (
	savedQueueScopeTeamWork          = "team_work"
	savedQueueScopeOrganizationTasks = "organization_tasks"
)

type SavedQueueView struct {
	ID          string
	UserID      core.UserID
	Scope       string
	Name        string
	Query       string
	StateFilter string
	TypeFilter  string
	Sort        string
}

type SavedQueueViewService interface {
	List(context.Context, core.UserID, string) SavedQueueViewsListResult
	Upsert(context.Context, SavedQueueView) SavedQueueViewMutationResult
}

type SavedQueueViewsListResult interface {
	savedQueueViewsListResult()
}

type SavedQueueViewsListed struct {
	Values []SavedQueueView
}

type SavedQueueViewsListRejected struct {
	Reason core.DomainError
}

func (SavedQueueViewsListed) savedQueueViewsListResult() {}

func (SavedQueueViewsListRejected) savedQueueViewsListResult() {}

type SavedQueueViewMutationResult interface {
	savedQueueViewMutationResult()
}

type SavedQueueViewSaved struct {
	Value SavedQueueView
}

type SavedQueueViewSaveRejected struct {
	Reason core.DomainError
}

func (SavedQueueViewSaved) savedQueueViewMutationResult() {}

func (SavedQueueViewSaveRejected) savedQueueViewMutationResult() {}

type memorySavedQueueViewService struct {
	mu    sync.Mutex
	next  int
	views []SavedQueueView
}

func newMemorySavedQueueViewService() *memorySavedQueueViewService {
	return &memorySavedQueueViewService{views: []SavedQueueView{}}
}

func (service *memorySavedQueueViewService) List(_ context.Context, userID core.UserID, scope string) SavedQueueViewsListResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	views := make([]SavedQueueView, 0)
	for index := range service.views {
		view := service.views[index]
		if view.UserID == userID && (scope == "" || view.Scope == scope) {
			views = append(views, view)
		}
	}
	return SavedQueueViewsListed{Values: views}
}

func (service *memorySavedQueueViewService) Upsert(_ context.Context, view SavedQueueView) SavedQueueViewMutationResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	for index := range service.views {
		existing := service.views[index]
		if existing.UserID == view.UserID && existing.Scope == view.Scope && existing.Name == view.Name {
			view.ID = existing.ID
			service.views[index] = view
			return SavedQueueViewSaved{Value: view}
		}
	}
	service.next++
	view.ID = "saved-view-" + strconv.Itoa(service.next)
	service.views = append(service.views, view)
	return SavedQueueViewSaved{Value: view}
}

func (server Server) listSavedQueueViews(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if scope != "" && !validSavedQueueScope(scope) {
		writeError(w, http.StatusBadRequest, "saved queue view scope is invalid")
		return
	}
	result := server.savedQueueViews.List(r.Context(), actor.subject.ID, scope)
	listed, listedMatched := result.(SavedQueueViewsListed)
	if !listedMatched {
		writeDomainError(w, result.(SavedQueueViewsListRejected).Reason)
		return
	}
	response := savedQueueViewsResponse{Views: make([]savedQueueViewResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		response.Views = append(response.Views, savedQueueViewToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) upsertSavedQueueView(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	var request savedQueueViewRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	viewResult := savedQueueViewFromRequest(actor.subject.ID, request)
	view, viewMatched := viewResult.(savedQueueViewAccepted)
	if !viewMatched {
		writeDomainError(w, viewResult.(savedQueueViewRejected).reason)
		return
	}
	saveResult := server.savedQueueViews.Upsert(r.Context(), view.value)
	saved, savedMatched := saveResult.(SavedQueueViewSaved)
	if !savedMatched {
		writeDomainError(w, saveResult.(SavedQueueViewSaveRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, savedQueueViewToResponse(saved.Value))
}

type savedQueueViewRequestResult interface {
	savedQueueViewRequestResult()
}

type savedQueueViewAccepted struct {
	value SavedQueueView
}

type savedQueueViewRejected struct {
	reason core.DomainError
}

func (savedQueueViewAccepted) savedQueueViewRequestResult() {}

func (savedQueueViewRejected) savedQueueViewRequestResult() {}

func savedQueueViewFromRequest(userID core.UserID, request savedQueueViewRequest) savedQueueViewRequestResult {
	scope := strings.TrimSpace(request.Scope)
	name := strings.TrimSpace(request.Name)
	if !validSavedQueueScope(scope) {
		return savedQueueViewRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "saved queue view scope is invalid")}
	}
	if name == "" {
		return savedQueueViewRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "saved queue view name is required")}
	}
	return savedQueueViewAccepted{value: SavedQueueView{
		UserID:      userID,
		Scope:       scope,
		Name:        name,
		Query:       strings.TrimSpace(request.Query),
		StateFilter: strings.TrimSpace(request.StateFilter),
		TypeFilter:  strings.TrimSpace(request.TypeFilter),
		Sort:        strings.TrimSpace(request.Sort),
	}}
}

func validSavedQueueScope(scope string) bool {
	return scope == savedQueueScopeTeamWork || scope == savedQueueScopeOrganizationTasks
}

func savedQueueViewToResponse(view SavedQueueView) savedQueueViewResponse {
	return savedQueueViewResponse{
		ID:          view.ID,
		Scope:       view.Scope,
		Name:        view.Name,
		Query:       view.Query,
		StateFilter: view.StateFilter,
		TypeFilter:  view.TypeFilter,
		Sort:        view.Sort,
	}
}
