//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mintOrgCredential(t *testing.T, server *httptest.Server, accessToken string, organizationID string, scopesJSON string) orgCredentialCreatedHTTPResponse {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/organizations/"+organizationID+"/credentials",
		[]byte(`{"label":"Submission automation","scopes":`+scopesJSON+`}`), accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeOrgCredentialCreatedHTTPResponse(t, response)
}

// TestOrgCredentialListsAndReviewsSubmissionsOnItsOwnOrgTasks closes the
// documented-but-missing org-credential submission surface: a credential
// holding submissions_read lists an org task's submissions, and one holding
// submissions_review accepts them — on its own organization's tasks only,
// with the user-only authorization rules unchanged for everyone else.
func TestOrgCredentialListsAndReviewsSubmissionsOnItsOwnOrgTasks(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "org-sub-owner")
	worker := registerUser(t, server, "org-sub-worker")
	outsider := registerUser(t, server, "org-sub-outsider")

	organizationID := createOrganization(t, server, owner, "Submission Review Org")
	credential := mintOrgCredential(t, server, owner.AccessToken, organizationID, `["submissions_read","submissions_review"]`)
	readOnly := mintOrgCredential(t, server, owner.AccessToken, organizationID, `["submissions_read"]`)

	otherOwner := registerUser(t, server, "org-sub-other-owner")
	otherOrganizationID := createOrganization(t, server, otherOwner, "Unrelated Org")
	foreign := mintOrgCredential(t, server, otherOwner.AccessToken, otherOrganizationID, `["submissions_read","submissions_review"]`)

	createTaskResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(organizationPublicTaskRequestJSON(organizationID)), owner.AccessToken)
	defer createTaskResponse.Body.Close()
	assertStatus(t, createTaskResponse, http.StatusCreated)
	task := decodeTaskHTTPResponse(t, createTaskResponse)
	openTask(t, server, owner.AccessToken, task.ID)

	submitted := submitAuthenticated(t, server, worker.AccessToken, task.ID)

	// The org credential lists the task's submissions.
	listResponse := getWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions", credential.Secret)
	defer listResponse.Body.Close()
	assertStatus(t, listResponse, http.StatusOK)
	var listed struct {
		Submissions []struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"submissions"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&listed); err != nil {
		t.Fatalf("decode submissions: %v", err)
	}
	if len(listed.Submissions) != 1 || listed.Submissions[0].ID != submitted.Submission.ID {
		t.Fatalf("org credential listing = %+v, want the worker's submission", listed.Submissions)
	}

	// Personal-scope denial is unchanged: an unrelated user cannot list, and
	// another organization's credential is denied on this org's task.
	outsiderDenied := getWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions", outsider.AccessToken)
	defer outsiderDenied.Body.Close()
	assertStatus(t, outsiderDenied, http.StatusForbidden)
	foreignDenied := getWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions", foreign.Secret)
	defer foreignDenied.Body.Close()
	assertStatus(t, foreignDenied, http.StatusForbidden)

	// A credential without submissions_review cannot review.
	scopeDenied := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions/"+submitted.Submission.ID+"/accept",
		[]byte(`{"idempotency_key":"org-accept-scope-`+task.ID+`"}`), readOnly.Secret)
	defer scopeDenied.Body.Close()
	assertStatus(t, scopeDenied, http.StatusForbidden)

	// An org credential cannot pay a tip: tips move personal credits.
	tipRefused := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions/"+submitted.Submission.ID+"/accept",
		[]byte(`{"idempotency_key":"org-accept-tip-`+task.ID+`","tip_amount":5}`), credential.Secret)
	defer tipRefused.Body.Close()
	assertStatus(t, tipRefused, http.StatusBadRequest)

	// Another organization's credential cannot review this org's task.
	foreignReview := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions/"+submitted.Submission.ID+"/accept",
		[]byte(`{"idempotency_key":"org-accept-foreign-`+task.ID+`"}`), foreign.Secret)
	defer foreignReview.Body.Close()
	assertStatus(t, foreignReview, http.StatusForbidden)

	// The reviewing credential accepts the submission.
	acceptResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions/"+submitted.Submission.ID+"/accept",
		[]byte(`{"idempotency_key":"org-accept-`+task.ID+`"}`), credential.Secret)
	defer acceptResponse.Body.Close()
	assertStatus(t, acceptResponse, http.StatusOK)
	var accepted acceptHTTPResponse
	if err := json.NewDecoder(acceptResponse.Body).Decode(&accepted); err != nil {
		t.Fatalf("decode accept: %v", err)
	}
	if accepted.SubmissionID != submitted.Submission.ID {
		t.Fatalf("accepted submission = %q, want %q", accepted.SubmissionID, submitted.Submission.ID)
	}

	// The worker learns about the review through the event-derived inbox.
	if !hasNotificationKind(t, server, worker.AccessToken, "submission_accepted") {
		t.Fatalf("worker inbox has no submission_accepted notification")
	}
}
