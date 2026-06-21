package schema

import (
	"strconv"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

type EncodeValueResult interface {
	encodeValueResult()
}

type ValueEncoded struct {
	Source string
}

type ValueEncodeRejected struct {
	Reason core.DomainError
}

func (ValueEncoded) encodeValueResult() {}

func (ValueEncodeRejected) encodeValueResult() {}

func EncodeValueJSON(value Value) EncodeValueResult {
	var builder strings.Builder
	result := writeValueJSON(&builder, value)
	if rejected, matched := result.(ValueEncodeRejected); matched {
		return rejected
	}
	return ValueEncoded{Source: builder.String()}
}

func writeValueJSON(builder *strings.Builder, value Value) EncodeValueResult {
	switch typed := value.(type) {
	case NullValue:
		builder.WriteString("null")
	case StringValue:
		writeJSONString(builder, typed.Value)
	case IntegerValue:
		builder.WriteString(strconv.FormatInt(typed.Value, 10))
	case DecimalStringValue:
		builder.WriteString(typed.Value)
	case ObjectValue:
		return writeObjectJSON(builder, typed)
	case ArrayValue:
		return writeArrayJSON(builder, typed)
	default:
		return ValueEncodeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "schema value is unsupported")}
	}
	return ValueEncoded{Source: builder.String()}
}

func writeObjectJSON(builder *strings.Builder, value ObjectValue) EncodeValueResult {
	builder.WriteString("{")
	for fieldIndex := range value.Fields {
		if fieldIndex > 0 {
			builder.WriteString(",")
		}
		field := value.Fields[fieldIndex]
		writeJSONString(builder, field.Name.String())
		builder.WriteString(":")
		result := writeValueJSON(builder, field.Value)
		if rejected, matched := result.(ValueEncodeRejected); matched {
			return rejected
		}
	}
	builder.WriteString("}")
	return ValueEncoded{Source: builder.String()}
}

func writeArrayJSON(builder *strings.Builder, value ArrayValue) EncodeValueResult {
	builder.WriteString("[")
	for itemIndex := range value.Items {
		if itemIndex > 0 {
			builder.WriteString(",")
		}
		result := writeValueJSON(builder, value.Items[itemIndex])
		if rejected, matched := result.(ValueEncodeRejected); matched {
			return rejected
		}
	}
	builder.WriteString("]")
	return ValueEncoded{Source: builder.String()}
}

func writeJSONString(builder *strings.Builder, value string) {
	builder.WriteString(strconv.Quote(value))
}
