package wasmdemo

import "testing"

func TestNewRequestRejectsUnsupportedMethod(t *testing.T) {
	result := NewRequest("PUT", "/api/privacy-requests", "")
	rejected, matched := result.(RequestRejected)
	if !matched {
		t.Fatalf("result = %T, want RequestRejected", result)
	}
	if rejected.Reason != "request method is unsupported" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestAdaptRecognizesPrivacyAndModerationRoutes(t *testing.T) {
	cases := []struct {
		name   string
		method Method
		path   string
		route  Route
	}{
		{name: "create privacy request", method: MethodPost, path: "/api/privacy-requests", route: RoutePrivacyRequests},
		{name: "list user privacy requests", method: MethodGet, path: "/api/privacy-requests", route: RoutePrivacyRequests},
		{name: "list admin privacy requests", method: MethodGet, path: "/api/admin/privacy-requests", route: RouteAdminPrivacyRequests},
		{name: "resolve admin privacy request", method: MethodPost, path: "/api/admin/privacy-requests/privacy-1/resolve", route: RouteAdminPrivacyRequests},
		{name: "run privacy retention", method: MethodPost, path: "/api/admin/privacy-retention/run", route: RouteAdminPrivacyRetention},
		{name: "refresh auth", method: MethodPost, path: "/api/auth/refresh", route: RouteAuth},
		{name: "account email verification", method: MethodPost, path: "/api/account/email-verification", route: RouteAccount},
		{name: "user selector", method: MethodGet, path: "/api/users?query=user&limit=2&offset=0", route: RouteUsers},
		{name: "create moderation report", method: MethodPost, path: "/api/moderation/reports", route: RouteModerationReports},
		{name: "list admin moderation reports", method: MethodGet, path: "/api/admin/moderation/reports", route: RouteAdminModerationReports},
		{name: "triage admin moderation report", method: MethodPost, path: "/api/admin/moderation/reports/audit-1/triage", route: RouteAdminModerationReports},
		{name: "list saved queue views", method: MethodGet, path: "/api/saved-queue-views", route: RouteSavedQueueViews},
		{name: "upsert saved queue view", method: MethodPost, path: "/api/saved-queue-views", route: RouteSavedQueueViews},
		{name: "create task", method: MethodPost, path: "/api/tasks", route: RouteTasks},
		{name: "get task", method: MethodGet, path: "/api/tasks/task-1", route: RouteTasks},
		{name: "list notifications", method: MethodGet, path: "/api/notifications?limit=1&offset=0", route: RouteNotifications},
		{name: "mark notification read", method: MethodPost, path: "/api/notifications/notification-1/read", route: RouteNotifications},
		{name: "create organization", method: MethodPost, path: "/api/organizations", route: RouteOrganizations},
		{name: "list organizations", method: MethodGet, path: "/api/organizations?query=field&limit=1&offset=0", route: RouteOrganizations},
		{name: "list organization members", method: MethodGet, path: "/api/organizations/org-1/members", route: RouteOrganizationMembers},
		{name: "provision organization member", method: MethodPost, path: "/api/organizations/org-1/members", route: RouteOrganizationMembers},
		{name: "update organization member roles", method: MethodPatch, path: "/api/organizations/org-1/members/user-1/roles", route: RouteOrganizationMembers},
		{name: "deactivate organization member", method: MethodPatch, path: "/api/organizations/org-1/members/user-1/deactivate", route: RouteOrganizationMembers},
		{name: "create organization team", method: MethodPost, path: "/api/organizations/org-1/teams", route: RouteOrganizationTeams},
		{name: "list organization teams", method: MethodGet, path: "/api/organizations/org-1/teams?query=crew", route: RouteOrganizationTeams},
		{name: "create standalone team", method: MethodPost, path: "/api/teams", route: RouteStandaloneTeams},
		{name: "list standalone teams", method: MethodGet, path: "/api/teams?query=crew", route: RouteStandaloneTeams},
		{name: "create task comment", method: MethodPost, path: "/api/tasks/task-1/comments", route: RouteTaskComments},
		{name: "list task comments", method: MethodGet, path: "/api/tasks/task-1/comments", route: RouteTaskComments},
		{name: "create submission comment", method: MethodPost, path: "/api/submissions/submission-1/comments", route: RouteSubmissionComments},
		{name: "list submission comments", method: MethodGet, path: "/api/submissions/submission-1/comments", route: RouteSubmissionComments},
		{name: "create reservation", method: MethodPost, path: "/api/tasks/task-1/reservations", route: RouteTaskReservations},
		{name: "list reservations", method: MethodGet, path: "/api/tasks/task-1/reservations?limit=1&offset=0", route: RouteTaskReservations},
		{name: "approve reservation", method: MethodPost, path: "/api/tasks/task-1/reservations/reservation-1/approve", route: RouteTaskReservations},
		{name: "create submission", method: MethodPost, path: "/api/tasks/task-1/submissions", route: RouteSubmissions},
		{name: "list task submissions", method: MethodGet, path: "/api/tasks/task-1/submissions", route: RouteSubmissions},
		{name: "accept submission", method: MethodPost, path: "/api/tasks/task-1/submissions/submission-1/accept", route: RouteSubmissions},
		{name: "list user submissions", method: MethodGet, path: "/api/users/user-1/submissions?limit=1&offset=0", route: RouteSubmissions},
		{name: "user balance", method: MethodGet, path: "/api/credits/balance", route: RouteLedger},
		{name: "user ledger", method: MethodGet, path: "/api/credits/ledger?limit=1&offset=0", route: RouteLedger},
		{name: "organization balance", method: MethodGet, path: "/api/organizations/org-1/credits/balance", route: RouteLedger},
		{name: "organization ledger", method: MethodGet, path: "/api/organizations/org-1/credits/ledger?limit=1&offset=0", route: RouteLedger},
		{name: "collectible catalog", method: MethodGet, path: "/api/collectibles/catalog", route: RouteCollectibles},
		{name: "mint collectible", method: MethodPost, path: "/api/collectibles", route: RouteCollectibles},
		{name: "admin operations", method: MethodGet, path: "/api/admin/operations", route: RouteAdminOperations},
		{name: "platform admins", method: MethodGet, path: "/api/admin/platform-admins", route: RoutePlatformAdmins},
		{name: "admin audit events", method: MethodGet, path: "/api/admin/audit-events?limit=1&offset=0", route: RouteAuditEvents},
		{name: "agent credentials", method: MethodGet, path: "/api/agent-credentials", route: RouteAgentCredentials},
		{name: "agent credential revoke", method: MethodPost, path: "/api/agent-credentials/credential-1/revoke", route: RouteAgentCredentials},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := Adapt(Request{Method: tc.method, Path: tc.path, Body: "{}"})
			adapted, matched := result.(RequestAdapted)
			if !matched {
				t.Fatalf("result = %T, want RequestAdapted", result)
			}
			if adapted.Route.String() != tc.route.String() {
				t.Fatalf("route = %q, want %q", adapted.Route.String(), tc.route.String())
			}
		})
	}
}

func TestAdaptRejectsUnimplementedRoute(t *testing.T) {
	result := Adapt(Request{Method: MethodGet, Path: "/api/not-implemented", Body: ""})
	unsupported, matched := result.(RequestUnsupported)
	if !matched {
		t.Fatalf("result = %T, want RequestUnsupported", result)
	}
	if unsupported.Reason != "request route is not implemented by the WASM demo adapter" {
		t.Fatalf("reason = %q", unsupported.Reason)
	}
}
