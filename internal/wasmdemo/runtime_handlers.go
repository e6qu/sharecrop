package wasmdemo

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

type RuntimeIDSource interface {
	NextUserID() string
	NextAuditEventID() string
	NextAccountToken() string
	NextCollectibleID() string
	NextNotificationID() string
	NextAgentCredentialID() string
	NextAgentCredentialSecret() string
}

type AuthHandler struct {
	storage BrowserStorage
	clock   HandlerClock
	actor   HandlerActor
	ids     RuntimeIDSource
}

func NewAuthHandler(storage BrowserStorage, clock HandlerClock, actor HandlerActor, ids RuntimeIDSource) AuthHandler {
	return AuthHandler{storage: storage, clock: clock, actor: actor, ids: ids}
}

func (handler AuthHandler) Handle(request Request) HandleResult {
	if handler.storage == nil || handler.clock == nil || handler.actor == nil || handler.ids == nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "auth handler dependencies are required")}
	}
	switch request.Path {
	case "/api/auth/refresh":
		return handler.authResponse(handler.actor.UserID(), 200)
	case "/api/auth/login", "/api/auth/register":
		var body authEmailBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "auth body is invalid")}
		}
		userID, err := LoadUserIDByEmail(handler.storage, body.Email)
		if err != nil {
			userID = strings.TrimSpace(handler.ids.NextUserID())
			if saveErr := SaveUser(handler.storage, StoredUser{ID: userID, Email: body.Email, Status: "active"}); saveErr != nil {
				return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, saveErr.Error())}
			}
		}
		status := 200
		if request.Path == "/api/auth/register" {
			status = 201
		}
		return handler.authResponse(userID, status)
	case "/api/auth/guest":
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "anonymous worker identity is not supported")}
	case "/api/auth/logout":
		return RequestHandled{Value: Response{Status: 204, Body: ""}}
	case "/api/auth/email-verification/confirm":
		return handler.consumeToken(request, "email_verification", "verified")
	case "/api/auth/password-reset/request":
		var body authEmailBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "password reset body is invalid")}
		}
		userID, err := LoadUserIDByEmail(handler.storage, body.Email)
		if err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "user was not found")}
		}
		return handler.issueToken(userID, "password_reset")
	case "/api/auth/password-reset/confirm":
		return handler.consumeToken(request, "password_reset", "password_reset")
	default:
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "request route is not implemented by the WASM demo handler")}
	}
}

func (handler AuthHandler) authResponse(userID string, status int) HandleResult {
	user, err := LoadUser(handler.storage, userID)
	if err != nil {
		user = StoredUser{ID: userID, Email: userID + "@sharecrop.demo", Status: "active"}
		if saveErr := SaveUser(handler.storage, user); saveErr != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, saveErr.Error())}
		}
	}
	encoded, err := json.Marshal(authResponseBody{SubjectKind: "user", SubjectID: user.ID, AccessToken: "wasm-access-" + user.ID, Role: "admin"})
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "auth response encoding failed")}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

func (handler AuthHandler) issueToken(userID string, kind string) HandleResult {
	token := strings.TrimSpace(handler.ids.NextAccountToken())
	if err := SaveAccountToken(handler.storage, StoredAccountToken{Token: token, Kind: kind, UserID: userID, State: "active"}); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	encoded, err := json.Marshal(accountTokenResponseBody{Token: token})
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "account token response encoding failed")}
	}
	return RequestHandled{Value: Response{Status: 201, Body: string(encoded)}}
}

func (handler AuthHandler) consumeToken(request Request, kind string, statusValue string) HandleResult {
	var body accountTokenBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "account token body is invalid")}
	}
	if err := ConsumeAccountToken(handler.storage, body.Token, kind); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	encoded, err := json.Marshal(statusResponseBody{Status: statusValue})
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "account token consume response encoding failed")}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

type AccountHandler struct {
	auth AuthHandler
}

