package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

type taskCommentResponse struct {
	ID           string `json:"id"`
	TaskID       string `json:"task_id"`
	AuthorUserID string `json:"author_user_id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

type taskCommentsResponse struct {
	Comments   []taskCommentResponse `json:"comments"`
	NextOffset int                   `json:"next_offset"`
}

func (taskCommentResponse) writableResponse() {}

func (taskCommentsResponse) writableResponse() {}

func (server Server) listTaskComments(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	taskID, ok := server.parseSeriesTaskID(w, r.PathValue("task_id"))
	if !ok {
		return
	}
	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.taskService.ListTaskComments(r.Context(), actor, taskID, page.Probe())
	listed, matched := result.(task.TaskCommentsListed)
	if !matched {
		writeDomainError(w, result.(task.TaskCommentsListRejected).Reason)
		return
	}
	visible, nextOffset := probeListWindow(len(listed.Values), page)
	writeJSON(w, http.StatusOK, taskCommentsResponse{Comments: taskCommentsToResponse(listed.Values[:visible]), NextOffset: nextOffset})
}

func (server Server) addTaskComment(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	if !server.allowBySubject(w, actor.ID.String()) {
		return
	}
	taskID, ok := server.parseSeriesTaskID(w, r.PathValue("task_id"))
	if !ok {
		return
	}
	var request seriesCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	bodyResult := task.NewCommentBody(request.Body)
	body, matched := bodyResult.(task.CommentBodyAccepted)
	if !matched {
		writeDomainError(w, bodyResult.(task.CommentBodyRejected).Reason)
		return
	}
	result := server.taskService.AddTaskComment(r.Context(), actor, taskID, body.Value)
	added, addedMatched := result.(task.TaskCommentAdded)
	if !addedMatched {
		writeDomainError(w, result.(task.TaskCommentRejected).Reason)
		return
	}
	writeJSON(w, http.StatusCreated, taskCommentToResponse(added.Value))
}

func taskCommentsToResponse(values []task.TaskComment) []taskCommentResponse {
	responses := make([]taskCommentResponse, 0, len(values))
	for index := range values {
		responses = append(responses, taskCommentToResponse(values[index]))
	}
	return responses
}

func taskCommentToResponse(value task.TaskComment) taskCommentResponse {
	return taskCommentResponse{
		ID:           value.ID.String(),
		TaskID:       value.TaskID.String(),
		AuthorUserID: value.AuthorID.String(),
		Body:         value.Body.String(),
		CreatedAt:    value.CreatedAt.UTC().Format(time.RFC3339),
	}
}
