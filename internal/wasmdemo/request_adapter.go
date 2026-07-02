package wasmdemo

import "strings"

type Request struct {
	Method Method
	Path   string
	Body   string
}

type Method struct {
	value string
}

var (
	MethodGet    = Method{value: "GET"}
	MethodPost   = Method{value: "POST"}
	MethodPatch  = Method{value: "PATCH"}
	MethodDelete = Method{value: "DELETE"}
)

func (method Method) String() string {
	return method.value
}

type RequestResult interface {
	requestResult()
}

type RequestAccepted struct {
	Value Request
}

type RequestRejected struct {
	Reason string
}

func (RequestAccepted) requestResult() {}
func (RequestRejected) requestResult() {}

func NewRequest(method string, path string, body string) RequestResult {
	cleanMethod := strings.TrimSpace(method)
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" || !strings.HasPrefix(cleanPath, "/") {
		return RequestRejected{Reason: "request path is invalid"}
	}
	switch cleanMethod {
	case MethodGet.String():
		return RequestAccepted{Value: Request{Method: MethodGet, Path: cleanPath, Body: body}}
	case MethodPost.String():
		return RequestAccepted{Value: Request{Method: MethodPost, Path: cleanPath, Body: body}}
	case MethodPatch.String():
		return RequestAccepted{Value: Request{Method: MethodPatch, Path: cleanPath, Body: body}}
	case MethodDelete.String():
		return RequestAccepted{Value: Request{Method: MethodDelete, Path: cleanPath, Body: body}}
	default:
		return RequestRejected{Reason: "request method is unsupported"}
	}
}

type Route struct {
	value string
}

var (
	RoutePrivacyRequests        = Route{value: "privacy_requests"}
	RouteAdminPrivacyRequests   = Route{value: "admin_privacy_requests"}
	RouteAdminPrivacyRetention  = Route{value: "admin_privacy_retention"}
	RouteModerationReports      = Route{value: "moderation_reports"}
	RouteAdminModerationReports = Route{value: "admin_moderation_reports"}
	RouteSavedQueueViews        = Route{value: "saved_queue_views"}
	RouteAuth                   = Route{value: "auth"}
	RouteAccount                = Route{value: "account"}
	RouteUsers                  = Route{value: "users"}
	RouteTasks                  = Route{value: "tasks"}
	RouteNotifications          = Route{value: "notifications"}
	RouteOrganizations          = Route{value: "organizations"}
	RouteOrganizationMembers    = Route{value: "organization_members"}
	RouteOrganizationTeams      = Route{value: "organization_teams"}
	RouteStandaloneTeams        = Route{value: "standalone_teams"}
	RouteTaskComments           = Route{value: "task_comments"}
	RouteSubmissionComments     = Route{value: "submission_comments"}
	RouteTaskReservations       = Route{value: "task_reservations"}
	RouteSubmissions            = Route{value: "submissions"}
	RouteLedger                 = Route{value: "ledger"}
	RouteCollectibles           = Route{value: "collectibles"}
	RouteAdminOperations        = Route{value: "admin_operations"}
	RoutePlatformAdmins         = Route{value: "platform_admins"}
	RouteAuditEvents            = Route{value: "audit_events"}
	RouteAgentCredentials       = Route{value: "agent_credentials"}
)

func (route Route) String() string {
	return route.value
}

type AdaptResult interface {
	adaptResult()
}

type RequestAdapted struct {
	Route Route
}

type RequestUnsupported struct {
	Reason string
}

func (RequestAdapted) adaptResult()     {}
func (RequestUnsupported) adaptResult() {}