func NewAccountHandler(storage BrowserStorage, clock HandlerClock, actor HandlerActor, ids RuntimeIDSource) AccountHandler {
	return AccountHandler{auth: NewAuthHandler(storage, clock, actor, ids)}
}

func (handler AccountHandler) Handle(request Request) HandleResult {
	switch request.Path {
	case "/api/account/email-verification":
		return handler.auth.issueToken(handler.auth.actor.UserID(), "email_verification")
	case "/api/account/password":
		return statusResponse("password_changed", 200)
	case "/api/account/profile":
		return statusResponse("profile_updated", 200)
	case "/api/account":
		if request.Method.String() != MethodDelete.String() {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "request method is unsupported for account")}
		}
		return statusResponse("deactivated", 200)
	default:
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "request route is not implemented by the WASM demo handler")}
	}
}

type UsersHandler struct {
	storage BrowserStorage
}

func NewUsersHandler(storage BrowserStorage) UsersHandler {
	return UsersHandler{storage: storage}
}

func (handler UsersHandler) Handle(request Request) HandleResult {
	if handler.storage == nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "browser storage is required")}
	}
	if request.Method.String() != MethodGet.String() {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "request method is unsupported for users")}
	}
	pageResult := storedListPageFromPath(request.Path, "user")
	page, pageMatched := pageResult.(storedListPageFromPathAccepted)
	if !pageMatched {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, pageResult.(storedListPageFromPathRejected).reason)}
	}
	pathOnly := strings.SplitN(request.Path, "?", 2)[0]
	if pathOnly != "/api/users" {
		userID := usersPath(request.Path)
		if _, err := LoadUser(handler.storage, userID); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		// The real backend's GET /api/users/{user_id} (getUserProfile) returns
		// {id, tasks: [...tasks this user created...]}, not the stored user
		// record itself; the browser's profile page decodes exactly that
		// shape and fails otherwise.
		listResult := ListTasks(handler.storage, "", "user", userID, "", "", DefaultStoredListPage())
		listed, listedMatched := listResult.(TasksStored)
		if !listedMatched {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, listResult.(TaskStorageRejected).Reason)}
		}
		encoded, err := json.Marshal(userProfileResponseBody{ID: userID, Tasks: taskSummaries(listed.Values)})
		if err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "user profile response encoding failed")}
		}
		return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
	}
	values, err := url.ParseQuery(queryStringFromPath(request.Path))
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "user query is invalid")}
	}
	users, err := ListUsers(handler.storage, values.Get("query"), page.value)
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	encoded, err := json.Marshal(usersResponseBody{Users: users})
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "users response encoding failed")}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

type AdminHandler struct {
	storage BrowserStorage
	clock   HandlerClock
	actor   HandlerActor
	ids     RuntimeIDSource
}

func NewAdminHandler(storage BrowserStorage, clock HandlerClock, actor HandlerActor, ids RuntimeIDSource) AdminHandler {
	return AdminHandler{storage: storage, clock: clock, actor: actor, ids: ids}
}

func (handler AdminHandler) Handle(request Request, route Route) HandleResult {
	if handler.storage == nil || handler.clock == nil || handler.actor == nil || handler.ids == nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "admin handler dependencies are required")}
	}
	switch route.String() {
	case RouteAdminOperations.String():
		return marshalResponse(operationsBody{Status: "ok", AccountTokenDelivery: "api", MCPStorage: "wasm_host", RateLimitStorage: "wasm_host", ActiveMCPSessions: 0, ActiveIPRateBuckets: 0, ActiveSubjectRateBuckets: 0, SecureCookies: "disabled"}, 200, "operations")
	case RoutePlatformAdmins.String():
		return handler.handlePlatformAdmins(request)
	case RouteAuditEvents.String():
		return handler.handleAuditEvents(request)
	case RouteAdminModerationReports.String():
		return handler.handleAdminModerationReports(request)
	case RouteAdminPrivacyRetention.String():
		if err := handler.recordAudit("privacy_retention_run", "privacy_retention", handler.actor.UserID(), `{"redacted_field_count":"0"}`); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		return marshalResponse(retentionResponseBody{RedactedFieldCount: 0}, 200, "privacy retention")
	default:
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "request route is not implemented by the WASM demo handler")}
	}
}

