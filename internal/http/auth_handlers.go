package httpserver

import (
	"encoding/json"
	"net/http"

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

	server.writeAuthResponse(w, http.StatusOK, responseAccepted.response)
}

func (server Server) logout(w http.ResponseWriter, r *http.Request) {
	// Revoke the session family server-side (not just clear the cookie) so the
	// refresh token cannot resume the session if it was captured.
	if cookie, err := r.Cookie("sharecrop_refresh_token"); err == nil && cookie.Value != "" {
		if parsed, matched := auth.ParseRefreshTokenPlain(cookie.Value).(auth.RefreshTokenPlainAccepted); matched {
			server.authService.Logout(r.Context(), parsed.Value)
		}
	}
	server.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

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