func Adapt(request Request) AdaptResult {
	switch {
	case strings.HasPrefix(request.Path, "/api/auth/"):
		return RequestAdapted{Route: RouteAuth}
	case strings.HasPrefix(request.Path, "/api/account"):
		return RequestAdapted{Route: RouteAccount}
	case request.Method.String() == MethodPost.String() && request.Path == "/api/privacy-requests":
		return RequestAdapted{Route: RoutePrivacyRequests}
	case request.Method.String() == MethodGet.String() && request.Path == "/api/privacy-requests":
		return RequestAdapted{Route: RoutePrivacyRequests}
	case request.Method.String() == MethodGet.String() && strings.SplitN(request.Path, "?", 2)[0] == "/api/admin/privacy-requests":
		return RequestAdapted{Route: RouteAdminPrivacyRequests}
	case adminPrivacyResolvePathID(request.Path) != "":
		return RequestAdapted{Route: RouteAdminPrivacyRequests}
	case request.Path == "/api/admin/privacy-retention/run":
		return RequestAdapted{Route: RouteAdminPrivacyRetention}
	case request.Method.String() == MethodPost.String() && request.Path == "/api/moderation/reports":
		return RequestAdapted{Route: RouteModerationReports}
	case request.Method.String() == MethodGet.String() && adminModerationReportsPathOnly(request.Path) == "/api/admin/moderation/reports":
		return RequestAdapted{Route: RouteAdminModerationReports}
	case moderationTriagePathOnly(request.Path) != "":
		return RequestAdapted{Route: RouteAdminModerationReports}
	case request.Method.String() == MethodGet.String() && savedQueueViewPathOnly(request.Path) == "/api/saved-queue-views":
		return RequestAdapted{Route: RouteSavedQueueViews}
	case request.Method.String() == MethodPost.String() && request.Path == "/api/saved-queue-views":
		return RequestAdapted{Route: RouteSavedQueueViews}
	case request.Path == "/api/tasks" || tasksPathOnly(request.Path) == "/api/tasks":
		return RequestAdapted{Route: RouteTasks}
	case taskActionPath(request.Path).taskID != "":
		return RequestAdapted{Route: RouteTasks}
	case request.Method.String() == MethodGet.String() && taskDetailPathID(request.Path) != "":
		return RequestAdapted{Route: RouteTasks}
	case request.Method.String() == MethodGet.String() && notificationsPathOnly(request.Path) == "/api/notifications":
		return RequestAdapted{Route: RouteNotifications}
	case request.Method.String() == MethodPost.String() && notificationReadPathID(request.Path) != "":
		return RequestAdapted{Route: RouteNotifications}
	case organizationCollectionPathOnly(request.Path) == "/api/organizations":
		return RequestAdapted{Route: RouteOrganizations}
	case organizationMemberRoute(request.Path) != "":
		return RequestAdapted{Route: RouteOrganizationMembers}
	case organizationTeamsRoute(request.Path) != "":
		return RequestAdapted{Route: RouteOrganizationTeams}
	case standaloneTeamsPathOnly(request.Path) == "/api/teams":
		return RequestAdapted{Route: RouteStandaloneTeams}
	case teamWorkPathID(request.Path) != "":
		return RequestAdapted{Route: RouteTasks}
	case taskCommentsPathID(request.Path) != "":
		return RequestAdapted{Route: RouteTaskComments}
	case submissionCommentsPathID(request.Path) != "":
		return RequestAdapted{Route: RouteSubmissionComments}
	case taskReservationsPath(request.Path).taskID != "":
		return RequestAdapted{Route: RouteTaskReservations}
	case taskSubmissionsPath(request.Path).taskID != "":
		return RequestAdapted{Route: RouteSubmissions}
	case userSubmissionsPath(request.Path) != "":
		return RequestAdapted{Route: RouteSubmissions}
	case userWorkPathID(request.Path) != "":
		return RequestAdapted{Route: RouteTasks}
	case usersPath(request.Path) != "":
		return RequestAdapted{Route: RouteUsers}
	case creditsPathOnly(request.Path) == "/api/credits/balance":
		return RequestAdapted{Route: RouteLedger}
	case creditsPathOnly(request.Path) == "/api/credits/ledger":
		return RequestAdapted{Route: RouteLedger}
	case organizationCreditsPath(request.Path).organizationID != "":
		return RequestAdapted{Route: RouteLedger}
	case collectibleRoute(request.Path) != "":
		return RequestAdapted{Route: RouteCollectibles}
	case request.Path == "/api/admin/operations":
		return RequestAdapted{Route: RouteAdminOperations}
	case platformAdminRoute(request.Path) != "":
		return RequestAdapted{Route: RoutePlatformAdmins}
	case adminAuditEventsPathOnly(request.Path) == "/api/admin/audit-events":
		return RequestAdapted{Route: RouteAuditEvents}
	case agentCredentialsRoute(request.Path) != "":
		return RequestAdapted{Route: RouteAgentCredentials}
	default:
		return RequestUnsupported{Reason: "request route is not implemented by the WASM demo adapter"}
	}
}

func tasksPathOnly(path string) string {
	return strings.SplitN(path, "?", 2)[0]
}

type taskActionRoute struct {
	taskID string
	action string
}

