package schema

import (
	"testing"
)

// schemaSeeds are valid and edge-case schema documents. They give the fuzzer a
// starting corpus that already exercises every schema kind, so mutation reaches
// interesting states quickly instead of rediscovering the grammar.
var schemaSeeds = []string{
	`{"kind":"freeform"}`,
	`{"kind":"string"}`,
	`{"kind":"integer"}`,
	`{"kind":"decimal_string"}`,
	`{"kind":"literal","value":"x"}`,
	`{"kind":"enum","values":["a","b"]}`,
	`{"kind":"array","item":{"kind":"string"}}`,
	`{"kind":"union","variants":[{"kind":"string"},{"kind":"integer"}]}`,
	`{"kind":"object","fields":[{"name":"a","presence":"required","schema":{"kind":"string"}}]}`,
	`{"kind":"object","fields":[{"name":"e","presence":"required","schema":{"kind":"string"},"sensitivity":{"category":"pii","retention":"delete_on_request","redaction":"replace"}}]}`,
	`{"kind":"union","variants":[{"kind":"union","variants":[{"kind":"object","fields":[{"name":"a","presence":"may_omit","schema":{"kind":"integer"}}]}]}]}`,
}

var valueSeeds = []string{
	`null`,
	`"text"`,
	`42`,
	`1.5`,
	`-0`,
	`[]`,
	`{}`,
	`[1,"two",null]`,
	`{"a":"b","c":[1,2,3]}`,
	`{"a":{"b":{"c":"deep"}}}`,
}

// FuzzParseSchemaJSON checks that schema parsing never panics and always
// returns one of the two declared result types regardless of input.
func FuzzParseSchemaJSON(f *testing.F) {
	for _, seed := range schemaSeeds {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		switch ParseSchemaJSON(raw).(type) {
		case SchemaParsed, SchemaParseRejected:
		default:
			t.Fatalf("ParseSchemaJSON returned an unexpected result type for %q", raw)
		}
	})
}

// FuzzParseValueJSON checks that value parsing never panics and that an
// accepted value re-encodes and re-parses to an accepted value (round-trip).
func FuzzParseValueJSON(f *testing.F) {
	for _, seed := range valueSeeds {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		result := ParseValueJSON(raw)
		parsed, ok := result.(ValueParsed)
		if !ok {
			if _, rejected := result.(ValueParseRejected); !rejected {
				t.Fatalf("ParseValueJSON returned an unexpected result type for %q", raw)
			}
			return
		}
		encoded, ok := EncodeValueJSON(parsed.Value).(ValueEncoded)
		if !ok {
			t.Fatalf("a parsed value failed to encode: %q", raw)
		}
		if _, ok := ParseValueJSON([]byte(encoded.Source)).(ValueParsed); !ok {
			t.Fatalf("re-parsing an encoded value was rejected: %q -> %q", raw, encoded.Source)
		}
	})
}

// FuzzValidate drives the full pipeline: parse an untrusted schema, parse an
// untrusted value, then validate, index sensitivity, and redact. None of these
// may panic, and validation must terminate (the fuzzer's per-input timeout
// catches a combinatorial blow-up in union validation).
func FuzzValidate(f *testing.F) {
	for _, schemaSeed := range schemaSeeds {
		for _, valueSeed := range valueSeeds {
			f.Add([]byte(schemaSeed), []byte(valueSeed))
		}
	}
	f.Fuzz(func(t *testing.T, rawSchema []byte, rawValue []byte) {
		schemaResult, ok := ParseSchemaJSON(rawSchema).(SchemaParsed)
		if !ok {
			return
		}
		valueResult, ok := ParseValueJSON(rawValue).(ValueParsed)
		if !ok {
			return
		}
		switch Validate(schemaResult.Value, valueResult.Value).(type) {
		case ValidationAccepted, ValidationRejected:
		default:
			t.Fatalf("Validate returned an unexpected result type")
		}
		BuildSensitiveIndex(schemaResult.Value, valueResult.Value)
		RedactSensitive(schemaResult.Value, valueResult.Value)
	})
}
