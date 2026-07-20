package httpserver

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/e6qu/sharecrop/internal/auth"
	"golang.org/x/oauth2"
)

const backchannelLogoutEvent = "http://schemas.openid.net/event/backchannel-logout"

type shauthConfig struct {
	issuer, clientID, clientSecret, publicURL string
	allowInsecure                             bool
	provider                                  *cachedShauthProvider
}

type cachedShauthProvider struct {
	mu                 sync.Mutex
	provider           *oidc.Provider
	verifier           *oidc.IDTokenVerifier
	endSessionEndpoint string
}

type shauthProviderMetadata struct {
	EndSessionEndpoint string `json:"end_session_endpoint"`
}

func (c shauthConfig) enabled() bool {
	return c.issuer != "" && c.clientID != "" && c.clientSecret != "" && c.publicURL != ""
}

func (c shauthConfig) validate() error {
	configured := 0
	for _, value := range []string{c.issuer, c.clientID, c.clientSecret, c.publicURL} {
		if value != "" {
			configured++
		}
	}
	if configured == 0 {
		return nil
	}
	if configured != 4 {
		return fmt.Errorf("all SHARECROP_SHAUTH_* and SHARECROP_PUBLIC_URL values must be configured together")
	}
	issuer, err := url.Parse(c.issuer)
	if err != nil || !c.validOIDCCoordinate(issuer) || issuer.User != nil || issuer.RawQuery != "" || issuer.Fragment != "" {
		return fmt.Errorf("Shauth issuer must be an absolute HTTPS URL, or an HTTP loopback URL when insecure cookies are explicitly enabled")
	}
	publicURL, err := url.Parse(c.publicURL)
	if err != nil || !c.validOIDCCoordinate(publicURL) || publicURL.User != nil || (publicURL.Path != "" && publicURL.Path != "/") || publicURL.RawQuery != "" || publicURL.Fragment != "" {
		return fmt.Errorf("Sharecrop public URL must be an absolute HTTPS origin, or an HTTP loopback origin when insecure cookies are explicitly enabled")
	}
	return nil
}
func shauthConfigFromEnv() shauthConfig {
	return shauthConfig{issuer: strings.TrimSpace(os.Getenv("SHARECROP_SHAUTH_ISSUER")), clientID: strings.TrimSpace(os.Getenv("SHARECROP_SHAUTH_CLIENT_ID")), clientSecret: strings.TrimSpace(os.Getenv("SHARECROP_SHAUTH_CLIENT_SECRET")), publicURL: strings.TrimRight(strings.TrimSpace(os.Getenv("SHARECROP_PUBLIC_URL")), "/"), allowInsecure: os.Getenv("SHARECROP_INSECURE_COOKIES") == "true", provider: &cachedShauthProvider{}}
}

func (c shauthConfig) validOIDCCoordinate(value *url.URL) bool {
	if value == nil || value.Host == "" {
		return false
	}
	if value.Scheme == "https" {
		return true
	}
	if !c.allowInsecure || value.Scheme != "http" {
		return false
	}
	host := value.Hostname()
	return strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
}

func (c shauthConfig) postLogoutRedirectURI() string {
	return c.publicURL + "/api/auth/signed-out"
}

