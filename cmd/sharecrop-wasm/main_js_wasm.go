//go:build js && wasm

package main

import (
	"encoding/json"
	"strings"
	"syscall/js"
	"time"

	"github.com/e6qu/sharecrop/internal/wasmdemo"
)

type wasmStatus struct {
	Name    string `json:"name"`
	Target  string `json:"target"`
	Runtime string `json:"runtime"`
}

type wasmHandleResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
	Error  string `json:"error"`
	Route  string `json:"route"`
}

type wasmConfigureResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

type jsHost struct {
	value js.Value
}

type jsHostStorage struct {
	host js.Value
}

type jsHostClock struct {
	host js.Value
}

type jsHostActor struct {
	host js.Value
}

type jsHostIDs struct {
	host js.Value
}

var configuredHost jsHost
var hostConfigured bool

func main() {
	js.Global().Set("sharecropWasmBackendStatus", js.FuncOf(func(js.Value, []js.Value) interface{} {
		return encodeStatus()
	}))
	js.Global().Set("sharecropConfigureHost", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			return encodeConfigureResponse(wasmConfigureResponse{Error: "host configuration argument is required"})
		}
		host := jsHost{value: args[0]}
		if reason := validateJSHost(host.value); reason != "" {
			return encodeConfigureResponse(wasmConfigureResponse{Error: reason})
		}
		configuredHost = host
		hostConfigured = true
		return encodeConfigureResponse(wasmConfigureResponse{Status: "configured"})
	}))
	js.Global().Set("sharecropHandleRequest", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) != 3 {
			return encodeHandleResponse(wasmHandleResponse{Status: 400, Error: "method, path, and body arguments are required"})
		}
		requestResult := wasmdemo.NewRequest(args[0].String(), args[1].String(), args[2].String())
		request, requestMatched := requestResult.(wasmdemo.RequestAccepted)
		if !requestMatched {
			return encodeHandleResponse(wasmHandleResponse{Status: 400, Error: requestResult.(wasmdemo.RequestRejected).Reason})
		}
		adaptResult := wasmdemo.Adapt(request.Value)
		adapted, adaptedMatched := adaptResult.(wasmdemo.RequestAdapted)
		if !adaptedMatched {
			return encodeHandleResponse(wasmHandleResponse{Status: 404, Error: adaptResult.(wasmdemo.RequestUnsupported).Reason})
		}
		if !hostConfigured {
			return encodeHandleResponse(wasmHandleResponse{Status: 500, Error: "host runtime is not configured", Route: adapted.Route.String()})
		}
		handleResult := handleWithConfiguredHost(request.Value, adapted.Route)
		handled, handledMatched := handleResult.(wasmdemo.RequestHandled)
		if !handledMatched {
			return encodeHandleResponse(wasmHandleResponse{Status: 500, Error: handleResult.(wasmdemo.RequestHandleRejected).Reason, Route: adapted.Route.String()})
		}
		return encodeHandleResponse(wasmHandleResponse{Status: handled.Value.Status, Body: handled.Value.Body, Route: adapted.Route.String()})
	}))
	select {}
}

