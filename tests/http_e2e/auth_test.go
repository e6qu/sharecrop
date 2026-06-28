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

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/ledger"
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

func TestRefreshTokenReuseRevokesFamilyHTTP(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	registerResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    "reuse-" + uniqueTestSuffix(t) + "@example.com",
		Password: "correct horse battery staple",
	}, nil)
	defer registerResponse.Body.Close()
	assertStatus(t, registerResponse, http.StatusCreated)
	originalCookie := findRefreshCookie(t, registerResponse)

	// Rotate: the original token is consumed and a new token is issued.
	rotateResponse := postRefresh(t, server, originalCookie)
	defer rotateResponse.Body.Close()
	assertStatus(t, rotateResponse, http.StatusOK)
	rotatedCookie := findRefreshCookie(t, rotateResponse)

	// Reusing the original (already consumed) token is detected and rejected.
	reuseResponse := postRefresh(t, server, originalCookie)
	defer reuseResponse.Body.Close()
	if reuseResponse.StatusCode == http.StatusOK {
		t.Fatalf("reused refresh token was accepted, want rejection")
	}

	// The rotated token belongs to the revoked family and can no longer refresh.
	revokedResponse := postRefresh(t, server, rotatedCookie)
	defer revokedResponse.Body.Close()
	if revokedResponse.StatusCode == http.StatusOK {
		t.Fatalf("rotated token still refreshed after family revocation, want rejection")
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	email := "logout-" + uniqueTestSuffix(t) + "@example.com"
	registerResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    email,
		Password: "correct horse battery staple",
	}, nil)
	defer registerResponse.Body.Close()
	assertStatus(t, registerResponse, http.StatusCreated)
	cookie := findRefreshCookie(t, registerResponse)

	// Logging out revokes the session family, so the refresh cookie can no longer
	// resume the session.
	logoutRequest, err := http.NewRequest(http.MethodPost, server.URL+"/api/auth/logout", http.NoBody)
	if err != nil {
		t.Fatalf("create logout request: %v", err)
	}
	logoutRequest.AddCookie(cookie)
	logoutResponse, err := http.DefaultClient.Do(logoutRequest)
	if err != nil {
		t.Fatalf("post logout: %v", err)
	}
	defer logoutResponse.Body.Close()
	assertStatus(t, logoutResponse, http.StatusNoContent)

	refreshResponse := postRefresh(t, server, cookie)
	defer refreshResponse.Body.Close()
	if refreshResponse.StatusCode == http.StatusOK {
		t.Fatalf("refresh succeeded after logout, want rejection")
	}

	// Logging in again is not blocked.
	loginResponse := postAuthJSON(t, server.URL+"/api/auth/login", authHTTPRequest{
		Email:    email,
		Password: "correct horse battery staple",
	}, nil)
	defer loginResponse.Body.Close()
	assertStatus(t, loginResponse, http.StatusOK)
}

