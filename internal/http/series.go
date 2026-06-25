package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

type taskSeriesResponse struct {
	ID          string `json:"id"`
	OwnerKind   string `json:"owner_kind"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedBy   string `json:"created_by"`
}

type seriesCommentResponse struct {
	ID           string `json:"id"`
	SeriesID     string `json:"series_id"`
	AuthorUserID string `json:"author_user_id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

type taskSeriesListResponse struct {
	Series []taskSeriesResponse `json:"series"`
}

type taskSeriesDetailResponse struct {
	Series   taskSeriesResponse      `json:"series"`
	Tasks    []taskResponse          `json:"tasks"`
	Comments []seriesCommentResponse `json:"comments"`
}

type seriesCommentsResponse struct {
	Comments []seriesCommentResponse `json:"comments"`
}

type createSeriesRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type addTaskToSeriesRequest struct {
	TaskID string `json:"task_id"`
}

type reorderSeriesRequest struct {
	TaskIDs []string `json:"task_ids"`
}

type seriesCommentRequest struct {
	Body string `json:"body"`
}

func (taskSeriesListResponse) writableResponse() {}

func (taskSeriesDetailResponse) writableResponse() {}

func (seriesCommentsResponse) writableResponse() {}

func (seriesCommentResponse) writableResponse() {}

func (server Server) listTaskSeries(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}

	result := server.taskService.ListSeries(r.Context(), actor, parsePage(r))
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
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	seriesID, ok := server.seriesPathID(w, r)
	if !ok {
		return
	}

	result := server.taskService.GetSeries(r.Context(), actor, seriesID)
	got, matched := result.(task.SeriesGot)
	if !matched {
		writeDomainError(w, result.(task.GetSeriesRejected).Reason)
		return
	}
	server.writeSeriesDetailStatus(w, r, actor, got.Value, http.StatusOK)
}

func (server Server) createTaskSeries(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	var request createSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	title, description, ok := server.seriesTitleAndDescription(w, request.Title, request.Description)
	if !ok {
		return
	}
	result := server.taskService.CreateSeries(r.Context(), actor, title, description)
	server.writeSeriesMutation(w, r, actor, result, http.StatusCreated)
}

func (server Server) updateTaskSeries(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	seriesID, ok := server.seriesPathID(w, r)
	if !ok {
		return
	}
	var request createSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	title, description, ok := server.seriesTitleAndDescription(w, request.Title, request.Description)
	if !ok {
		return
	}
	result := server.taskService.UpdateSeries(r.Context(), actor, seriesID, title, description)
	server.writeSeriesMutation(w, r, actor, result, http.StatusOK)
}

func (server Server) changeTaskSeriesState(w http.ResponseWriter, r *http.Request, transition task.SeriesStateTransition) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	seriesID, ok := server.seriesPathID(w, r)
	if !ok {
		return
	}
	result := server.taskService.ChangeSeriesState(r.Context(), actor, seriesID, transition)
	server.writeSeriesMutation(w, r, actor, result, http.StatusOK)
}

func (server Server) publishTaskSeries(w http.ResponseWriter, r *http.Request) {
	server.changeTaskSeriesState(w, r, task.PublishSeriesState)
}

func (server Server) unpublishTaskSeries(w http.ResponseWriter, r *http.Request) {
	server.changeTaskSeriesState(w, r, task.UnpublishSeriesState)
}

func (server Server) closeTaskSeries(w http.ResponseWriter, r *http.Request) {
	server.changeTaskSeriesState(w, r, task.CloseSeriesState)
}

func (server Server) reopenTaskSeries(w http.ResponseWriter, r *http.Request) {
	server.changeTaskSeriesState(w, r, task.ReopenSeriesState)
}

func (server Server) addTaskToSeriesHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	seriesID, ok := server.seriesPathID(w, r)
	if !ok {
		return
	}
	var request addTaskToSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	taskID, ok := server.parseSeriesTaskID(w, request.TaskID)
	if !ok {
		return
	}
	result := server.taskService.AddTaskToSeries(r.Context(), actor, seriesID, taskID)
	server.writeSeriesMutation(w, r, actor, result, http.StatusOK)
}

func (server Server) removeTaskFromSeriesHandler(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	seriesID, ok := server.seriesPathID(w, r)
	if !ok {
		return
	}
	taskID, ok := server.parseSeriesTaskID(w, r.PathValue("task_id"))
	if !ok {
		return
	}
	result := server.taskService.RemoveTaskFromSeries(r.Context(), actor, seriesID, taskID)
	server.writeSeriesMutation(w, r, actor, result, http.StatusOK)
}

func (server Server) reorderTaskSeries(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	seriesID, ok := server.seriesPathID(w, r)
	if !ok {
		return
	}
	var request reorderSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	order := make([]core.TaskID, 0, len(request.TaskIDs))
	for index := range request.TaskIDs {
		taskID, ok := server.parseSeriesTaskID(w, request.TaskIDs[index])
		if !ok {
			return
		}
		order = append(order, taskID)
	}
	result := server.taskService.ReorderSeries(r.Context(), actor, seriesID, order)
	server.writeSeriesMutation(w, r, actor, result, http.StatusOK)
}