func (handler AdminHandler) handleAdminModerationReports(request Request) HandleResult {
	if request.Method.String() == MethodPost.String() {
		reportID := moderationTriagePathOnly(request.Path)
		if reportID == "" {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation triage route is invalid")}
		}
		var body moderationTriageBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation triage request body is invalid")}
		}
		triage := StoredModerationTriage{ReportID: reportID, State: strings.TrimSpace(body.State), ResolutionNote: strings.TrimSpace(body.ResolutionNote), UpdatedBy: handler.actor.UserID(), UpdatedAt: handler.clock.Now().UTC().Format(time.RFC3339)}
		if triage.State == "" {
			triage.State = "open"
		}
		saveResult := SaveModerationTriage(handler.storage, triage)
		if _, matched := saveResult.(ModerationTriageStored); !matched {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, saveResult.(ModerationTriageStorageRejected).Reason)}
		}
		if err := handler.recordAudit("moderation_report_triaged", "moderation_report", reportID, `{"state":"`+triage.State+`"}`); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		report, err := handler.moderationReportFromID(reportID)
		if err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		return marshalResponse(report, 200, "moderation report")
	}
	pageResult := storedListPageFromPath(request.Path, "moderation report")
	page, pageMatched := pageResult.(storedListPageFromPathAccepted)
	if !pageMatched {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, pageResult.(storedListPageFromPathRejected).reason)}
	}
	values, err := url.ParseQuery(queryStringFromPath(request.Path))
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation report query is invalid")}
	}
	events, err := ListAuditEvents(handler.storage, "moderation_report_created", "", "", DefaultStoredListPage())
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	reports := make([]moderationReportResponse, 0, len(events))
	for index := range events {
		report, err := handler.moderationReportFromEvent(events[index])
		if err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		if values.Get("state") == "" || report.State == values.Get("state") {
			reports = append(reports, report)
		}
	}
	start, end := pageBounds(len(reports), page.value)
	return marshalResponse(moderationReportsBody{Reports: reports[start:end]}, 200, "moderation reports")
}

func (handler AdminHandler) moderationReportFromID(reportID string) (moderationReportResponse, error) {
	events, err := ListAuditEvents(handler.storage, "moderation_report_created", "", "", DefaultStoredListPage())
	if err != nil {
		return moderationReportResponse{}, err
	}
	for index := range events {
		if events[index].ID == strings.TrimSpace(reportID) {
			return handler.moderationReportFromEvent(events[index])
		}
	}
	return moderationReportResponse{}, errString("moderation report was not found")
}

func (handler AdminHandler) moderationReportFromEvent(event StoredAuditEvent) (moderationReportResponse, error) {
	triageResult := LoadModerationTriage(handler.storage, event.ID)
	triage, matched := triageResult.(ModerationTriageStored)
	if !matched {
		return moderationReportResponse{}, errString(triageResult.(ModerationTriageStorageRejected).Reason)
	}
	metadata := moderationMetadata{}
	if err := json.Unmarshal([]byte(event.MetadataJSON), &metadata); err != nil {
		return moderationReportResponse{}, errString("moderation report metadata is invalid")
	}
	return moderationReportResponse{
		ID:             event.ID,
		SubjectKind:    event.SubjectKind,
		SubjectID:      event.SubjectID,
		Reason:         metadata.Reason,
		Details:        metadata.Details,
		ReporterUserID: event.ActorID,
		SubjectHref:    moderationSubjectHref(event.SubjectKind, event.SubjectID),
		State:          triage.Value.State,
		ResolutionNote: triage.Value.ResolutionNote,
		UpdatedBy:      triage.Value.UpdatedBy,
		UpdatedAt:      triage.Value.UpdatedAt,
		CreatedAt:      event.CreatedAt,
	}, nil
}

