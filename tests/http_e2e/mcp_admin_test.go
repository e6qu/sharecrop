//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestMCPAdminScopeAloneIsNotEnough is the regression test for the double
// -check mechanism this phase added: an agent credential minted with the
// platform_admin scope must still be rejected if the underlying user is not
// actually a platform admin. The scope check alone (already enforced by
// handleToolsCall before dispatch) is not sufficient.
func TestMCPAdminScopeAloneIsNotEnough(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	nonAdmin := registerUser(t, server, "mcp-admin-scope-nonadmin")
	nonAdminAgent := createAgentCredential(t, server, nonAdmin.AccessToken, []string{"platform_admin"})
	session := initializeMCPSession(t, server, nonAdminAgent)

	response := decodeRPC(t, mcpCall(t, server, nonAdminAgent, session, `1`, "sharecrop.list_platform_admins", `{}`))
	if response.Error != nil {
		t.Fatalf("expected a tool-level failure, not a protocol error: %+v", response.Error)
	}
	var result struct {
		IsError bool `json:"isError"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected list_platform_admins to be denied for a non-admin user despite the platform_admin scope")
	}
	if len(result.Content) != 1 || !strings.Contains(result.Content[0].Text, "platform admin") {
		t.Fatalf("expected a platform-admin-denied message, got: %+v", result.Content)
	}
}

func TestMCPPlatformAdminToolsForARealAdmin(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "mcp-admin-real")
	target := registerUser(t, bootstrap, "mcp-admin-target")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	adminAgent := createAgentCredential(t, server, admin.AccessToken, []string{"platform_admin"})
	session := initializeMCPSession(t, server, adminAgent)

	granted := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `1`, "sharecrop.grant_platform_admin", `{"user_id":"`+target.SubjectID+`"}`)))
	if !strings.Contains(granted, target.SubjectID) {
		t.Fatalf("grant_platform_admin missing the granted user: %s", granted)
	}

	listed := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `2`, "sharecrop.list_platform_admins", `{}`)))
	if !strings.Contains(listed, target.SubjectID) {
		t.Fatalf("list_platform_admins missing the granted admin: %s", listed)
	}

	revoked := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `3`, "sharecrop.revoke_platform_admin", `{"user_id":"`+target.SubjectID+`"}`)))
	if !strings.Contains(revoked, target.SubjectID) {
		t.Fatalf("revoke_platform_admin missing the revoked user: %s", revoked)
	}
}

func TestMCPModerationLifecycle(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "mcp-moderation-admin")
	reporter := registerUser(t, bootstrap, "mcp-moderation-reporter")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	reporterAgent := createAgentCredential(t, server, reporter.AccessToken, []string{"tasks_read"})
	reporterSession := initializeMCPSession(t, server, reporterAgent)

	created := toolText(t, decodeRPC(t, mcpCall(t, server, reporterAgent, reporterSession, `1`, "sharecrop.create_moderation_report", `{"subject_kind":"user","subject_id":"`+admin.SubjectID+`","reason":"spam","details":"testing"}`)))
	var report struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(created), &report); err != nil {
		t.Fatalf("decode create_moderation_report: %v (%s)", err, created)
	}

	adminAgent := createAgentCredential(t, server, admin.AccessToken, []string{"moderation_read", "moderation_manage"})
	adminSession := initializeMCPSession(t, server, adminAgent)

	listed := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, adminSession, `2`, "sharecrop.list_admin_moderation_reports", `{}`)))
	if !strings.Contains(listed, report.ID) {
		t.Fatalf("list_admin_moderation_reports missing the created report: %s", listed)
	}

	triaged := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, adminSession, `3`, "sharecrop.triage_moderation_report", `{"report_id":"`+report.ID+`","state":"resolved","resolution_note":"handled"}`)))
	if !strings.Contains(triaged, `"state":"resolved"`) {
		t.Fatalf("triage_moderation_report did not report resolved state: %s", triaged)
	}
}

func TestMCPPrivacyLifecycle(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "mcp-privacy-admin")
	requester := registerUser(t, bootstrap, "mcp-privacy-requester")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	requesterAgent := createAgentCredential(t, server, requester.AccessToken, []string{"privacy_read"})
	requesterSession := initializeMCPSession(t, server, requesterAgent)

	created := toolText(t, decodeRPC(t, mcpCall(t, server, requesterAgent, requesterSession, `1`, "sharecrop.create_privacy_request", `{"kind":"data_export"}`)))
	var request struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(created), &request); err != nil {
		t.Fatalf("decode create_privacy_request: %v (%s)", err, created)
	}

	ownList := toolText(t, decodeRPC(t, mcpCall(t, server, requesterAgent, requesterSession, `2`, "sharecrop.list_privacy_requests", `{}`)))
	if !strings.Contains(ownList, request.ID) {
		t.Fatalf("list_privacy_requests missing the created request: %s", ownList)
	}

	adminAgent := createAgentCredential(t, server, admin.AccessToken, []string{"privacy_read", "privacy_manage"})
	adminSession := initializeMCPSession(t, server, adminAgent)

	adminList := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, adminSession, `3`, "sharecrop.list_admin_privacy_requests", `{}`)))
	if !strings.Contains(adminList, request.ID) {
		t.Fatalf("list_admin_privacy_requests missing the created request: %s", adminList)
	}

	resolved := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, adminSession, `4`, "sharecrop.resolve_admin_privacy_request", `{"privacy_request_id":"`+request.ID+`","resolution_note":"exported"}`)))
	if !strings.Contains(resolved, `"state":"resolved"`) {
		t.Fatalf("resolve_admin_privacy_request did not report resolved state: %s", resolved)
	}

	retention := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, adminSession, `5`, "sharecrop.run_privacy_retention", `{}`)))
	if !strings.Contains(retention, "redacted_field_count") {
		t.Fatalf("run_privacy_retention missing redacted_field_count: %s", retention)
	}
}

func TestMCPAuditEventTools(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "mcp-audit-admin")
	orgOwner := registerUser(t, bootstrap, "mcp-audit-org-owner")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	orgResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"MCP Audit Org"}`), orgOwner.AccessToken)
	defer orgResponse.Body.Close()
	assertStatus(t, orgResponse, http.StatusCreated)
	organization := decodeOrganizationHTTPResponse(t, orgResponse)

	ownerAgent := createAgentCredential(t, server, orgOwner.AccessToken, []string{"org_read"})
	ownerSession := initializeMCPSession(t, server, ownerAgent)

	orgEvents := decodeRPC(t, mcpCall(t, server, ownerAgent, ownerSession, `1`, "sharecrop.list_organization_audit_events", `{"organization_id":"`+organization.ID+`"}`))
	orgEventsText := toolText(t, orgEvents)
	if !strings.Contains(orgEventsText, `"events"`) {
		t.Fatalf("list_organization_audit_events missing events key: %s", orgEventsText)
	}

	adminAgent := createAgentCredential(t, server, admin.AccessToken, []string{"platform_admin"})
	adminSession := initializeMCPSession(t, server, adminAgent)

	platformEvents := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, adminSession, `2`, "sharecrop.list_admin_audit_events", `{"subject_kind":"organization"}`)))
	if !strings.Contains(platformEvents, organization.ID) {
		t.Fatalf("list_admin_audit_events missing the organization creation event: %s", platformEvents)
	}
}

