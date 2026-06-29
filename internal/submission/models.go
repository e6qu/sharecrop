package submission

import "github.com/e6qu/sharecrop/internal/core"

type Submission struct {
	ID              core.SubmissionID
	TaskID          core.TaskID
	SubmitterID     core.UserID
	State           State
	ResponseSource  ResponseSource
	Validation      ValidationOutcome
	SensitiveFields []SensitiveField
	ReviewNote      ReviewNote
}

type Receipt struct {
	ID           core.SubmissionReceiptTokenID
	SubmissionID core.SubmissionID
}

type State struct {
	value string
}

var (
	StateSubmitted        = State{value: "submitted"}
	StateInvalid          = State{value: "invalid"}
	StateAccepted         = State{value: "accepted"}
	StateRejected         = State{value: "rejected"}
	StateChangesRequested = State{value: "changes_requested"}
)

type StateResult interface {
	stateResult()
}

type StateParsed struct {
	Value State
}

type StateParseRejected struct {
	Reason core.DomainError
}

func (StateParsed) stateResult() {}

func (StateParseRejected) stateResult() {}

func ParseState(raw string) StateResult {
	switch raw {
	case StateSubmitted.value:
		return StateParsed{Value: StateSubmitted}
	case StateInvalid.value:
		return StateParsed{Value: StateInvalid}
	case StateAccepted.value:
		return StateParsed{Value: StateAccepted}
	case StateRejected.value:
		return StateParsed{Value: StateRejected}
	case StateChangesRequested.value:
		return StateParsed{Value: StateChangesRequested}
	default:
		return StateParseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "submission state is invalid")}
	}
}

func (state State) String() string {
	return state.value
}

type ValidationOutcome interface {
	validationOutcome()
}

type ValidationPassed struct{}

type ValidationFailed struct {
	Errors []ValidationError
}

func (ValidationPassed) validationOutcome() {}

func (ValidationFailed) validationOutcome() {}

type ValidationError struct {
	Path    string
	Message string
}

func StateForValidation(outcome ValidationOutcome) State {
	if _, matched := outcome.(ValidationPassed); matched {
		return StateSubmitted
	}
	return StateInvalid
}

type SensitiveField struct {
	Path       string
	Category   string
	Retention  string
	Redaction  string
	State      string
	RedactedAt string
}

type ReviewNote struct {
	value string
}

type ReviewNoteResult interface {
	reviewNoteResult()
}

type ReviewNoteAccepted struct {
	Value ReviewNote
}

type ReviewNoteRejected struct {
	Reason core.DomainError
}

func (ReviewNoteAccepted) reviewNoteResult() {}

func (ReviewNoteRejected) reviewNoteResult() {}

func NewRequiredReviewNote(raw string) ReviewNoteResult {
	if raw == "" {
		return ReviewNoteRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "review note is required")}
	}
	if len(raw) > 2000 {
		return ReviewNoteRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "review note is too long")}
	}
	return ReviewNoteAccepted{Value: ReviewNote{value: raw}}
}

func NewStoredReviewNote(raw string) ReviewNoteResult {
	if len(raw) > 2000 {
		return ReviewNoteRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "review note is too long")}
	}
	return ReviewNoteAccepted{Value: ReviewNote{value: raw}}
}

func EmptyReviewNote() ReviewNote {
	return ReviewNote{value: ""}
}

func (note ReviewNote) String() string {
	return note.value
}