func handleWithConfiguredHost(request wasmdemo.Request, route wasmdemo.Route) wasmdemo.HandleResult {
	runtimeResult := wasmdemo.ValidateHostRuntime(configuredHost)
	runtime, runtimeMatched := runtimeResult.(wasmdemo.HostRuntimeAccepted)
	if !runtimeMatched {
		return wasmdemo.RequestHandleRejected{Reason: runtimeResult.(wasmdemo.HostRuntimeRejected).Reason}
	}
	switch route.String() {
	case wasmdemo.RouteAuth.String():
		runtimeIDs, matched := runtime.InteractionIDs.(wasmdemo.RuntimeIDSource)
		if !matched {
			return wasmdemo.RequestHandleRejected{Reason: "runtime id source is required"}
		}
		handler := wasmdemo.NewAuthHandler(runtime.Storage, runtime.Clock, runtime.Actor, runtimeIDs)
		return handler.Handle(request)
	case wasmdemo.RouteAccount.String():
		runtimeIDs, matched := runtime.InteractionIDs.(wasmdemo.RuntimeIDSource)
		if !matched {
			return wasmdemo.RequestHandleRejected{Reason: "runtime id source is required"}
		}
		handler := wasmdemo.NewAccountHandler(runtime.Storage, runtime.Clock, runtime.Actor, runtimeIDs)
		return handler.Handle(request)
	case wasmdemo.RouteUsers.String():
		handler := wasmdemo.NewUsersHandler(runtime.Storage)
		return handler.Handle(request)
	case wasmdemo.RouteTasks.String():
		taskIDs, taskIDMatched := runtime.InteractionIDs.(wasmdemo.TaskIDSource)
		if !taskIDMatched {
			return wasmdemo.RequestHandleRejected{Reason: "host task id adapter is required"}
		}
		handler := wasmdemo.NewTaskHandler(runtime.Storage, runtime.Actor, taskIDs)
		return handler.Handle(request)
	case wasmdemo.RoutePrivacyRequests.String(), wasmdemo.RouteAdminPrivacyRequests.String():
		privacyIDs, privacyIDMatched := runtime.InteractionIDs.(wasmdemo.PrivacyRequestIDSource)
		if !privacyIDMatched {
			return wasmdemo.RequestHandleRejected{Reason: "host privacy request id adapter is required"}
		}
		handler := wasmdemo.NewPrivacyRequestHandler(runtime.Storage, runtime.Clock, runtime.Actor, privacyIDs)
		return handler.Handle(request)
	case wasmdemo.RouteAdminPrivacyRetention.String(),
		wasmdemo.RouteAdminOperations.String(),
		wasmdemo.RoutePlatformAdmins.String(),
		wasmdemo.RouteAuditEvents.String():
		runtimeIDs, matched := runtime.InteractionIDs.(wasmdemo.RuntimeIDSource)
		if !matched {
			return wasmdemo.RequestHandleRejected{Reason: "runtime id source is required"}
		}
		handler := wasmdemo.NewAdminHandler(runtime.Storage, runtime.Clock, runtime.Actor, runtimeIDs)
		return handler.Handle(request, route)
	case wasmdemo.RouteModerationReports.String():
		runtimeIDs, matched := runtime.InteractionIDs.(wasmdemo.RuntimeIDSource)
		if !matched {
			return wasmdemo.RequestHandleRejected{Reason: "runtime id source is required"}
		}
		handler := wasmdemo.NewModerationReportHandler(runtime.Storage, runtime.Clock, runtime.Actor, runtimeIDs)
		return handler.Handle(request)
	case wasmdemo.RouteAdminModerationReports.String():
		runtimeIDs, matched := runtime.InteractionIDs.(wasmdemo.RuntimeIDSource)
		if !matched {
			return wasmdemo.RequestHandleRejected{Reason: "runtime id source is required"}
		}
		handler := wasmdemo.NewAdminHandler(runtime.Storage, runtime.Clock, runtime.Actor, runtimeIDs)
		return handler.Handle(request, route)
	case wasmdemo.RouteSavedQueueViews.String():
		savedQueueIDs, savedQueueIDMatched := runtime.InteractionIDs.(wasmdemo.SavedQueueViewIDSource)
		if !savedQueueIDMatched {
			return wasmdemo.RequestHandleRejected{Reason: "host saved queue view id adapter is required"}
		}
		handler := wasmdemo.NewSavedQueueViewHandler(runtime.Storage, runtime.Actor, savedQueueIDs)
		return handler.Handle(request)
	case wasmdemo.RouteNotifications.String():
		handler := wasmdemo.NewNotificationHandler(runtime.Storage, runtime.Actor)
		return handler.Handle(request)
	case wasmdemo.RouteOrganizations.String(),
		wasmdemo.RouteOrganizationMembers.String(),
		wasmdemo.RouteOrganizationTeams.String(),
		wasmdemo.RouteStandaloneTeams.String():
		organizationIDs, organizationIDMatched := runtime.InteractionIDs.(wasmdemo.OrganizationIDSource)
		if !organizationIDMatched {
			return wasmdemo.RequestHandleRejected{Reason: "host organization id adapter is required"}
		}
		handler := wasmdemo.NewOrganizationHandler(runtime.Storage, runtime.Actor, organizationIDs, configuredHost)
		return handler.Handle(request)
	case wasmdemo.RouteTaskComments.String(),
		wasmdemo.RouteSubmissionComments.String(),
		wasmdemo.RouteTaskReservations.String(),
		wasmdemo.RouteSubmissions.String(),
		wasmdemo.RouteLedger.String():
		handler := wasmdemo.NewInteractionHandler(runtime.Storage, runtime.Clock, runtime.Actor, runtime.InteractionIDs)
		return handler.Handle(request)
	case wasmdemo.RouteCollectibles.String():
		runtimeIDs, matched := runtime.InteractionIDs.(wasmdemo.RuntimeIDSource)
		if !matched {
			return wasmdemo.RequestHandleRejected{Reason: "runtime id source is required"}
		}
		handler := wasmdemo.NewCollectibleHandler(runtime.Storage, runtime.Actor, runtimeIDs)
		return handler.Handle(request)
	case wasmdemo.RouteAgentCredentials.String():
		runtimeIDs, matched := runtime.InteractionIDs.(wasmdemo.RuntimeIDSource)
		if !matched {
			return wasmdemo.RequestHandleRejected{Reason: "runtime id source is required"}
		}
		handler := wasmdemo.NewAgentCredentialHandler(runtime.Storage, runtime.Actor, runtimeIDs)
		return handler.Handle(request)
	default:
		return wasmdemo.RequestHandleRejected{Reason: "configured WASM host does not execute this route"}
	}
}

func (host jsHost) Storage() wasmdemo.BrowserStorage {
	return jsHostStorage{host: host.value}
}

func (host jsHost) Clock() wasmdemo.HandlerClock {
	return jsHostClock{host: host.value}
}

