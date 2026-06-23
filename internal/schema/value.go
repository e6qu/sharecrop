package schema

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/e6qu/sharecrop/internal/core"
)

type Value interface {
	value()
}

type NullValue struct{}

type StringValue struct {
	Value string
}

type IntegerValue struct {
	Value int64
}

type DecimalStringValue struct {
	Value string
}

type ObjectFieldValue struct {
	Name  FieldName
	Value Value
}

type ObjectValue struct {
	Fields []ObjectFieldValue
}

type ArrayValue struct {
	Items []Value
}

func (NullValue) value() {}

func (StringValue) value() {}

func (IntegerValue) value() {}

func (DecimalStringValue) value() {}

func (ObjectValue) value() {}

func (ArrayValue) value() {}

type ValueParseResult interface {
	valueParseResult()
}

type ValueParsed struct {
	Value Value
}

type ValueParseRejected struct {
	Reason core.DomainError
}

func (ValueParsed) valueParseResult() {}

func (ValueParseRejected) valueParseResult() {}

func ParseValueJSON(raw []byte) ValueParseResult {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	valueResult := parseDecoderValue(decoder, 0)
	valueParsed, matched := valueResult.(ValueParsed)
	if !matched {
		return valueResult
	}

	if decoder.More() {
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "response JSON has trailing values")}
	}

	return valueParsed
}

func parseDecoderValue(decoder *json.Decoder, depth int) ValueParseResult {
	if depth > maxNestingDepth {
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "response JSON nesting is too deep")}
	}

	token, err := decoder.Token()
	if err != nil {
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "response JSON is invalid")}
	}

	switch typed := token.(type) {
	case json.Delim:
		return parseDelimitedValue(decoder, typed, depth)
	case string:
		return ValueParsed{Value: StringValue{Value: typed}}
	case json.Number:
		return parseJSONNumber(typed)
	case nil:
		return ValueParsed{Value: NullValue{}}
	default:
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "response JSON token is unsupported")}
	}
}

func parseDelimitedValue(decoder *json.Decoder, delimiter json.Delim, depth int) ValueParseResult {
	switch delimiter {
	case '{':
		return parseObjectValue(decoder, depth)
	case '[':
		return parseArrayValue(decoder, depth)
	default:
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "response JSON delimiter is unsupported")}
	}
}

func parseObjectValue(decoder *json.Decoder, depth int) ValueParseResult {
	fields := make([]ObjectFieldValue, 0)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "object field name is invalid")}
		}
		rawName, matched := token.(string)
		if !matched {
			return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "object field name is invalid")}
		}
		nameResult := NewFieldName(rawName)
		nameAccepted, nameMatched := nameResult.(FieldNameAccepted)
		if !nameMatched {
			rejected := nameResult.(FieldNameRejected)
			return ValueParseRejected{Reason: rejected.Reason}
		}

		valueResult := parseDecoderValue(decoder, depth+1)
		valueParsed, valueMatched := valueResult.(ValueParsed)
		if !valueMatched {
			return valueResult
		}
		fields = append(fields, ObjectFieldValue{Name: nameAccepted.Value, Value: valueParsed.Value})
	}

	endToken, err := decoder.Token()
	if err != nil {
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "object end is invalid")}
	}
	if endToken != json.Delim('}') {
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "object end is invalid")}
	}

	return ValueParsed{Value: ObjectValue{Fields: fields}}
}

func parseArrayValue(decoder *json.Decoder, depth int) ValueParseResult {
	items := make([]Value, 0)
	for decoder.More() {
		valueResult := parseDecoderValue(decoder, depth+1)
		valueParsed, matched := valueResult.(ValueParsed)
		if !matched {
			return valueResult
		}
		items = append(items, valueParsed.Value)
	}

	endToken, err := decoder.Token()
	if err != nil {
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "array end is invalid")}
	}
	if endToken != json.Delim(']') {
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "array end is invalid")}
	}

	return ValueParsed{Value: ArrayValue{Items: items}}
}

func parseJSONNumber(number json.Number) ValueParseResult {
	if integer, err := number.Int64(); err == nil {
		return ValueParsed{Value: IntegerValue{Value: integer}}
	}

	parsedDecimal, err := strconv.ParseFloat(number.String(), 64)
	if err != nil {
		return ValueParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "number is invalid")}
	}
	if parsedDecimal == 0 {
		return ValueParsed{Value: DecimalStringValue{Value: number.String()}}
	}
	return ValueParsed{Value: DecimalStringValue{Value: number.String()}}
}