type ModerationReportHandler struct {
	storage BrowserStorage
	clock   HandlerClock
	actor   HandlerActor
	ids     RuntimeIDSource
}

func NewModerationReportHandler(storage BrowserStorage, clock HandlerClock, actor HandlerActor, ids RuntimeIDSource) ModerationReportHandler {
	return ModerationReportHandler{storage: storage, clock: clock, actor: actor, ids: ids}
}

func (handler ModerationReportHandler) Handle(request Request) HandleResult {
	if handler.storage == nil || handler.clock == nil || handler.actor == nil || handler.ids == nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "moderation report handler dependencies are required")}
	}
	if request.Method.String() != MethodPost.String() || request.Path != "/api/moderation/reports" {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "request route is not implemented by the WASM demo handler")}
	}
	var body moderationReportBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation report body is invalid")}
	}
	reportID := handler.ids.NextAuditEventID()
	now := handler.clock.Now().UTC().Format(time.RFC3339)
	metadata, err := json.Marshal(moderationMetadata{Reason: body.Reason, Details: body.Details})
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "moderation report metadata encoding failed")}
	}
	if err := SaveAuditEvent(handler.storage, StoredAuditEvent{ID: reportID, ActorID: handler.actor.UserID(), Action: "moderation_report_created", SubjectKind: body.SubjectKind, SubjectID: body.SubjectID, MetadataJSON: string(metadata), CreatedAt: now}); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	saveResult := SaveModerationTriage(handler.storage, StoredModerationTriage{ReportID: reportID, State: "open", ResolutionNote: "", UpdatedBy: "", UpdatedAt: now})
	if _, matched := saveResult.(ModerationTriageStored); !matched {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, saveResult.(ModerationTriageStorageRejected).Reason)}
	}
	response := moderationReportResponse{ID: reportID, SubjectKind: body.SubjectKind, SubjectID: body.SubjectID, Reason: body.Reason, Details: body.Details, ReporterUserID: handler.actor.UserID(), SubjectHref: moderationSubjectHref(body.SubjectKind, body.SubjectID), State: "open", ResolutionNote: "", UpdatedBy: "", UpdatedAt: now, CreatedAt: now}
	return marshalResponse(response, 201, "moderation report")
}

func (handler AdminHandler) handlePlatformAdmins(request Request) HandleResult {
	pageResult := storedListPageFromPath(request.Path, "platform admin")
	page, pageMatched := pageResult.(storedListPageFromPathAccepted)
	if !pageMatched {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, pageResult.(storedListPageFromPathRejected).reason)}
	}
	if request.Method.String() == MethodGet.String() {
		admins, err := ListPlatformAdmins(handler.storage, page.value)
		if err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		return marshalResponse(platformAdminsResponseBody{Admins: admins}, 200, "platform admins")
	}
	if request.Method.String() != MethodPost.String() {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "request method is unsupported for platform admins")}
	}
	revokeID := platformAdminRoute(request.Path)
	if revokeID != "" && revokeID != "collection" {
		admin, err := LoadPlatformAdmin(handler.storage, revokeID)
		if err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		admin.State = "revoked"
		if err := SavePlatformAdmin(handler.storage, admin); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		if err := handler.recordAudit("platform_admin_revoked", "user", admin.UserID, `{"source":"`+admin.Source+`"}`); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		return marshalResponse(admin, 200, "platform admin revoke")
	}
	var body platformAdminGrantBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "platform admin body is invalid")}
	}
	admin := StoredPlatformAdmin{UserID: strings.TrimSpace(body.UserID), Source: "granted", State: "active", CreatedAt: handler.clock.Now().UTC().Format(time.RFC3339)}
	if err := SavePlatformAdmin(handler.storage, admin); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	if err := handler.recordAudit("platform_admin_granted", "user", admin.UserID, `{"source":"granted"}`); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	return marshalResponse(admin, 201, "platform admin grant")
}

