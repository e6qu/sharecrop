package schema

import "testing"

func TestParseValidateIndexAndRedactSensitiveObject(t *testing.T) {
	schema := parsedSchema(t, []byte(`{
		"kind": "object",
		"fields": [
			{
				"name": "email",
				"presence": "required",
				"schema": {
					"kind": "string"
				},
				"sensitivity": {
					"category": "pii",
					"retention": "delete_on_request",
					"redaction": "replace"
				}
			},
			{
				"name": "age",
				"presence": "may_omit",
				"schema": {
					"kind": "integer"
				}
			}
		]
	}`))
	value := parsedValue(t, []byte(`{"email":"person@example.com","age":42}`))

	validation := Validate(schema, value)
	validationAccepted, matched := validation.(ValidationAccepted)
	if !matched {
		t.Fatalf("validation = %T, want ValidationAccepted", validation)
	}
	if acceptedType := validationAccepted; acceptedType != (ValidationAccepted{}) {
		t.Fatalf("validation accepted marker changed")
	}

	index := BuildSensitiveIndex(schema, value).(SensitiveIndexBuilt)
	if len(index.Fields) != 1 {
		t.Fatalf("sensitive fields = %d, want 1", len(index.Fields))
	}
	if index.Fields[0].Path.String() != "email" {
		t.Fatalf("sensitive path = %q, want email", index.Fields[0].Path.String())
	}

	redacted := RedactSensitive(schema, value).(ValueRedacted)
	object, matched := redacted.Value.(ObjectValue)
	if !matched {
		t.Fatalf("redacted value = %T, want ObjectValue", redacted.Value)
	}
	email := findObjectField(object, acceptedFieldName(t, "email")).(ObjectFieldFound)
	emailString, matched := email.Value.(StringValue)
	if !matched {
		t.Fatalf("email value = %T, want StringValue", email.Value)
	}
	if emailString.Value != "[redacted]" {
		t.Fatalf("email = %q, want [redacted]", emailString.Value)
	}
}

func TestValidateRejectsMissingRequiredField(t *testing.T) {
	schema := parsedSchema(t, []byte(`{
		"kind": "object",
		"fields": [
			{
				"name": "email",
				"presence": "required",
				"schema": {
					"kind": "string"
				}
			}
		]
	}`))
	value := parsedValue(t, []byte(`{}`))

	validation := Validate(schema, value)
	rejected, matched := validation.(ValidationRejected)
	if !matched {
		t.Fatalf("validation = %T, want ValidationRejected", validation)
	}
	if rejected.Errors[0].Path.String() != "email" {
		t.Fatalf("error path = %q, want email", rejected.Errors[0].Path.String())
	}
}

func TestParseRejectsUnsupportedSchemaKind(t *testing.T) {
	result := ParseSchemaJSON([]byte(`{"kind":"number"}`))
	rejected, matched := result.(SchemaParseRejected)
	if !matched {
		t.Fatalf("result = %T, want SchemaParseRejected", result)
	}
	if rejected.Reason.Description() == "" {
		t.Fatalf("rejection reason is empty")
	}
}

func TestFreeformSchemaAcceptsArbitraryValue(t *testing.T) {
	schema := parsedSchema(t, []byte(`{"kind":"freeform"}`))
	value := parsedValue(t, []byte(`{"anything":[1,"two",3.5]}`))

	validation := Validate(schema, value)
	accepted, matched := validation.(ValidationAccepted)
	if !matched {
		t.Fatalf("validation = %T, want ValidationAccepted", validation)
	}
	if accepted != (ValidationAccepted{}) {
		t.Fatalf("validation accepted marker changed")
	}
}

func TestUnionSchemaAcceptsMatchingVariant(t *testing.T) {
	schema := parsedSchema(t, []byte(`{
		"kind": "union",
		"variants": [
			{"kind":"integer"},
			{"kind":"literal","value":"done"}
		]
	}`))
	value := parsedValue(t, []byte(`"done"`))

	validation := Validate(schema, value)
	accepted, matched := validation.(ValidationAccepted)
	if !matched {
		t.Fatalf("validation = %T, want ValidationAccepted", validation)
	}
	if accepted != (ValidationAccepted{}) {
		t.Fatalf("validation accepted marker changed")
	}
}

func TestEnumSchemaRejectsUnexpectedMember(t *testing.T) {
	schema := parsedSchema(t, []byte(`{"kind":"enum","values":["red","blue"]}`))
	value := parsedValue(t, []byte(`"green"`))

	validation := Validate(schema, value)
	rejected, matched := validation.(ValidationRejected)
	if !matched {
		t.Fatalf("validation = %T, want ValidationRejected", validation)
	}
	if len(rejected.Errors) == 0 {
		t.Fatalf("validation rejection did not include errors")
	}
}

func TestRedactionRemoveDropsSensitiveField(t *testing.T) {
	schema := parsedSchema(t, []byte(`{
		"kind": "object",
		"fields": [
			{
				"name": "secret",
				"presence": "required",
				"schema": {
					"kind": "string"
				},
				"sensitivity": {
					"category": "secret",
					"retention": "standard",
					"redaction": "remove"
				}
			}
		]
	}`))
	value := parsedValue(t, []byte(`{"secret":"abc","kept":"visible"}`))

	redacted := RedactSensitive(schema, value).(ValueRedacted)
	object := redacted.Value.(ObjectValue)
	lookup := findObjectField(object, acceptedFieldName(t, "secret"))
	missing, matched := lookup.(ObjectFieldMissing)
	if !matched {
		t.Fatalf("secret lookup = %T, want ObjectFieldMissing", lookup)
	}
	if missing != (ObjectFieldMissing{}) {
		t.Fatalf("missing marker changed")
	}
}

func parsedSchema(t *testing.T, raw []byte) Schema {
	t.Helper()
	result := ParseSchemaJSON(raw)
	parsed, matched := result.(SchemaParsed)
	if !matched {
		t.Fatalf("schema result = %T, want SchemaParsed", result)
	}
	return parsed.Value
}

func parsedValue(t *testing.T, raw []byte) Value {
	t.Helper()
	result := ParseValueJSON(raw)
	parsed, matched := result.(ValueParsed)
	if !matched {
		t.Fatalf("value result = %T, want ValueParsed", result)
	}
	return parsed.Value
}

func acceptedFieldName(t *testing.T, raw string) FieldName {
	t.Helper()
	result := NewFieldName(raw)
	accepted, matched := result.(FieldNameAccepted)
	if !matched {
		t.Fatalf("field name result = %T, want FieldNameAccepted", result)
	}
	return accepted.Value
}
