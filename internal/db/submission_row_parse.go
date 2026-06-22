package db

import (
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
)

type validationErrorDTO struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func parseSubmissionRow(rawSubmissionID string, rawTaskID string, rawUserID string, rawState string, rawResponse string, rawReviewNote string, rawValidationErrors string) submissionRowResult {
	submissionIDResult := core.ParseSubmissionID(rawSubmissionID)
	submissionID, submissionIDMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionIDMatched {
		rejected := submissionIDResult.(core.SubmissionIDRejected)
		return submissionRowRejected{reason: rejected.Reason}
	}

	taskIDResult := core.ParseTaskID(rawTaskID)
	taskID, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		rejected := taskIDResult.(core.TaskIDRejected)
		return submissionRowRejected{reason: rejected.Reason}
	}

	submitterResult := core.ParseUserID(rawUserID)
	submitterID, submitterMatched := submitterResult.(core.UserIDCreated)
	if !submitterMatched {
		rejected := submitterResult.(core.UserIDRejected)
		return submissionRowRejected{reason: rejected.Reason}
	}

	stateResult := submission.ParseState(rawState)
	state, stateMatched := stateResult.(submission.StateParsed)
	if !stateMatched {
		rejected := stateResult.(submission.StateParseRejected)
		return submissionRowRejected{reason: rejected.Reason}
	}

	sourceResult := submission.NewResponseSource(rawResponse)
	source, sourceMatched := sourceResult.(submission.ResponseSourceAccepted)
	if !sourceMatched {
		rejected := sourceResult.(submission.ResponseSourceRejected)
		return submissionRowRejected{reason: rejected.Reason}
	}

	outcomeResult := parseValidationOutcome(rawValidationErrors)
	outcome, outcomeMatched := outcomeResult.(validationOutcomeAccepted)
	if !outcomeMatched {
		rejected := outcomeResult.(validationOutcomeRejected)
		return submissionRowRejected{reason: rejected.reason}
	}

	noteResult := submission.NewStoredReviewNote(rawReviewNote)
	note, noteMatched := noteResult.(submission.ReviewNoteAccepted)
	if !noteMatched {
		rejected := noteResult.(submission.ReviewNoteRejected)
		return submissionRowRejected{reason: rejected.Reason}
	}

	return submissionRowAccepted{value: submission.Submission{
		ID:             submissionID.Value,
		TaskID:         taskID.Value,
		SubmitterID:    submitterID.Value,
		State:          state.Value,
		ResponseSource: source.Value,
		Validation:     outcome.value,
		ReviewNote:     note.Value,
	}}
}

type validationOutcomeResult interface {
	validationOutcomeResult()
}

type validationOutcomeAccepted struct {
	value submission.ValidationOutcome
}

type validationOutcomeRejected struct {
	reason core.DomainError
}

func (validationOutcomeAccepted) validationOutcomeResult() {}

func (validationOutcomeRejected) validationOutcomeResult() {}

func parseValidationOutcome(raw string) validationOutcomeResult {
	var values []validationErrorDTO
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return validationOutcomeRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "submission validation errors are invalid")}
	}
	if len(values) == 0 {
		return validationOutcomeAccepted{value: submission.ValidationPassed{}}
	}
	errors := make([]submission.ValidationError, 0, len(values))
	for valueIndex := range values {
		value := values[valueIndex]
		errors = append(errors, submission.ValidationError{Path: value.Path, Message: value.Message})
	}
	return validationOutcomeAccepted{value: submission.ValidationFailed{Errors: errors}}
}