func (handler AdminHandler) handleAuditEvents(request Request) HandleResult {
	pageResult := storedListPageFromPath(request.Path, "audit")
	page, pageMatched := pageResult.(storedListPageFromPathAccepted)
	if !pageMatched {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, pageResult.(storedListPageFromPathRejected).reason)}
	}
	values, err := url.ParseQuery(queryStringFromPath(request.Path))
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "audit query is invalid")}
	}
	// GET /api/organizations/{organization_id}/audit-events scopes the list to
	// that organization's own subject id, matching internal/http's
	// listOrganizationAuditEvents; the platform-admin-wide
	// GET /api/admin/audit-events instead takes subject_id as a free filter.
	subjectID := values.Get("subject_id")
	if organizationID := organizationAuditEventsPathID(request.Path); organizationID != "" {
		subjectID = organizationID
	}
	events, err := ListAuditEvents(handler.storage, values.Get("action"), values.Get("subject_kind"), subjectID, page.value)
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	return marshalResponse(auditEventsResponseBody{Events: events}, 200, "audit events")
}

func (handler AdminHandler) recordAudit(action string, subjectKind string, subjectID string, metadataJSON string) error {
	return SaveAuditEvent(handler.storage, StoredAuditEvent{ID: strings.TrimSpace(handler.ids.NextAuditEventID()), ActorID: handler.actor.UserID(), Action: action, SubjectKind: subjectKind, SubjectID: subjectID, MetadataJSON: metadataJSON, CreatedAt: handler.clock.Now().UTC().Format(time.RFC3339)})
}

type CollectibleHandler struct {
	storage BrowserStorage
	actor   HandlerActor
	ids     RuntimeIDSource
}

func NewCollectibleHandler(storage BrowserStorage, actor HandlerActor, ids RuntimeIDSource) CollectibleHandler {
	return CollectibleHandler{storage: storage, actor: actor, ids: ids}
}

type AgentCredentialHandler struct {
	storage BrowserStorage
	actor   HandlerActor
	ids     RuntimeIDSource
}

func NewAgentCredentialHandler(storage BrowserStorage, actor HandlerActor, ids RuntimeIDSource) AgentCredentialHandler {
	return AgentCredentialHandler{storage: storage, actor: actor, ids: ids}
}

func (handler AgentCredentialHandler) Handle(request Request) HandleResult {
	if handler.storage == nil || handler.actor == nil || handler.ids == nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "agent credential handler dependencies are required")}
	}
	if request.Method.String() == MethodGet.String() && agentCredentialsRoute(request.Path) == "collection" {
		pageResult := storedListPageFromPath(request.Path, "agent credential")
		page, pageMatched := pageResult.(storedListPageFromPathAccepted)
		if !pageMatched {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, pageResult.(storedListPageFromPathRejected).reason)}
		}
		credentials, err := ListAgentCredentials(handler.storage, handler.actor.UserID(), page.value)
		if err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		return marshalResponse(agentCredentialsResponseBody{Credentials: agentCredentialsToResponse(credentials)}, 200, "agent credentials")
	}
	if request.Method.String() == MethodPost.String() && agentCredentialsRoute(request.Path) == "collection" {
		var body agentCredentialRequestBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential body is invalid")}
		}
		label := strings.TrimSpace(body.Label)
		if label == "" {
			label = "Agent"
		}
		credential := StoredAgentCredential{ID: strings.TrimSpace(handler.ids.NextAgentCredentialID()), OwnerID: handler.actor.UserID(), Label: label, Scopes: body.Scopes, State: "active", ExpiresAt: strings.TrimSpace(body.ExpiresAt)}
		if err := SaveAgentCredential(handler.storage, credential); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		response := agentCredentialCreatedResponseBody{
			Credential: agentCredentialToResponse(credential),
			Secret:     strings.TrimSpace(handler.ids.NextAgentCredentialSecret()),
		}
		if response.Secret == "" {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential secret is required")}
		}
		return marshalResponse(response, 201, "agent credential")
	}
	revokeID := agentCredentialsRoute(request.Path)
	if request.Method.String() == MethodPost.String() && revokeID != "" && revokeID != "collection" {
		credential, err := LoadAgentCredential(handler.storage, revokeID)
		if err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		if credential.OwnerID != handler.actor.UserID() {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "agent credential owner mismatch")}
		}
		credential.State = "revoked"
		if err := SaveAgentCredential(handler.storage, credential); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		return marshalResponse(agentCredentialToResponse(credential), 200, "agent credential revoke")
	}
	return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "request route is not implemented by the WASM demo handler")}
}

