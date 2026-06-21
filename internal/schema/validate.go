package schema

import (
	"strconv"
)

type ValidationResult interface {
	validationResult()
}

type ValidationAccepted struct{}

type ValidationRejected struct {
	Errors []ValidationError
}

func (ValidationAccepted) validationResult() {}

func (ValidationRejected) validationResult() {}

type ValidationError struct {
	Path    Path
	Message string
}

func Validate(schema Schema, value Value) ValidationResult {
	errors := validateAt(schema, value, RootPath())
	if len(errors) > 0 {
		return ValidationRejected{Errors: errors}
	}
	return ValidationAccepted{}
}

func validateAt(schema Schema, value Value, path Path) []ValidationError {
	switch typed := schema.(type) {
	case ObjectSchema:
		return validateObject(typed, value, path)
	case ArraySchema:
		return validateArray(typed, value, path)
	case StringSchema:
		return validateString(value, path)
	case IntegerSchema:
		return validateInteger(value, path)
	case DecimalStringSchema:
		return validateDecimalString(value, path)
	case EnumSchema:
		return validateEnum(typed, value, path)
	case LiteralSchema:
		return validateLiteral(typed, value, path)
	case UnionSchema:
		return validateUnion(typed, value, path)
	case FreeformSchema:
		return []ValidationError{}
	default:
		return []ValidationError{{Path: path, Message: "schema is unsupported"}}
	}
}

func validateObject(schema ObjectSchema, value Value, path Path) []ValidationError {
	object, matched := value.(ObjectValue)
	if !matched {
		return []ValidationError{{Path: path, Message: "value must be an object"}}
	}

	errors := make([]ValidationError, 0)
	for fieldIndex := range schema.Fields {
		field := schema.Fields[fieldIndex]
		lookup := findObjectField(object, field.Name)
		found, foundMatched := lookup.(ObjectFieldFound)
		if !foundMatched {
			if field.Presence == FieldRequired {
				errors = append(errors, ValidationError{Path: path.Append(field.Name), Message: "required field is missing"})
			}
			continue
		}
		errors = append(errors, validateAt(field.Schema, found.Value, path.Append(field.Name))...)
	}
	return errors
}

func validateArray(schema ArraySchema, value Value, path Path) []ValidationError {
	array, matched := value.(ArrayValue)
	if !matched {
		return []ValidationError{{Path: path, Message: "value must be an array"}}
	}

	errors := make([]ValidationError, 0)
	for itemIndex := range array.Items {
		item := array.Items[itemIndex]
		errors = append(errors, validateAt(schema.Item, item, path)...)
	}
	return errors
}

func validateString(value Value, path Path) []ValidationError {
	switch value.(type) {
	case StringValue:
		return []ValidationError{}
	default:
		return []ValidationError{{Path: path, Message: "value must be a string"}}
	}
}

func validateInteger(value Value, path Path) []ValidationError {
	switch value.(type) {
	case IntegerValue:
		return []ValidationError{}
	default:
		return []ValidationError{{Path: path, Message: "value must be an integer"}}
	}
}

func validateDecimalString(value Value, path Path) []ValidationError {
	stringValue, matched := value.(StringValue)
	if !matched {
		return []ValidationError{{Path: path, Message: "value must be a decimal string"}}
	}
	parsedDecimal, err := strconv.ParseFloat(stringValue.Value, 64)
	if err != nil {
		return []ValidationError{{Path: path, Message: "value must be a decimal string"}}
	}
	if parsedDecimal == 0 {
		return []ValidationError{}
	}
	return []ValidationError{}
}

func validateEnum(schema EnumSchema, value Value, path Path) []ValidationError {
	stringValue, matched := value.(StringValue)
	if !matched {
		return []ValidationError{{Path: path, Message: "value must be an enum string"}}
	}
	for allowedIndex := range schema.Values {
		allowed := schema.Values[allowedIndex]
		if stringValue.Value == allowed {
			return []ValidationError{}
		}
	}
	return []ValidationError{{Path: path, Message: "value is not an allowed enum member"}}
}

func validateLiteral(schema LiteralSchema, value Value, path Path) []ValidationError {
	stringValue, matched := value.(StringValue)
	if !matched {
		return []ValidationError{{Path: path, Message: "value must be a literal string"}}
	}
	if stringValue.Value != schema.Value {
		return []ValidationError{{Path: path, Message: "value does not match literal"}}
	}
	return []ValidationError{}
}

func validateUnion(schema UnionSchema, value Value, path Path) []ValidationError {
	for variantIndex := range schema.Variants {
		variant := schema.Variants[variantIndex]
		errors := validateAt(variant, value, path)
		if len(errors) == 0 {
			return []ValidationError{}
		}
	}
	return []ValidationError{{Path: path, Message: "value did not match a union variant"}}
}

type ObjectFieldLookupResult interface {
	objectFieldLookupResult()
}

type ObjectFieldFound struct {
	Value Value
}

type ObjectFieldMissing struct{}

func (ObjectFieldFound) objectFieldLookupResult() {}

func (ObjectFieldMissing) objectFieldLookupResult() {}

func findObjectField(object ObjectValue, name FieldName) ObjectFieldLookupResult {
	for fieldIndex := range object.Fields {
		field := object.Fields[fieldIndex]
		if field.Name.String() == name.String() {
			return ObjectFieldFound{Value: field.Value}
		}
	}
	return ObjectFieldMissing{}
}