func taskActionPath(path string) taskActionRoute {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "tasks" {
		return taskActionRoute{}
	}
	switch strings.TrimSpace(parts[3]) {
	case "open", "cancel", "unpublish", "funding", "refund", "collectible-refund", "collectible-reward":
		return taskActionRoute{taskID: strings.TrimSpace(parts[2]), action: strings.TrimSpace(parts[3])}
	default:
		return taskActionRoute{}
	}
}

func taskDetailPathID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "api" || parts[1] != "tasks" {
		return ""
	}
	return strings.TrimSpace(parts[2])
}

func notificationsPathOnly(path string) string {
	return strings.SplitN(path, "?", 2)[0]
}

func usersPath(path string) string {
	pathOnly := strings.SplitN(path, "?", 2)[0]
	if pathOnly == "/api/users" {
		return "collection"
	}
	parts := strings.Split(strings.Trim(pathOnly, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "users" {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

func notificationReadPathID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "notifications" || parts[3] != "read" {
		return ""
	}
	return strings.TrimSpace(parts[2])
}

func organizationCollectionPathOnly(path string) string {
	return strings.SplitN(path, "?", 2)[0]
}

func organizationTeamsRoute(path string) string {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "organizations" || parts[3] != "teams" {
		return ""
	}
	return strings.TrimSpace(parts[2])
}

func organizationMemberRoute(path string) string {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "organizations" && parts[3] == "members" {
		return strings.TrimSpace(parts[2])
	}
	if len(parts) == 6 && parts[0] == "api" && parts[1] == "organizations" && parts[3] == "members" {
		return strings.TrimSpace(parts[2]) + ":" + strings.TrimSpace(parts[4]) + ":" + strings.TrimSpace(parts[5])
	}
	return ""
}

func standaloneTeamsPathOnly(path string) string {
	return strings.SplitN(path, "?", 2)[0]
}

func teamWorkPathID(path string) string {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "teams" && parts[3] == "work" {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

func userWorkPathID(path string) string {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "users" && parts[3] == "work" {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

func adminPrivacyResolvePathID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 5 && parts[0] == "api" && parts[1] == "admin" && parts[2] == "privacy-requests" && parts[4] == "resolve" {
		return strings.TrimSpace(parts[3])
	}
	return ""
}

func adminModerationReportsPathOnly(path string) string {
	return strings.SplitN(path, "?", 2)[0]
}

func moderationTriagePathOnly(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 6 && parts[0] == "api" && parts[1] == "admin" && parts[2] == "moderation" && parts[3] == "reports" && parts[5] == "triage" {
		return strings.TrimSpace(parts[4])
	}
	return ""
}

func collectibleRoute(path string) string {
	pathOnly := strings.SplitN(path, "?", 2)[0]
	if pathOnly == "/api/collectibles" || pathOnly == "/api/collectibles/catalog" || pathOnly == "/api/collectibles/award" {
		return pathOnly
	}
	parts := strings.Split(strings.Trim(pathOnly, "/"), "/")
	if len(parts) == 5 && parts[0] == "api" && parts[1] == "tasks" && (parts[3] == "collectible-refund" || parts[3] == "collectible-reward") {
		return strings.TrimSpace(parts[2]) + ":" + strings.TrimSpace(parts[3])
	}
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "collectibles" && parts[3] == "transfer" {
		return strings.TrimSpace(parts[2])
	}
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "organizations" && parts[3] == "collectibles" {
		return strings.TrimSpace(parts[2])
	}
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "teams" && parts[3] == "collectibles" {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

func platformAdminRoute(path string) string {
	pathOnly := strings.SplitN(path, "?", 2)[0]
	if pathOnly == "/api/admin/platform-admins" {
		return "collection"
	}
	parts := strings.Split(strings.Trim(pathOnly, "/"), "/")
	if len(parts) == 5 && parts[0] == "api" && parts[1] == "admin" && parts[2] == "platform-admins" && parts[4] == "revoke" {
		return strings.TrimSpace(parts[3])
	}
	return ""
}

func adminAuditEventsPathOnly(path string) string {
	return strings.SplitN(path, "?", 2)[0]
}

func agentCredentialsRoute(path string) string {
	pathOnly := strings.SplitN(path, "?", 2)[0]
	if pathOnly == "/api/agent-credentials" {
		return "collection"
	}
	parts := strings.Split(strings.Trim(pathOnly, "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "agent-credentials" && parts[3] == "revoke" {
		return strings.TrimSpace(parts[2])
	}
	return ""
}