func (host jsHost) Actor() wasmdemo.HandlerActor {
	return jsHostActor{host: host.value}
}

func (host jsHost) InteractionIDs() wasmdemo.InteractionIDSource {
	return jsHostIDs{host: host.value}
}

func (host jsHost) UserIDForEmail(email string) (string, bool) {
	result := host.value.Get("userIDForEmail").Invoke(email)
	if result.Type() != js.TypeString {
		return "", false
	}
	userID := strings.TrimSpace(result.String())
	if userID == "" {
		return "", false
	}
	return userID, true
}

func (storage jsHostStorage) Put(key wasmdemo.StorageKey, value string) wasmdemo.StorageWriteResult {
	result := storage.host.Get("storagePut").Invoke(key.String(), value)
	if result.Type() != js.TypeBoolean || !result.Bool() {
		return wasmdemo.StorageWriteRejected{Reason: "host storage put failed"}
	}
	return wasmdemo.StorageWritten{}
}

func (storage jsHostStorage) Get(key wasmdemo.StorageKey) wasmdemo.StorageReadResult {
	has := storage.host.Get("storageHas").Invoke(key.String())
	if has.Type() != js.TypeBoolean {
		return wasmdemo.StorageReadRejected{Reason: "host storage has returned an invalid value"}
	}
	if !has.Bool() {
		return wasmdemo.StorageMissing{Reason: "host storage key was not found"}
	}
	value := storage.host.Get("storageGet").Invoke(key.String())
	if value.Type() != js.TypeString {
		return wasmdemo.StorageReadRejected{Reason: "host storage get returned an invalid value"}
	}
	return wasmdemo.StorageRead{Value: value.String()}
}

func (clock jsHostClock) Now() time.Time {
	raw := strings.TrimSpace(clock.host.Get("now").Invoke().String())
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		panic("host clock returned an invalid RFC3339 time")
	}
	return value
}

func (actor jsHostActor) UserID() string {
	return strings.TrimSpace(actor.host.Get("actorID").Invoke().String())
}

func (ids jsHostIDs) NextSubmissionID() string {
	return ids.next("submission")
}

func (ids jsHostIDs) NextCommentID() string {
	return ids.next("comment")
}

func (ids jsHostIDs) NextReservationID() string {
	return ids.next("reservation")
}

func (ids jsHostIDs) NextLedgerEntryID() string {
	return ids.next("ledger")
}

func (ids jsHostIDs) NextTaskID() string {
	return ids.next("task")
}

func (ids jsHostIDs) NextPrivacyRequestID() string {
	return ids.next("privacy")
}

func (ids jsHostIDs) NextSavedQueueViewID() string {
	return ids.next("saved_view")
}

func (ids jsHostIDs) NextOrganizationID() string {
	return ids.next("organization")
}

func (ids jsHostIDs) NextOrganizationMemberID() string {
	return ids.next("organization_member")
}

func (ids jsHostIDs) NextTeamID() string {
	return ids.next("team")
}

func (ids jsHostIDs) NextUserID() string {
	return ids.next("user")
}

func (ids jsHostIDs) NextAuditEventID() string {
	return ids.next("audit")
}

func (ids jsHostIDs) NextAccountToken() string {
	return ids.next("account_token")
}

func (ids jsHostIDs) NextCollectibleID() string {
	return ids.next("collectible")
}

func (ids jsHostIDs) NextNotificationID() string {
	return ids.next("notification")
}

func (ids jsHostIDs) NextAgentCredentialID() string {
	return ids.next("agent_credential")
}

func (ids jsHostIDs) NextAgentCredentialSecret() string {
	return "scrop_agent_" + ids.next("agent_secret")
}

func (ids jsHostIDs) next(kind string) string {
	return strings.TrimSpace(ids.host.Get("nextID").Invoke(kind).String())
}

func validateJSHost(host js.Value) string {
	if host.Type() != js.TypeObject {
		return "host configuration must be an object"
	}
	requiredFunctions := []string{"storageHas", "storageGet", "storagePut", "now", "actorID", "nextID", "userIDForEmail"}
	for index := range requiredFunctions {
		if host.Get(requiredFunctions[index]).Type() != js.TypeFunction {
			return "host function is missing: " + requiredFunctions[index]
		}
	}
	return ""
}

func encodeStatus() string {
	runtime := "unconfigured"
	if hostConfigured {
		runtime = "configured"
	}
	encoded, err := json.Marshal(wasmStatus{Name: "sharecrop-wasm", Target: "js/wasm", Runtime: runtime})
	if err != nil {
		panic("wasm status encoding failed")
	}
	return string(encoded)
}

func encodeConfigureResponse(response wasmConfigureResponse) string {
	encoded, err := json.Marshal(response)
	if err != nil {
		panic("wasm configure response encoding failed")
	}
	return string(encoded)
}

func encodeHandleResponse(response wasmHandleResponse) string {
	encoded, err := json.Marshal(response)
	if err != nil {
		panic("wasm response encoding failed")
	}
	return string(encoded)
}
