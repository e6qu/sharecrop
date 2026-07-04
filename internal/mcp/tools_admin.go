package mcp

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
)

const (
	toolListPlatformAdmins  = "sharecrop.list_platform_admins"
	toolGrantPlatformAdmin  = "sharecrop.grant_platform_admin"
	toolRevokePlatformAdmin = "sharecrop.revoke_platform_admin"

	toolCreateModerationReport     = "sharecrop.create_moderation_report"
	toolListAdminModerationReports = "sharecrop.list_admin_moderation_reports"
	toolTriageModerationReport     = "sharecrop.triage_moderation_report"

	toolCreatePrivacyRequest       = "sharecrop.create_privacy_request"
	toolListPrivacyRequests        = "sharecrop.list_privacy_requests"
	toolListAdminPrivacyRequests   = "sharecrop.list_admin_privacy_requests"
	toolResolveAdminPrivacyRequest = "sharecrop.resolve_admin_privacy_request"
	toolRunPrivacyRetention        = "sharecrop.run_privacy_retention"

	toolListOrganizationAuditEvents = "sharecrop.list_organization_audit_events"
	toolListAdminAuditEvents        = "sharecrop.list_admin_audit_events"
)

func adminToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        toolListPlatformAdmins,
			Description: "List platform administrators. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolGrantPlatformAdmin,
			Description: "Grant platform admin access to a user. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"string"}},"required":["user_id"]}`),
		},
		{
			Name:        toolRevokePlatformAdmin,
			Description: "Revoke a granted user's platform admin access (bootstrap admins cannot be revoked). Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"string"}},"required":["user_id"]}`),
		},
		{
			Name:        toolCreateModerationReport,
			Description: "Report a task, submission, comment, user, organization, team, or collectible for moderation review. reason is one of spam, abuse, pii, policy, other.",
			Scope:       agent.ScopeTasksRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"subject_kind":{"type":"string"},"subject_id":{"type":"string"},"reason":{"type":"string"},"details":{"type":"string"}},"required":["subject_kind","subject_id","reason"]}`),
		},
		{
			Name:        toolListAdminModerationReports,
			Description: "List moderation reports, optionally filtered by triage state (open, resolved, dismissed). Requires platform admin access.",
			Scope:       agent.ScopeModerationRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"state":{"type":"string"}}}`),
		},
		{
			Name:        toolTriageModerationReport,
			Description: "Triage a moderation report to open, resolved, or dismissed with a resolution note. Requires platform admin access.",
			Scope:       agent.ScopeModerationManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"report_id":{"type":"string"},"state":{"type":"string"},"resolution_note":{"type":"string"}},"required":["report_id","state"]}`),
		},
		{
			Name:        toolCreatePrivacyRequest,
			Description: "File a privacy request for the agent's user's own data: data_export or sensitive_field_deletion.",
			Scope:       agent.ScopePrivacyRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"kind":{"type":"string"}},"required":["kind"]}`),
		},
		{
			Name:        toolListPrivacyRequests,
			Description: "List the agent's user's own privacy requests.",
			Scope:       agent.ScopePrivacyRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolListAdminPrivacyRequests,
			Description: "List every privacy request on the platform. Requires platform admin access.",
			Scope:       agent.ScopePrivacyRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolResolveAdminPrivacyRequest,
			Description: "Resolve a privacy request with a resolution note. Requires platform admin access.",
			Scope:       agent.ScopePrivacyManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"privacy_request_id":{"type":"string"},"resolution_note":{"type":"string"}},"required":["privacy_request_id"]}`),
		},
		{
			Name:        toolRunPrivacyRetention,
			Description: "Run the privacy retention sweep, redacting sensitive fields past their retention window. Requires platform admin access.",
			Scope:       agent.ScopePrivacyManage,
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        toolListOrganizationAuditEvents,
			Description: "List an organization's audit events. Requires PermissionManageMembers on the organization.",
			Scope:       agent.ScopeOrgRead,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"organization_id":{"type":"string"}},"required":["organization_id"]}`),
		},
		{
			Name:        toolListAdminAuditEvents,
			Description: "List platform-wide audit events, optionally filtered by action, subject_kind, or subject_id. Requires platform admin access.",
			Scope:       agent.ScopePlatformAdmin,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"action":{"type":"string"},"subject_kind":{"type":"string"},"subject_id":{"type":"string"}}}`),
		},
	}
}
