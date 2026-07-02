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

// Schema models the subset of the JSON Schema object OpenAPI uses that this
// package can derive from a Go DTO struct: a type, its object properties
// (recursively), which of those are required (a field is required unless
// its json tag has `,omitempty`), and array item shape. A field or a whole
// body whose Go type this package cannot confidently classify (see
// FieldKind) gets the zero Schema — an empty `{}`, meaning an unconstrained
// JSON value — rather than a guessed type.
type Schema struct {
	Type       string            `json:"type,omitempty"`
	Properties map[string]Schema `json:"properties,omitempty"`
	Required   []string          `json:"required,omitempty"`
	Items      *Schema           `json:"items,omitempty"`
}

type MediaType struct {
	Schema Schema `json:"schema"`
}

type RequestBody struct {
	Content map[string]MediaType `json:"content"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
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

// Generate builds an OpenAPI document from extracted routes and the DTO
// struct registry extracted alongside them. It cannot fail: every route
// already carries a valid method/path/operationId by the time it reaches
// this function.
func Generate(routes []Route, structs map[string]StructShape) Document {
	paths := map[string]map[string]Operation{}
	for _, route := range routes {
		operation := Operation{
			OperationID: route.OperationID,
			Responses: map[string]Response{
				"default": responseFor(route.ResponseType, structs),
			},
		}
		if !route.RequiresAuth {
			noSecurity := []map[string][]string{}
			operation.Security = &noSecurity
		}
		if requestBodyMethod(route.Method) {
			operation.RequestBody = &RequestBody{
				Content: map[string]MediaType{
					"application/json": {Schema: schemaFor(route.RequestType, structs)},
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
				"Request/response body schemas are derived from the Go DTO struct each " +
				"handler actually decodes/writes where that struct could be resolved; an " +
				"empty schema (`{}`, an unconstrained JSON value) means the handler did not match one " +
				"of the standard decode/write patterns, not that the body is untyped. See " +
				"docs/api_reference.md for prose request/response descriptions per route.",
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

func responseFor(typeName string, structs map[string]StructShape) Response {
	return Response{
		Description: "response",
		Content: map[string]MediaType{
			"application/json": {Schema: schemaFor(typeName, structs)},
		},
	}
}

// schemaFor resolves the named DTO type into a schema, falling back to an
// unconstrained JSON object when typeName is empty or unresolved.
func schemaFor(typeName string, structs map[string]StructShape) Schema {
	if typeName == "" {
		return Schema{Type: "object"}
	}
	return schemaForStruct(typeName, structs, map[string]bool{})
}

// schemaForStruct builds an object schema from a StructShape, recursing into
// struct-typed fields. stack tracks structs currently being expanded on this
// call chain (not a global visited set) so the same struct can appear more
// than once as sibling fields without being mistaken for a cycle; a genuine
// cycle falls back to an unconstrained object rather than recursing forever.
func schemaForStruct(name string, structs map[string]StructShape, stack map[string]bool) Schema {
	shape, found := structs[name]
	if !found || stack[name] {
		return Schema{Type: "object"}
	}
	stack[name] = true
	defer delete(stack, name)

	properties := map[string]Schema{}
	var required []string
	for _, field := range shape.Fields {
		properties[field.JSONName] = schemaForField(field, structs, stack)
		if field.Required {
			required = append(required, field.JSONName)
		}
	}
	return Schema{Type: "object", Properties: properties, Required: required}
}

func schemaForField(field FieldShape, structs map[string]StructShape, stack map[string]bool) Schema {
	switch field.Kind {
	case FieldString:
		return Schema{Type: "string"}
	case FieldInteger:
		return Schema{Type: "integer"}
	case FieldNumber:
		return Schema{Type: "number"}
	case FieldBoolean:
		return Schema{Type: "boolean"}
	case FieldStruct:
		return schemaForStruct(field.StructName, structs, stack)
	case FieldArray:
		itemSchema := schemaForArrayElement(field, structs, stack)
		return Schema{Type: "array", Items: &itemSchema}
	default:
		return Schema{}
	}
}

func schemaForArrayElement(field FieldShape, structs map[string]StructShape, stack map[string]bool) Schema {
	switch field.ElemKind {
	case FieldString:
		return Schema{Type: "string"}
	case FieldInteger:
		return Schema{Type: "integer"}
	case FieldNumber:
		return Schema{Type: "number"}
	case FieldBoolean:
		return Schema{Type: "boolean"}
	case FieldStruct:
		return schemaForStruct(field.StructName, structs, stack)
	default:
		return Schema{}
	}
}
