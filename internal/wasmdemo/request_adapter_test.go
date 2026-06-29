package wasmdemo

import "testing"

func TestNewRequestRejectsUnsupportedMethod(t *testing.T) {
	result := NewRequest("PATCH", "/api/privacy-requests", "")
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
