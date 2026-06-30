package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

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

	result := server.authService.ListUsers(r.Context(), r.URL.Query().Get("query"), parsePage(r))
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

func (server Server) getUserProfile(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userIDCreated, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		writeError(w, http.StatusBadRequest, userIDResult.(core.UserIDRejected).Reason.Description())
		return
	}

	result := server.taskService.List(r.Context(), actor.subject, task.CreatorListScope{CreatorID: userIDCreated.Value}, task.NoListFilters(), parsePage(r))
	listed, matched := result.(task.TasksListed)
	if !matched {
		writeDomainError(w, result.(task.ListRejected).Reason)
		return
	}

	response := userProfileResponse{ID: userIDCreated.Value.String(), Tasks: make([]taskListItemResponse, 0, len(listed.Values))}
	for valueIndex := range listed.Values {
		response.Tasks = append(response.Tasks, taskListItemToResponse(listed.Values[valueIndex]))
	}
	writeUserProfileResponse(w, http.StatusOK, response)
}

func (server Server) getUserWork(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userIDCreated, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		writeError(w, http.StatusBadRequest, userIDResult.(core.UserIDRejected).Reason.Description())
		return
	}

	result := server.taskService.List(r.Context(), actor.subject, task.AssigneeListScope{AssigneeID: userIDCreated.Value}, task.NoListFilters(), parsePage(r))
	listed, matched := result.(task.TasksListed)
	if !matched {
		writeDomainError(w, result.(task.ListRejected).Reason)
		return
	}
	writeTasksResponse(w, http.StatusOK, tasksToResponse(listed.Values))
}

func (server Server) getUserSubmissions(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userIDCreated, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		writeError(w, http.StatusBadRequest, userIDResult.(core.UserIDRejected).Reason.Description())
		return
	}

	pageResult := parsePageStrict(r)
	page, pageMatched := pageResult.(pageParseAccepted)
	if !pageMatched {
		writeError(w, http.StatusBadRequest, pageResult.(pageParseRejected).reason)
		return
	}

	result := server.submissionService.ListForSubmitter(r.Context(), actor.subject, userIDCreated.Value, page.value)
	listed, matched := result.(submission.SubmissionsListed)
	if !matched {
		writeDomainError(w, result.(submission.ListRejected).Reason)
		return
	}

	response := submissionsResponse{Submissions: make([]submissionResponse, 0, len(listed.Values))}
	for _, value := range listed.Values {
		if !server.recordSensitiveFieldAccess(w, r, actor.subject.ID, value) {
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
