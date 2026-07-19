package httpserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"
)

type recordingExternalLogoutAuth struct {
	testAuth
	issuer  string
	subject string
}

type signedBackchannelLogoutClaims struct {
	Issuer   string              `json:"iss"`
	Audience string              `json:"aud"`
	Subject  string              `json:"sub"`
	SID      string              `json:"sid"`
	IssuedAt int64               `json:"iat"`
	Expires  int64               `json:"exp"`
	JWTID    string              `json:"jti"`
	Nonce    string              `json:"nonce,omitempty"`
	Events   map[string]struct{} `json:"events"`
}

func (service *recordingExternalLogoutAuth) LogoutExternalIdentity(_ context.Context, issuer, subject string) auth.LogoutResult {
	service.issuer = issuer
	service.subject = subject
	return auth.LogoutDone{}
}

func TestSHAUTHBackchannelLogoutVerifiesTokenAndRevokesIdentitySessions(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const keyID = "logout-key"
	discoveryRequests := 0
	var issuer *httptest.Server
	issuer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			discoveryRequests++
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": issuer.URL, "authorization_endpoint": issuer.URL + "/oauth2/auth", "token_endpoint": issuer.URL + "/oauth2/token", "jwks_uri": issuer.URL + "/.well-known/jwks.json"})
		case "/.well-known/jwks.json":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: keyID, Algorithm: string(jose.RS256), Use: "sig"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer issuer.Close()

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("logout+jwt").WithHeader("kid", keyID))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	raw, err := jwt.Signed(signer).Claims(signedBackchannelLogoutClaims{Issuer: issuer.URL, Audience: "sharecrop", Subject: "sha-subject", SID: "sha-session", IssuedAt: now.Unix(), Expires: now.Add(time.Minute).Unix(), JWTID: "logout-id", Events: map[string]struct{}{backchannelLogoutEvent: {}}}).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	authService := &recordingExternalLogoutAuth{}
	server := Server{authService: authService, shauth: shauthConfig{issuer: issuer.URL, clientID: "sharecrop", clientSecret: "secret", publicURL: "https://sharecrop.example.test", logoutVerifier: &cachedLogoutVerifier{}}}
	if _, _, err := server.verifyBackchannelLogout(context.Background(), raw); err != nil {
		t.Fatalf("verify logout token: %v", err)
	}
	form := url.Values{"logout_token": {raw}}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/shauth/backchannel-logout", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	server.shauthBackchannelLogout(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusNoContent, response.Body.String())
	}
	if authService.issuer != issuer.URL || authService.subject != "sha-subject" {
		t.Fatalf("revoked identity = %q/%q", authService.issuer, authService.subject)
	}

	for name, claims := range map[string]signedBackchannelLogoutClaims{
		"missing sid": {Issuer: issuer.URL, Audience: "sharecrop", Subject: "sha-subject", IssuedAt: now.Unix(), Expires: now.Add(time.Minute).Unix(), JWTID: "logout-id-missing-sid", Events: map[string]struct{}{backchannelLogoutEvent: {}}},
		"nonce":       {Issuer: issuer.URL, Audience: "sharecrop", Subject: "sha-subject", SID: "sha-session", IssuedAt: now.Unix(), Expires: now.Add(time.Minute).Unix(), JWTID: "logout-id-nonce", Nonce: "must-not-appear", Events: map[string]struct{}{backchannelLogoutEvent: {}}},
	} {
		t.Run(name, func(t *testing.T) {
			invalid, err := jwt.Signed(signer).Claims(claims).Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := server.verifyBackchannelLogout(context.Background(), invalid); err == nil {
				t.Fatal("invalid logout token was accepted")
			}
		})
	}
	if discoveryRequests != 1 {
		t.Fatalf("discovery requests = %d, want one cached verifier", discoveryRequests)
	}
}

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
