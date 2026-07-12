//go:build js && wasm

package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall/js"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/sqlitex"
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
	"github.com/e6qu/sharecrop/internal/wasmseed"
	"github.com/e6qu/sharecrop/migrations"
)

// wasmAccessTokenSecret is a fixed, demo-only HMAC secret for signing access
// tokens inside the browser. It never leaves the browser tab and protects
// nothing a user with devtools access doesn't already fully control.
const wasmAccessTokenSecret = "sharecrop-wasm-demo-access-token-secret-not-for-production-use"

// The browser demo persists the whole SQLite database as one base64 snapshot
// plus the seeded session's refresh token, so clicking through the demo and
// refreshing the page keeps the state. Reset clears both.
const (
	snapshotStorageKey = "sharecrop_sqlite_snapshot"
	sessionStorageKey  = "sharecrop_refresh_cookie"
	demoMemdbName      = "sharecrop-demo"
	demoDataSourceName = "file:/sharecrop-demo?vfs=memdb&_pragma=foreign_keys(off)"
	refreshCookieName  = "sharecrop_refresh_token"
	// seedSnapshotGlobal is the window global the page loader sets to the
	// base64 pre-generated seed snapshot before configuring the host.
	seedSnapshotGlobal = "__sharecropSeedSnapshot"
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
}

type wasmConfigureResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

var (
	configuredMux        http.Handler
	hostConfigured       bool
	currentRefreshCookie *http.Cookie
	sqliteHandle         *sql.DB
	hostStorage          js.Value
)

func main() {
	js.Global().Set("sharecropWasmBackendStatus", js.FuncOf(func(js.Value, []js.Value) interface{} {
		return encodeStatus()
	}))
	js.Global().Set("sharecropConfigureHost", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			return encodeConfigureResponse(wasmConfigureResponse{Error: "host configuration argument is required"})
		}
		if reason := validateJSHost(args[0]); reason != "" {
			return encodeConfigureResponse(wasmConfigureResponse{Error: reason})
		}
		mux, err := buildConfiguredMux(args[0])
		if err != "" {
			return encodeConfigureResponse(wasmConfigureResponse{Error: err})
		}
		hostStorage = args[0]
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
		captureRefreshCookie(recorder.Result().Cookies())
		return encodeHandleResponse(wasmHandleResponse{Status: recorder.Code, Body: recorder.Body.String()})
	}))
	js.Global().Set("sharecropPersistSnapshot", js.FuncOf(func(js.Value, []js.Value) interface{} {
		// Called from the page's beforeunload handler: snapshotting the whole
		// database is too expensive to do after every request, so state is saved
		// once when the page is about to unload (refresh or close).
		if hostConfigured {
			persistSnapshot()
		}
		return js.ValueOf(true)
	}))
	js.Global().Set("sharecropResetDemo", js.FuncOf(func(js.Value, []js.Value) interface{} {
		if hostConfigured {
			storageDelete(snapshotStorageKey)
			storageDelete(sessionStorageKey)
		}
		return js.ValueOf(true)
	}))
	select {}
}

// buildConfiguredMux opens a browser-local SQLite database, restores the demo
// from a saved snapshot or seeds it fresh, and builds the same app mux
// production runs (appmux.New) over stores backed by that database.
func buildConfiguredMux(host js.Value) (http.Handler, string) {
	// The browser demo has no email and no server log, so it reads account
	// tokens (email verification, password reset) straight from the HTTP
	// response. Safe here (browser-local, no real accounts); production defaults
	// to log delivery, so opt into api delivery explicitly.
	if err := os.Setenv("SHARECROP_ACCOUNT_TOKEN_DELIVERY", "api"); err != nil {
		return nil, "set demo account token delivery: " + err.Error()
	}

	// A snapshot must be loaded into the shared memdb before the database is
	// opened, so every pooled connection sees the restored data. Prefer the
	// user's saved snapshot (their clicked-through state); otherwise fall back to
	// the pre-generated seed snapshot, which is far faster than re-running the
	// seed. Only when neither is available (e.g. the scenario-parity harness)
	// does the demo migrate and seed from scratch.
	userSnapshot, hasUserSnapshot := readSnapshot(host)
	seedSnapshot, hasSeedSnapshot := readSeedSnapshot()
	switch {
	case hasUserSnapshot:
		sqlitex.LoadSnapshot(demoMemdbName, userSnapshot)
	case hasSeedSnapshot:
		sqlitex.LoadSnapshot(demoMemdbName, seedSnapshot)
	}

	handle, err := sqlitex.Open(demoDataSourceName)
	if err != nil {
		return nil, "open demo database: " + err.Error()
	}
	// The memdb VFS shares one database across every pooled connection (its name
	// starts with "/"), so all requests see the same in-memory data without
	// pinning to a single connection.
	sqliteHandle = handle

	tokenSecretResult := auth.NewAccessTokenSecret(wasmAccessTokenSecret)
	tokenSecret, matched := tokenSecretResult.(auth.AccessTokenSecretAccepted)
	if !matched {
		return nil, tokenSecretResult.(auth.AccessTokenSecretRejected).Reason.Description()
	}

	stores := wasmseed.StoresFromHandle(db.NewSQLite(handle))

	switch {
	case hasUserSnapshot:
		if token, present := storageGet(host, sessionStorageKey); present {
			currentRefreshCookie = &http.Cookie{Name: refreshCookieName, Value: token}
		}
	case hasSeedSnapshot:
		login := wasmseed.LoginDemoAdmin(context.Background(), tokenSecret.Value, stores)
		if login.Err != "" {
			return nil, login.Err
		}
		currentRefreshCookie = login.AdminRefreshToken
	default:
		if reason := seedFreshDatabase(host, handle, tokenSecret.Value, stores); reason != "" {
			return nil, reason
		}
	}

	return appmux.New(tokenSecret.Value, stores), ""
}

