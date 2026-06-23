package httpserver

import (
	"net/http"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

type taskSeriesResponse struct {
	ID        string `json:"id"`
	OwnerKind string `json:"owner_kind"`
	Title     string `json:"title"`
	CreatedBy string `json:"created_by"`
}

type taskSeriesListResponse struct {
	Series []taskSeriesResponse `json:"series"`
}

type taskSeriesDetailResponse struct {
	Series taskSeriesResponse `json:"series"`
	Tasks  []taskResponse     `json:"tasks"`
}

func (taskSeriesListResponse) writableResponse() {}

func (taskSeriesDetailResponse) writableResponse() {}

func (server Server) listTaskSeries(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	result := server.taskService.ListSeries(r.Context(), actor.subject, parsePage(r))
	listed, matched := result.(task.SeriesListed)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(task.ListSeriesRejected).Reason.Description())
		return
	}

	response := taskSeriesListResponse{Series: make([]taskSeriesResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		response.Series = append(response.Series, seriesToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) getTaskSeries(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	seriesIDResult := core.ParseTaskSeriesID(r.PathValue("series_id"))
	seriesID, seriesMatched := seriesIDResult.(core.TaskSeriesIDCreated)
	if !seriesMatched {
		writeError(w, http.StatusBadRequest, seriesIDResult.(core.TaskSeriesIDRejected).Reason.Description())
		return
	}

	result := server.taskService.GetSeries(r.Context(), actor.subject, seriesID.Value)
	got, matched := result.(task.SeriesGot)
	if !matched {
		writeError(w, http.StatusForbidden, result.(task.GetSeriesRejected).Reason.Description())
		return
	}

	detail := taskSeriesDetailResponse{
		Series: seriesToResponse(got.Value.Series),
		Tasks:  make([]taskResponse, 0, len(got.Value.Tasks)),
	}
	for index := range got.Value.Tasks {
		detail.Tasks = append(detail.Tasks, taskToResponse(got.Value.Tasks[index]))
	}
	writeJSON(w, http.StatusOK, detail)
}

func seriesToResponse(value task.Series) taskSeriesResponse {
	return taskSeriesResponse{
		ID:        value.ID.String(),
		OwnerKind: taskOwnerResponseParts(value.Owner).kind,
		Title:     value.Title.String(),
		CreatedBy: value.CreatedBy.String(),
	}
}
