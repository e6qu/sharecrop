package submission

import "github.com/e6qu/sharecrop/internal/core"

type Submission struct {
	ID             core.SubmissionID
	TaskID         core.TaskID
	Submitter      Submitter
	State          State
	ResponseSource ResponseSource
	Validation     ValidationOutcome
}

type Receipt struct {
	ID           core.SubmissionReceiptTokenID
	SubmissionID core.SubmissionID
}

type Submitter interface {
	submitter()
}

type AuthenticatedSubmitter struct {
	UserID core.UserID
}

type AnonymousSubmitter struct {
	WalletAddress WalletAddress
}

func (AuthenticatedSubmitter) submitter() {}

func (AnonymousSubmitter) submitter() {}

type SubmitterKind struct {
	value string
}

var (
	SubmitterKindAuthenticated = SubmitterKind{value: "authenticated"}
	SubmitterKindAnonymous     = SubmitterKind{value: "anonymous"}
)

func (kind SubmitterKind) String() string {
	return kind.value
}

type State struct {
	value string
}

var (
	StateSubmitted = State{value: "submitted"}
	StateInvalid   = State{value: "invalid"}
	StateAccepted  = State{value: "accepted"}
	StateRejected  = State{value: "rejected"}
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
	Path      string
	Category  string
	Retention string
	Redaction string
}
