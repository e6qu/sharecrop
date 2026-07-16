package httpserver

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/e6qu/sharecrop/internal/auth"
	"golang.org/x/oauth2"
)

type shauthConfig struct{ issuer, clientID, clientSecret, publicURL string }

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
	for _, value := range []string{c.issuer, c.publicURL} {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("Shauth issuer and public URL must be absolute HTTPS URLs")
		}
	}
	return nil
}
func shauthConfigFromEnv() shauthConfig {
	return shauthConfig{issuer: strings.TrimRight(os.Getenv("SHARECROP_SHAUTH_ISSUER"), "/"), clientID: os.Getenv("SHARECROP_SHAUTH_CLIENT_ID"), clientSecret: os.Getenv("SHARECROP_SHAUTH_CLIENT_SECRET"), publicURL: strings.TrimRight(os.Getenv("SHARECROP_PUBLIC_URL"), "/")}
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
	provider, err := oidc.NewProvider(r.Context(), server.shauth.issuer)
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
	config := oauth2.Config{ClientID: server.shauth.clientID, ClientSecret: server.shauth.clientSecret, Endpoint: provider.Endpoint(), RedirectURL: server.shauth.publicURL + "/api/auth/shauth/callback", Scopes: []string{oidc.ScopeOpenID, "profile", "email", "offline_access"}}
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
	provider, err := oidc.NewProvider(r.Context(), server.shauth.issuer)
	if err != nil {
		writeError(w, 502, "Shauth discovery failed")
		return
	}
	config := oauth2.Config{ClientID: server.shauth.clientID, ClientSecret: server.shauth.clientSecret, Endpoint: provider.Endpoint(), RedirectURL: server.shauth.publicURL + "/api/auth/shauth/callback"}
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
	token, err := provider.Verifier(&oidc.Config{ClientID: server.shauth.clientID}).Verify(r.Context(), rawID)
	if err != nil {
		writeError(w, 401, "Shauth ID token verification failed")
		return
	}
	var claims struct {
		Nonce string `json:"nonce"`
		Email string `json:"email"`
	}
	if token.Claims(&claims) != nil || claims.Nonce != tx.Nonce {
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
	server.setRefreshCookie(w, login.RefreshToken)
	http.Redirect(w, r, "/", http.StatusFound)
}
