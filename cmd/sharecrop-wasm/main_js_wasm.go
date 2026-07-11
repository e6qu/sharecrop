//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall/js"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasmdemo"
	"github.com/e6qu/sharecrop/web"
)

// wasmAccessTokenSecret is a fixed, demo-only HMAC secret for signing access
// tokens inside the browser. It never leaves the browser tab and protects
// nothing a user with devtools access doesn't already fully control, so a
// random per-session secret would add ceremony without adding security.
const wasmAccessTokenSecret = "sharecrop-wasm-demo-access-token-secret-not-for-production-use"

type wasmStatus struct {
	Name    string `json:"name"`
	Target  string `json:"target"`
	Runtime string `json:"runtime"`
}

type wasmHandleResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
	Error  string `json:"error"`
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

type jsHostIDs struct {
	host js.Value
}

var (
	configuredMux        http.Handler
	hostConfigured       bool
	currentRefreshCookie *http.Cookie
)

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
		mux, err := buildConfiguredMux(host)
		if err != "" {
			return encodeConfigureResponse(wasmConfigureResponse{Error: err})
		}
		configuredMux = mux
		hostConfigured = true
		return encodeConfigureResponse(wasmConfigureResponse{Status: "configured"})
	}))
	js.Global().Set("sharecropHandleRequest", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) != 4 {
			return encodeHandleResponse(wasmHandleResponse{Status: 400, Error: "method, path, body, and authorization arguments are required"})
		}
		if !hostConfigured || configuredMux == nil {
			return encodeHandleResponse(wasmHandleResponse{Status: 500, Error: "host runtime is not configured"})
		}
		method, path, body, authorization := args[0].String(), args[1].String(), args[2].String(), args[3].String()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		if currentRefreshCookie != nil {
			req.AddCookie(currentRefreshCookie)
		}
		recorder := httptest.NewRecorder()
		configuredMux.ServeHTTP(recorder, req)
		for _, cookie := range recorder.Result().Cookies() {
			if cookie.Name == "sharecrop_refresh_token" {
				currentRefreshCookie = cookie
			}
		}
		return encodeHandleResponse(wasmHandleResponse{Status: recorder.Code, Body: recorder.Body.String()})
	}))
	select {}
}

// buildConfiguredMux constructs the real internal/http mux over browser-
// storage-backed domain services (the same services cmd/sharecrop wires
// against Postgres), seeds the demo scenario the first time it runs against
// fresh storage, and pre-authenticates the seeded admin user so the demo
// loads already logged in - matching the UX of the browser demo's original
// hand-rolled backend, now backed by real business logic end to end.
func buildConfiguredMux(host jsHost) (http.Handler, string) {
	// The browser demo has no email and no accessible server log, so it reads
	// account tokens (email verification, password reset) straight from the
	// HTTP response. That is safe here - browser-local storage, no real
	// accounts - but the default is log delivery (fail closed) for production,
	// so opt into api delivery explicitly for the demo.
	if err := os.Setenv("SHARECROP_ACCOUNT_TOKEN_DELIVERY", "api"); err != nil {
		return nil, "set demo account token delivery: " + err.Error()
	}

	storage := host.Storage()
	ids := host.InteractionIDs()
	clock := host.Clock()

	tokenSecretResult := auth.NewAccessTokenSecret(wasmAccessTokenSecret)
	tokenSecret, tokenSecretMatched := tokenSecretResult.(auth.AccessTokenSecretAccepted)
	if !tokenSecretMatched {
		return nil, tokenSecretResult.(auth.AccessTokenSecretRejected).Reason.Description()
	}

	authServiceResult := auth.NewService(wasmdemo.NewAuthBrowserStore(storage, ids), tokenSecret.Value, clock)
	authService, authServiceMatched := authServiceResult.(auth.ServiceCreated)
	if !authServiceMatched {
		return nil, authServiceResult.(auth.ServiceRejected).Reason.Description()
	}
	tokenVerifier := auth.NewAccessTokenVerifier(tokenSecret.Value, clock)

	organizationService := org.NewService(wasmdemo.NewOrgBrowserStore(storage, ids))
	agentService := agent.NewService(wasmdemo.NewAgentBrowserStore(storage))
	orgCredentialService := orgcred.NewService(wasmdemo.NewOrgCredentialBrowserStore(storage))
	taskStore := wasmdemo.NewTaskBrowserStore(storage, ids, clock)
	taskService := task.NewService(taskStore, organizationService, agentService)
	submissionStore := wasmdemo.NewSubmissionBrowserStore(storage, ids)
	submissionService := submission.NewService(submissionStore, taskStore, organizationService)
	ledgerService := ledger.NewService(wasmdemo.NewLedgerBrowserStore(storage, ids))
	assetService := assets.NewService(wasmdemo.NewAssetBrowserStore(storage, ids))
	notificationService := notification.NewService(wasmdemo.NewNotificationBrowserStore(storage))

	staticFiles, staticErr := web.StaticFiles()
	if staticErr != nil {
		return nil, staticErr.Error()
	}

	runtime := httpserver.DefaultRuntimeState(map[string]bool{})
	runtime.NotificationService = notificationService
	runtime.PrivacyService = httpserver.NewMemoryPrivacyService(submissionStore)

	seedResult := wasmdemo.SeedDemoScenario(context.Background(), authService.Value, organizationService, taskService, ledgerService, submissionService, assetService, notificationService)
	if seedResult.Err != "" {
		return nil, seedResult.Err
	}
	runtime.PlatformAdmins.Grant(context.Background(), seedResult.AdminUserID, seedResult.AdminUserID)

	mux := httpserver.NewWithRuntimeState(staticFiles, authService.Value, tokenVerifier, organizationService, taskService, submissionService, ledgerService, agentService, orgCredentialService, assetService, runtime)

	currentRefreshCookie = seedResult.AdminRefreshToken

	return mux, ""
}

func (host jsHost) Storage() wasmdemo.BrowserStorage {
	return jsHostStorage{host: host.value}
}

func (host jsHost) Clock() wasmdemo.HandlerClock {
	return jsHostClock{host: host.value}
}

func (host jsHost) InteractionIDs() wasmdemo.InteractionIDSource {
	return jsHostIDs{host: host.value}
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

func (ids jsHostIDs) NextSubmissionID() string  { return ids.next("submission") }
func (ids jsHostIDs) NextCommentID() string     { return ids.next("comment") }
func (ids jsHostIDs) NextReservationID() string { return ids.next("reservation") }

// NextLedgerEntryID generates a real UUID directly in Go rather than
// delegating to the host's counter-based nextID: ledger entries round-trip
// through core.ParseLedgerEntryID when listed back (ListEntries/
// ListOrganizationEntries), which requires UUID-shaped ids - a "ledger-1"
// style counter id would fail to parse.
func (ids jsHostIDs) NextLedgerEntryID() string {
	result := core.NewLedgerEntryID()
	created, matched := result.(core.LedgerEntryIDCreated)
	if !matched {
		panic("generate ledger entry id failed")
	}
	return created.Value.String()
}

func (ids jsHostIDs) next(kind string) string {
	return strings.TrimSpace(ids.host.Get("nextID").Invoke(kind).String())
}

func validateJSHost(host js.Value) string {
	if host.Type() != js.TypeObject {
		return "host configuration must be an object"
	}
	requiredFunctions := []string{"storageHas", "storageGet", "storagePut", "now", "nextID"}
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
