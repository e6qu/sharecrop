package wasmdemo

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type fixedRuntimeIDs struct {
	userID           string
	auditEventIDs    []string
	accountTokens    []string
	collectibleID    string
	notificationID   string
	credentialID     string
	credentialSecret string
}

func (ids *fixedRuntimeIDs) NextUserID() string {
	return ids.userID
}

func (ids *fixedRuntimeIDs) NextAuditEventID() string {
	if len(ids.auditEventIDs) == 0 {
		return "audit-fallback"
	}
	value := ids.auditEventIDs[0]
	ids.auditEventIDs = ids.auditEventIDs[1:]
	return value
}

func (ids *fixedRuntimeIDs) NextAccountToken() string {
	if len(ids.accountTokens) == 0 {
		return "account-token-fallback"
	}
	value := ids.accountTokens[0]
	ids.accountTokens = ids.accountTokens[1:]
	return value
}

func (ids *fixedRuntimeIDs) NextCollectibleID() string {
	return ids.collectibleID
}

func (ids *fixedRuntimeIDs) NextNotificationID() string {
	return ids.notificationID
}

func (ids *fixedRuntimeIDs) NextAgentCredentialID() string {
	return ids.credentialID
}

func (ids *fixedRuntimeIDs) NextAgentCredentialSecret() string {
	return ids.credentialSecret
}

func handledRuntimeResponse(t *testing.T, result HandleResult, expectedStatus int) Response {
	t.Helper()
	handled, matched := result.(RequestHandled)
	if !matched {
		t.Fatalf("result = %T, want RequestHandled", result)
	}
	if handled.Value.Status != expectedStatus {
		t.Fatalf("status = %d, want %d; body=%s", handled.Value.Status, expectedStatus, handled.Value.Body)
	}
	return handled.Value
}

func rejectedRuntimeResponse(t *testing.T, result HandleResult, expectedReason string) {
	t.Helper()
	rejected, matched := result.(RequestHandleRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestHandleRejected", result)
	}
	if rejected.Reason != expectedReason {
		t.Fatalf("reason = %q, want %q", rejected.Reason, expectedReason)
	}
}