func TestMCPAwardCollectible(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "mcp-award-admin")
	recipient := registerUser(t, bootstrap, "mcp-award-recipient")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	adminAgent := createAgentCredential(t, server, admin.AccessToken, []string{"platform_admin", "collectibles_read"})
	session := initializeMCPSession(t, server, adminAgent)

	catalog := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `1`, "sharecrop.collectible_catalog", `{}`)))
	var catalogPayload struct {
		Entries []struct {
			Slug string `json:"slug"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(catalog), &catalogPayload); err != nil || len(catalogPayload.Entries) == 0 {
		t.Fatalf("decode collectible_catalog: %v (%s)", err, catalog)
	}

	awarded := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `2`, "sharecrop.award_collectible", `{"slug":"`+catalogPayload.Entries[0].Slug+`","recipient_kind":"user","recipient_id":"`+recipient.SubjectID+`"}`)))
	if !strings.Contains(awarded, recipient.SubjectID) {
		t.Fatalf("award_collectible did not report the recipient: %s", awarded)
	}

	// A non-admin is denied even with the collectibles_manage scope.
	recipientAgent := createAgentCredential(t, server, recipient.AccessToken, []string{"collectibles_manage"})
	recipientSession := initializeMCPSession(t, server, recipientAgent)
	denied := decodeRPC(t, mcpCall(t, server, recipientAgent, recipientSession, `1`, "sharecrop.award_collectible", `{"slug":"`+catalogPayload.Entries[0].Slug+`","recipient_kind":"user","recipient_id":"`+recipient.SubjectID+`"}`))
	if denied.Error == nil {
		t.Fatalf("expected award_collectible to require the platform_admin scope")
	}
	if denied.Error.Code != -32001 {
		t.Fatalf("error code = %d, want -32001 (scope denied)", denied.Error.Code)
	}
}