func (handler CollectibleHandler) Handle(request Request) HandleResult {
	if handler.storage == nil || handler.actor == nil || handler.ids == nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible handler dependencies are required")}
	}
	pathOnly := strings.SplitN(request.Path, "?", 2)[0]
	switch {
	case pathOnly == "/api/collectibles/catalog":
		return marshalResponse(collectibleCatalogBody{Entries: defaultCollectibleCatalog()}, 200, "collectible catalog")
	case pathOnly == "/api/collectibles" && request.Method.String() == MethodGet.String():
		pageResult := storedListPageFromPath(request.Path, "collectible")
		page, pageMatched := pageResult.(storedListPageFromPathAccepted)
		if !pageMatched {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, pageResult.(storedListPageFromPathRejected).reason)}
		}
		collectibles, err := ListCollectibles(handler.storage, "user", handler.actor.UserID(), page.value)
		if err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		return marshalResponse(collectiblesResponseBody{Collectibles: collectibles}, 200, "collectibles")
	case pathOnly == "/api/collectibles" && request.Method.String() == MethodPost.String():
		var body collectibleBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible body is invalid")}
		}
		collectible := StoredCollectible{ID: handler.ids.NextCollectibleID(), Name: body.Name, Kind: body.Kind, State: "minted", TransferPolicy: body.TransferPolicy, OwnerID: handler.actor.UserID(), OwnerKind: "user", Art: body.Art}
		if collectible.Name == "" {
			collectible.Name = "Collectible"
		}
		if collectible.Kind == "" {
			collectible.Kind = "badge"
		}
		if collectible.TransferPolicy == "" {
			collectible.TransferPolicy = "transferable_between_users"
		}
		if err := SaveCollectible(handler.storage, collectible); err != nil {
			return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
		}
		return marshalResponse(collectible, 201, "collectible")
	case pathOnly == "/api/collectibles/award":
		return handler.handleAward(request)
	case strings.HasPrefix(pathOnly, "/api/collectibles/"):
		return handler.handleTransfer(request)
	case strings.Contains(pathOnly, "/collectibles"):
		return handler.handleScopedList(request)
	default:
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "request route is not implemented by the WASM demo handler")}
	}
}

func (handler CollectibleHandler) handleAward(request Request) HandleResult {
	var body collectibleAwardBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible award body is invalid")}
	}
	entry := defaultCollectibleCatalog()[0]
	for _, candidate := range defaultCollectibleCatalog() {
		if candidate.Slug == body.Slug {
			entry = candidate
		}
	}
	collectible := StoredCollectible{ID: handler.ids.NextCollectibleID(), Name: entry.Name, Kind: entry.Kind, State: "awarded", TransferPolicy: entry.TransferPolicy, OwnerID: body.RecipientID, OwnerKind: body.RecipientKind, OrganizationID: body.OrganizationID, Art: entry.Art}
	if collectible.OwnerKind == "" {
		collectible.OwnerKind = "user"
	}
	if err := SaveCollectible(handler.storage, collectible); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	return marshalResponse(collectible, 201, "collectible award")
}