func decodeAuthResponseBody(t *testing.T, body string) authResponseBody {
	t.Helper()
	var value authResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeAccountTokenResponseBody(t *testing.T, body string) accountTokenResponseBody {
	t.Helper()
	var value accountTokenResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeStatusResponseBody(t *testing.T, body string) statusResponseBody {
	t.Helper()
	var value statusResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeUsersResponseBody(t *testing.T, body string) usersResponseBody {
	t.Helper()
	var value usersResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeUserProfileResponseBody(t *testing.T, body string) userProfileResponseBody {
	t.Helper()
	var value userProfileResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeModerationReportResponse(t *testing.T, body string) moderationReportResponse {
	t.Helper()
	var value moderationReportResponse
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeModerationReportsBody(t *testing.T, body string) moderationReportsBody {
	t.Helper()
	var value moderationReportsBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeOperationsBody(t *testing.T, body string) operationsBody {
	t.Helper()
	var value operationsBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeRetentionResponseBody(t *testing.T, body string) retentionResponseBody {
	t.Helper()
	var value retentionResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeStoredPlatformAdmin(t *testing.T, body string) StoredPlatformAdmin {
	t.Helper()
	var value StoredPlatformAdmin
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodePlatformAdminsResponseBody(t *testing.T, body string) platformAdminsResponseBody {
	t.Helper()
	var value platformAdminsResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeAuditEventsResponseBody(t *testing.T, body string) auditEventsResponseBody {
	t.Helper()
	var value auditEventsResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeCollectibleCatalogBody(t *testing.T, body string) collectibleCatalogBody {
	t.Helper()
	var value collectibleCatalogBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeStoredCollectible(t *testing.T, body string) StoredCollectible {
	t.Helper()
	var value StoredCollectible
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeCollectiblesResponseBody(t *testing.T, body string) collectiblesResponseBody {
	t.Helper()
	var value collectiblesResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeAgentCredentialCreatedResponseBody(t *testing.T, body string) agentCredentialCreatedResponseBody {
	t.Helper()
	var value agentCredentialCreatedResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeAgentCredentialsResponseBody(t *testing.T, body string) agentCredentialsResponseBody {
	t.Helper()
	var value agentCredentialsResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func decodeAgentCredentialResponseBody(t *testing.T, body string) agentCredentialResponseBody {
	t.Helper()
	var value agentCredentialResponseBody
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("decode response: %v\n%s", err, body)
	}
	return value
}

func TestRuntimeStorageRoundTripsTypedRecords(t *testing.T) {
	storage := newTestBrowserStorage()
	if err := SaveUser(storage, StoredUser{ID: " user-a ", Email: "USER-A@EXAMPLE.COM", Status: ""}); err != nil {
		t.Fatalf("save user: %v", err)
	}
	user, err := LoadUser(storage, "user-a")
	if err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Email != "user-a@example.com" || user.Status != "active" {
		t.Fatalf("user = %#v", user)
	}
	userID, err := LoadUserIDByEmail(storage, "USER-A@example.com")
	if err != nil {
		t.Fatalf("load user id by email: %v", err)
	}
	if userID != "user-a" {
		t.Fatalf("user id = %q", userID)
	}
	users, err := ListUsers(storage, "USER-A", DefaultStoredListPage())
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 1 || users[0].ID != "user-a" {
		t.Fatalf("users = %#v", users)
	}

	event := StoredAuditEvent{ID: "audit-1", ActorID: "user-a", Action: "thing_done", SubjectKind: "task", SubjectID: "task-1", MetadataJSON: "{}", CreatedAt: "2026-07-01T00:00:00Z"}
	if err := SaveAuditEvent(storage, event); err != nil {
		t.Fatalf("save audit event: %v", err)
	}
	events, err := ListAuditEvents(storage, "thing_done", "task", "task-1", DefaultStoredListPage())
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) != 1 || events[0].ID != "audit-1" {
		t.Fatalf("events = %#v", events)
	}

	admin := StoredPlatformAdmin{UserID: "user-a", Source: "bootstrap", State: "active", CreatedAt: "2026-07-01T00:00:00Z"}
	if err := SavePlatformAdmin(storage, admin); err != nil {
		t.Fatalf("save platform admin: %v", err)
	}
	loadedAdmin, err := LoadPlatformAdmin(storage, "user-a")
	if err != nil {
		t.Fatalf("load platform admin: %v", err)
	}
	if loadedAdmin.Source != "bootstrap" {
		t.Fatalf("platform admin = %#v", loadedAdmin)
	}
	admins, err := ListPlatformAdmins(storage, DefaultStoredListPage())
	if err != nil {
		t.Fatalf("list platform admins: %v", err)
	}
	if len(admins) != 1 || admins[0].UserID != "user-a" {
		t.Fatalf("admins = %#v", admins)
	}

	token := StoredAccountToken{Token: "token-1", Kind: "email_verification", UserID: "user-a", State: "active"}
	if err := SaveAccountToken(storage, token); err != nil {
		t.Fatalf("save account token: %v", err)
	}
	if err := ConsumeAccountToken(storage, "token-1", "email_verification"); err != nil {
		t.Fatalf("consume account token: %v", err)
	}
	if err := ConsumeAccountToken(storage, "token-1", "email_verification"); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("second consume error = %v, want invalid token", err)
	}

	collectible := StoredCollectible{ID: "collectible-1", Name: "Harvest Star", Kind: "badge", State: "minted", TransferPolicy: "transferable_between_users", OwnerID: "user-a", OwnerKind: "user", Art: "harvest-star"}
	if err := SaveCollectible(storage, collectible); err != nil {
		t.Fatalf("save collectible: %v", err)
	}
	loadedCollectible, err := LoadCollectible(storage, "collectible-1")
	if err != nil {
		t.Fatalf("load collectible: %v", err)
	}
	if loadedCollectible.Name != "Harvest Star" {
		t.Fatalf("collectible = %#v", loadedCollectible)
	}
	collectibles, err := ListCollectibles(storage, "user", "user-a", DefaultStoredListPage())
	if err != nil {
		t.Fatalf("list collectibles: %v", err)
	}
	if len(collectibles) != 1 || collectibles[0].ID != "collectible-1" {
		t.Fatalf("collectibles = %#v", collectibles)
	}

	credential := StoredAgentCredential{ID: "credential-1", OwnerID: "user-a", Label: "Local agent", Scopes: []string{"tasks_read"}, State: "active"}
	if err := SaveAgentCredential(storage, credential); err != nil {
		t.Fatalf("save agent credential: %v", err)
	}
	loadedCredential, err := LoadAgentCredential(storage, "credential-1")
	if err != nil {
		t.Fatalf("load agent credential: %v", err)
	}
	if loadedCredential.Label != "Local agent" {
		t.Fatalf("agent credential = %#v", loadedCredential)
	}
	credentials, err := ListAgentCredentials(storage, "user-a", DefaultStoredListPage())
	if err != nil {
		t.Fatalf("list agent credentials: %v", err)
	}
	if len(credentials) != 1 || credentials[0].ID != "credential-1" {
		t.Fatalf("credentials = %#v", credentials)
	}
}

func TestAuthAndAccountHandlersUseExplicitRuntimeStorage(t *testing.T) {
	storage := newTestBrowserStorage()
	clock := fixedHandlerClock{value: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)}
	actor := fixedHandlerActor{userID: "user-a"}
	ids := &fixedRuntimeIDs{
		userID:        "user-b",
		accountTokens: []string{"verify-token", "reset-token"},
	}
	if err := SaveUser(storage, StoredUser{ID: "user-a", Email: "a@example.com", Status: "active"}); err != nil {
		t.Fatalf("save user: %v", err)
	}

	auth := NewAuthHandler(storage, clock, actor, ids)
	login := handledRuntimeResponse(t, auth.Handle(Request{Method: MethodPost, Path: "/api/auth/login", Body: `{"email":"a@example.com"}`}), 200)
	loginBody := decodeAuthResponseBody(t, login.Body)
	if loginBody.AccessToken != "wasm-access-user-a" {
		t.Fatalf("login body = %#v", loginBody)
	}
	register := handledRuntimeResponse(t, auth.Handle(Request{Method: MethodPost, Path: "/api/auth/register", Body: `{"email":"b@example.com"}`}), 201)
	registerBody := decodeAuthResponseBody(t, register.Body)
	if registerBody.SubjectID != "user-b" {
		t.Fatalf("register body = %#v", registerBody)
	}
	rejectedRuntimeResponse(t, auth.Handle(Request{Method: MethodPost, Path: "/api/auth/guest", Body: `{}`}), "anonymous worker identity is not supported")

	account := NewAccountHandler(storage, clock, actor, ids)
	verifyToken := handledRuntimeResponse(t, account.Handle(Request{Method: MethodPost, Path: "/api/account/email-verification", Body: `{}`}), 201)
	verifyBody := decodeAccountTokenResponseBody(t, verifyToken.Body)
	if verifyBody.Token != "verify-token" {
		t.Fatalf("verify body = %#v", verifyBody)
	}
	handledRuntimeResponse(t, auth.Handle(Request{Method: MethodPost, Path: "/api/auth/email-verification/confirm", Body: `{"token":"verify-token"}`}), 200)

	resetToken := handledRuntimeResponse(t, auth.Handle(Request{Method: MethodPost, Path: "/api/auth/password-reset/request", Body: `{"email":"a@example.com"}`}), 201)
	resetBody := decodeAccountTokenResponseBody(t, resetToken.Body)
	if resetBody.Token != "reset-token" {
		t.Fatalf("reset body = %#v", resetBody)
	}
	handledRuntimeResponse(t, auth.Handle(Request{Method: MethodPost, Path: "/api/auth/password-reset/confirm", Body: `{"token":"reset-token"}`}), 200)

	password := handledRuntimeResponse(t, account.Handle(Request{Method: MethodPatch, Path: "/api/account/password", Body: `{}`}), 200)
	if decodeStatusResponseBody(t, password.Body).Status != "password_changed" {
		t.Fatalf("password body = %s", password.Body)
	}
	profile := handledRuntimeResponse(t, account.Handle(Request{Method: MethodPatch, Path: "/api/account/profile", Body: `{}`}), 200)
	if decodeStatusResponseBody(t, profile.Body).Status != "profile_updated" {
		t.Fatalf("profile body = %s", profile.Body)
	}
	deactivate := handledRuntimeResponse(t, account.Handle(Request{Method: MethodDelete, Path: "/api/account", Body: `{}`}), 200)
	if decodeStatusResponseBody(t, deactivate.Body).Status != "deactivated" {
		t.Fatalf("deactivate body = %s", deactivate.Body)
	}
}

// TestUsersHandlerListsAndLoadsUsers also locks in a real bug found by
// hand-testing the demo: GET /api/users/{user_id} returned the raw stored
// user record ({id, email, status}), but the real backend's getUserProfile
// (and the browser's profile page decoder) expects {id, tasks: [...tasks
// this user created...]}, so the profile page's "Public tasks" section
// failed with a JSON decode error on every profile view.
func TestUsersHandlerListsAndLoadsUsers(t *testing.T) {
	storage := newTestBrowserStorage()
	if err := SaveUser(storage, StoredUser{ID: "user-a", Email: "a@example.com", Status: "active"}); err != nil {
		t.Fatalf("save user a: %v", err)
	}
	if err := SaveUser(storage, StoredUser{ID: "user-b", Email: "b@example.com", Status: "active"}); err != nil {
		t.Fatalf("save user b: %v", err)
	}
	taskHandler := NewTaskHandler(storage, fixedHandlerActor{userID: "user-a"}, fixedTaskIDs{value: "task-1"})
	createResult := taskHandler.Handle(Request{
		Method: MethodPost,
		Path:   "/api/tasks",
		Body:   `{"owner":{"kind":"user","user_id":"user-a"},"title":"Created by user-a","description":"desc","reward":{"kind":"none","credit_amount":0,"collectible_ids":[]},"participation":{"policy":"open","assignee_scope":"user","reservation_expiry_hours":48},"visibility":{"kind":"public"},"placement":{"kind":"standalone"},"response_schema_json":"{\"kind\":\"freeform\"}","payload":{"kind":"none","json":""},"task_type":"general","attachments":[]}`,
	})
	if _, matched := createResult.(RequestHandled); !matched {
		t.Fatalf("create task = %#v, want RequestHandled", createResult)
	}

	handler := NewUsersHandler(storage)
	list := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodGet, Path: "/api/users?query=b&limit=10&offset=0", Body: ""}), 200)
	listBody := decodeUsersResponseBody(t, list.Body)
	if len(listBody.Users) != 1 || listBody.Users[0].ID != "user-b" {
		t.Fatalf("list body = %#v", listBody)
	}

	detail := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodGet, Path: "/api/users/user-a", Body: ""}), 200)
	profile := decodeUserProfileResponseBody(t, detail.Body)
	if profile.ID != "user-a" || len(profile.Tasks) != 1 || profile.Tasks[0].ID != "task-1" {
		t.Fatalf("profile detail = %#v", profile)
	}

	emptyProfile := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodGet, Path: "/api/users/user-b", Body: ""}), 200)
	emptyProfileBody := decodeUserProfileResponseBody(t, emptyProfile.Body)
	if emptyProfileBody.ID != "user-b" || len(emptyProfileBody.Tasks) != 0 {
		t.Fatalf("empty profile detail = %#v", emptyProfileBody)
	}
}

func TestModerationAndAdminHandlersPersistRuntimeRecords(t *testing.T) {
	storage := newTestBrowserStorage()
	clock := fixedHandlerClock{value: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)}
	actor := fixedHandlerActor{userID: "user-admin"}
	ids := &fixedRuntimeIDs{
		auditEventIDs: []string{"audit-report", "audit-triage", "audit-retention", "audit-grant", "audit-revoke"},
	}

	reportHandler := NewModerationReportHandler(storage, clock, actor, ids)
	report := handledRuntimeResponse(t, reportHandler.Handle(Request{Method: MethodPost, Path: "/api/moderation/reports", Body: `{"subject_kind":"task","subject_id":"task-1","reason":"pii","details":"contains data"}`}), 201)
	reportBody := decodeModerationReportResponse(t, report.Body)
	if reportBody.ID != "audit-report" || reportBody.SubjectHref != "#/tasks/task-1" {
		t.Fatalf("report body = %#v", reportBody)
	}

	admin := NewAdminHandler(storage, clock, actor, ids)
	listReports := handledRuntimeResponse(t, admin.Handle(Request{Method: MethodGet, Path: "/api/admin/moderation/reports?state=open&limit=10&offset=0", Body: ""}, RouteAdminModerationReports), 200)
	reportsBody := decodeModerationReportsBody(t, listReports.Body)
	if len(reportsBody.Reports) != 1 || reportsBody.Reports[0].State != "open" {
		t.Fatalf("reports body = %#v", reportsBody)
	}
	triaged := handledRuntimeResponse(t, admin.Handle(Request{Method: MethodPost, Path: "/api/admin/moderation/reports/audit-report/triage", Body: `{"state":"resolved","resolution_note":"handled"}`}, RouteAdminModerationReports), 200)
	triagedBody := decodeModerationReportResponse(t, triaged.Body)
	if triagedBody.State != "resolved" || triagedBody.ResolutionNote != "handled" {
		t.Fatalf("triaged body = %#v", triagedBody)
	}
	operations := handledRuntimeResponse(t, admin.Handle(Request{Method: MethodGet, Path: "/api/admin/operations", Body: ""}, RouteAdminOperations), 200)
	if decodeOperationsBody(t, operations.Body).Status != "ok" {
		t.Fatalf("operations body = %s", operations.Body)
	}
	retention := handledRuntimeResponse(t, admin.Handle(Request{Method: MethodPost, Path: "/api/admin/privacy-retention/run", Body: `{}`}, RouteAdminPrivacyRetention), 200)
	if decodeRetentionResponseBody(t, retention.Body).RedactedFieldCount != 0 {
		t.Fatalf("retention body = %s", retention.Body)
	}
	grant := handledRuntimeResponse(t, admin.Handle(Request{Method: MethodPost, Path: "/api/admin/platform-admins", Body: `{"user_id":"user-reviewer"}`}, RoutePlatformAdmins), 201)
	if decodeStoredPlatformAdmin(t, grant.Body).State != "active" {
		t.Fatalf("grant body = %s", grant.Body)
	}
	admins := handledRuntimeResponse(t, admin.Handle(Request{Method: MethodGet, Path: "/api/admin/platform-admins?limit=10&offset=0", Body: ""}, RoutePlatformAdmins), 200)
	if len(decodePlatformAdminsResponseBody(t, admins.Body).Admins) != 1 {
		t.Fatalf("admins body = %s", admins.Body)
	}
	revoke := handledRuntimeResponse(t, admin.Handle(Request{Method: MethodPost, Path: "/api/admin/platform-admins/user-reviewer/revoke", Body: `{}`}, RoutePlatformAdmins), 200)
	if decodeStoredPlatformAdmin(t, revoke.Body).State != "revoked" {
		t.Fatalf("revoke body = %s", revoke.Body)
	}
	audit := handledRuntimeResponse(t, admin.Handle(Request{Method: MethodGet, Path: "/api/admin/audit-events?action=platform_admin_revoked&limit=10&offset=0", Body: ""}, RouteAuditEvents), 200)
	auditBody := decodeAuditEventsResponseBody(t, audit.Body)
	if len(auditBody.Events) != 1 || auditBody.Events[0].SubjectID != "user-reviewer" {
		t.Fatalf("audit body = %#v", auditBody)
	}

	// GET /api/organizations/{organization_id}/audit-events reproduces a real
	// bug found by hand-testing the demo: the route was unclassified
	// (RequestUnsupported, 404), so the organization detail page's audit
	// section always failed to load. The route now maps to the same
	// RouteAuditEvents/AdminHandler as the platform-admin-wide listing, scoped
	// to the organization's own subject id.
	adapted, adaptedMatched := Adapt(Request{Method: MethodGet, Path: "/api/organizations/org-field/audit-events?limit=10&offset=0"}).(RequestAdapted)
	if !adaptedMatched || adapted.Route != RouteAuditEvents {
		t.Fatalf("adapt organization audit-events route = %#v, want RouteAuditEvents", adapted)
	}
	if err := SaveAuditEvent(storage, StoredAuditEvent{ID: "audit-org-funded", ActorID: "user-admin", Action: "organization_funded", SubjectKind: "organization", SubjectID: "org-field", MetadataJSON: "{}", CreatedAt: "2026-07-01T10:00:00Z"}); err != nil {
		t.Fatalf("save organization audit event: %v", err)
	}
	orgAudit := handledRuntimeResponse(t, admin.Handle(Request{Method: MethodGet, Path: "/api/organizations/org-field/audit-events?limit=10&offset=0", Body: ""}, RouteAuditEvents), 200)
	orgAuditBody := decodeAuditEventsResponseBody(t, orgAudit.Body)
	if len(orgAuditBody.Events) != 1 || orgAuditBody.Events[0].SubjectID != "org-field" {
		t.Fatalf("organization audit body = %#v, want only the org-field event", orgAuditBody)
	}
}

func TestCollectibleHandlerMintsAwardsTransfersAndLists(t *testing.T) {
	storage := newTestBrowserStorage()
	actor := fixedHandlerActor{userID: "user-owner"}
	ids := &fixedRuntimeIDs{collectibleID: "collectible-1"}
	handler := NewCollectibleHandler(storage, actor, ids)

	catalog := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodGet, Path: "/api/collectibles/catalog", Body: ""}), 200)
	if len(decodeCollectibleCatalogBody(t, catalog.Body).Entries) != 25 {
		t.Fatalf("catalog body = %s", catalog.Body)
	}
	minted := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodPost, Path: "/api/collectibles", Body: `{"name":"Minted","kind":"badge","transfer_policy":"transferable_between_users","art":"minted"}`}), 201)
	if decodeStoredCollectible(t, minted.Body).OwnerID != "user-owner" {
		t.Fatalf("minted body = %s", minted.Body)
	}
	list := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodGet, Path: "/api/collectibles?limit=10&offset=0", Body: ""}), 200)
	if len(decodeCollectiblesResponseBody(t, list.Body).Collectibles) != 1 {
		t.Fatalf("list body = %s", list.Body)
	}
	transferred := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodPost, Path: "/api/collectibles/collectible-1/transfer", Body: `{"recipient_id":"user-recipient"}`}), 200)
	if decodeStoredCollectible(t, transferred.Body).OwnerID != "user-recipient" {
		t.Fatalf("transfer body = %s", transferred.Body)
	}

	ids.collectibleID = "collectible-2"
	award := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodPost, Path: "/api/collectibles/award", Body: `{"slug":"seedling","recipient_kind":"organization","recipient_id":"org-1","organization_id":"org-1"}`}), 201)
	if decodeStoredCollectible(t, award.Body).OwnerKind != "organization" {
		t.Fatalf("award body = %s", award.Body)
	}
	scoped := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodGet, Path: "/api/organizations/org-1/collectibles", Body: ""}), 200)
	if len(decodeCollectiblesResponseBody(t, scoped.Body).Collectibles) != 1 {
		t.Fatalf("scoped body = %s", scoped.Body)
	}
}

