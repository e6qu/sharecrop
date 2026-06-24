package httpserver

import (
	"net/http"

	"github.com/e6qu/sharecrop/internal/auth"
)

func (server Server) register(w http.ResponseWriter, r *http.Request) {
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
	writeAuthResponse(w, http.StatusCreated, authResponse{
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
	writeAuthResponse(w, http.StatusOK, authResponse{
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

	writeAuthResponse(w, http.StatusOK, responseAccepted.response)
}

func (server Server) logout(w http.ResponseWriter, r *http.Request) {
	server.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (server Server) guest(w http.ResponseWriter, r *http.Request) {
	result := server.authService.CreateGuest(r.Context())
	accepted, matched := result.(auth.GuestAccepted)
	if !matched {
		rejected := result.(auth.GuestRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	server.setRefreshCookie(w, accepted.RefreshToken)
	writeAuthResponse(w, http.StatusCreated, authResponse{
		SubjectKind: "guest",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}
