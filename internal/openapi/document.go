package openapi

import "strings"

const bearerSchemeName = "bearerAuth"

// Info, ServerEntry, SecurityScheme, Components, MediaType, RequestBody,
// Response, Operation, and Document model the subset of the OpenAPI 3.0
// object model this package emits.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type ServerEntry struct {
	URL string `json:"url"`
}

type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme"`
	BearerFormat string `json:"bearerFormat"`
}

type Components struct {
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes"`
}

// Schema is deliberately a fixed placeholder (an unconstrained JSON object)
// rather than a typed per-route schema: this generator proves method, path,
// operationId, and bearer-auth requirements from the actual route table,
// not per-route request/response field shapes. See docs/api_reference.md
// for prose per-route request/response descriptions.
type Schema struct {
	Type string `json:"type"`
}

type MediaType struct {
	Schema Schema `json:"schema"`
}

type RequestBody struct {
	Content map[string]MediaType `json:"content"`
}

type Response struct {
	Description string `json:"description"`
}

// Operation.Security is a pointer so encoding/json can distinguish "not set,
// inherit the document-level default" (nil, omitted) from "explicitly no
// auth" (pointer to an empty slice, encoded as `[]`). A plain non-pointer
// empty slice would be omitted by `omitempty` too, making both cases
// indistinguishable in the generated JSON.
type Operation struct {
	OperationID string                 `json:"operationId"`
	Security    *[]map[string][]string `json:"security,omitempty"`
	RequestBody *RequestBody           `json:"requestBody,omitempty"`
	Responses   map[string]Response    `json:"responses"`
}

type Document struct {
	OpenAPI    string                          `json:"openapi"`
	Info       Info                            `json:"info"`
	Servers    []ServerEntry                   `json:"servers"`
	Security   []map[string][]string           `json:"security"`
	Paths      map[string]map[string]Operation `json:"paths"`
	Components Components                      `json:"components"`
}

// Generate builds an OpenAPI document from extracted routes. It cannot fail:
// every route already carries a valid method/path/operationId by the time it
// reaches this function.
func Generate(routes []Route) Document {
	paths := map[string]map[string]Operation{}
	for _, route := range routes {
		operation := Operation{
			OperationID: route.OperationID,
			Responses: map[string]Response{
				"default": {Description: "response"},
			},
		}
		if !route.RequiresAuth {
			noSecurity := []map[string][]string{}
			operation.Security = &noSecurity
		}
		if requestBodyMethod(route.Method) {
			operation.RequestBody = &RequestBody{
				Content: map[string]MediaType{
					"application/json": {Schema: Schema{Type: "object"}},
				},
			}
		}
		if paths[route.Path] == nil {
			paths[route.Path] = map[string]Operation{}
		}
		paths[route.Path][strings.ToLower(route.Method)] = operation
	}

	return Document{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:   "Sharecrop HTTP API",
			Version: "unversioned",
			Description: "Generated from the route registrations in internal/http/server.go " +
				"(see internal/openapi). Method, path, operationId, and bearer-auth " +
				"requirements are derived from the actual mux table and handler bodies. " +
				"Request/response bodies are generic JSON object placeholders, not typed " +
				"per-route schemas; see docs/api_reference.md for prose request/response " +
				"descriptions per route.",
		},
		Servers:  []ServerEntry{{URL: "/"}},
		Security: []map[string][]string{{bearerSchemeName: {}}},
		Paths:    paths,
		Components: Components{
			SecuritySchemes: map[string]SecurityScheme{
				bearerSchemeName: {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			},
		},
	}
}

func requestBodyMethod(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}
