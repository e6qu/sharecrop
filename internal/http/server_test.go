package httpserver

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	New(testStaticFiles(), testAuthService()).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestIndex(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	New(testStaticFiles(), testAuthService()).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	contentType := response.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("content type = %q, want html", contentType)
	}
}

func TestRegisterEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"email":"person@example.com","password":"correct horse battery staple"}`))
	response := httptest.NewRecorder()

	New(testStaticFiles(), testAuthService()).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	assertAuthResponse(t, response, "user")
	assertRefreshCookie(t, response)
}

func TestGuestEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/guest", nil)
	response := httptest.NewRecorder()

	New(testStaticFiles(), testAuthService()).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	assertAuthResponse(t, response, "guest")
	assertRefreshCookie(t, response)
}

func TestRefreshEndpointRequiresCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	response := httptest.NewRecorder()

	New(testStaticFiles(), testAuthService()).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func testStaticFiles() fs.FS {
	return fstest.MapFS{
		"index.html": {
			Data: []byte("<!doctype html><title>Sharecrop</title>"),
		},
	}
}

func assertAuthResponse(t *testing.T, response *httptest.ResponseRecorder, subjectKind string) {
	t.Helper()

	var body authResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.SubjectKind != subjectKind {
		t.Fatalf("subject kind = %q, want %q", body.SubjectKind, subjectKind)
	}

	if body.SubjectID == "" {
		t.Fatalf("subject id is empty")
	}

	if body.AccessToken == "" {
		t.Fatalf("access token is empty")
	}
}

func assertRefreshCookie(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	cookies := response.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "sharecrop_refresh_token" && cookie.HttpOnly {
			return
		}
	}

	t.Fatalf("refresh cookie was not set")
}

type testAuth struct{}

func testAuthService() testAuth {
	return testAuth{}
}

func (testAuth) Register(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.RegisterResult {
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.RegisterAccepted{
		Subject:      auth.UserSubject{ID: idCreated.Value},
		AccessToken:  testAccessToken(),
		RefreshToken: testRefreshToken(),
	}
}

func (testAuth) Login(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.LoginResult {
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.LoginAccepted{
		Subject:      auth.UserSubject{ID: idCreated.Value},
		AccessToken:  testAccessToken(),
		RefreshToken: testRefreshToken(),
	}
}

func (testAuth) Refresh(context.Context, auth.RefreshTokenPlain) auth.RefreshResult {
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.RefreshAccepted{
		Subject:      auth.UserSubject{ID: idCreated.Value},
		AccessToken:  testAccessToken(),
		RefreshToken: testRefreshToken(),
	}
}

func (testAuth) CreateGuest(context.Context) auth.GuestResult {
	idResult := core.NewGuestID()
	idCreated := idResult.(core.GuestIDCreated)
	return auth.GuestAccepted{
		Subject:      auth.GuestSubject{ID: idCreated.Value},
		AccessToken:  testAccessToken(),
		RefreshToken: testRefreshToken(),
	}
}

func testAccessToken() auth.AccessToken {
	secretResult := auth.NewAccessTokenSecret("01234567890123456789012345678901")
	secretAccepted := secretResult.(auth.AccessTokenSecretAccepted)
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	tokenResult := auth.SignAccessToken(secretAccepted.Value, auth.UserSubject{ID: idCreated.Value}, timeForTest())
	tokenAccepted := tokenResult.(auth.AccessTokenAccepted)
	return tokenAccepted.Value
}

func testRefreshToken() auth.RefreshTokenPlain {
	tokenResult := auth.ParseRefreshTokenPlain("test-refresh-token")
	tokenAccepted := tokenResult.(auth.RefreshTokenPlainAccepted)
	return tokenAccepted.Value
}

func timeForTest() time.Time {
	return time.Unix(1_700_000_000, 0).UTC()
}