func (handler CollectibleHandler) handleTransfer(request Request) HandleResult {
	parts := strings.Split(strings.Trim(strings.SplitN(request.Path, "?", 2)[0], "/"), "/")
	if len(parts) != 4 || parts[3] != "transfer" {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible transfer route is invalid")}
	}
	collectible, err := LoadCollectible(handler.storage, parts[2])
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	var body collectibleTransferBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible transfer body is invalid")}
	}
	if collectible.TransferPolicy != "transferable_between_users" {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "this collectible cannot be traded")}
	}
	collectible.OwnerKind = "user"
	collectible.OwnerID = strings.TrimSpace(body.RecipientID)
	if err := SaveCollectible(handler.storage, collectible); err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	return marshalResponse(collectible, 200, "collectible transfer")
}

func (handler CollectibleHandler) handleScopedList(request Request) HandleResult {
	parts := strings.Split(strings.Trim(strings.SplitN(request.Path, "?", 2)[0], "/"), "/")
	if len(parts) != 4 {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible scoped route is invalid")}
	}
	ownerKind := "organization"
	if parts[1] == "teams" {
		ownerKind = "team"
	}
	collectibles, err := ListCollectibles(handler.storage, ownerKind, parts[2], DefaultStoredListPage())
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, err.Error())}
	}
	return marshalResponse(collectiblesResponseBody{Collectibles: collectibles}, 200, "collectibles")
}

func statusResponse(statusValue string, status int) HandleResult {
	return marshalResponse(statusResponseBody{Status: statusValue}, status, "status")
}

type runtimeResponseValue interface {
	operationsBody |
		retentionResponseBody |
		moderationReportResponse |
		moderationReportsBody |
		platformAdminsResponseBody |
		StoredPlatformAdmin |
		auditEventsResponseBody |
		collectibleCatalogBody |
		collectiblesResponseBody |
		StoredCollectible |
		agentCredentialResponseBody |
		agentCredentialCreatedResponseBody |
		agentCredentialsResponseBody |
		statusResponseBody
}

func marshalResponse[T runtimeResponseValue](value T, status int, label string) HandleResult {
	encoded, err := json.Marshal(value)
	if err != nil {
		return RequestHandleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, label + " response encoding failed")}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

type authEmailBody struct {
	Email string `json:"email"`
}

type authResponseBody struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	AccessToken string `json:"access_token"`
	Role        string `json:"role"`
}

type accountTokenBody struct {
	Token string `json:"token"`
}

type accountTokenResponseBody struct {
	Token string `json:"token"`
}

type statusResponseBody struct {
	Status string `json:"status"`
}

type usersResponseBody struct {
	Users []StoredUser `json:"users"`
}

type operationsBody struct {
	Status                   string `json:"status"`
	AccountTokenDelivery     string `json:"account_token_delivery"`
	MCPStorage               string `json:"mcp_storage"`
	RateLimitStorage         string `json:"rate_limit_storage"`
	ActiveMCPSessions        int    `json:"active_mcp_sessions"`
	ActiveIPRateBuckets      int    `json:"active_ip_rate_buckets"`
	ActiveSubjectRateBuckets int    `json:"active_subject_rate_buckets"`
	SecureCookies            string `json:"secure_cookies"`
}

type platformAdminGrantBody struct {
	UserID string `json:"user_id"`
}

type platformAdminsResponseBody struct {
	Admins []StoredPlatformAdmin `json:"admins"`
}

type auditEventsResponseBody struct {
	Events []StoredAuditEvent `json:"events"`
}

type retentionResponseBody struct {
	RedactedFieldCount int `json:"redacted_field_count"`
}