func (c shauthConfig) discoveredProvider(ctx context.Context) (*oidc.Provider, *oidc.IDTokenVerifier, string, error) {
	cache := c.provider
	if cache == nil {
		cache = &cachedShauthProvider{}
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.provider != nil {
		return cache.provider, cache.verifier, cache.endSessionEndpoint, nil
	}
	provider, err := oidc.NewProvider(ctx, c.issuer)
	if err != nil {
		return nil, nil, "", fmt.Errorf("discover Shauth: %w", err)
	}
	var metadata shauthProviderMetadata
	if err := provider.Claims(&metadata); err != nil {
		return nil, nil, "", fmt.Errorf("decode Shauth discovery metadata: %w", err)
	}
	endpoint, err := url.Parse(metadata.EndSessionEndpoint)
	if err != nil || !c.validOIDCCoordinate(endpoint) || endpoint.User != nil {
		return nil, nil, "", fmt.Errorf("Shauth discovery omitted a valid end_session_endpoint")
	}
	issuer, err := url.Parse(c.issuer)
	if err != nil || !sameURLOrigin(issuer, endpoint) {
		return nil, nil, "", fmt.Errorf("Shauth end_session_endpoint must use the configured issuer origin")
	}
	cache.provider = provider
	cache.verifier = provider.Verifier(&oidc.Config{ClientID: c.clientID})
	cache.endSessionEndpoint = endpoint.String()
	return cache.provider, cache.verifier, cache.endSessionEndpoint, nil
}

func sameURLOrigin(left, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func (c shauthConfig) oauthConfig(endpoint oauth2.Endpoint) oauth2.Config {
	endpoint.AuthStyle = oauth2.AuthStyleInParams
	return oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     endpoint,
		RedirectURL:  c.publicURL + "/api/auth/shauth/callback",
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}
}

type shauthTransaction struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Expires  int64  `json:"e"`
}

func randomSHAUTHValue() (string, error) {
	raw := make([]byte, 32)
	_, err := rand.Read(raw)
	return base64.RawURLEncoding.EncodeToString(raw), err
}

func (c shauthConfig) encodeTransaction(tx shauthTransaction) (string, error) {
	payload, err := json.Marshal(tx)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(c.clientSecret))
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (c shauthConfig) decodeTransaction(value string) (shauthTransaction, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return shauthTransaction{}, fmt.Errorf("invalid transaction")
	}
	provided, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return shauthTransaction{}, fmt.Errorf("invalid transaction")
	}
	mac := hmac.New(sha256.New, []byte(c.clientSecret))
	_, _ = mac.Write([]byte(parts[0]))
	if subtle.ConstantTimeCompare(provided, mac.Sum(nil)) != 1 {
		return shauthTransaction{}, fmt.Errorf("invalid transaction")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return shauthTransaction{}, fmt.Errorf("invalid transaction")
	}
	var tx shauthTransaction
	if err := json.Unmarshal(payload, &tx); err != nil {
		return shauthTransaction{}, fmt.Errorf("invalid transaction")
	}
	return tx, nil
}

