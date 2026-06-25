package task

import (
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

// SeriesState is the lifecycle state of a first-class series. A draft series is
// visible only to its creator and its member tasks cannot be worked; publishing
// makes it and its open tasks available; closing retires it.
type SeriesState struct {
	value string
}

var (
	SeriesStateDraft     = SeriesState{value: "draft"}
	SeriesStatePublished = SeriesState{value: "published"}
	SeriesStateClosed    = SeriesState{value: "closed"}
)

type SeriesStateResult interface {
	seriesStateResult()
}

type SeriesStateAccepted struct {
	Value SeriesState
}

type SeriesStateRejected struct {
	Reason core.DomainError
}

func (SeriesStateAccepted) seriesStateResult() {}

func (SeriesStateRejected) seriesStateResult() {}

func ParseSeriesState(raw string) SeriesStateResult {
	switch raw {
	case SeriesStateDraft.value:
		return SeriesStateAccepted{Value: SeriesStateDraft}
	case SeriesStatePublished.value:
		return SeriesStateAccepted{Value: SeriesStatePublished}
	case SeriesStateClosed.value:
		return SeriesStateAccepted{Value: SeriesStateClosed}
	default:
		return SeriesStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task series state is invalid")}
	}
}

func (state SeriesState) String() string {
	return state.value
}

// SeriesStateTransition maps a current series state to the next state, or
// rejects the move.
type SeriesStateTransition func(SeriesState) SeriesStateTransitionResult

type SeriesStateTransitionResult interface {
	seriesStateTransitionResult()
}

type SeriesStateTransitionAccepted struct {
	Value SeriesState
}

type SeriesStateTransitionRejected struct {
	Reason core.DomainError
}

func (SeriesStateTransitionAccepted) seriesStateTransitionResult() {}

func (SeriesStateTransitionRejected) seriesStateTransitionResult() {}

func PublishSeriesState(current SeriesState) SeriesStateTransitionResult {
	if current != SeriesStateDraft {
		return SeriesStateTransitionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only a draft series can be published")}
	}
	return SeriesStateTransitionAccepted{Value: SeriesStatePublished}
}

func UnpublishSeriesState(current SeriesState) SeriesStateTransitionResult {
	if current != SeriesStatePublished {
		return SeriesStateTransitionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only a published series can be moved back to draft")}
	}
	return SeriesStateTransitionAccepted{Value: SeriesStateDraft}
}

func CloseSeriesState(current SeriesState) SeriesStateTransitionResult {
	switch current {
	case SeriesStateDraft, SeriesStatePublished:
		return SeriesStateTransitionAccepted{Value: SeriesStateClosed}
	default:
		return SeriesStateTransitionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "series cannot be closed from this state")}
	}
}

func ReopenSeriesState(current SeriesState) SeriesStateTransitionResult {
	if current != SeriesStateClosed {
		return SeriesStateTransitionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only a closed series can be reopened to draft")}
	}
	return SeriesStateTransitionAccepted{Value: SeriesStateDraft}
}

// SeriesDescription is the optional prose describing a series.
type SeriesDescription struct {
	value string
}

type SeriesDescriptionResult interface {
	seriesDescriptionResult()
}

type SeriesDescriptionAccepted struct {
	Value SeriesDescription
}

type SeriesDescriptionRejected struct {
	Reason core.DomainError
}

func (SeriesDescriptionAccepted) seriesDescriptionResult() {}

func (SeriesDescriptionRejected) seriesDescriptionResult() {}

func NewSeriesDescription(raw string) SeriesDescriptionResult {
	if len(raw) > 8000 {
		return SeriesDescriptionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task series description is too long")}
	}
	return SeriesDescriptionAccepted{Value: SeriesDescription{value: raw}}
}

func (description SeriesDescription) String() string {
	return description.value
}

// CommentBody is the text of a series comment.
type CommentBody struct {
	value string
}

type CommentBodyResult interface {
	commentBodyResult()
}

type CommentBodyAccepted struct {
	Value CommentBody
}

type CommentBodyRejected struct {
	Reason core.DomainError
}

func (CommentBodyAccepted) commentBodyResult() {}

func (CommentBodyRejected) commentBodyResult() {}

func NewCommentBody(raw string) CommentBodyResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return CommentBodyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "comment body is required")}
	}
	if len(trimmed) > 4000 {
		return CommentBodyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "comment body is too long")}
	}
	return CommentBodyAccepted{Value: CommentBody{value: trimmed}}
}

func (body CommentBody) String() string {
	return body.value
}

// SeriesComment is one message on a series discussion thread.
type SeriesComment struct {
	ID        core.SeriesCommentID
	SeriesID  core.TaskSeriesID
	AuthorID  core.UserID
	Body      CommentBody
	CreatedAt time.Time
}