func TestAccountLifecycleHTTP(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	email := "account-" + uniqueTestSuffix(t) + "@example.com"
	user := registerUserWithEmail(t, server, email)

	directoryResponse := getWithBearer(t, server.URL+"/api/users?query="+email, user.AccessToken)
	defer directoryResponse.Body.Close()
	assertStatus(t, directoryResponse, http.StatusOK)
	var directory usersHTTPResponse
	if err := json.NewDecoder(directoryResponse.Body).Decode(&directory); err != nil {
		t.Fatalf("decode user directory: %v", err)
	}
	if len(directory.Users) != 1 || directory.Users[0].ID != user.SubjectID {
		t.Fatalf("directory users = %+v, want registered user %q", directory.Users, user.SubjectID)
	}

	verifyResponse := postJSONWithBearer(t, server.URL+"/api/account/email-verification", []byte(`{}`), user.AccessToken)
	defer verifyResponse.Body.Close()
	assertStatus(t, verifyResponse, http.StatusCreated)
	verifyToken := decodeAccountTokenHTTPResponse(t, verifyResponse).Token

	confirmVerify := postEncodedJSON(t, server.URL+"/api/auth/email-verification/confirm", []byte(`{"token":"`+verifyToken+`"}`), nil)
	defer confirmVerify.Body.Close()
	assertStatus(t, confirmVerify, http.StatusOK)

	resetResponse := postEncodedJSON(t, server.URL+"/api/auth/password-reset/request", []byte(`{"email":"`+email+`"}`), nil)
	defer resetResponse.Body.Close()
	assertStatus(t, resetResponse, http.StatusCreated)
	resetToken := decodeAccountTokenHTTPResponse(t, resetResponse).Token

	confirmReset := postEncodedJSON(t, server.URL+"/api/auth/password-reset/confirm", []byte(`{"token":"`+resetToken+`","password":"changed horse battery staple"}`), nil)
	defer confirmReset.Body.Close()
	assertStatus(t, confirmReset, http.StatusOK)

	oldLogin := postAuthJSON(t, server.URL+"/api/auth/login", authHTTPRequest{Email: email, Password: "correct horse battery staple"}, nil)
	defer oldLogin.Body.Close()
	if oldLogin.StatusCode == http.StatusOK {
		t.Fatalf("old password still logged in after reset")
	}
	newLogin := postAuthJSON(t, server.URL+"/api/auth/login", authHTTPRequest{Email: email, Password: "changed horse battery staple"}, nil)
	defer newLogin.Body.Close()
	assertStatus(t, newLogin, http.StatusOK)
	loggedIn := decodeAuthHTTPResponse(t, newLogin)

	changePassword := patchJSONWithBearer(t, server.URL+"/api/account/password", []byte(`{"current_password":"changed horse battery staple","new_password":"final horse battery staple"}`), loggedIn.AccessToken)
	defer changePassword.Body.Close()
	assertStatus(t, changePassword, http.StatusOK)

	newEmail := "account-updated-" + uniqueTestSuffix(t) + "@example.com"
	updateProfile := patchJSONWithBearer(t, server.URL+"/api/account/profile", []byte(`{"email":"`+newEmail+`"}`), loggedIn.AccessToken)
	defer updateProfile.Body.Close()
	assertStatus(t, updateProfile, http.StatusOK)

	loginUpdated := postAuthJSON(t, server.URL+"/api/auth/login", authHTTPRequest{Email: newEmail, Password: "final horse battery staple"}, nil)
	defer loginUpdated.Body.Close()
	assertStatus(t, loginUpdated, http.StatusOK)
	updated := decodeAuthHTTPResponse(t, loginUpdated)

	deactivateRequest, err := http.NewRequest(http.MethodDelete, server.URL+"/api/account", http.NoBody)
	if err != nil {
		t.Fatalf("create deactivate request: %v", err)
	}
	deactivateRequest.Header.Set("Authorization", "Bearer "+updated.AccessToken)
	deactivateResponse, err := http.DefaultClient.Do(deactivateRequest)
	if err != nil {
		t.Fatalf("delete account: %v", err)
	}
	defer deactivateResponse.Body.Close()
	assertStatus(t, deactivateResponse, http.StatusOK)

	afterDeactivate := postAuthJSON(t, server.URL+"/api/auth/login", authHTTPRequest{Email: newEmail, Password: "final horse battery staple"}, nil)
	defer afterDeactivate.Body.Close()
	if afterDeactivate.StatusCode == http.StatusOK {
		t.Fatalf("deactivated account still logged in")
	}
}

type accountTokenHTTPResponse struct {
	Token string `json:"token"`
}

type usersHTTPResponse struct {
	Users []struct {
		ID     string `json:"id"`
		Email  string `json:"email"`
		Status string `json:"status"`
	} `json:"users"`
}

func decodeAccountTokenHTTPResponse(t *testing.T, response *http.Response) accountTokenHTTPResponse {
	t.Helper()
	var body accountTokenHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode account token response: %v", err)
	}
	if body.Token == "" {
		t.Fatalf("account token is empty")
	}
	return body
}

func postRefresh(t *testing.T, server *httptest.Server, cookie *http.Cookie) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/auth/refresh", http.NoBody)
	if err != nil {
		t.Fatalf("create refresh request: %v", err)
	}
	request.AddCookie(cookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post refresh: %v", err)
	}
	return response
}

func TestOversizedRequestBodyIsRejected(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	huge := strings.Repeat("a", 3<<20)
	body := `{"email":"big-` + uniqueTestSuffix(t) + `@example.com","password":"` + huge + `"}`
	response, err := http.Post(server.URL+"/api/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("post oversized body: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest && response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d, want 400 or 413", response.StatusCode)
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
	submissionService := submission.NewService(db.NewSubmissionStore(pool), taskStore, organizationService)
	ledgerService := ledger.NewService(db.NewLedgerStore(pool))
	agentService := agent.NewService(db.NewAgentStore(pool))
	assetService := assets.NewService(db.NewCollectibleStore(pool))
	return httptest.NewServer(httpserver.New(staticFiles, serviceCreated.Value, verifier, organizationService, taskService, submissionService, ledgerService, agentService, assetService))
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
