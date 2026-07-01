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
	MethodGet  = Method{value: "GET"}
	MethodPost = Method{value: "POST"}
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
	RouteModerationReports      = Route{value: "moderation_reports"}
	RouteAdminModerationReports = Route{value: "admin_moderation_reports"}
	RouteSavedQueueViews        = Route{value: "saved_queue_views"}
	RouteTasks                  = Route{value: "tasks"}
	RouteNotifications          = Route{value: "notifications"}
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
	case request.Method.String() == MethodPost.String() && request.Path == "/api/privacy-requests":
		return RequestAdapted{Route: RoutePrivacyRequests}
	case request.Method.String() == MethodGet.String() && request.Path == "/api/admin/privacy-requests":
		return RequestAdapted{Route: RouteAdminPrivacyRequests}
	case request.Method.String() == MethodPost.String() && request.Path == "/api/moderation/reports":
		return RequestAdapted{Route: RouteModerationReports}
	case request.Method.String() == MethodGet.String() && request.Path == "/api/admin/moderation/reports":
		return RequestAdapted{Route: RouteAdminModerationReports}
	case request.Method.String() == MethodGet.String() && request.Path == "/api/saved-queue-views":
		return RequestAdapted{Route: RouteSavedQueueViews}
	case request.Method.String() == MethodPost.String() && request.Path == "/api/saved-queue-views":
		return RequestAdapted{Route: RouteSavedQueueViews}
	case request.Method.String() == MethodPost.String() && request.Path == "/api/tasks":
		return RequestAdapted{Route: RouteTasks}
	case request.Method.String() == MethodGet.String() && taskDetailPathID(request.Path) != "":
		return RequestAdapted{Route: RouteTasks}
	case request.Method.String() == MethodGet.String() && notificationsPathOnly(request.Path) == "/api/notifications":
		return RequestAdapted{Route: RouteNotifications}
	case request.Method.String() == MethodPost.String() && notificationReadPathID(request.Path) != "":
		return RequestAdapted{Route: RouteNotifications}
	default:
		return RequestUnsupported{Reason: "request route is not implemented by the WASM demo adapter"}
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

func notificationReadPathID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "notifications" || parts[3] != "read" {
		return ""
	}
	return strings.TrimSpace(parts[2])
}
