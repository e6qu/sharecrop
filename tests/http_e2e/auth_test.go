//go:build http_e2e

package http_e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/web"
)

func TestAuthHTTPFlow(t *testing.T) {
	ctx := context.Background()
	server := newAuthHTTPServer(t, ctx)
	defer server.Close()

	email := "person-" + uniqueTestSuffix(t) + "@example.com"
	registerResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    email,
		Password: "correct horse battery staple",
	}, nil)
	defer registerResponse.Body.Close()
	assertStatus(t, registerResponse, http.StatusCreated)
	registerBody := decodeAuthHTTPResponse(t, registerResponse)
	if registerBody.SubjectKind != "user" {
		t.Fatalf("register subject kind = %q, want user", registerBody.SubjectKind)
	}

	loginResponse := postAuthJSON(t, server.URL+"/api/auth/login", authHTTPRequest{
		Email:    email,
		Password: "correct horse battery staple",
	}, nil)
	defer loginResponse.Body.Close()
	assertStatus(t, loginResponse, http.StatusOK)

	refreshCookie := findRefreshCookie(t, loginResponse)
	refreshRequest, err := http.NewRequest(http.MethodPost, server.URL+"/api/auth/refresh", http.NoBody)
	if err != nil {
		t.Fatalf("create refresh request: %v", err)
	}
	refreshRequest.AddCookie(refreshCookie)
	refreshResponse, err := http.DefaultClient.Do(refreshRequest)
	if err != nil {
		t.Fatalf("post refresh: %v", err)
	}
	defer refreshResponse.Body.Close()
	assertStatus(t, refreshResponse, http.StatusOK)

	guestResponse := postEmptyJSON(t, server.URL+"/api/auth/guest", nil)
	defer guestResponse.Body.Close()
	assertStatus(t, guestResponse, http.StatusCreated)
	guestBody := decodeAuthHTTPResponse(t, guestResponse)
	if guestBody.SubjectKind != "guest" {
		t.Fatalf("guest subject kind = %q, want guest", guestBody.SubjectKind)
	}
}

type authHTTPRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authHTTPResponse struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	AccessToken string `json:"access_token"`
}

func newAuthHTTPServer(t *testing.T, ctx context.Context) *httptest.Server {
	t.Helper()

	pool, err := db.Open(ctx, requiredEnv(t, "DATABASE_URL"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.MigrateUp(ctx, pool, requiredEnv(t, "SHARECROP_MIGRATIONS_DIR")); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	secretResult := auth.NewAccessTokenSecret(requiredEnv(t, "SHARECROP_ACCESS_TOKEN_SECRET"))
	secretAccepted, secretMatched := secretResult.(auth.AccessTokenSecretAccepted)
	if !secretMatched {
		rejected := secretResult.(auth.AccessTokenSecretRejected)
		t.Fatalf("access token secret rejected: %s", rejected.Reason.Description())
	}

	serviceResult := auth.NewService(db.NewAuthStore(pool), secretAccepted.Value, auth.SystemClock{})
	serviceCreated, serviceMatched := serviceResult.(auth.ServiceCreated)
	if !serviceMatched {
		rejected := serviceResult.(auth.ServiceRejected)
		t.Fatalf("service rejected: %s", rejected.Reason.Description())
	}

	staticFiles, err := web.StaticFiles()
	if err != nil {
		t.Fatalf("static files: %v", err)
	}

	verifier := auth.NewAccessTokenVerifier(secretAccepted.Value, auth.SystemClock{})
	organizationService := org.NewService(db.NewOrgStore(pool))
	taskStore := db.NewTaskStore(pool)
	taskService := task.NewService(taskStore, organizationService)
	submissionService := submission.NewService(db.NewSubmissionStore(pool), taskStore)
	return httptest.NewServer(httpserver.New(staticFiles, serviceCreated.Value, verifier, organizationService, taskService, submissionService))
}

func postEmptyJSON(t *testing.T, url string, cookies []*http.Cookie) *http.Response {
	t.Helper()
	return postEncodedJSON(t, url, []byte("{}"), cookies)
}

func postAuthJSON(t *testing.T, url string, body authHTTPRequest, cookies []*http.Cookie) *http.Response {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	return postEncodedJSON(t, url, encoded, cookies)
}

func postEncodedJSON(t *testing.T, url string, encoded []byte, cookies []*http.Cookie) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post json: %v", err)
	}
	return response
}

func decodeAuthHTTPResponse(t *testing.T, response *http.Response) authHTTPResponse {
	t.Helper()
	var body authHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.SubjectID == "" {
		t.Fatalf("subject id is empty")
	}
	if body.AccessToken == "" {
		t.Fatalf("access token is empty")
	}
	return body
}

func findRefreshCookie(t *testing.T, response *http.Response) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Cookies() {
		if cookie.Name == "sharecrop_refresh_token" {
			return cookie
		}
	}
	t.Fatalf("refresh cookie missing")
	return &http.Cookie{}
}

func assertStatus(t *testing.T, response *http.Response, want int) {
	t.Helper()
	if response.StatusCode != want {
		t.Fatalf("status = %d, want %d", response.StatusCode, want)
	}
}

func requiredEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if strings.TrimSpace(value) == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}

func uniqueTestSuffix(t *testing.T) string {
	t.Helper()
	result := core.NewUserID()
	created, matched := result.(core.UserIDCreated)
	if !matched {
		rejected := result.(core.UserIDRejected)
		t.Fatalf("user id rejected: %s", rejected.Reason.Description())
	}
	return created.Value.String()
}
