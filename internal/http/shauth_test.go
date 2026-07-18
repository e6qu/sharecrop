package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSHAUTHTransactionIsAuthenticated(t *testing.T) {
	config := shauthConfig{clientSecret: "test-client-secret"}
	want := shauthTransaction{State: "state", Nonce: "nonce", Verifier: "verifier", Expires: time.Now().Add(time.Minute).Unix()}
	encoded, err := config.encodeTransaction(want)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}
	got, err := config.decodeTransaction(encoded)
	if err != nil {
		t.Fatalf("decode transaction: %v", err)
	}
	if got != want {
		t.Fatalf("transaction = %#v, want %#v", got, want)
	}
	parts := strings.Split(encoded, ".")
	if len(parts) != 2 {
		t.Fatalf("encoded transaction = %q", encoded)
	}
	if _, err := config.decodeTransaction(parts[0] + "." + strings.Repeat("A", len(parts[1]))); err == nil {
		t.Fatal("tampered transaction was accepted")
	}
}

func TestSHAUTHLogoutReturnsIssuerFrontChannelURL(t *testing.T) {
	server := Server{shauth: shauthConfig{issuer: "https://auth.dev.e6qu.dev", clientID: "sharecrop", clientSecret: "test-client-secret", publicURL: "https://sharecrop.dev.e6qu.dev"}}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	response := httptest.NewRecorder()

	server.logout(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", response.Code, http.StatusOK)
	}
	var body logoutResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode logout response: %v", err)
	}
	if want := "https://auth.dev.e6qu.dev/oauth2/sessions/logout"; body.LogoutURL != want {
		t.Fatalf("logout URL = %q, want %q", body.LogoutURL, want)
	}
}

func TestSHAUTHConfigRequiresCompleteHTTPSCoordinates(t *testing.T) {
	for _, config := range []shauthConfig{
		{issuer: "https://auth.dev.e6qu.dev", clientID: "client"},
		{issuer: "http://auth.dev.e6qu.dev", clientID: "client", clientSecret: "secret", publicURL: "https://sharecrop.dev.e6qu.dev"},
		{issuer: "https://auth.dev.e6qu.dev", clientID: "client", clientSecret: "secret", publicURL: "http://sharecrop.dev.e6qu.dev"},
	} {
		if err := config.validate(); err == nil {
			t.Fatalf("config %#v was accepted", config)
		}
	}
	if err := (shauthConfig{}).validate(); err != nil {
		t.Fatalf("disabled config: %v", err)
	}
	if err := (shauthConfig{issuer: "https://auth.dev.e6qu.dev", clientID: "client", clientSecret: "secret", publicURL: "https://sharecrop.dev.e6qu.dev"}).validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
}

func TestSHAUTHOAuthConfigUsesClientSecretPost(t *testing.T) {
	config := shauthConfig{
		clientID:     "sharecrop",
		clientSecret: "secret",
		publicURL:    "https://sharecrop.dev.e6qu.dev",
	}.oauthConfig(oauth2.Endpoint{AuthURL: "https://auth.dev.e6qu.dev/oauth2/auth", TokenURL: "https://auth.dev.e6qu.dev/oauth2/token"})

	if config.Endpoint.AuthStyle != oauth2.AuthStyleInParams {
		t.Fatalf("token endpoint auth style = %v, want client_secret_post", config.Endpoint.AuthStyle)
	}
	if config.RedirectURL != "https://sharecrop.dev.e6qu.dev/api/auth/shauth/callback" {
		t.Fatalf("redirect URL = %q", config.RedirectURL)
	}
}
