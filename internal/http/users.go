package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

// maxUserDirectoryQueryLength bounds the user-directory search term so a
// caller cannot force an expensive database scan with a very long pattern.
const maxUserDirectoryQueryLength = 160

type userProfileResponse struct {
	ID    string                 `json:"id"`
	Tasks []taskListItemResponse `json:"tasks"`
}

func (usersResponse) writableResponse() {}

func (server Server) listUsers(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	if _, actorMatched := actorResult.(userSubjectAccepted); !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	query := r.URL.Query().Get("query")
	if len(query) > maxUserDirectoryQueryLength {
		writeError(w, http.StatusBadRequest, "query is too long")
		return
	}
	result := server.authService.ListUsers(r.Context(), query, page)
	listed, matched := result.(auth.UsersListed)
	if !matched {
		writeDomainError(w, result.(auth.UserDirectoryRejected).Reason)
		return
	}

	response := usersResponse{Users: make([]userDirectoryEntryResponse, 0, len(listed.Values))}
	for _, value := range listed.Values {
		response.Users = append(response.Users, userDirectoryEntryResponse{ID: value.ID.String(), Email: value.Email.String(), Status: value.Status})
	}
	writeJSON(w, http.StatusOK, response)
}

// userPathRequest resolves the shared prologue of the per-user list
// endpoints: the authenticated actor, the {user_id} path value, and strict
// paging. It reports false after writing the error response itself.
func (server Server) userPathRequest(w http.ResponseWriter, r *http.Request) (auth.UserSubject, core.UserID, core.Page, bool) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return auth.UserSubject{}, core.UserID{}, core.Page{}, false
	}

	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userIDCreated, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		writeError(w, http.StatusBadRequest, userIDResult.(core.UserIDRejected).Reason.Description())
		return auth.UserSubject{}, core.UserID{}, core.Page{}, false
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return auth.UserSubject{}, core.UserID{}, core.Page{}, false
	}
	return actor.subject, userIDCreated.Value, page, true
}

func (server Server) getUserProfile(w http.ResponseWriter, r *http.Request) {
	actor, userID, page, ok := server.userPathRequest(w, r)
	if !ok {
		return
	}
	result := server.taskService.List(r.Context(), actor, task.CreatorListScope{CreatorID: userID}, task.NoListFilters(), page)
	listed, matched := result.(task.TasksListed)
	if !matched {
		writeDomainError(w, result.(task.ListRejected).Reason)
		return
	}

	response := userProfileResponse{ID: userID.String(), Tasks: make([]taskListItemResponse, 0, len(listed.Values))}
	for valueIndex := range listed.Values {
		response.Tasks = append(response.Tasks, taskListItemToResponse(listed.Values[valueIndex]))
	}
	writeUserProfileResponse(w, http.StatusOK, response)
}

func (server Server) getUserWork(w http.ResponseWriter, r *http.Request) {
	actor, userID, page, ok := server.userPathRequest(w, r)
	if !ok {
		return
	}
	result := server.taskService.List(r.Context(), actor, task.AssigneeListScope{AssigneeID: userID}, task.NoListFilters(), page)
	listed, matched := result.(task.TasksListed)
	if !matched {
		writeDomainError(w, result.(task.ListRejected).Reason)
		return
	}
	writeTasksResponse(w, http.StatusOK, tasksToResponse(listed.Values))
}

func (server Server) getUserSubmissions(w http.ResponseWriter, r *http.Request) {
	actor, userID, page, ok := server.userPathRequest(w, r)
	if !ok {
		return
	}
	result := server.submissionService.ListForSubmitter(r.Context(), actor, userID, page)
	listed, matched := result.(submission.SubmissionsListed)
	if !matched {
		writeDomainError(w, result.(submission.ListRejected).Reason)
		return
	}

	response := submissionsResponse{Submissions: make([]submissionResponse, 0, len(listed.Values))}
	for _, value := range listed.Values {
		if !server.recordSensitiveFieldAccess(w, r, actor.ID, value) {
			return
		}
		response.Submissions = append(response.Submissions, submissionToResponse(value))
	}
	writeSubmissionsResponse(w, http.StatusOK, response)
}

func writeUserProfileResponse(w http.ResponseWriter, status int, response userProfileResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
