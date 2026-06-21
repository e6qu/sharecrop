package httpserver

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
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

type SubjectVerifier interface {
	Verify(auth.AccessToken) auth.SubjectVerifyResult
}

type OrganizationService interface {
	CreateOrganization(context.Context, auth.UserSubject, org.OrganizationName) org.CreateOrganizationResult
	ListOrganizations(context.Context, auth.UserSubject) org.ListOrganizationsResult
	ProvisionMember(context.Context, auth.UserSubject, core.OrganizationID, auth.EmailAddress, []org.Role) org.ProvisionMemberResult
	DeactivateMember(context.Context, auth.UserSubject, core.OrganizationID, core.UserID) org.DeactivateMemberResult
	CreateOrganizationTeam(context.Context, auth.UserSubject, core.OrganizationID, org.TeamName) org.CreateTeamResult
	ListOrganizationTeams(context.Context, auth.UserSubject, core.OrganizationID) org.ListTeamsResult
}

type Server struct {
	staticFiles         fs.FS
	authService         AuthService
	subjectVerifier     SubjectVerifier
	organizationService OrganizationService
}

func New(staticFiles fs.FS, authService AuthService, subjectVerifier SubjectVerifier, organizationService OrganizationService) http.Handler {
	server := Server{staticFiles: staticFiles, authService: authService, subjectVerifier: subjectVerifier, organizationService: organizationService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("POST /api/auth/register", server.register)
	mux.HandleFunc("POST /api/auth/login", server.login)
	mux.HandleFunc("POST /api/auth/refresh", server.refresh)
	mux.HandleFunc("POST /api/auth/guest", server.guest)
	mux.HandleFunc("GET /api/organizations", server.listOrganizations)
	mux.HandleFunc("POST /api/organizations", server.createOrganization)
	mux.HandleFunc("POST /api/organizations/{organization_id}/members", server.provisionOrganizationMember)
	mux.HandleFunc("PATCH /api/organizations/{organization_id}/members/{user_id}/deactivate", server.deactivateOrganizationMember)
	mux.HandleFunc("GET /api/organizations/{organization_id}/teams", server.listOrganizationTeams)
	mux.HandleFunc("POST /api/organizations/{organization_id}/teams", server.createOrganizationTeam)
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

type organizationRequest struct {
	Name string `json:"name"`
}

type provisionMemberRequest struct {
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type teamRequest struct {
	Name string `json:"name"`
}

type organizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

type organizationsResponse struct {
	Organizations []organizationResponse `json:"organizations"`
}

type organizationMemberResponse struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Roles          []string `json:"roles"`
}

type teamResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	CreatedBy      string `json:"created_by"`
}

type teamsResponse struct {
	Teams []teamResponse `json:"teams"`
}

type emptyResponse struct {
	Status string `json:"status"`
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

func (server Server) createOrganization(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	var request organizationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	nameResult := org.NewOrganizationName(request.Name)
	nameAccepted, nameMatched := nameResult.(org.OrganizationNameAccepted)
	if !nameMatched {
		rejected := nameResult.(org.OrganizationNameRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.CreateOrganization(r.Context(), actor.subject, nameAccepted.Value)
	created, matched := result.(org.OrganizationCreated)
	if !matched {
		rejected := result.(org.CreateOrganizationRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	writeOrganizationResponse(w, http.StatusCreated, organizationToResponse(created.Value))
}

func (server Server) listOrganizations(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	result := server.organizationService.ListOrganizations(r.Context(), actor.subject)
	listed, matched := result.(org.OrganizationsListed)
	if !matched {
		rejected := result.(org.ListOrganizationsRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	response := organizationsResponse{Organizations: make([]organizationResponse, 0, len(listed.Values))}
	for _, organization := range listed.Values {
		response.Organizations = append(response.Organizations, organizationToResponse(organization))
	}
	writeOrganizationsResponse(w, http.StatusOK, response)
}

func (server Server) provisionOrganizationMember(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	requestResult := decodeProvisionMemberRequest(r)
	requestAccepted, requestMatched := requestResult.(provisionMemberAccepted)
	if !requestMatched {
		rejected := requestResult.(provisionMemberRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.organizationService.ProvisionMember(r.Context(), actor.subject, organizationIDAccepted.value, requestAccepted.email, requestAccepted.roles)
	provisioned, matched := result.(org.MemberProvisioned)
	if !matched {
		rejected := result.(org.ProvisionMemberRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeOrganizationMemberResponse(w, http.StatusCreated, memberToResponse(provisioned.Value))
}

func (server Server) deactivateOrganizationMember(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userIDAccepted, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		rejected := userIDResult.(core.UserIDRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.DeactivateMember(r.Context(), actor.subject, organizationIDAccepted.value, userIDAccepted.Value)
	if _, matched := result.(org.MemberDeactivationAccepted); !matched {
		rejected := result.(org.DeactivateMemberRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "deactivated"})
}

func (server Server) createOrganizationTeam(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	var request teamRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	nameResult := org.NewTeamName(request.Name)
	nameAccepted, nameMatched := nameResult.(org.TeamNameAccepted)
	if !nameMatched {
		rejected := nameResult.(org.TeamNameRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.CreateOrganizationTeam(r.Context(), actor.subject, organizationIDAccepted.value, nameAccepted.Value)
	created, matched := result.(org.TeamCreated)
	if !matched {
		rejected := result.(org.CreateTeamRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeTeamResponse(w, http.StatusCreated, teamToResponse(created.Value))
}

func (server Server) listOrganizationTeams(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.organizationService.ListOrganizationTeams(r.Context(), actor.subject, organizationIDAccepted.value)
	listed, matched := result.(org.OrganizationTeamsListed)
	if !matched {
		rejected := result.(org.ListTeamsRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	response := teamsResponse{Teams: make([]teamResponse, 0, len(listed.Values))}
	for _, team := range listed.Values {
		response.Teams = append(response.Teams, teamToResponse(team))
	}
	writeTeamsResponse(w, http.StatusOK, response)
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

type userSubjectResult interface {
	userSubjectResult()
}

type userSubjectAccepted struct {
	subject auth.UserSubject
}

type userSubjectRejected struct {
	reason string
}

func (userSubjectAccepted) userSubjectResult() {}

func (userSubjectRejected) userSubjectResult() {}

func (server Server) requireUserSubject(r *http.Request) userSubjectResult {
	rawHeader := r.Header.Get("Authorization")
	rawToken, matched := strings.CutPrefix(rawHeader, "Bearer ")
	if !matched {
		return userSubjectRejected{reason: "bearer access token is required"}
	}

	tokenResult := auth.ParseAccessToken(rawToken)
	tokenAccepted, tokenMatched := tokenResult.(auth.AccessTokenParsed)
	if !tokenMatched {
		rejected := tokenResult.(auth.AccessTokenParseRejected)
		return userSubjectRejected{reason: rejected.Reason.Description()}
	}

	verifyResult := server.subjectVerifier.Verify(tokenAccepted.Value)
	verified, verifyMatched := verifyResult.(auth.SubjectVerified)
	if !verifyMatched {
		rejected := verifyResult.(auth.SubjectVerifyRejected)
		return userSubjectRejected{reason: rejected.Reason.Description()}
	}

	subject, subjectMatched := verified.Value.(auth.UserSubject)
	if !subjectMatched {
		return userSubjectRejected{reason: "user access token is required"}
	}

	return userSubjectAccepted{subject: subject}
}

type organizationIDResult interface {
	organizationIDResult()
}

type organizationIDAccepted struct {
	value core.OrganizationID
}

type organizationIDRejected struct {
	reason string
}

func (organizationIDAccepted) organizationIDResult() {}

func (organizationIDRejected) organizationIDResult() {}

func parseOrganizationPathValue(r *http.Request) organizationIDResult {
	result := core.ParseOrganizationID(r.PathValue("organization_id"))
	accepted, matched := result.(core.OrganizationIDCreated)
	if !matched {
		rejected := result.(core.OrganizationIDRejected)
		return organizationIDRejected{reason: rejected.Reason.Description()}
	}
	return organizationIDAccepted{value: accepted.Value}
}

type provisionMemberResult interface {
	provisionMemberResult()
}

type provisionMemberAccepted struct {
	email auth.EmailAddress
	roles []org.Role
}

type provisionMemberRejected struct {
	reason string
}

func (provisionMemberAccepted) provisionMemberResult() {}

func (provisionMemberRejected) provisionMemberResult() {}

func decodeProvisionMemberRequest(r *http.Request) provisionMemberResult {
	var request provisionMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return provisionMemberRejected{reason: "request body is invalid"}
	}

	emailResult := auth.NewEmailAddress(request.Email)
	emailAccepted, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		rejected := emailResult.(auth.EmailAddressRejected)
		return provisionMemberRejected{reason: rejected.Reason.Description()}
	}

	roles := make([]org.Role, 0, len(request.Roles))
	for _, rawRole := range request.Roles {
		roleResult := org.ParseRole(rawRole)
		roleAccepted, roleMatched := roleResult.(org.RoleAccepted)
		if !roleMatched {
			rejected := roleResult.(org.RoleRejected)
			return provisionMemberRejected{reason: rejected.Reason.Description()}
		}
		roles = append(roles, roleAccepted.Value)
	}

	if len(roles) == 0 {
		return provisionMemberRejected{reason: "at least one organization role is required"}
	}

	return provisionMemberAccepted{email: emailAccepted.Value, roles: roles}
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

func organizationToResponse(value org.Organization) organizationResponse {
	return organizationResponse{ID: value.ID.String(), Name: value.Name.String(), CreatedBy: value.CreatedBy.String()}
}

func memberToResponse(value org.OrganizationMember) organizationMemberResponse {
	roles := make([]string, 0, len(value.Roles))
	for _, role := range value.Roles {
		roles = append(roles, role.String())
	}
	return organizationMemberResponse{
		ID:             value.ID.String(),
		OrganizationID: value.OrganizationID.String(),
		UserID:         value.UserID.String(),
		Status:         value.Status.String(),
		Roles:          roles,
	}
}

func teamToResponse(value org.Team) teamResponse {
	return teamResponse{ID: value.ID.String(), OrganizationID: value.OrganizationID.String(), Name: value.Name.String(), CreatedBy: value.CreatedBy.String()}
}

func writeAuthResponse(w http.ResponseWriter, status int, response authResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeOrganizationResponse(w http.ResponseWriter, status int, response organizationResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeOrganizationsResponse(w http.ResponseWriter, status int, response organizationsResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeOrganizationMemberResponse(w http.ResponseWriter, status int, response organizationMemberResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeTeamResponse(w http.ResponseWriter, status int, response teamResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeTeamsResponse(w http.ResponseWriter, status int, response teamsResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeEmptyResponse(w http.ResponseWriter, status int, response emptyResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}
