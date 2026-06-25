package task

import (
	"net/url"
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

// TaskType is a pre-baked category for a task. It lets developer-oriented work
// (code review, security review, and so on) be templated and filtered, beyond
// the free-text title and description.
type TaskType struct {
	value string
}

var (
	TaskTypeGeneral        = TaskType{value: "general"}
	TaskTypeCodeReview     = TaskType{value: "code_review"}
	TaskTypeSecurityReview = TaskType{value: "security_review"}
	TaskTypeProductReview  = TaskType{value: "product_review"}
	TaskTypeUIUXReview     = TaskType{value: "ui_ux_review"}
	TaskTypeQATesting      = TaskType{value: "qa_testing"}
)

type TaskTypeResult interface {
	taskTypeResult()
}

type TaskTypeAccepted struct {
	Value TaskType
}

type TaskTypeRejected struct {
	Reason core.DomainError
}

func (TaskTypeAccepted) taskTypeResult() {}

func (TaskTypeRejected) taskTypeResult() {}

// ParseTaskType accepts the known developer task types. An empty value defaults
// to "general".
func ParseTaskType(raw string) TaskTypeResult {
	switch raw {
	case "", TaskTypeGeneral.value:
		return TaskTypeAccepted{Value: TaskTypeGeneral}
	case TaskTypeCodeReview.value:
		return TaskTypeAccepted{Value: TaskTypeCodeReview}
	case TaskTypeSecurityReview.value:
		return TaskTypeAccepted{Value: TaskTypeSecurityReview}
	case TaskTypeProductReview.value:
		return TaskTypeAccepted{Value: TaskTypeProductReview}
	case TaskTypeUIUXReview.value:
		return TaskTypeAccepted{Value: TaskTypeUIUXReview}
	case TaskTypeQATesting.value:
		return TaskTypeAccepted{Value: TaskTypeQATesting}
	default:
		return TaskTypeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task type is invalid")}
	}
}

func (taskType TaskType) String() string {
	return taskType.value
}

// ReferenceURL is an optional external reference for a task, such as the pull
// request to review. When present it must be an absolute http(s) URL so the
// instruction points at a specific resource.
type ReferenceURL struct {
	value string
}

type ReferenceURLResult interface {
	referenceURLResult()
}

type ReferenceURLAccepted struct {
	Value ReferenceURL
}

type ReferenceURLRejected struct {
	Reason core.DomainError
}

func (ReferenceURLAccepted) referenceURLResult() {}

func (ReferenceURLRejected) referenceURLResult() {}

func NewReferenceURL(raw string) ReferenceURLResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ReferenceURLAccepted{Value: ReferenceURL{value: ""}}
	}
	if len(trimmed) > 2000 {
		return ReferenceURLRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "reference URL is too long")}
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ReferenceURLRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "reference URL must be an absolute http or https URL")}
	}
	return ReferenceURLAccepted{Value: ReferenceURL{value: trimmed}}
}

func (reference ReferenceURL) String() string {
	return reference.value
}