type collectibleBody struct {
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	TransferPolicy string `json:"transfer_policy"`
	Art            string `json:"art"`
}

type collectibleAwardBody struct {
	Slug           string `json:"slug"`
	RecipientKind  string `json:"recipient_kind"`
	RecipientID    string `json:"recipient_id"`
	OrganizationID string `json:"organization_id"`
}

type collectibleTransferBody struct {
	RecipientID string `json:"recipient_id"`
}

type collectibleCatalogEntry struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	TransferPolicy string `json:"transfer_policy"`
	Art            string `json:"art"`
}

type collectibleCatalogBody struct {
	Entries []collectibleCatalogEntry `json:"entries"`
}

type collectiblesResponseBody struct {
	Collectibles []StoredCollectible `json:"collectibles"`
}

type moderationReportBody struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	Reason      string `json:"reason"`
	Details     string `json:"details"`
}

type moderationMetadata struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

type moderationReportResponse struct {
	ID             string `json:"id"`
	SubjectKind    string `json:"subject_kind"`
	SubjectID      string `json:"subject_id"`
	Reason         string `json:"reason"`
	Details        string `json:"details"`
	ReporterUserID string `json:"reporter_user_id"`
	SubjectHref    string `json:"subject_href"`
	State          string `json:"state"`
	ResolutionNote string `json:"resolution_note"`
	UpdatedBy      string `json:"updated_by"`
	UpdatedAt      string `json:"updated_at"`
	CreatedAt      string `json:"created_at"`
}

type moderationReportsBody struct {
	Reports []moderationReportResponse `json:"reports"`
}

type agentCredentialRequestBody struct {
	Label     string   `json:"label"`
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expires_at"`
}

type agentCredentialResponseBody struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Scopes    []string `json:"scopes"`
	State     string   `json:"state"`
	ExpiresAt string   `json:"expires_at"`
	TaskID    string   `json:"task_id"`
}

type agentCredentialCreatedResponseBody struct {
	Credential agentCredentialResponseBody `json:"credential"`
	Secret     string                      `json:"secret"`
}

type agentCredentialsResponseBody struct {
	Credentials []agentCredentialResponseBody `json:"credentials"`
}

func agentCredentialToResponse(credential StoredAgentCredential) agentCredentialResponseBody {
	return agentCredentialResponseBody{
		ID:        credential.ID,
		Label:     credential.Label,
		Scopes:    credential.Scopes,
		State:     credential.State,
		ExpiresAt: credential.ExpiresAt,
		TaskID:    credential.TaskID,
	}
}

func agentCredentialsToResponse(credentials []StoredAgentCredential) []agentCredentialResponseBody {
	responses := make([]agentCredentialResponseBody, 0, len(credentials))
	for index := range credentials {
		responses = append(responses, agentCredentialToResponse(credentials[index]))
	}
	return responses
}

func defaultCollectibleCatalog() []collectibleCatalogEntry {
	entries := []collectibleCatalogEntry{
		{Slug: "harvest-star", Name: "Harvest Star", Kind: "badge", TransferPolicy: "transferable_between_users", Art: "harvest-star"},
		{Slug: "seedling", Name: "Seedling", Kind: "badge", TransferPolicy: "non_transferable_except_payout", Art: "seedling"},
	}
	for len(entries) < 25 {
		index := len(entries) + 1
		entries = append(entries, collectibleCatalogEntry{
			Slug:           "catalog-" + strconv.Itoa(index),
			Name:           "Catalog " + strconv.Itoa(index),
			Kind:           "badge",
			TransferPolicy: "transferable_between_users",
			Art:            "catalog-" + strconv.Itoa(index),
		})
	}
	return entries
}

func moderationSubjectHref(kind string, id string) string {
	switch strings.TrimSpace(kind) {
	case "task":
		return "#/tasks/" + strings.TrimSpace(id)
	default:
		return ""
	}
}

func errString(message string) error {
	return errors.New(message)
}