func (server Server) shauthLogin(w http.ResponseWriter, r *http.Request) {
	if !server.shauth.enabled() {
		writeError(w, http.StatusServiceUnavailable, "Shauth sign-in is not configured")
		return
	}
	provider, _, _, err := server.shauth.discoveredProvider(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "Shauth discovery failed")
		return
	}
	state, err := randomSHAUTHValue()
	if err != nil {
		writeError(w, 500, "could not create Shauth transaction")
		return
	}
	nonce, err := randomSHAUTHValue()
	if err != nil {
		writeError(w, 500, "could not create Shauth transaction")
		return
	}
	verifier, err := randomSHAUTHValue()
	if err != nil {
		writeError(w, 500, "could not create Shauth transaction")
		return
	}
	tx := shauthTransaction{State: state, Nonce: nonce, Verifier: verifier, Expires: time.Now().Add(10 * time.Minute).Unix()}
	encoded, err := server.shauth.encodeTransaction(tx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create Shauth transaction")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "sharecrop_shauth_tx", Value: encoded, Path: "/api/auth/shauth", HttpOnly: true, Secure: server.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	config := server.shauth.oauthConfig(provider.Endpoint())
	http.Redirect(w, r, config.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}
func (server Server) shauthCallback(w http.ResponseWriter, r *http.Request) {
	if !server.shauth.enabled() {
		writeError(w, http.StatusServiceUnavailable, "Shauth sign-in is not configured")
		return
	}
	cookie, err := r.Cookie("sharecrop_shauth_tx")
	if err != nil {
		writeError(w, 400, "Shauth transaction is missing")
		return
	}
	tx, err := server.shauth.decodeTransaction(cookie.Value)
	if err != nil || tx.Expires < time.Now().Unix() || subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("state")), []byte(tx.State)) != 1 {
		writeError(w, 400, "Shauth transaction is invalid")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "sharecrop_shauth_tx", Path: "/api/auth/shauth", MaxAge: -1, HttpOnly: true, Secure: server.secureCookies, SameSite: http.SameSiteLaxMode})
	provider, verifier, endSessionEndpoint, err := server.shauth.discoveredProvider(r.Context())
	if err != nil {
		writeError(w, 502, "Shauth discovery failed")
		return
	}
	config := server.shauth.oauthConfig(provider.Endpoint())
	tokens, err := config.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(tx.Verifier))
	if err != nil {
		writeError(w, 401, "Shauth code exchange failed")
		return
	}
	rawID, ok := tokens.Extra("id_token").(string)
	if !ok {
		writeError(w, 401, "Shauth did not return an ID token")
		return
	}
	token, err := verifier.Verify(r.Context(), rawID)
	if err != nil {
		writeError(w, 401, "Shauth ID token verification failed")
		return
	}
	var claims struct {
		Nonce string `json:"nonce"`
		Email string `json:"email"`
		SID   string `json:"sid"`
	}
	if token.Claims(&claims) != nil || claims.Nonce != tx.Nonce || token.Subject == "" {
		writeError(w, 401, "Shauth ID token claims were invalid")
		return
	}
	email := auth.NewEmailAddress(claims.Email)
	accepted, ok := email.(auth.EmailAddressAccepted)
	if !ok {
		writeError(w, 401, "Shauth ID token did not contain a valid email")
		return
	}
	result := server.authService.LoginExternal(r.Context(), token.Issuer, token.Subject, accepted.Value)
	login, ok := result.(auth.ExternalLoginAccepted)
	if !ok {
		writeError(w, 401, result.(auth.ExternalLoginRejected).Reason.Description())
		return
	}
	session := auth.OpenIDConnectSession{
		Provider: "shauth", Issuer: token.Issuer, Subject: token.Subject, SID: claims.SID,
		RawIDToken: rawID, ClientID: server.shauth.clientID, EndSessionEndpoint: endSessionEndpoint,
		PostLogoutRedirectURI: server.shauth.postLogoutRedirectURI(), ExpiresAt: token.Expiry,
	}
	stored := server.oidcSessions.StoreOpenIDConnectSession(r.Context(), auth.HashRefreshToken(login.RefreshToken), session)
	if _, ok := stored.(auth.OpenIDConnectSessionStored); !ok {
		server.authService.Logout(r.Context(), login.RefreshToken)
		writeError(w, http.StatusServiceUnavailable, "Sharecrop OpenID Connect session could not be stored")
		return
	}
	server.setRefreshCookie(w, login.RefreshToken)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (server Server) shauthFrontchannelLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	frameAncestor := "'none'"
	if issuer, err := url.Parse(server.shauth.issuer); err == nil && server.shauth.validOIDCCoordinate(issuer) {
		frameAncestor = issuer.Scheme + "://" + issuer.Host
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors "+frameAncestor)
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	issuer := r.URL.Query().Get("iss")
	sid := r.URL.Query().Get("sid")
	if server.shauth.enabled() && issuer == server.shauth.issuer && sid != "" {
		result := server.oidcSessions.ApplyFrontchannelLogout(r.Context(), auth.OpenIDConnectFrontchannelLogout{
			Provider: "shauth", Issuer: issuer, ClientID: server.shauth.clientID, SID: sid,
		})
		if _, ok := result.(auth.FrontchannelLogoutApplied); !ok {
			writeError(w, http.StatusServiceUnavailable, "logout could not be completed")
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Signed out</title></head><body></body></html>"))
}

type backchannelLogoutClaims struct {
	Events   map[string]json.RawMessage `json:"events"`
	IssuedAt int64                      `json:"iat"`
	JWTID    string                     `json:"jti"`
	Expires  int64                      `json:"exp"`
	Nonce    json.RawMessage            `json:"nonce"`
	SID      string                     `json:"sid"`
}

func (server Server) verifyBackchannelLogout(ctx context.Context, raw string) (*oidc.IDToken, backchannelLogoutClaims, error) {
	_, verifier, _, err := server.shauth.discoveredProvider(ctx)
	if err != nil {
		return nil, backchannelLogoutClaims{}, err
	}
	token, err := verifier.Verify(ctx, raw)
	if err != nil {
		return nil, backchannelLogoutClaims{}, fmt.Errorf("verify logout token: %w", err)
	}
	var claims backchannelLogoutClaims
	if err := token.Claims(&claims); err != nil {
		return nil, backchannelLogoutClaims{}, fmt.Errorf("decode logout token: %w", err)
	}
	event, eventPresent := claims.Events[backchannelLogoutEvent]
	if (token.Subject == "" && claims.SID == "") || claims.IssuedAt == 0 || claims.Expires == 0 || claims.JWTID == "" || len(claims.Nonce) != 0 || !eventPresent {
		return nil, backchannelLogoutClaims{}, fmt.Errorf("logout token claims are invalid")
	}
	var eventObject map[string]json.RawMessage
	if err := json.Unmarshal(event, &eventObject); err != nil || eventObject == nil || len(eventObject) != 0 {
		return nil, backchannelLogoutClaims{}, fmt.Errorf("logout token event is invalid")
	}
	now := time.Now()
	issuedAt := time.Unix(claims.IssuedAt, 0)
	if issuedAt.Before(now.Add(-5*time.Minute)) || issuedAt.After(now.Add(time.Minute)) {
		return nil, backchannelLogoutClaims{}, fmt.Errorf("logout token is stale")
	}
	return token, claims, nil
}

func (server Server) shauthBackchannelLogout(w http.ResponseWriter, r *http.Request) {
	if !server.shauth.enabled() {
		writeError(w, http.StatusServiceUnavailable, "Shauth sign-in is not configured")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "logout token is invalid")
		return
	}
	token, claims, err := server.verifyBackchannelLogout(r.Context(), strings.TrimSpace(r.Form.Get("logout_token")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "logout token is invalid")
		return
	}
	result := server.oidcSessions.ApplyBackchannelLogout(r.Context(), auth.OpenIDConnectLogoutClaim{
		Provider: "shauth", Issuer: token.Issuer, ClientID: server.shauth.clientID,
		JWTID: claims.JWTID, ExpiresAt: token.Expiry, SID: claims.SID, Subject: token.Subject,
	}, time.Now())
	switch result.(type) {
	case auth.BackchannelLogoutApplied:
	case auth.BackchannelLogoutReplay:
		writeError(w, http.StatusBadRequest, "logout token was already used")
		return
	default:
		writeError(w, http.StatusServiceUnavailable, "logout could not be completed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (server Server) shauthSignedOut(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("sharecrop_refresh_token"); err == nil && cookie.Value != "" {
		parsed, accepted := auth.ParseRefreshTokenPlain(cookie.Value).(auth.RefreshTokenPlainAccepted)
		if accepted {
			if _, done := server.authService.Logout(r.Context(), parsed.Value).(auth.LogoutDone); !done {
				writeError(w, http.StatusServiceUnavailable, "Sharecrop session could not be revoked")
				return
			}
		}
	}
	server.clearRefreshCookie(w)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light dark"><title>Signed out · Sharecrop</title><style>:root{font:16px system-ui,sans-serif;color-scheme:light dark}body{min-height:100vh;margin:0;display:grid;place-items:center;background:#f6f8fa;color:#1f2328}main{width:min(28rem,calc(100% - 3rem));padding:2rem;border:1px solid #d0d7de;border-radius:1rem;background:#fff;box-shadow:0 1rem 3rem #1f23281f}h1{margin-top:0}a{display:inline-block;padding:.7rem 1rem;border-radius:.5rem;background:#1f883d;color:#fff;font-weight:700;text-decoration:none}a:focus-visible{outline:3px solid #0969da;outline-offset:3px}@media(prefers-color-scheme:dark){body{background:#0d1117;color:#e6edf3}main{background:#161b22;border-color:#30363d}a{background:#238636}}</style></head><body><main><h1>You are signed out</h1><p>The sign-out flow ended this Sharecrop browser session.</p><a href="/api/auth/shauth">Sign in again</a></main></body></html>`))
}
