package httpserver

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
)

type healthResponse struct {
	Status string `json:"status"`
}

type AuthService interface {
	Register(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.RegisterResult
	Login(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.LoginResult
	Refresh(context.Context, auth.RefreshTokenPlain) auth.RefreshResult
	CreateGuest(context.Context) auth.GuestResult
}

type Server struct {
	staticFiles fs.FS
	authService AuthService
}

func New(staticFiles fs.FS, authService AuthService) http.Handler {
	server := Server{staticFiles: staticFiles, authService: authService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("POST /api/auth/register", server.register)
	mux.HandleFunc("POST /api/auth/login", server.login)
	mux.HandleFunc("POST /api/auth/refresh", server.refresh)
	mux.HandleFunc("POST /api/auth/guest", server.guest)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	mux.HandleFunc("GET /", index(staticFiles))
	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}

func index(staticFiles fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(staticFiles, "index.html")
		if err != nil {
			http.Error(w, "index not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	AccessToken string `json:"access_token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type authRequestResult interface {
	authRequestResult()
}

type authRequestAccepted struct {
	email    auth.EmailAddress
	password auth.PasswordSecret
}

type authRequestRejected struct {
	reason string
}

func (authRequestAccepted) authRequestResult() {}

func (authRequestRejected) authRequestResult() {}

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

	setRefreshCookie(w, accepted.RefreshToken)
	writeAuthResponse(w, http.StatusCreated, authResponse{
		SubjectKind: "user",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}

func (server Server) login(w http.ResponseWriter, r *http.Request) {
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

	setRefreshCookie(w, accepted.RefreshToken)
	writeAuthResponse(w, http.StatusOK, authResponse{
		SubjectKind: "user",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}

func (server Server) refresh(w http.ResponseWriter, r *http.Request) {
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

	setRefreshCookie(w, accepted.RefreshToken)
	responseResult := authResponseForSubject(accepted.Subject, accepted.AccessToken)
	responseAccepted, responseMatched := responseResult.(authResponseAccepted)
	if !responseMatched {
		rejected := responseResult.(authResponseRejected)
		writeError(w, http.StatusInternalServerError, rejected.reason)
		return
	}

	writeAuthResponse(w, http.StatusOK, responseAccepted.response)
}

func (server Server) guest(w http.ResponseWriter, r *http.Request) {
	result := server.authService.CreateGuest(r.Context())
	accepted, matched := result.(auth.GuestAccepted)
	if !matched {
		rejected := result.(auth.GuestRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	setRefreshCookie(w, accepted.RefreshToken)
	writeAuthResponse(w, http.StatusCreated, authResponse{
		SubjectKind: "guest",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}

func decodeAuthRequest(r *http.Request) authRequestResult {
	var request authRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return authRequestRejected{reason: "request body is invalid"}
	}

	emailResult := auth.NewEmailAddress(request.Email)
	emailAccepted, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		rejected := emailResult.(auth.EmailAddressRejected)
		return authRequestRejected{reason: rejected.Reason.Description()}
	}

	passwordResult := auth.NewPasswordSecret(request.Password)
	passwordAccepted, passwordMatched := passwordResult.(auth.PasswordSecretAccepted)
	if !passwordMatched {
		rejected := passwordResult.(auth.PasswordSecretRejected)
		return authRequestRejected{reason: rejected.Reason.Description()}
	}

	return authRequestAccepted{email: emailAccepted.Value, password: passwordAccepted.Value}
}

type authResponseResult interface {
	authResponseResult()
}

type authResponseAccepted struct {
	response authResponse
}

type authResponseRejected struct {
	reason string
}

func (authResponseAccepted) authResponseResult() {}

func (authResponseRejected) authResponseResult() {}

func authResponseForSubject(subject auth.Subject, accessToken auth.AccessToken) authResponseResult {
	switch typed := subject.(type) {
	case auth.UserSubject:
		return authResponseAccepted{response: authResponse{SubjectKind: "user", SubjectID: typed.ID.String(), AccessToken: accessToken.String()}}
	case auth.GuestSubject:
		return authResponseAccepted{response: authResponse{SubjectKind: "guest", SubjectID: typed.ID.String(), AccessToken: accessToken.String()}}
	default:
		return authResponseRejected{reason: "subject is invalid"}
	}
}

func setRefreshCookie(w http.ResponseWriter, refreshToken auth.RefreshTokenPlain) {
	http.SetCookie(w, &http.Cookie{
		Name:     "sharecrop_refresh_token",
		Value:    refreshToken.String(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(30 * 24 * time.Hour),
	})
}

func writeAuthResponse(w http.ResponseWriter, status int, response authResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}
