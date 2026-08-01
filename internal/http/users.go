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
	ID string `json:"id"`
	// DisplayName names the profiled user; empty when the user has no
	// visible directory entry (a deactivated account).
	DisplayName string                 `json:"display_name"`
	Tasks       []taskListItemResponse `json:"tasks"`
}

func (usersResponse) writableResponse() {}

func (server Server) listUsers(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	if _, actorMatched := actorResult.(userSubjectAccepted); !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, rejected.reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	query := r.URL.Query().Get("query")
	if len(query) > maxUserDirectoryQueryLength {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "query is too long")
		return
	}
	result := server.authService.ListUsers(r.Context(), query, page.Probe())
	listed, matched := result.(auth.UsersListed)
	if !matched {
		writeDomainError(w, result.(auth.UserDirectoryRejected).Reason)
		return
	}

	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := usersResponse{Users: make([]userDirectoryEntryResponse, 0, visible), NextOffset: nextOffset}
	for _, value := range listed.Values[:visible] {
		response.Users = append(response.Users, userDirectoryEntryResponse{ID: value.ID.String(), Email: value.Email.String(), DisplayName: value.DisplayName.String(), Status: value.Status})
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
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return auth.UserSubject{}, core.UserID{}, core.Page{}, false
	}

	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userIDCreated, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		writeDomainError(w, userIDResult.(core.UserIDRejected).Reason)
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

	displayName := ""
	switch entry := server.findOwnDirectoryEntry(r.Context(), userID).(type) {
	case directoryEntryFound:
		displayName = entry.value.DisplayName.String()
	case directoryEntryMissing:
		// A deactivated account has no visible directory entry; its profile
		// keeps an empty display name.
	case directoryEntryRejected:
		writeDomainError(w, entry.reason)
		return
	}
	response := userProfileResponse{ID: userID.String(), DisplayName: displayName, Tasks: make([]taskListItemResponse, 0, len(listed.Values))}
	for valueIndex := range listed.Values {
		response.Tasks = append(response.Tasks, taskListItemToResponse(listed.Values[valueIndex], actor))
	}
	writeUserProfileResponse(w, http.StatusOK, response)
}

func (server Server) getUserWork(w http.ResponseWriter, r *http.Request) {
	actor, userID, page, ok := server.userPathRequest(w, r)
	if !ok {
		return
	}
	result := server.taskService.List(r.Context(), actor, task.AssigneeListScope{AssigneeID: userID}, task.NoListFilters(), page.Probe())
	listed, matched := result.(task.TasksListed)
	if !matched {
		writeDomainError(w, result.(task.ListRejected).Reason)
		return
	}
	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := tasksToResponse(listed.Values[:visible], actor)
	response.NextOffset = nextOffset
	writeTasksResponse(w, http.StatusOK, response)
}

func (server Server) getUserSubmissions(w http.ResponseWriter, r *http.Request) {
	actor, userID, page, ok := server.userPathRequest(w, r)
	if !ok {
		return
	}
	result := server.submissionService.ListForSubmitter(r.Context(), actor, userID, page.Probe())
	listed, matched := result.(submission.SubmissionsListed)
	if !matched {
		writeDomainError(w, result.(submission.ListRejected).Reason)
		return
	}

	visible, nextOffset := probeListWindow(len(listed.Values), page)
	server.recordSensitiveFieldAccessForList(r.Context(), actor.ID, listed.Values[:visible])
	response := submissionsResponse{Submissions: make([]submissionResponse, 0, visible), NextOffset: nextOffset}
	for _, value := range listed.Values[:visible] {
		response.Submissions = append(response.Submissions, submissionToResponse(value))
	}
	writeSubmissionsResponse(w, http.StatusOK, response)
}

func writeUserProfileResponse(w http.ResponseWriter, status int, response userProfileResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
