package wasmdemo

import "testing"

func TestNewRequestRejectsUnsupportedMethod(t *testing.T) {
	result := NewRequest("DELETE", "/api/privacy-requests", "")
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
		{name: "list admin privacy requests", method: MethodGet, path: "/api/admin/privacy-requests", route: RouteAdminPrivacyRequests},
		{name: "create moderation report", method: MethodPost, path: "/api/moderation/reports", route: RouteModerationReports},
		{name: "list admin moderation reports", method: MethodGet, path: "/api/admin/moderation/reports", route: RouteAdminModerationReports},
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
	result := Adapt(Request{Method: MethodGet, Path: "/api/tasks", Body: ""})
	unsupported, matched := result.(RequestUnsupported)
	if !matched {
		t.Fatalf("result = %T, want RequestUnsupported", result)
	}
	if unsupported.Reason != "request route is not implemented by the WASM demo adapter" {
		t.Fatalf("reason = %q", unsupported.Reason)
	}
}
