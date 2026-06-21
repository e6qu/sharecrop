package task

import (
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

type Title struct {
	value string
}

type TitleResult interface {
	titleResult()
}

type TitleAccepted struct {
	Value Title
}

type TitleRejected struct {
	Reason core.DomainError
}

func (TitleAccepted) titleResult() {}

func (TitleRejected) titleResult() {}

func NewTitle(raw string) TitleResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return TitleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task title is required")}
	}
	if len(trimmed) > 160 {
		return TitleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task title is too long")}
	}
	return TitleAccepted{Value: Title{value: trimmed}}
}

func (title Title) String() string {
	return title.value
}

type Description struct {
	value string
}

type DescriptionResult interface {
	descriptionResult()
}

type DescriptionAccepted struct {
	Value Description
}

type DescriptionRejected struct {
	Reason core.DomainError
}

func (DescriptionAccepted) descriptionResult() {}

func (DescriptionRejected) descriptionResult() {}

func NewDescription(raw string) DescriptionResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DescriptionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task description is required")}
	}
	if len(trimmed) > 8000 {
		return DescriptionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task description is too long")}
	}
	return DescriptionAccepted{Value: Description{value: trimmed}}
}

func (description Description) String() string {
	return description.value
}

type ResponseSchemaSource struct {
	value string
}

type ResponseSchemaSourceResult interface {
	responseSchemaSourceResult()
}

type ResponseSchemaSourceAccepted struct {
	Value ResponseSchemaSource
}

type ResponseSchemaSourceRejected struct {
	Reason core.DomainError
}

func (ResponseSchemaSourceAccepted) responseSchemaSourceResult() {}

func (ResponseSchemaSourceRejected) responseSchemaSourceResult() {}

func NewResponseSchemaSource(raw string) ResponseSchemaSourceResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ResponseSchemaSourceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "response schema is required")}
	}
	return ResponseSchemaSourceAccepted{Value: ResponseSchemaSource{value: trimmed}}
}

func (source ResponseSchemaSource) String() string {
	return source.value
}

type PayloadSource struct {
	value string
}

type PayloadSourceResult interface {
	payloadSourceResult()
}

type PayloadSourceAccepted struct {
	Value PayloadSource
}

type PayloadSourceRejected struct {
	Reason core.DomainError
}

func (PayloadSourceAccepted) payloadSourceResult() {}

func (PayloadSourceRejected) payloadSourceResult() {}

func NewPayloadSource(raw string) PayloadSourceResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return PayloadSourceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "payload JSON is required")}
	}
	return PayloadSourceAccepted{Value: PayloadSource{value: trimmed}}
}

func (source PayloadSource) String() string {
	return source.value
}

type SeriesTitle struct {
	value string
}

type SeriesTitleResult interface {
	seriesTitleResult()
}

type SeriesTitleAccepted struct {
	Value SeriesTitle
}

type SeriesTitleRejected struct {
	Reason core.DomainError
}

func (SeriesTitleAccepted) seriesTitleResult() {}

func (SeriesTitleRejected) seriesTitleResult() {}

func NewSeriesTitle(raw string) SeriesTitleResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return SeriesTitleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task series title is required")}
	}
	if len(trimmed) > 160 {
		return SeriesTitleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task series title is too long")}
	}
	return SeriesTitleAccepted{Value: SeriesTitle{value: trimmed}}
}

func (title SeriesTitle) String() string {
	return title.value
}

type SeriesPosition struct {
	value int
}

type SeriesPositionResult interface {
	seriesPositionResult()
}

type SeriesPositionAccepted struct {
	Value SeriesPosition
}

type SeriesPositionRejected struct {
	Reason core.DomainError
}

func (SeriesPositionAccepted) seriesPositionResult() {}

func (SeriesPositionRejected) seriesPositionResult() {}

func NewSeriesPosition(raw int) SeriesPositionResult {
	if raw < 1 {
		return SeriesPositionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task series position must be positive")}
	}
	return SeriesPositionAccepted{Value: SeriesPosition{value: raw}}
}

func (position SeriesPosition) Int() int {
	return position.value
}