func (server Server) listTaskSeriesComments(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	seriesID, ok := server.seriesPathID(w, r)
	if !ok {
		return
	}
	result := server.taskService.ListSeriesComments(r.Context(), actor, seriesID)
	listed, matched := result.(task.SeriesCommentsListed)
	if !matched {
		writeDomainError(w, result.(task.SeriesCommentsListRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, seriesCommentsResponse{Comments: commentsToResponse(listed.Values)})
}

func (server Server) addTaskSeriesComment(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	seriesID, ok := server.seriesPathID(w, r)
	if !ok {
		return
	}
	var request seriesCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	bodyResult := task.NewCommentBody(request.Body)
	body, matched := bodyResult.(task.CommentBodyAccepted)
	if !matched {
		writeError(w, http.StatusBadRequest, bodyResult.(task.CommentBodyRejected).Reason.Description())
		return
	}
	result := server.taskService.AddSeriesComment(r.Context(), actor, seriesID, body.Value)
	added, addedMatched := result.(task.SeriesCommentAdded)
	if !addedMatched {
		writeDomainError(w, result.(task.SeriesCommentRejected).Reason)
		return
	}
	writeJSON(w, http.StatusCreated, commentToResponse(added.Value))
}

// helpers

func (server Server) seriesActor(w http.ResponseWriter, r *http.Request) (auth.UserSubject, bool) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return auth.UserSubject{}, false
	}
	return actor.subject, true
}

func (server Server) seriesPathID(w http.ResponseWriter, r *http.Request) (core.TaskSeriesID, bool) {
	result := core.ParseTaskSeriesID(r.PathValue("series_id"))
	seriesID, matched := result.(core.TaskSeriesIDCreated)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(core.TaskSeriesIDRejected).Reason.Description())
		return core.TaskSeriesID{}, false
	}
	return seriesID.Value, true
}

func (server Server) parseSeriesTaskID(w http.ResponseWriter, raw string) (core.TaskID, bool) {
	result := core.ParseTaskID(raw)
	taskID, matched := result.(core.TaskIDCreated)
	if !matched {
		writeError(w, http.StatusBadRequest, result.(core.TaskIDRejected).Reason.Description())
		return core.TaskID{}, false
	}
	return taskID.Value, true
}

func (server Server) seriesTitleAndDescription(w http.ResponseWriter, rawTitle string, rawDescription string) (task.SeriesTitle, task.SeriesDescription, bool) {
	titleResult := task.NewSeriesTitle(rawTitle)
	title, titleMatched := titleResult.(task.SeriesTitleAccepted)
	if !titleMatched {
		writeError(w, http.StatusBadRequest, titleResult.(task.SeriesTitleRejected).Reason.Description())
		return task.SeriesTitle{}, task.SeriesDescription{}, false
	}
	descriptionResult := task.NewSeriesDescription(rawDescription)
	description, descriptionMatched := descriptionResult.(task.SeriesDescriptionAccepted)
	if !descriptionMatched {
		writeError(w, http.StatusBadRequest, descriptionResult.(task.SeriesDescriptionRejected).Reason.Description())
		return task.SeriesTitle{}, task.SeriesDescription{}, false
	}
	return title.Value, description.Value, true
}

func (server Server) writeSeriesMutation(w http.ResponseWriter, r *http.Request, actor auth.UserSubject, result task.SeriesMutationResult, status int) {
	mutated, matched := result.(task.SeriesMutated)
	if !matched {
		writeDomainError(w, result.(task.SeriesMutationRejected).Reason)
		return
	}
	server.writeSeriesDetailStatus(w, r, actor, mutated.Value, status)
}

func (server Server) writeSeriesDetailStatus(w http.ResponseWriter, r *http.Request, actor auth.UserSubject, detail task.SeriesDetail, status int) {
	response := taskSeriesDetailResponse{
		Series:   seriesToResponse(detail.Series),
		Tasks:    make([]taskResponse, 0, len(detail.Tasks)),
		Comments: []seriesCommentResponse{},
	}
	for index := range detail.Tasks {
		response.Tasks = append(response.Tasks, taskToResponse(detail.Tasks[index]))
	}
	commentsResult := server.taskService.ListSeriesComments(r.Context(), actor, detail.Series.ID)
	if listed, matched := commentsResult.(task.SeriesCommentsListed); matched {
		response.Comments = commentsToResponse(listed.Values)
	}
	writeJSON(w, status, response)
}

func seriesToResponse(value task.Series) taskSeriesResponse {
	return taskSeriesResponse{
		ID:          value.ID.String(),
		OwnerKind:   taskOwnerResponseParts(value.Owner).kind,
		Title:       value.Title.String(),
		Description: value.Description.String(),
		State:       value.State.String(),
		CreatedBy:   value.CreatedBy.String(),
	}
}

func commentsToResponse(values []task.SeriesComment) []seriesCommentResponse {
	responses := make([]seriesCommentResponse, 0, len(values))
	for index := range values {
		responses = append(responses, commentToResponse(values[index]))
	}
	return responses
}

func commentToResponse(value task.SeriesComment) seriesCommentResponse {
	return seriesCommentResponse{
		ID:           value.ID.String(),
		SeriesID:     value.SeriesID.String(),
		AuthorUserID: value.AuthorID.String(),
		Body:         value.Body.String(),
		CreatedAt:    value.CreatedAt.UTC().Format(time.RFC3339),
	}
}
