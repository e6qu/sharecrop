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

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"golang.org/x/oauth2"
)

type logoutTokenFixture struct {
	Issuer   string                     `json:"iss"`
	Audience string                     `json:"aud"`
	Subject  string                     `json:"sub,omitempty"`
	SID      string                     `json:"sid,omitempty"`
	IssuedAt int64                      `json:"iat"`
	Expires  int64                      `json:"exp,omitempty"`
	JWTID    string                     `json:"jti"`
	Events   map[string]json.RawMessage `json:"events"`
	Nonce    json.RawMessage            `json:"nonce,omitempty"`
}

func (fixture logoutTokenFixture) clone() logoutTokenFixture {
	cloned := fixture
	cloned.Events = make(map[string]json.RawMessage, len(fixture.Events))
	for name, event := range fixture.Events {
		cloned.Events[name] = append(json.RawMessage(nil), event...)
	}
	cloned.Nonce = append(json.RawMessage(nil), fixture.Nonce...)
	return cloned
}

type IDTokenFixture struct {
	Issuer   string `json:"iss"`
	Audience string `json:"aud"`
	Subject  string `json:"sub"`
	Nonce    string `json:"nonce"`
	Email    string `json:"email"`
	IssuedAt int64  `json:"iat"`
	Expires  int64  `json:"exp"`
}

type OAuthTokenResponseFixture struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	IDToken     string `json:"id_token"`
}

