package schema

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/core"
)

type ParseResult interface {
	parseResult()
}

type SchemaParsed struct {
	Value Schema
}

type SchemaParseRejected struct {
	Reason core.DomainError
}

func (SchemaParsed) parseResult() {}

func (SchemaParseRejected) parseResult() {}

func ParseSchemaJSON(raw []byte) ParseResult {
	return parseSchemaRaw(raw)
}

type schemaDTO struct {
	Kind        string            `json:"kind"`
	Fields      []fieldDTO        `json:"fields"`
	Item        json.RawMessage   `json:"item"`
	Values      []string          `json:"values"`
	Value       string            `json:"value"`
	Variants    []json.RawMessage `json:"variants"`
	Sensitivity sensitivityDTO    `json:"sensitivity"`
}

type fieldDTO struct {
	Name        string          `json:"name"`
	Presence    string          `json:"presence"`
	Schema      json.RawMessage `json:"schema"`
	Sensitivity sensitivityDTO  `json:"sensitivity"`
}

type sensitivityDTO struct {
	Category  string `json:"category"`
	Retention string `json:"retention"`
	Redaction string `json:"redaction"`
}

func parseSchemaRaw(raw json.RawMessage) ParseResult {
	var dto schemaDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return SchemaParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "schema JSON is invalid")}
	}

	switch dto.Kind {
	case "object":
		return parseObjectSchema(dto)
	case "array":
		return parseArraySchema(dto)
	case "string":
		return parseStringSchema(dto)
	case "integer":
		return SchemaParsed{Value: IntegerSchema{}}
	case "decimal_string":
		return SchemaParsed{Value: DecimalStringSchema{}}
	case "enum":
		return parseEnumSchema(dto)
	case "literal":
		return SchemaParsed{Value: LiteralSchema{Value: dto.Value}}
	case "union":
		return parseUnionSchema(dto)
	case "freeform":
		return SchemaParsed{Value: FreeformSchema{}}
	default:
		return SchemaParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "schema kind is unsupported")}
	}
}

func parseObjectSchema(dto schemaDTO) ParseResult {
	fields := make([]ObjectField, 0, len(dto.Fields))
	for fieldIndex := range dto.Fields {
		field := dto.Fields[fieldIndex]
		nameResult := NewFieldName(field.Name)
		nameAccepted, nameMatched := nameResult.(FieldNameAccepted)
		if !nameMatched {
			rejected := nameResult.(FieldNameRejected)
			return SchemaParseRejected{Reason: rejected.Reason}
		}

		presenceResult := ParseFieldPresence(field.Presence)
		presenceAccepted, presenceMatched := presenceResult.(FieldPresenceAccepted)
		if !presenceMatched {
			rejected := presenceResult.(FieldPresenceRejected)
			return SchemaParseRejected{Reason: rejected.Reason}
		}

		schemaResult := parseSchemaRaw(field.Schema)
		schemaParsed, schemaMatched := schemaResult.(SchemaParsed)
		if !schemaMatched {
			return schemaResult
		}

		sensitivityResult := parseSensitivity(field.Sensitivity)
		sensitivityParsed, sensitivityMatched := sensitivityResult.(SensitivityParsed)
		if !sensitivityMatched {
			rejected := sensitivityResult.(SensitivityRejected)
			return SchemaParseRejected{Reason: rejected.Reason}
		}

		fields = append(fields, ObjectField{
			Name:        nameAccepted.Value,
			Presence:    presenceAccepted.Value,
			Schema:      schemaParsed.Value,
			Sensitivity: sensitivityParsed.Value,
		})
	}

	return SchemaParsed{Value: ObjectSchema{Fields: fields}}
}

func parseArraySchema(dto schemaDTO) ParseResult {
	result := parseSchemaRaw(dto.Item)
	parsed, matched := result.(SchemaParsed)
	if !matched {
		return result
	}
	return SchemaParsed{Value: ArraySchema{Item: parsed.Value}}
}

func parseStringSchema(dto schemaDTO) ParseResult {
	sensitivityResult := parseSensitivity(dto.Sensitivity)
	sensitivityParsed, matched := sensitivityResult.(SensitivityParsed)
	if !matched {
		rejected := sensitivityResult.(SensitivityRejected)
		return SchemaParseRejected{Reason: rejected.Reason}
	}
	return SchemaParsed{Value: StringSchema{Sensitivity: sensitivityParsed.Value}}
}

func parseEnumSchema(dto schemaDTO) ParseResult {
	if len(dto.Values) == 0 {
		return SchemaParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "enum schema requires values")}
	}
	return SchemaParsed{Value: EnumSchema{Values: dto.Values}}
}

func parseUnionSchema(dto schemaDTO) ParseResult {
	if len(dto.Variants) == 0 {
		return SchemaParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "union schema requires variants")}
	}

	variants := make([]Schema, 0, len(dto.Variants))
	for variantIndex := range dto.Variants {
		rawVariant := dto.Variants[variantIndex]
		result := parseSchemaRaw(rawVariant)
		parsed, matched := result.(SchemaParsed)
		if !matched {
			return result
		}
		variants = append(variants, parsed.Value)
	}
	return SchemaParsed{Value: UnionSchema{Variants: variants}}
}

type SensitivityParseResult interface {
	sensitivityParseResult()
}

type SensitivityParsed struct {
	Value Sensitivity
}

type SensitivityRejected struct {
	Reason core.DomainError
}

func (SensitivityParsed) sensitivityParseResult() {}

func (SensitivityRejected) sensitivityParseResult() {}

func parseSensitivity(dto sensitivityDTO) SensitivityParseResult {
	if dto.Category == "" && dto.Retention == "" && dto.Redaction == "" {
		return SensitivityParsed{Value: NotSensitive{}}
	}

	categoryResult := ParseSensitivityCategory(dto.Category)
	categoryAccepted, categoryMatched := categoryResult.(SensitivityCategoryAccepted)
	if !categoryMatched {
		rejected := categoryResult.(SensitivityCategoryRejected)
		return SensitivityRejected{Reason: rejected.Reason}
	}

	retentionResult := ParseRetentionPolicy(dto.Retention)
	retentionAccepted, retentionMatched := retentionResult.(RetentionPolicyAccepted)
	if !retentionMatched {
		rejected := retentionResult.(RetentionPolicyRejected)
		return SensitivityRejected{Reason: rejected.Reason}
	}

	redactionResult := ParseRedactionPolicy(dto.Redaction)
	redactionAccepted, redactionMatched := redactionResult.(RedactionPolicyAccepted)
	if !redactionMatched {
		rejected := redactionResult.(RedactionPolicyRejected)
		return SensitivityRejected{Reason: rejected.Reason}
	}

	return SensitivityParsed{
		Value: Sensitive{
			Category:  categoryAccepted.Value,
			Retention: retentionAccepted.Value,
			Redaction: redactionAccepted.Value,
		},
	}
}
