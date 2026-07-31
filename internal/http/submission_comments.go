package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

type submissionCommentResponse struct {
	ID           string `json:"id"`
	SubmissionID string `json:"submission_id"`
	AuthorUserID string `json:"author_user_id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

type submissionCommentsResponse struct {
	Comments   []submissionCommentResponse `json:"comments"`
	NextOffset int                         `json:"next_offset"`
}

type submissionCommentRequest struct {
	Body string `json:"body"`
}

func (submissionCommentResponse) writableResponse() {}

func (submissionCommentsResponse) writableResponse() {}

func (server Server) listSubmissionComments(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	submissionID, ok := server.parseSubmissionCommentID(w, r.PathValue("submission_id"))
	if !ok {
		return
	}
	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.submissionService.ListSubmissionComments(r.Context(), actor, submissionID, page.Probe())
	listed, matched := result.(submission.SubmissionCommentsListed)
	if !matched {
		writeDomainError(w, result.(submission.SubmissionCommentsListRejected).Reason)
		return
	}
	visible, nextOffset := probeListWindow(len(listed.Values), page)
	writeJSON(w, http.StatusOK, submissionCommentsResponse{Comments: submissionCommentsToResponse(listed.Values[:visible]), NextOffset: nextOffset})
}

func (server Server) addSubmissionComment(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.seriesActor(w, r)
	if !ok {
		return
	}
	if !server.allowBySubject(w, actor.ID.String()) {
		return
	}
	submissionID, ok := server.parseSubmissionCommentID(w, r.PathValue("submission_id"))
	if !ok {
		return
	}
	var request submissionCommentRequest
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
	// The counterparty's notification is fanned out by the submission
	// service's event emission.
	result := server.submissionService.AddSubmissionComment(r.Context(), actor, submissionID, body.Value)
	added, addedMatched := result.(submission.SubmissionCommentAdded)
	if !addedMatched {
		writeDomainError(w, result.(submission.SubmissionCommentRejected).Reason)
		return
	}
	writeJSON(w, http.StatusCreated, submissionCommentToResponse(added.Value))
}

func (server Server) parseSubmissionCommentID(w http.ResponseWriter, raw string) (core.SubmissionID, bool) {
	result := core.ParseSubmissionID(raw)
	submissionID, matched := result.(core.SubmissionIDCreated)
	if !matched {
		writeDomainError(w, result.(core.SubmissionIDRejected).Reason)
		return core.SubmissionID{}, false
	}
	return submissionID.Value, true
}

func submissionCommentsToResponse(values []submission.SubmissionComment) []submissionCommentResponse {
	responses := make([]submissionCommentResponse, 0, len(values))
	for index := range values {
		responses = append(responses, submissionCommentToResponse(values[index]))
	}
	return responses
}

func submissionCommentToResponse(value submission.SubmissionComment) submissionCommentResponse {
	return submissionCommentResponse{
		ID:           value.ID.String(),
		SubmissionID: value.SubmissionID.String(),
		AuthorUserID: value.AuthorID.String(),
		Body:         value.Body.String(),
		CreatedAt:    value.CreatedAt.UTC().Format(time.RFC3339),
	}
}
