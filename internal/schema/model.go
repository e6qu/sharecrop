package schema

import (
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

type FieldName struct {
	value string
}

type FieldNameResult interface {
	fieldNameResult()
}

type FieldNameAccepted struct {
	Value FieldName
}

type FieldNameRejected struct {
	Reason core.DomainError
}

func (FieldNameAccepted) fieldNameResult() {}

func (FieldNameRejected) fieldNameResult() {}

func NewFieldName(raw string) FieldNameResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return FieldNameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "field name is required")}
	}
	return FieldNameAccepted{Value: FieldName{value: trimmed}}
}

func (name FieldName) String() string {
	return name.value
}

type Path struct {
	Segments []FieldName
}

func RootPath() Path {
	return Path{Segments: []FieldName{}}
}

func (path Path) Append(name FieldName) Path {
	segments := make([]FieldName, 0, len(path.Segments)+1)
	segments = append(segments, path.Segments...)
	segments = append(segments, name)
	return Path{Segments: segments}
}

func (path Path) String() string {
	parts := make([]string, 0, len(path.Segments))
	for segmentIndex := range path.Segments {
		segment := path.Segments[segmentIndex]
		parts = append(parts, segment.String())
	}
	return strings.Join(parts, ".")
}

type Schema interface {
	schema()
}

type ObjectField struct {
	Name        FieldName
	Presence    FieldPresence
	Schema      Schema
	Sensitivity Sensitivity
}

type ObjectSchema struct {
	Fields []ObjectField
}

type ArraySchema struct {
	Item Schema
}

type StringSchema struct {
	Sensitivity Sensitivity
}

type IntegerSchema struct{}

type DecimalStringSchema struct{}

type EnumSchema struct {
	Values []string
}

type LiteralSchema struct {
	Value string
}

type UnionSchema struct {
	Variants []Schema
}

type FreeformSchema struct{}

func (ObjectSchema) schema() {}

func (ArraySchema) schema() {}

func (StringSchema) schema() {}

func (IntegerSchema) schema() {}

func (DecimalStringSchema) schema() {}

func (EnumSchema) schema() {}

func (LiteralSchema) schema() {}

func (UnionSchema) schema() {}

func (FreeformSchema) schema() {}

type FieldPresence struct {
	value string
}

var (
	FieldRequired = FieldPresence{value: "required"}
	FieldMayOmit  = FieldPresence{value: "may_omit"}
)

type FieldPresenceResult interface {
	fieldPresenceResult()
}

type FieldPresenceAccepted struct {
	Value FieldPresence
}

type FieldPresenceRejected struct {
	Reason core.DomainError
}

func (FieldPresenceAccepted) fieldPresenceResult() {}

func (FieldPresenceRejected) fieldPresenceResult() {}

func ParseFieldPresence(raw string) FieldPresenceResult {
	switch raw {
	case FieldRequired.value:
		return FieldPresenceAccepted{Value: FieldRequired}
	case FieldMayOmit.value:
		return FieldPresenceAccepted{Value: FieldMayOmit}
	default:
		return FieldPresenceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "field presence is invalid")}
	}
}

func (presence FieldPresence) String() string {
	return presence.value
}
