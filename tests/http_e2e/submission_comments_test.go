//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestSubmissionCommentThread(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "submission-comment-owner")
	worker := registerUser(t, server, "submission-comment-worker")
	stranger := registerUser(t, server, "submission-comment-stranger")

	task := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, task.ID)
	submission := submitAuthenticated(t, server, worker.AccessToken, task.ID)
	submissionID := submission.Submission.ID

	// The worker who authored the submission may comment.
	workerComment := postJSONWithBearer(t, server.URL+"/api/submissions/"+submissionID+"/comments",
		[]byte(`{"body":"I left a note about the edge case."}`), worker.AccessToken)
	defer workerComment.Body.Close()
	assertStatus(t, workerComment, http.StatusCreated)

	// The owner of the submission's task may also comment.
	ownerComment := postJSONWithBearer(t, server.URL+"/api/submissions/"+submissionID+"/comments",
		[]byte(`{"body":"Thanks, could you confirm the input range?"}`), owner.AccessToken)
	defer ownerComment.Body.Close()
	assertStatus(t, ownerComment, http.StatusCreated)

	// Both parties read the same thread back, in order.
	for _, party := range []struct {
		name        string
		accessToken string
	}{
		{name: "worker", accessToken: worker.AccessToken},
		{name: "owner", accessToken: owner.AccessToken},
	} {
		listResponse := getWithBearer(t, server.URL+"/api/submissions/"+submissionID+"/comments", party.accessToken)
		assertStatus(t, listResponse, http.StatusOK)
		var comments struct {
			Comments []struct {
				Body string `json:"body"`
			} `json:"comments"`
		}
		if err := json.NewDecoder(listResponse.Body).Decode(&comments); err != nil {
			listResponse.Body.Close()
			t.Fatalf("decode %s comments: %v", party.name, err)
		}
		listResponse.Body.Close()
		if len(comments.Comments) != 2 {
			t.Fatalf("%s sees %d comments, want 2", party.name, len(comments.Comments))
		}
		if !strings.Contains(comments.Comments[0].Body, "edge case") || !strings.Contains(comments.Comments[1].Body, "input range") {
			t.Fatalf("%s comment thread = %+v, want both posted comments in order", party.name, comments.Comments)
		}
	}

	// An unrelated user can neither post nor read.
	strangerComment := postJSONWithBearer(t, server.URL+"/api/submissions/"+submissionID+"/comments",
		[]byte(`{"body":"Let me peek."}`), stranger.AccessToken)
	defer strangerComment.Body.Close()
	assertStatus(t, strangerComment, http.StatusForbidden)

	strangerList := getWithBearer(t, server.URL+"/api/submissions/"+submissionID+"/comments", stranger.AccessToken)
	defer strangerList.Body.Close()
	assertStatus(t, strangerList, http.StatusForbidden)
}
