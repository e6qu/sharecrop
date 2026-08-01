package openapi

import "testing"

// The status tests feed minimal handler sources through Extract so the
// derivation is exercised end to end: mux registration, call graph, and the
// generated responses.
const statusTestServerSource = `package httpserver

import "net/http"

func routes(server Server) {
	mux.HandleFunc("POST /api/things", server.createThing)
	mux.HandleFunc("GET /api/things", server.listThings)
	mux.HandleFunc("POST /api/things/{thing_id}/open", server.openThing)
	mux.HandleFunc("DELETE /api/sessions", server.deleteSession)
	mux.HandleFunc("GET /login", server.startLogin)
	mux.HandleFunc("GET /healthz", server.healthz)
}
`

const statusTestHandlersSource = `package httpserver

import "net/http"

func (server Server) createThing(w http.ResponseWriter, r *http.Request) {
	server.requireUserSubject(r)
	writeError(w, http.StatusBadRequest, code, "bad request")
	writeJSON(w, http.StatusCreated, response)
}

func (server Server) listThings(w http.ResponseWriter, r *http.Request) {
	server.requireUserSubject(r)
	writeThingsResponse(w, http.StatusOK, response)
}

func (server Server) openThing(w http.ResponseWriter, r *http.Request) {
	server.changeThingState(w, r)
}

func (server Server) changeThingState(w http.ResponseWriter, r *http.Request) {
	server.requireUserSubject(r)
	writeDomainError(w, reason)
	writeJSON(w, http.StatusOK, response)
}

func (server Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	server.requireUserSubject(r)
	w.WriteHeader(http.StatusNoContent)
}

func (server Server) startLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/next", http.StatusFound)
}

func (server Server) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{})
}
`

func statusTestRoutes(t *testing.T) map[string]Route {
	t.Helper()
	result := Extract(map[string][]byte{
		"server.go":   []byte(statusTestServerSource),
		"handlers.go": []byte(statusTestHandlersSource),
	})
	extracted, matched := result.(Extracted)
	if !matched {
		t.Fatalf("extract rejected: %#v", result)
	}
	routes := map[string]Route{}
	for _, route := range extracted.Routes {
		routes[route.OperationID] = route
	}
	return routes
}

func TestExtractDerivesSuccessStatusesFromResponseWriters(t *testing.T) {
	routes := statusTestRoutes(t)

	cases := []struct {
		operationID string
		want        []int
	}{
		{operationID: "createThing", want: []int{201}},
		{operationID: "listThings", want: []int{200}},
		{operationID: "deleteSession", want: []int{204}},
		{operationID: "startLogin", want: []int{302}},
		{operationID: "healthz", want: []int{200}},
	}
	for _, testCase := range cases {
		got := routes[testCase.operationID].SuccessStatuses
		if len(got) != len(testCase.want) {
			t.Fatalf("%s success statuses = %v, want %v", testCase.operationID, got, testCase.want)
		}
		for index := range got {
			if got[index] != testCase.want[index] {
				t.Fatalf("%s success statuses = %v, want %v", testCase.operationID, got, testCase.want)
			}
		}
	}
}

func TestExtractDerivesStatusesThroughSharedHelpers(t *testing.T) {
	routes := statusTestRoutes(t)

	openThing := routes["openThing"]
	if len(openThing.SuccessStatuses) != 1 || openThing.SuccessStatuses[0] != 200 {
		t.Fatalf("openThing success statuses = %v, want the helper's 200", openThing.SuccessStatuses)
	}
	if !openThing.EmitsErrors {
		t.Fatalf("openThing must report the helper's writeDomainError")
	}
	wantErrors := []int{400, 401, 403, 404, 409}
	if len(openThing.ErrorStatuses) != len(wantErrors) {
		t.Fatalf("openThing error statuses = %v, want %v", openThing.ErrorStatuses, wantErrors)
	}
	for index := range wantErrors {
		if openThing.ErrorStatuses[index] != wantErrors[index] {
			t.Fatalf("openThing error statuses = %v, want %v", openThing.ErrorStatuses, wantErrors)
		}
	}
}

func TestExtractDerivesErrorStatusesAndAuthImplies401(t *testing.T) {
	routes := statusTestRoutes(t)

	createThing := routes["createThing"]
	wantErrors := []int{400, 401}
	if len(createThing.ErrorStatuses) != len(wantErrors) {
		t.Fatalf("createThing error statuses = %v, want %v", createThing.ErrorStatuses, wantErrors)
	}
	for index := range wantErrors {
		if createThing.ErrorStatuses[index] != wantErrors[index] {
			t.Fatalf("createThing error statuses = %v, want %v", createThing.ErrorStatuses, wantErrors)
		}
	}

	healthz := routes["healthz"]
	if healthz.EmitsErrors || len(healthz.ErrorStatuses) != 0 {
		t.Fatalf("healthz must emit no error responses, got %v (emits=%v)", healthz.ErrorStatuses, healthz.EmitsErrors)
	}
}

func TestGenerateEmitsSuccessErrorAndDefaultResponses(t *testing.T) {
	document := Generate([]Route{
		{Method: "POST", Path: "/api/things", OperationID: "createThing", RequiresAuth: true, SuccessStatuses: []int{201}, ErrorStatuses: []int{400, 401}, EmitsErrors: true},
		{Method: "GET", Path: "/healthz", OperationID: "healthz"},
	}, nil, nil)

	createThing := document.Paths["/api/things"]["post"].Responses
	if _, found := createThing["201"]; !found {
		t.Fatalf("createThing responses = %#v, want a 201 entry", createThing)
	}
	if _, found := createThing["default"]; !found {
		t.Fatalf("createThing responses = %#v, want a default error entry", createThing)
	}
	for _, status := range []string{"400", "401", "default"} {
		schema := createThing[status].Content["application/json"].Schema
		if schema.Ref != errorResponseSchemaRef {
			t.Fatalf("createThing %s schema = %#v, want the ErrorResponse reference", status, schema)
		}
	}

	healthz := document.Paths["/healthz"]["get"].Responses
	if len(healthz) != 1 {
		t.Fatalf("healthz responses = %#v, want only the 200 success entry", healthz)
	}
	if _, found := healthz["200"]; !found {
		t.Fatalf("healthz responses = %#v, want a 200 entry", healthz)
	}
}

func TestGenerateEmitsErrorResponseComponentWithEveryCode(t *testing.T) {
	document := Generate([]Route{{Method: "GET", Path: "/healthz", OperationID: "healthz"}}, nil, nil)

	schema, found := document.Components.Schemas[errorResponseSchemaName]
	if !found {
		t.Fatalf("components.schemas is missing %q", errorResponseSchemaName)
	}
	if schema.Type != "object" || schema.Properties["error"].Type != "string" {
		t.Fatalf("ErrorResponse schema = %#v, want an object with a string error", schema)
	}
	code := schema.Properties["code"]
	if code.Type != "string" || len(code.Enum) != 10 {
		t.Fatalf("ErrorResponse code schema = %#v, want a string with the 10-code enum", code)
	}
	wantCodes := map[string]bool{
		"invalid_id": true, "invalid_enum": true, "invalid_state": true,
		"invalid_argument": true, "not_found": true, "permission_denied": true,
		"conflict": true, "unauthenticated": true, "rate_limited": true, "unavailable": true,
	}
	for _, value := range code.Enum {
		if !wantCodes[value] {
			t.Fatalf("unexpected error code %q in %v", value, code.Enum)
		}
		delete(wantCodes, value)
	}
	if len(wantCodes) != 0 {
		t.Fatalf("missing error codes: %v", wantCodes)
	}
}