func TestAgentCredentialHandlerCreatesListsAndRevokes(t *testing.T) {
	storage := newTestBrowserStorage()
	actor := fixedHandlerActor{userID: "user-owner"}
	ids := &fixedRuntimeIDs{credentialID: "credential-1", credentialSecret: "scrop_agent_secret-1"}
	handler := NewAgentCredentialHandler(storage, actor, ids)

	created := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodPost, Path: "/api/agent-credentials", Body: `{"label":"Task worker token","scopes":["tasks_read","submissions_write"]}`}), 201)
	createdBody := decodeAgentCredentialCreatedResponseBody(t, created.Body)
	if createdBody.Secret != "scrop_agent_secret-1" || createdBody.Credential.ID != "credential-1" {
		t.Fatalf("created body = %#v", createdBody)
	}
	list := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodGet, Path: "/api/agent-credentials?limit=10&offset=0", Body: ""}), 200)
	listBody := decodeAgentCredentialsResponseBody(t, list.Body)
	if len(listBody.Credentials) != 1 || listBody.Credentials[0].State != "active" {
		t.Fatalf("list body = %#v", listBody)
	}
	revoked := handledRuntimeResponse(t, handler.Handle(Request{Method: MethodPost, Path: "/api/agent-credentials/credential-1/revoke", Body: `{}`}), 200)
	if decodeAgentCredentialResponseBody(t, revoked.Body).State != "revoked" {
		t.Fatalf("revoked body = %s", revoked.Body)
	}
}
