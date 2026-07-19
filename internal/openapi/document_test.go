package openapi

import (
	"encoding/json"
	"testing"
)

func TestGenerateMarksPublicRoutesWithEmptySecurity(t *testing.T) {
	document := Generate([]Route{
		{Method: "POST", Path: "/api/auth/login", OperationID: "login", RequiresAuth: false},
		{Method: "POST", Path: "/api/tasks", OperationID: "createTask", RequiresAuth: true},
	}, nil)

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
	}, nil)

	if document.Paths["/api/tasks"]["get"].RequestBody != nil {
		t.Fatalf("GET should not have a request body")
	}
	if document.Paths["/api/account"]["delete"].RequestBody != nil {
		t.Fatalf("DELETE should not have a request body")
	}
}

func TestGenerateUsesExtractedFormMediaType(t *testing.T) {
	document := Generate([]Route{{Method: "POST", Path: "/logout", OperationID: "logout", RequestMediaType: "application/x-www-form-urlencoded"}}, nil)
	content := document.Paths["/logout"]["post"].RequestBody.Content
	if _, ok := content["application/x-www-form-urlencoded"]; !ok {
		t.Fatalf("request content = %#v", content)
	}
	if _, ok := content["application/json"]; ok {
		t.Fatalf("form route was documented as JSON: %#v", content)
	}
}

func TestGenerateUsesExtractedResponseMediaType(t *testing.T) {
	document := Generate([]Route{{Method: "GET", Path: "/signed-out", OperationID: "signedOut", ResponseMediaType: "text/html"}}, nil)
	content := document.Paths["/signed-out"]["get"].Responses["default"].Content
	media, ok := content["text/html"]
	if !ok || media.Schema.Type != "string" {
		t.Fatalf("response content = %#v", content)
	}
	if _, ok := content["application/json"]; ok {
		t.Fatalf("HTML route was documented as JSON: %#v", content)
	}
}

func TestGenerateJSONDistinguishesPublicFromDefaultSecurity(t *testing.T) {
	document := Generate([]Route{
		{Method: "POST", Path: "/api/auth/login", OperationID: "login", RequiresAuth: false},
		{Method: "POST", Path: "/api/tasks", OperationID: "createTask", RequiresAuth: true},
	}, nil)

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
	document := Generate([]Route{{Method: "GET", Path: "/api/tasks", OperationID: "listTasks", RequiresAuth: true}}, nil)

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

func TestGenerateUsesGenericObjectWhenTypeUnresolvedOrUnknown(t *testing.T) {
	document := Generate([]Route{
		{Method: "POST", Path: "/api/tasks", OperationID: "createTask", RequiresAuth: true, RequestType: "", ResponseType: "notInRegistry"},
	}, map[string]StructShape{})

	operation := document.Paths["/api/tasks"]["post"]
	requestSchema := operation.RequestBody.Content["application/json"].Schema
	if requestSchema.Type != "object" || requestSchema.Properties != nil {
		t.Fatalf("unresolved request schema = %#v, want a bare object placeholder", requestSchema)
	}

	responseSchema := operation.Responses["default"].Content["application/json"].Schema
	if responseSchema.Type != "object" || responseSchema.Properties != nil {
		t.Fatalf("unresolved response schema = %#v, want a bare object placeholder", responseSchema)
	}
}

func TestGenerateBuildsTypedSchemaWithRequiredFieldsAndNestedStruct(t *testing.T) {
	structs := map[string]StructShape{
		"taskResponse": {Fields: []FieldShape{
			{JSONName: "title", Required: true, Kind: FieldString},
			{JSONName: "credit_amount", Required: false, Kind: FieldInteger},
			{JSONName: "owner", Required: true, Kind: FieldStruct, StructName: "ownerResponse"},
			{JSONName: "tags", Required: true, Kind: FieldArray, ElemKind: FieldString},
		}},
		"ownerResponse": {Fields: []FieldShape{
			{JSONName: "kind", Required: true, Kind: FieldString},
		}},
	}

	document := Generate([]Route{
		{Method: "GET", Path: "/api/tasks/{task_id}", OperationID: "getTask", RequiresAuth: true, ResponseType: "taskResponse"},
	}, structs)

	schema := document.Paths["/api/tasks/{task_id}"]["get"].Responses["default"].Content["application/json"].Schema
	if schema.Type != "object" {
		t.Fatalf("schema type = %q, want object", schema.Type)
	}
	if len(schema.Properties) != 4 {
		t.Fatalf("properties = %#v, want 4 fields", schema.Properties)
	}
	wantRequired := map[string]bool{"title": true, "owner": true, "tags": true}
	for _, name := range schema.Required {
		if !wantRequired[name] {
			t.Fatalf("unexpected required field %q in %#v", name, schema.Required)
		}
		delete(wantRequired, name)
	}
	if len(wantRequired) != 0 {
		t.Fatalf("missing required fields: %#v", wantRequired)
	}
	if creditAmount := schema.Properties["credit_amount"]; creditAmount.Type != "integer" {
		t.Fatalf("credit_amount = %#v, want integer", creditAmount)
	}
	if stringSliceContains(schema.Required, "credit_amount") {
		t.Fatalf("credit_amount must not be required (omitempty), required = %#v", schema.Required)
	}

	owner := schema.Properties["owner"]
	if owner.Type != "object" || owner.Properties["kind"].Type != "string" {
		t.Fatalf("owner = %#v, want nested object with a string kind field", owner)
	}

	tags := schema.Properties["tags"]
	if tags.Type != "array" || tags.Items == nil || tags.Items.Type != "string" {
		t.Fatalf("tags = %#v, want array of string", tags)
	}
}

func TestGenerateSchemaForStructBreaksCycles(t *testing.T) {
	structs := map[string]StructShape{
		"nodeResponse": {Fields: []FieldShape{
			{JSONName: "child", Required: true, Kind: FieldStruct, StructName: "nodeResponse"},
		}},
	}

	schema := schemaForStruct("nodeResponse", structs, map[string]bool{})
	child := schema.Properties["child"]
	if child.Type != "object" || child.Properties != nil {
		t.Fatalf("cyclic field = %#v, want a bare object placeholder instead of infinite recursion", child)
	}
}

func TestGenerateSchemaForStructAllowsSameStructAsSiblingFields(t *testing.T) {
	structs := map[string]StructShape{
		"pairResponse": {Fields: []FieldShape{
			{JSONName: "left", Required: true, Kind: FieldStruct, StructName: "leafResponse"},
			{JSONName: "right", Required: true, Kind: FieldStruct, StructName: "leafResponse"},
		}},
		"leafResponse": {Fields: []FieldShape{
			{JSONName: "value", Required: true, Kind: FieldString},
		}},
	}

	schema := schemaForStruct("pairResponse", structs, map[string]bool{})
	left := schema.Properties["left"]
	right := schema.Properties["right"]
	if left.Properties["value"].Type != "string" || right.Properties["value"].Type != "string" {
		t.Fatalf("left = %#v, right = %#v, want both fully expanded (not treated as a cycle)", left, right)
	}
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