func TestSHAUTHBackchannelLogoutVerifiesTokenAndRevokesIdentitySessions(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const keyID = "logout-key"
	discoveryRequests := 0
	var issuer *httptest.Server
	issuer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			discoveryRequests++
			_ = json.NewEncoder(w).Encode(map[string]string{"issuer": issuer.URL, "authorization_endpoint": issuer.URL + "/oauth2/auth", "token_endpoint": issuer.URL + "/oauth2/token", "jwks_uri": issuer.URL + "/.well-known/jwks.json", "end_session_endpoint": issuer.URL + "/oauth2/logout"})
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
	claims := logoutTokenFixture{
		Issuer: issuer.URL, Audience: "sharecrop", Subject: "sha-subject", SID: "sha-session",
		IssuedAt: now.Unix(), Expires: now.Add(time.Minute).Unix(), JWTID: "logout-id",
		Events: map[string]json.RawMessage{backchannelLogoutEvent: json.RawMessage(`{}`), "https://example.test/event": json.RawMessage(`{}`)},
	}
	raw, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	sessions := newMemoryOpenIDConnectSessionStore()
	refreshHash := auth.HashRefreshToken(testRefreshToken())
	sessions.StoreOpenIDConnectSession(context.Background(), refreshHash, auth.OpenIDConnectSession{Provider: "shauth", Issuer: issuer.URL, Subject: "sha-subject", SID: "sha-session", ClientID: "sharecrop"})
	server := Server{authService: testAuth{}, oidcSessions: sessions, shauth: shauthConfig{issuer: issuer.URL, clientID: "sharecrop", clientSecret: "secret", publicURL: "https://sharecrop.example.test", provider: &cachedShauthProvider{}}}
	clientContext := oidc.ClientContext(context.Background(), issuer.Client())
	if _, _, err := server.verifyBackchannelLogout(clientContext, raw); err != nil {
		t.Fatalf("verify logout token: %v", err)
	}
	form := url.Values{"logout_token": {raw}}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/shauth/backchannel-logout", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(clientContext)
	response := httptest.NewRecorder()

	server.shauthBackchannelLogout(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusNoContent, response.Body.String())
	}
	if found := sessions.FindOpenIDConnectSession(context.Background(), refreshHash); found != (auth.OpenIDConnectSessionNotFound{}) {
		t.Fatalf("back-channel logout retained session: %#v", found)
	}

	for name, mutate := range map[string]func(*logoutTokenFixture){
		"missing exp": func(value *logoutTokenFixture) { value.Expires = 0 },
		"empty nonce": func(value *logoutTokenFixture) { value.Nonce = json.RawMessage(`""`) },
		"null nonce":  func(value *logoutTokenFixture) { value.Nonce = json.RawMessage(`null`) },
		"event array": func(value *logoutTokenFixture) {
			value.Events = map[string]json.RawMessage{backchannelLogoutEvent: json.RawMessage(`[]`)}
		},
		"nonempty event": func(value *logoutTokenFixture) {
			value.Events = map[string]json.RawMessage{backchannelLogoutEvent: json.RawMessage(`{"reason":"logout"}`)}
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalidClaims := claims.clone()
			invalidClaims.JWTID = "invalid-" + name
			mutate(&invalidClaims)
			invalid, err := jwt.Signed(signer).Claims(invalidClaims).Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := server.verifyBackchannelLogout(clientContext, invalid); err == nil {
				t.Fatal("invalid logout token was accepted")
			}
		})
	}
	for name, mutate := range map[string]func(*logoutTokenFixture){
		"subject only": func(value *logoutTokenFixture) { value.SID = "" },
		"session only": func(value *logoutTokenFixture) { value.Subject = "" },
	} {
		t.Run(name, func(t *testing.T) {
			validClaims := claims.clone()
			validClaims.JWTID = "valid-" + name
			mutate(&validClaims)
			valid, err := jwt.Signed(signer).Claims(validClaims).Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := server.verifyBackchannelLogout(clientContext, valid); err != nil {
				t.Fatalf("valid logout token was rejected: %v", err)
			}
		})
	}
	replayRequest := httptest.NewRequest(http.MethodPost, "/api/auth/shauth/backchannel-logout", strings.NewReader(form.Encode()))
	replayRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	replayResponse := httptest.NewRecorder()
	server.shauthBackchannelLogout(replayResponse, replayRequest.WithContext(clientContext))
	if replayResponse.Code != http.StatusBadRequest {
		t.Fatalf("logout replay status = %d, want 400", replayResponse.Code)
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

func TestSHAUTHCallbackRetainsSignedLogoutSessionMetadata(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const keyID = "callback-key"
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", keyID))
	if err != nil {
		t.Fatal(err)
	}
	const nonce = "callback-nonce"
	var provider *httptest.Server
	provider = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		issuer := provider.URL + "/"
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer": issuer, "authorization_endpoint": provider.URL + "/oauth2/auth",
				"token_endpoint": provider.URL + "/oauth2/token", "jwks_uri": provider.URL + "/jwks",
				"end_session_endpoint": provider.URL + "/oauth2/logout",
			})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: keyID, Algorithm: string(jose.RS256), Use: "sig"}}})
		case "/oauth2/token":
			if err := r.ParseForm(); err != nil || r.PostForm.Get("code") == "" || r.PostForm.Get("code_verifier") == "" {
				http.Error(w, "invalid token request", http.StatusBadRequest)
				return
			}
			rawIDToken, err := jwt.Signed(signer).Claims(IDTokenFixture{
				Issuer: issuer, Audience: "sharecrop", Subject: "subject-1",
				Nonce: nonce, Email: "person@example.com", IssuedAt: time.Now().Unix(), Expires: time.Now().Add(time.Hour).Unix(),
			}).Serialize()
			if err != nil {
				http.Error(w, "sign ID token", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(OAuthTokenResponseFixture{AccessToken: "access", TokenType: "Bearer", ExpiresIn: 3600, IDToken: rawIDToken})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()

	config := shauthConfig{issuer: provider.URL + "/", clientID: "sharecrop", clientSecret: "secret", publicURL: "https://sharecrop.example.test", provider: &cachedShauthProvider{}}
	encoded, err := config.encodeTransaction(shauthTransaction{State: "callback-state", Nonce: nonce, Verifier: "callback-verifier", Expires: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	sessions := newMemoryOpenIDConnectSessionStore()
	server := Server{authService: testAuth{}, oidcSessions: sessions, shauth: config}
	request := httptest.NewRequest(http.MethodGet, "/api/auth/shauth/callback?code=code&state=callback-state", nil)
	request.AddCookie(&http.Cookie{Name: "sharecrop_shauth_tx", Value: encoded})
	request = request.WithContext(oidc.ClientContext(request.Context(), provider.Client()))
	response := httptest.NewRecorder()
	server.shauthCallback(response, request)
	if response.Code != http.StatusFound || response.Header().Get("Location") != "/" {
		t.Fatalf("callback = %d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	found, ok := sessions.FindOpenIDConnectSession(context.Background(), auth.HashRefreshToken(testRefreshToken())).(auth.OpenIDConnectSessionFound)
	if !ok {
		t.Fatalf("signed OpenID Connect session was not retained")
	}
	if found.Session.Issuer != provider.URL+"/" || found.Session.Subject != "subject-1" || found.Session.SID != "" || found.Session.RawIDToken == "" || found.Session.ClientID != "sharecrop" || found.Session.EndSessionEndpoint != provider.URL+"/oauth2/logout" || found.Session.PostLogoutRedirectURI != "https://sharecrop.example.test/api/auth/signed-out" {
		t.Fatalf("retained session = %#v", found.Session)
	}
}

func TestSHAUTHLogoutReturnsIssuerFrontChannelURL(t *testing.T) {
	server := Server{shauth: shauthConfig{issuer: "https://auth.dev.e6qu.dev/", clientID: "sharecrop", clientSecret: "test-client-secret", publicURL: "https://sharecrop.dev.e6qu.dev", provider: &cachedShauthProvider{provider: new(oidc.Provider), endSessionEndpoint: "https://auth.dev.e6qu.dev/oauth2/sessions/logout"}}}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set("Origin", "https://sharecrop.dev.e6qu.dev")
	response := httptest.NewRecorder()

	server.logout(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d", response.Code, http.StatusOK)
	}
	var body logoutResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode logout response: %v", err)
	}
	parsed, err := url.Parse(body.LogoutURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme+"://"+parsed.Host+parsed.Path != "https://auth.dev.e6qu.dev/oauth2/sessions/logout" || parsed.Query().Get("client_id") != "sharecrop" || parsed.Query().Get("post_logout_redirect_uri") != "https://sharecrop.dev.e6qu.dev/api/auth/signed-out" {
		t.Fatalf("logout URL = %q", body.LogoutURL)
	}
}

func TestSHAUTHLogoutUsesIDTokenHintAndExactSignedOutLanding(t *testing.T) {
	sessions := newMemoryOpenIDConnectSessionStore()
	sessions.StoreOpenIDConnectSession(context.Background(), auth.HashRefreshToken(testRefreshToken()), auth.OpenIDConnectSession{
		Provider: "shauth", Issuer: "https://auth.dev.e6qu.dev/", Subject: "subject-1", SID: "sid-1",
		RawIDToken: "signed.id.token", ClientID: "sharecrop", EndSessionEndpoint: "https://auth.dev.e6qu.dev/oauth2/logout",
		PostLogoutRedirectURI: "https://sharecrop.dev.e6qu.dev/api/auth/signed-out", ExpiresAt: time.Now().Add(time.Hour),
	})
	server := Server{authService: testAuth{}, oidcSessions: sessions, shauth: shauthConfig{issuer: "https://auth.dev.e6qu.dev/", clientID: "sharecrop", clientSecret: "secret", publicURL: "https://sharecrop.dev.e6qu.dev"}}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set("Origin", "https://sharecrop.dev.e6qu.dev")
	request.AddCookie(&http.Cookie{Name: "sharecrop_refresh_token", Value: testRefreshToken().String()})
	response := httptest.NewRecorder()
	server.logout(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("logout = %d: %s", response.Code, response.Body.String())
	}
	var body logoutResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(body.LogoutURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("id_token_hint") != "signed.id.token" || parsed.Query().Get("client_id") != "sharecrop" || parsed.Query().Get("post_logout_redirect_uri") != "https://sharecrop.dev.e6qu.dev/api/auth/signed-out" {
		t.Fatalf("RP-Initiated Logout URL = %q", body.LogoutURL)
	}
	cleared := false
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "sharecrop_refresh_token" && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("logout did not clear Sharecrop refresh cookie")
	}
}

func TestSHAUTHLogoutRevokesLocallyBeforeProviderDiscoveryFailure(t *testing.T) {
	provider := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "discovery unavailable", http.StatusServiceUnavailable)
	}))
	defer provider.Close()
	recorder := &recordingLogoutAuth{}
	server := Server{
		authService: recorder, oidcSessions: newMemoryOpenIDConnectSessionStore(),
		shauth: shauthConfig{issuer: provider.URL, clientID: "sharecrop", clientSecret: "secret", publicURL: "https://sharecrop.example.test", provider: &cachedShauthProvider{}},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set("Origin", "https://sharecrop.example.test")
	request.AddCookie(&http.Cookie{Name: "sharecrop_refresh_token", Value: testRefreshToken().String()})
	request = request.WithContext(oidc.ClientContext(request.Context(), provider.Client()))
	response := httptest.NewRecorder()
	server.logout(response, request)
	if response.Code != http.StatusBadGateway {
		t.Fatalf("logout = %d, want 502: %s", response.Code, response.Body.String())
	}
	if recorder.calls != 1 {
		t.Fatalf("local revocations = %d, want 1", recorder.calls)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != "sharecrop_refresh_token" || cookies[0].MaxAge >= 0 {
		t.Fatalf("provider failure did not clear local cookie: %#v", cookies)
	}
}

func TestSHAUTHFrontchannelLogoutRequiresTrustedIssuerAndSessionID(t *testing.T) {
	sessions := newMemoryOpenIDConnectSessionStore()
	hash := auth.HashRefreshToken(testRefreshToken())
	session := auth.OpenIDConnectSession{Provider: "shauth", Issuer: "https://auth.example.test", ClientID: "sharecrop", SID: "provider-session"}
	sessions.StoreOpenIDConnectSession(context.Background(), hash, session)
	server := Server{oidcSessions: sessions, shauth: shauthConfig{issuer: session.Issuer, clientID: session.ClientID, clientSecret: "secret", publicURL: "https://sharecrop.example.test"}}

	untrusted := httptest.NewRecorder()
	server.shauthFrontchannelLogout(untrusted, httptest.NewRequest(http.MethodGet, "/api/auth/shauth/frontchannel-logout?iss=https%3A%2F%2Fattacker.example&sid=provider-session", nil))
	if untrusted.Code != http.StatusOK {
		t.Fatalf("untrusted front-channel response = %d", untrusted.Code)
	}
	if _, found := sessions.FindOpenIDConnectSession(context.Background(), hash).(auth.OpenIDConnectSessionFound); !found {
		t.Fatal("untrusted front-channel request revoked the session")
	}

	trusted := httptest.NewRecorder()
	server.shauthFrontchannelLogout(trusted, httptest.NewRequest(http.MethodGet, "/api/auth/shauth/frontchannel-logout?iss=https%3A%2F%2Fauth.example.test&sid=provider-session", nil))
	if trusted.Code != http.StatusOK {
		t.Fatalf("trusted front-channel response = %d: %s", trusted.Code, trusted.Body.String())
	}
	if _, found := sessions.FindOpenIDConnectSession(context.Background(), hash).(auth.OpenIDConnectSessionNotFound); !found {
		t.Fatal("trusted front-channel request retained the session")
	}
	if trusted.Header().Get("Cache-Control") != "no-store" || !strings.Contains(trusted.Header().Get("Content-Security-Policy"), "https://auth.example.test") {
		t.Fatalf("front-channel security headers = %#v", trusted.Header())
	}
}

func TestSHAUTHSignedOutLandingDoesNotStartAuthentication(t *testing.T) {
	recorder := &recordingLogoutAuth{}
	server := Server{authService: recorder}
	request := httptest.NewRequest(http.MethodGet, "/api/auth/signed-out", nil)
	request.AddCookie(&http.Cookie{Name: "sharecrop_refresh_token", Value: testRefreshToken().String()})
	response := httptest.NewRecorder()
	server.shauthSignedOut(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Location") != "" {
		t.Fatalf("signed-out landing = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if response.Header().Get("Content-Security-Policy") == "" || response.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("signed-out security headers = %#v", response.Header())
	}
	if !strings.Contains(response.Body.String(), "You are signed out") || strings.Contains(response.Body.String(), "window.location") {
		t.Fatalf("signed-out landing body was not static: %s", response.Body.String())
	}
	if recorder.calls != 1 {
		t.Fatalf("residual session revocations = %d, want 1", recorder.calls)
	}
	cleared := false
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "sharecrop_refresh_token" && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("signed-out landing did not clear the refresh cookie")
	}
}

type recordingLogoutAuth struct {
	testAuth
	calls int
}

func (authService *recordingLogoutAuth) Logout(context.Context, auth.RefreshTokenPlain) auth.LogoutResult {
	authService.calls++
	return auth.LogoutDone{}
}

func TestSHAUTHSignedOutLandingFailsClosedWhenResidualSessionCannotBeRevoked(t *testing.T) {
	server := Server{authService: rejectingLogoutAuth{}}
	request := httptest.NewRequest(http.MethodGet, "/api/auth/signed-out", nil)
	request.AddCookie(&http.Cookie{Name: "sharecrop_refresh_token", Value: testRefreshToken().String()})
	response := httptest.NewRecorder()
	server.shauthSignedOut(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("signed-out landing = %d, want 503", response.Code)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("failed signed-out revocation changed cookies: %#v", cookies)
	}
}

func TestSHAUTHLogoutRejectsCrossOriginWithoutClearingCookie(t *testing.T) {
	server := Server{shauth: shauthConfig{issuer: "https://auth.dev.e6qu.dev", clientID: "sharecrop", clientSecret: "secret", publicURL: "https://sharecrop.dev.e6qu.dev", provider: &cachedShauthProvider{provider: new(oidc.Provider), endSessionEndpoint: "https://auth.dev.e6qu.dev/oauth2/logout"}}}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	server.logout(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-origin logout status = %d, want 403", response.Code)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("cross-origin logout changed cookies: %#v", cookies)
	}
}

type rejectingLogoutAuth struct{ testAuth }

func (rejectingLogoutAuth) Logout(context.Context, auth.RefreshTokenPlain) auth.LogoutResult {
	return auth.LogoutRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "refresh family was not revoked")}
}

func TestSHAUTHLogoutFailsClosedWhenLocalSessionCannotBeRevoked(t *testing.T) {
	server := Server{
		authService:  rejectingLogoutAuth{},
		oidcSessions: newMemoryOpenIDConnectSessionStore(),
		shauth: shauthConfig{
			issuer: "https://auth.dev.e6qu.dev", clientID: "sharecrop", clientSecret: "secret", publicURL: "https://sharecrop.dev.e6qu.dev",
			provider: &cachedShauthProvider{provider: new(oidc.Provider), endSessionEndpoint: "https://auth.dev.e6qu.dev/oauth2/logout"},
		},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set("Origin", "https://sharecrop.dev.e6qu.dev")
	request.AddCookie(&http.Cookie{Name: "sharecrop_refresh_token", Value: testRefreshToken().String()})
	response := httptest.NewRecorder()
	server.logout(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("logout status = %d, want 503", response.Code)
	}
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("failed logout changed cookies: %#v", cookies)
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
	if err := (shauthConfig{issuer: "http://localhost:8080", clientID: "client", clientSecret: "secret", publicURL: "http://127.0.0.1:29180", allowInsecure: true}).validate(); err != nil {
		t.Fatalf("explicit loopback development config: %v", err)
	}
	if err := (shauthConfig{issuer: "http://auth.example.test", clientID: "client", clientSecret: "secret", publicURL: "http://sharecrop.example.test", allowInsecure: true}).validate(); err == nil {
		t.Fatal("non-loopback insecure config was accepted")
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

func TestSHAUTHConfigPreservesExactIssuer(t *testing.T) {
	t.Setenv("SHARECROP_SHAUTH_ISSUER", "https://auth.dev.e6qu.dev/")
	if got := shauthConfigFromEnv().issuer; got != "https://auth.dev.e6qu.dev/" {
		t.Fatalf("issuer = %q, want exact trailing slash", got)
	}
}

func TestSHAUTHLogoutRejectsEndSessionEndpointOutsideIssuerOrigin(t *testing.T) {
	config := shauthConfig{issuer: "https://auth.dev.e6qu.dev/", clientID: "sharecrop", publicURL: "https://sharecrop.dev.e6qu.dev"}
	if _, err := config.logoutURL("https://attacker.example/oauth2/logout", ""); err == nil {
		t.Fatal("cross-origin end-session endpoint was accepted")
	}
}

func TestSHAUTHDiscoveryRejectsEndSessionEndpointOutsideIssuerOrigin(t *testing.T) {
	var provider *httptest.Server
	provider = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": provider.URL, "authorization_endpoint": provider.URL + "/oauth2/auth",
			"token_endpoint": provider.URL + "/oauth2/token", "jwks_uri": provider.URL + "/jwks",
			"end_session_endpoint": "https://attacker.example/oauth2/logout",
		})
	}))
	defer provider.Close()
	config := shauthConfig{issuer: provider.URL, clientID: "sharecrop", clientSecret: "secret", publicURL: "https://sharecrop.example.test", provider: &cachedShauthProvider{}}
	ctx := oidc.ClientContext(context.Background(), provider.Client())
	if _, _, _, err := config.discoveredProvider(ctx); err == nil {
		t.Fatal("cross-origin discovered end-session endpoint was accepted")
	}
}
