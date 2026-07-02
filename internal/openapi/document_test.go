package openapi

import (
	"encoding/json"
	"testing"
)

func TestGenerateMarksPublicRoutesWithEmptySecurity(t *testing.T) {
	document := Generate([]Route{
		{Method: "POST", Path: "/api/auth/login", OperationID: "login", RequiresAuth: false},
		{Method: "POST", Path: "/api/tasks", OperationID: "createTask", RequiresAuth: true},
	})

	login := document.Paths["/api/auth/login"]["post"]
	if login.Security == nil || len(*login.Security) != 0 {
		t.Fatalf("login security = %#v, want pointer to empty slice", login.Security)
	}
	if login.RequestBody == nil {
		t.Fatalf("login request body should be present for POST")
	}

	createTask := document.Paths["/api/tasks"]["post"]
	if createTask.Security != nil {
		t.Fatalf("createTask security = %#v, want nil (inherits document default)", createTask.Security)
	}
}

func TestGenerateOmitsRequestBodyForGetAndDelete(t *testing.T) {
	document := Generate([]Route{
		{Method: "GET", Path: "/api/tasks", OperationID: "listTasks", RequiresAuth: false},
		{Method: "DELETE", Path: "/api/account", OperationID: "deactivateAccount", RequiresAuth: true},
	})

	if document.Paths["/api/tasks"]["get"].RequestBody != nil {
		t.Fatalf("GET should not have a request body")
	}
	if document.Paths["/api/account"]["delete"].RequestBody != nil {
		t.Fatalf("DELETE should not have a request body")
	}
}

func TestGenerateJSONDistinguishesPublicFromDefaultSecurity(t *testing.T) {
	document := Generate([]Route{
		{Method: "POST", Path: "/api/auth/login", OperationID: "login", RequiresAuth: false},
		{Method: "POST", Path: "/api/tasks", OperationID: "createTask", RequiresAuth: true},
	})

	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}
	var decoded Document
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal document: %v", err)
	}

	login := decoded.Paths["/api/auth/login"]["post"]
	if login.Security == nil || len(*login.Security) != 0 {
		t.Fatalf("public route JSON must round-trip an explicit empty security override, got %#v", login.Security)
	}

	createTask := decoded.Paths["/api/tasks"]["post"]
	if createTask.Security != nil {
		t.Fatalf("protected route must omit security so it inherits the document default, got %#v", createTask.Security)
	}
}

func TestGenerateDocumentHasGlobalBearerSecurity(t *testing.T) {
	document := Generate([]Route{{Method: "GET", Path: "/api/tasks", OperationID: "listTasks", RequiresAuth: true}})

	if document.OpenAPI == "" {
		t.Fatalf("openapi version must be set")
	}
	if len(document.Security) != 1 || document.Security[0][bearerSchemeName] == nil {
		t.Fatalf("document security = %#v, want global bearerAuth requirement", document.Security)
	}
	if _, ok := document.Components.SecuritySchemes[bearerSchemeName]; !ok {
		t.Fatalf("components.securitySchemes missing %q", bearerSchemeName)
	}
}
