package schema

type SensitiveField struct {
	Path        Path
	Sensitivity Sensitive
}

type SensitiveIndexResult interface {
	sensitiveIndexResult()
}

type SensitiveIndexBuilt struct {
	Fields []SensitiveField
}

func (SensitiveIndexBuilt) sensitiveIndexResult() {}

func BuildSensitiveIndex(schema Schema, value Value) SensitiveIndexResult {
	fields := collectSensitive(schema, value, RootPath())
	return SensitiveIndexBuilt{Fields: fields}
}

func collectSensitive(schema Schema, value Value, path Path) []SensitiveField {
	switch typed := schema.(type) {
	case ObjectSchema:
		object, matched := value.(ObjectValue)
		if !matched {
			return []SensitiveField{}
		}
		fields := make([]SensitiveField, 0)
		for schemaFieldIndex := range typed.Fields {
			schemaField := typed.Fields[schemaFieldIndex]
			lookup := findObjectField(object, schemaField.Name)
			found, foundMatched := lookup.(ObjectFieldFound)
			if !foundMatched {
				continue
			}
			fieldPath := path.Append(schemaField.Name)
			if sensitive, matched := schemaField.Sensitivity.(Sensitive); matched {
				fields = append(fields, SensitiveField{Path: fieldPath, Sensitivity: sensitive})
			}
			fields = append(fields, collectSensitive(schemaField.Schema, found.Value, fieldPath)...)
		}
		return fields
	case ArraySchema:
		array, matched := value.(ArrayValue)
		if !matched {
			return []SensitiveField{}
		}
		fields := make([]SensitiveField, 0)
		for itemIndex := range array.Items {
			item := array.Items[itemIndex]
			fields = append(fields, collectSensitive(typed.Item, item, path)...)
		}
		return fields
	case StringSchema:
		if sensitive, matched := typed.Sensitivity.(Sensitive); matched {
			return []SensitiveField{{Path: path, Sensitivity: sensitive}}
		}
		return []SensitiveField{}
	default:
		return []SensitiveField{}
	}
}

type RedactionResult interface {
	redactionResult()
}

type ValueRedacted struct {
	Value Value
}

func (ValueRedacted) redactionResult() {}

func RedactSensitive(schema Schema, value Value) RedactionResult {
	return ValueRedacted{Value: redactAt(schema, value)}
}

func redactAt(schema Schema, value Value) Value {
	switch typed := schema.(type) {
	case ObjectSchema:
		object, matched := value.(ObjectValue)
		if !matched {
			return value
		}
		fields := make([]ObjectFieldValue, 0, len(object.Fields))
		for valueFieldIndex := range object.Fields {
			valueField := object.Fields[valueFieldIndex]
			schemaLookup := findSchemaField(typed, valueField.Name)
			schemaFound, schemaMatched := schemaLookup.(SchemaFieldFound)
			if !schemaMatched {
				fields = append(fields, valueField)
				continue
			}
			if sensitive, matched := schemaFound.Field.Sensitivity.(Sensitive); matched {
				if sensitive.Redaction == RedactionRemove {
					continue
				}
				fields = append(fields, ObjectFieldValue{Name: valueField.Name, Value: StringValue{Value: "[redacted]"}})
				continue
			}
			fields = append(fields, ObjectFieldValue{Name: valueField.Name, Value: redactAt(schemaFound.Field.Schema, valueField.Value)})
		}
		return ObjectValue{Fields: fields}
	case ArraySchema:
		array, matched := value.(ArrayValue)
		if !matched {
			return value
		}
		items := make([]Value, 0, len(array.Items))
		for itemIndex := range array.Items {
			item := array.Items[itemIndex]
			items = append(items, redactAt(typed.Item, item))
		}
		return ArrayValue{Items: items}
	case StringSchema:
		if sensitive, matched := typed.Sensitivity.(Sensitive); matched {
			if sensitive.Redaction == RedactionReplace {
				return StringValue{Value: "[redacted]"}
			}
		}
		return value
	default:
		return value
	}
}

type SchemaFieldLookupResult interface {
	schemaFieldLookupResult()
}

type SchemaFieldFound struct {
	Field ObjectField
}

type SchemaFieldMissing struct{}

func (SchemaFieldFound) schemaFieldLookupResult() {}

func (SchemaFieldMissing) schemaFieldLookupResult() {}

func findSchemaField(schema ObjectSchema, name FieldName) SchemaFieldLookupResult {
	for fieldIndex := range schema.Fields {
		field := schema.Fields[fieldIndex]
		if field.Name.String() == name.String() {
			return SchemaFieldFound{Field: field}
		}
	}
	return SchemaFieldMissing{}
}
