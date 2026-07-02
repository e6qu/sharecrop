package openapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseStructShapes(t *testing.T, src string) map[string]StructShape {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return collectStructShapes(map[string]*ast.File{"x.go": file})
}

func TestCollectStructShapesClassifiesPrimitivesAndArrays(t *testing.T) {
	shapes := parseStructShapes(t, `package httpserver

type taskResponse struct {
	Title       string   `+"`json:\"title\"`"+`
	CreditAmount int64   `+"`json:\"credit_amount,omitempty\"`"+`
	Tags        []string `+"`json:\"tags\"`"+`
	Owner       ownerResponse `+"`json:\"owner\"`"+`
	Attachments []attachmentResponse `+"`json:\"attachments\"`"+`
	Internal    string   `+"`json:\"-\"`"+`
	NoTag       string
}

type ownerResponse struct {
	Kind string `+"`json:\"kind\"`"+`
}

type attachmentResponse struct {
	Name string `+"`json:\"name\"`"+`
}
`)

	shape, ok := shapes["taskResponse"]
	if !ok {
		t.Fatalf("taskResponse not found in %#v", shapes)
	}
	byName := map[string]FieldShape{}
	for _, field := range shape.Fields {
		byName[field.JSONName] = field
	}

	if len(shape.Fields) != 5 {
		t.Fatalf("fields = %#v, want 5 (Internal and NoTag excluded)", shape.Fields)
	}
	if _, excluded := byName["Internal"]; excluded {
		t.Fatalf("json:\"-\" field must be excluded")
	}

	title := byName["title"]
	if title.Kind != FieldString || !title.Required {
		t.Fatalf("title = %#v, want required string", title)
	}

	credit := byName["credit_amount"]
	if credit.Kind != FieldInteger || credit.Required {
		t.Fatalf("credit_amount = %#v, want optional integer (omitempty)", credit)
	}

	tags := byName["tags"]
	if tags.Kind != FieldArray || tags.ElemKind != FieldString {
		t.Fatalf("tags = %#v, want array of string", tags)
	}

	owner := byName["owner"]
	if owner.Kind != FieldStruct || owner.StructName != "ownerResponse" {
		t.Fatalf("owner = %#v, want struct ownerResponse", owner)
	}

	attachments := byName["attachments"]
	if attachments.Kind != FieldArray || attachments.ElemKind != FieldStruct || attachments.StructName != "attachmentResponse" {
		t.Fatalf("attachments = %#v, want array of struct attachmentResponse", attachments)
	}
}

func TestCollectStructShapesHandlesPointerFields(t *testing.T) {
	shapes := parseStructShapes(t, `package httpserver

type withPointer struct {
	Note *string `+"`json:\"note,omitempty\"`"+`
}
`)
	shape := shapes["withPointer"]
	if len(shape.Fields) != 1 || shape.Fields[0].Kind != FieldString {
		t.Fatalf("fields = %#v, want a single optional string field (pointer unwrapped)", shape.Fields)
	}
}

func TestCollectStructShapesUsesGoNameWhenTagOmitsName(t *testing.T) {
	shapes := parseStructShapes(t, `package httpserver

type bareTag struct {
	Status string `+"`json:\",omitempty\"`"+`
}
`)
	shape := shapes["bareTag"]
	if len(shape.Fields) != 1 || shape.Fields[0].JSONName != "Status" {
		t.Fatalf("fields = %#v, want JSONName fallback to the Go field name", shape.Fields)
	}
}