// readSeedSnapshot decodes the pre-generated seed snapshot the page fetched into
// a global before configuring, if present.
func readSeedSnapshot() ([]byte, bool) {
	value := js.Global().Get(seedSnapshotGlobal)
	if value.Type() != js.TypeString {
		return nil, false
	}
	snapshot, err := base64.StdEncoding.DecodeString(value.String())
	if err != nil {
		return nil, false
	}
	return snapshot, true
}

// readSnapshot reads and decodes the saved database snapshot from browser
// storage, if any.
func readSnapshot(host js.Value) ([]byte, bool) {
	encoded, present := storageGet(host, snapshotStorageKey)
	if !present {
		return nil, false
	}
	snapshot, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, false
	}
	return snapshot, true
}

// seedFreshDatabase migrates and seeds a new demo database, then saves the first
// snapshot so the seeded state survives the next page load.
func seedFreshDatabase(host js.Value, handle *sql.DB, secret auth.AccessTokenSecret, stores appmux.Stores) string {
	ctx := context.Background()
	if migrateErr := db.MigrateUpSQLite(ctx, handle, migrations.FS); migrateErr != nil {
		return "migrate demo database: " + migrateErr.Error()
	}

	seedResult := wasmseed.Seed(ctx, secret, stores)
	if seedResult.Err != "" {
		return seedResult.Err
	}
	if _, ok := stores.PlatformAdmins.Grant(ctx, seedResult.AdminUserID, seedResult.AdminUserID).(httpserver.PlatformAdminSaved); !ok {
		return "grant demo admin failed"
	}
	currentRefreshCookie = seedResult.AdminRefreshToken

	persistSnapshotWithHost(host)
	return ""
}

func captureRefreshCookie(cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie.Name == refreshCookieName {
			currentRefreshCookie = cookie
		}
	}
}

func persistSnapshot() {
	persistSnapshotWithHost(hostStorage)
}

func persistSnapshotWithHost(host js.Value) {
	if sqliteHandle == nil {
		return
	}
	snapshot, err := sqlitex.Snapshot(context.Background(), sqliteHandle)
	if err != nil {
		return
	}
	storagePut(host, snapshotStorageKey, base64.StdEncoding.EncodeToString(snapshot))
	if currentRefreshCookie != nil {
		storagePut(host, sessionStorageKey, currentRefreshCookie.Value)
	}
}

func storageGet(host js.Value, key string) (string, bool) {
	has := host.Get("storageHas").Invoke(key)
	if has.Type() != js.TypeBoolean || !has.Bool() {
		return "", false
	}
	value := host.Get("storageGet").Invoke(key)
	if value.Type() != js.TypeString {
		return "", false
	}
	return value.String(), true
}

func storagePut(host js.Value, key string, value string) {
	host.Get("storagePut").Invoke(key, value)
}

func storageDelete(key string) {
	if hostStorage.Get("storageDelete").Type() == js.TypeFunction {
		hostStorage.Get("storageDelete").Invoke(key)
	}
}

func validateJSHost(host js.Value) string {
	if host.Type() != js.TypeObject {
		return "host configuration must be an object"
	}
	requiredFunctions := []string{"storageHas", "storageGet", "storagePut"}
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
