package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
)

func (server Server) register(w http.ResponseWriter, r *http.Request) {
	if !server.allowByIP(w, r) {
		return
	}
	requestResult := decodeAuthRequest(r)
	requestAccepted, requestMatched := requestResult.(authRequestAccepted)
	if !requestMatched {
		rejected := requestResult.(authRequestRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.authService.Register(r.Context(), requestAccepted.email, requestAccepted.password)
	accepted, matched := result.(auth.RegisterAccepted)
	if !matched {
		rejected := result.(auth.RegisterRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	server.setRefreshCookie(w, accepted.RefreshToken)
	server.writeAuthResponse(w, http.StatusCreated, authResponse{
		SubjectKind: "user",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}

func (server Server) login(w http.ResponseWriter, r *http.Request) {
	if !server.allowByIP(w, r) {
		return
	}
	requestResult := decodeAuthRequest(r)
	requestAccepted, requestMatched := requestResult.(authRequestAccepted)
	if !requestMatched {
		rejected := requestResult.(authRequestRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.authService.Login(r.Context(), requestAccepted.email, requestAccepted.password)
	accepted, matched := result.(auth.LoginAccepted)
	if !matched {
		rejected := result.(auth.LoginRejected)
		writeError(w, http.StatusUnauthorized, rejected.Reason.Description())
		return
	}

	server.setRefreshCookie(w, accepted.RefreshToken)
	server.writeAuthResponse(w, http.StatusOK, authResponse{
		SubjectKind: "user",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}

func (server Server) refresh(w http.ResponseWriter, r *http.Request) {
	if !server.allowByIP(w, r) {
		return
	}
	cookie, err := r.Cookie("sharecrop_refresh_token")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "refresh token is required")
		return
	}

	tokenResult := auth.ParseRefreshTokenPlain(cookie.Value)
	tokenAccepted, tokenMatched := tokenResult.(auth.RefreshTokenPlainAccepted)
	if !tokenMatched {
		rejected := tokenResult.(auth.RefreshTokenPlainRejected)
		writeError(w, http.StatusUnauthorized, rejected.Reason.Description())
		return
	}

	result := server.authService.Refresh(r.Context(), tokenAccepted.Value)
	accepted, matched := result.(auth.RefreshAccepted)
	if !matched {
		rejected := result.(auth.RefreshRejected)
		writeError(w, http.StatusUnauthorized, rejected.Reason.Description())
		return
	}

	server.setRefreshCookie(w, accepted.RefreshToken)
	responseResult := authResponseForSubject(accepted.Subject, accepted.AccessToken)
	responseAccepted, responseMatched := responseResult.(authResponseAccepted)
	if !responseMatched {
		rejected := responseResult.(authResponseRejected)
		writeError(w, http.StatusInternalServerError, rejected.reason)
		return
	}
	// Refreshing replaces the browser's refresh token, so the provider session
	// has to follow it or both the end-session coordinates and the signed-in
	// username become unreachable for the rest of the browser session.
	response := responseAccepted.response
	switch rotated := server.oidcSessions.RotateOpenIDConnectSession(r.Context(), auth.HashRefreshToken(tokenAccepted.Value), auth.HashRefreshToken(accepted.RefreshToken)).(type) {
	case auth.OpenIDConnectSessionRotated:
		response.Username = rotated.Session.Username
	case auth.OpenIDConnectSessionNotRotated:
		// A session that never came from the provider (a guest) has no
		// provider username, and correctly reports none.
	default:
		writeError(w, http.StatusServiceUnavailable, "Sharecrop OpenID Connect session could not be carried across the refresh")
		return
	}

	server.writeAuthResponse(w, http.StatusOK, response)
}

func (server Server) logout(w http.ResponseWriter, r *http.Request) {
	logoutURL, ok := server.completeBrowserLogout(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, logoutResponse{LogoutURL: logoutURL})
}

func (server Server) completeBrowserLogout(w http.ResponseWriter, r *http.Request) (string, bool) {
	w.Header().Set("Cache-Control", "no-store")
	logoutURL := ""
	if server.shauth.enabled() {
		if r.Header.Get("Origin") != server.shauth.publicURL || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			writeError(w, http.StatusForbidden, "cross-origin logout denied")
			return "", false
		}
	}
	var refreshToken auth.RefreshTokenPlain
	hasRefreshToken := false
	if cookie, err := r.Cookie("sharecrop_refresh_token"); err == nil && cookie.Value != "" {
		parsed, matched := auth.ParseRefreshTokenPlain(cookie.Value).(auth.RefreshTokenPlainAccepted)
		if !matched {
			writeError(w, http.StatusUnauthorized, "refresh token is invalid")
			return "", false
		}
		refreshToken = parsed.Value
		hasRefreshToken = true
	}
	endpoint := ""
	idTokenHint := ""
	coordinateError := ""
	if server.shauth.enabled() {
		if hasRefreshToken {
			sessionResult := server.oidcSessions.FindOpenIDConnectSession(r.Context(), auth.HashRefreshToken(refreshToken))
			switch session := sessionResult.(type) {
			case auth.OpenIDConnectSessionFound:
				if session.Session.Provider != "shauth" || session.Session.Issuer != server.shauth.issuer || session.Session.ClientID != server.shauth.clientID || session.Session.PostLogoutRedirectURI != server.shauth.logoutBridgeURI() {
					coordinateError = "OpenID Connect logout session coordinates are invalid"
					break
				}
				endpoint = session.Session.EndSessionEndpoint
				idTokenHint = session.Session.RawIDToken
			case auth.OpenIDConnectSessionNotFound:
			default:
				coordinateError = "OpenID Connect logout session is unavailable"
			}
		}
	}
	if hasRefreshToken {
		if _, ok := server.authService.Logout(r.Context(), refreshToken).(auth.LogoutDone); !ok {
			writeError(w, http.StatusServiceUnavailable, "Sharecrop session could not be revoked")
			return "", false
		}
	}
	server.clearRefreshCookie(w)
	if coordinateError != "" {
		writeError(w, http.StatusServiceUnavailable, coordinateError)
		return "", false
	}
	if server.shauth.enabled() {
		if endpoint == "" {
			_, _, discoveredEndpoint, discoveryErr := server.shauth.discoveredProvider(r.Context())
			if discoveryErr != nil {
				writeError(w, http.StatusBadGateway, "Shauth discovery failed")
				return "", false
			}
			endpoint = discoveredEndpoint
		}
		var err error
		logoutURL, err = server.shauth.logoutURL(endpoint, idTokenHint)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Shauth logout endpoint is unavailable")
			return "", false
		}
	}
	return logoutURL, true
}

func (server Server) requireActiveBrowserSession(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("sharecrop_refresh_token")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/api/auth/shauth", http.StatusFound)
		return false
	}
	parsed, ok := auth.ParseRefreshTokenPlain(cookie.Value).(auth.RefreshTokenPlainAccepted)
	if !ok {
		server.clearRefreshCookie(w)
		http.Redirect(w, r, "/api/auth/shauth", http.StatusFound)
		return false
	}
	switch server.authService.ValidateSession(r.Context(), parsed.Value).(type) {
	case auth.RefreshTokenActive:
		return true
	case auth.RefreshTokenInactive:
		server.clearRefreshCookie(w)
		http.Redirect(w, r, "/api/auth/shauth", http.StatusFound)
		return false
	default:
		writeError(w, http.StatusServiceUnavailable, "Sharecrop session could not be validated")
		return false
	}
}

func (c shauthConfig) logoutURL(rawEndpoint, idTokenHint string) (string, error) {
	endpoint, err := url.Parse(rawEndpoint)
	if err != nil || !c.validOIDCCoordinate(endpoint) || endpoint.User != nil || endpoint.Fragment != "" {
		return "", errInvalidLogoutEndpoint
	}
	issuer, err := url.Parse(c.issuer)
	if err != nil || !sameURLOrigin(issuer, endpoint) {
		return "", errInvalidLogoutEndpoint
	}
	query := endpoint.Query()
	query.Set("client_id", c.clientID)
	if idTokenHint != "" {
		query.Set("id_token_hint", idTokenHint)
	}
	query.Set("post_logout_redirect_uri", c.logoutBridgeURI())
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

var errInvalidLogoutEndpoint = errors.New("invalid OpenID Connect end-session endpoint")

func (server Server) guest(w http.ResponseWriter, r *http.Request) {
	if !server.allowByIP(w, r) {
		return
	}
	result := server.authService.CreateGuest(r.Context())
	accepted, matched := result.(auth.GuestAccepted)
	if !matched {
		rejected := result.(auth.GuestRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	server.setRefreshCookie(w, accepted.RefreshToken)
	server.writeAuthResponse(w, http.StatusCreated, authResponse{
		SubjectKind: "guest",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}

func (server Server) requestEmailVerification(w http.ResponseWriter, r *http.Request) {
	if !server.allowByIP(w, r) {
		return
	}
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	result := server.authService.RequestEmailVerification(r.Context(), actor.subject.ID)
	issued, ok := result.(auth.AccountTokenIssued)
	if !ok {
		writeDomainError(w, result.(auth.AccountTokenIssueRejected).Reason)
		return
	}
	server.accountTokens.write(w, auth.AccountTokenKindEmailVerification, actor.subject.ID.String(), issued)
}

func (server Server) confirmEmailVerification(w http.ResponseWriter, r *http.Request) {
	if !server.allowByIP(w, r) {
		return
	}
	var request accountTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	tokenResult := auth.ParseAccountTokenPlain(request.Token)
	token, matched := tokenResult.(auth.AccountTokenPlainAccepted)
	if !matched {
		writeDomainError(w, tokenResult.(auth.AccountTokenPlainRejected).Reason)
		return
	}
	result := server.authService.VerifyEmail(r.Context(), token.Value)
	if _, ok := result.(auth.AccountActionAccepted); !ok {
		writeDomainError(w, result.(auth.AccountActionRejected).Reason)
		return
	}
	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "verified"})
}

func (server Server) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if !server.allowByIP(w, r) {
		return
	}
	var request passwordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	emailResult := auth.NewEmailAddress(request.Email)
	email, matched := emailResult.(auth.EmailAddressAccepted)
	if !matched {
		writeDomainError(w, emailResult.(auth.EmailAddressRejected).Reason)
		return
	}
	result := server.authService.RequestPasswordReset(r.Context(), email.Value)
	switch outcome := result.(type) {
	case auth.AccountTokenIssued:
		server.accountTokens.write(w, auth.AccountTokenKindPasswordReset, email.Value.String(), outcome)
	case auth.AccountTokenIssueIgnored:
		// Unknown email: respond exactly as a successful log-mode delivery so
		// the response never reveals whether the account exists.
		writeJSON(w, http.StatusCreated, accountTokenSentResponse{Status: "sent"})
	default:
		writeDomainError(w, result.(auth.AccountTokenIssueRejected).Reason)
	}
}

func (server Server) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	if !server.allowByIP(w, r) {
		return
	}
	var request passwordResetConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	tokenResult := auth.ParseAccountTokenPlain(request.Token)
	token, tokenMatched := tokenResult.(auth.AccountTokenPlainAccepted)
	if !tokenMatched {
		writeDomainError(w, tokenResult.(auth.AccountTokenPlainRejected).Reason)
		return
	}
	passwordResult := auth.NewPasswordSecret(request.Password)
	password, passwordMatched := passwordResult.(auth.PasswordSecretAccepted)
	if !passwordMatched {
		writeDomainError(w, passwordResult.(auth.PasswordSecretRejected).Reason)
		return
	}
	result := server.authService.ResetPassword(r.Context(), token.Value, password.Value)
	if _, ok := result.(auth.AccountActionAccepted); !ok {
		writeDomainError(w, result.(auth.AccountActionRejected).Reason)
		return
	}
	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "password_reset"})
}

func (server Server) changePassword(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	var request passwordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	current := auth.NewPasswordSecret(request.CurrentPassword)
	currentAccepted, currentMatched := current.(auth.PasswordSecretAccepted)
	if !currentMatched {
		writeDomainError(w, current.(auth.PasswordSecretRejected).Reason)
		return
	}
	next := auth.NewPasswordSecret(request.NewPassword)
	nextAccepted, nextMatched := next.(auth.PasswordSecretAccepted)
	if !nextMatched {
		writeDomainError(w, next.(auth.PasswordSecretRejected).Reason)
		return
	}
	result := server.authService.ChangePassword(r.Context(), actor.subject.ID, currentAccepted.Value, nextAccepted.Value)
	if _, ok := result.(auth.AccountActionAccepted); !ok {
		writeDomainError(w, result.(auth.AccountActionRejected).Reason)
		return
	}
	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "password_changed"})
}

func (server Server) updateAccountProfile(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	var request accountProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	emailResult := auth.NewEmailAddress(request.Email)
	email, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		writeDomainError(w, emailResult.(auth.EmailAddressRejected).Reason)
		return
	}
	result := server.authService.UpdateProfile(r.Context(), actor.subject.ID, email.Value)
	if _, ok := result.(auth.AccountActionAccepted); !ok {
		writeDomainError(w, result.(auth.AccountActionRejected).Reason)
		return
	}
	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "profile_updated"})
}

func (server Server) deactivateAccount(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	result := server.authService.DeactivateAccount(r.Context(), actor.subject.ID)
	if _, ok := result.(auth.AccountActionAccepted); !ok {
		writeDomainError(w, result.(auth.AccountActionRejected).Reason)
		return
	}
	if !server.recordAudit(w, r.Context(), actor.subject.ID, audit.ActionAccountDeactivated, audit.Subject{Kind: "user", ID: actor.subject.ID.String()}, audit.EmptyMetadata()) {
		return
	}
	server.clearRefreshCookie(w)
	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "deactivated"})
}
