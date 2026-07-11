package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

func (server Server) createAuthenticatedSubmission(w http.ResponseWriter, r *http.Request) {
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		rejected := taskIDResult.(taskIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	actorResult := server.requireWorkerSubject(r, agent.ScopeSubmissionsWrite, taskIDAccepted.value)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	requestResult := decodeAuthenticatedSubmissionRequest(r, actor.subject, taskIDAccepted.value)
	requestAccepted, requestMatched := requestResult.(submissionRequestAccepted)
	if !requestMatched {
		rejected := requestResult.(submissionRequestRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	server.submitResponse(w, r, requestAccepted.command)
}
func (server Server) submitResponse(w http.ResponseWriter, r *http.Request, command submission.SubmitCommand) {
	taskResult := server.taskService.Get(r.Context(), auth.UserSubject{ID: command.SubmitterID}, command.TaskID)
	taskFound, taskMatched := taskResult.(task.TaskGot)
	if !taskMatched {
		rejected := taskResult.(task.GetRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	result := server.submissionService.Submit(r.Context(), command)
	created, matched := result.(submission.SubmissionCreated)
	if !matched {
		rejected := result.(submission.SubmitRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	if !server.notify(w, r.Context(), taskFound.Value.CreatedBy, command.SubmitterID, notification.KindSubmissionCreated, notificationSubjectForSubmission(created.Value.ID), taskNotificationMetadata(command.TaskID)) {
		return
	}

	writeSubmissionCreatedResponse(w, http.StatusCreated, submissionCreatedResponse{
		Submission:   submissionToResponse(created.Value),
		ReceiptToken: created.ReceiptToken.String(),
	})
}
func (server Server) findSubmissionReceipt(w http.ResponseWriter, r *http.Request) {
	if !server.allowByIP(w, r) {
		return
	}
	tokenResult := submission.ParseReceiptTokenPlain(r.PathValue("receipt_token"))
	tokenAccepted, tokenMatched := tokenResult.(submission.ReceiptTokenPlainAccepted)
	if !tokenMatched {
		rejected := tokenResult.(submission.ReceiptTokenPlainRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.submissionService.FindByReceipt(r.Context(), tokenAccepted.Value)
	found, matched := result.(submission.ReceiptStatusFound)
	if !matched {
		rejected := result.(submission.ReceiptStatusRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeSubmissionResponse(w, http.StatusOK, submissionToResponse(found.Value))
}
func (server Server) listTaskSubmissions(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		rejected := taskIDResult.(taskIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.submissionService.ListForTask(r.Context(), actor.subject, taskIDAccepted.value, page)
	listed, matched := result.(submission.SubmissionsListed)
	if !matched {
		rejected := result.(submission.ListRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	response := submissionsResponse{Submissions: make([]submissionResponse, 0, len(listed.Values))}
	for _, value := range listed.Values {
		if !server.recordSensitiveFieldAccess(w, r, actor.subject.ID, value) {
			return
		}
		response.Submissions = append(response.Submissions, submissionToResponse(value))
	}
	writeSubmissionsResponse(w, http.StatusOK, response)
}

func (server Server) recordSensitiveFieldAccess(w http.ResponseWriter, r *http.Request, actor core.UserID, value submission.Submission) bool {
	if len(value.SensitiveFields) == 0 {
		return true
	}
	result := server.privacyService.RecordSensitiveFieldAccess(r.Context(), actor, value)
	if rejected, matched := result.(PrivacyRequestMutationRejected); matched {
		writeDomainError(w, rejected.Reason)
		return false
	}
	return true
}
func decodeAuthenticatedSubmissionRequest(r *http.Request, actor auth.UserSubject, taskID core.TaskID) submissionRequestResult {
	var request submissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return submissionRequestRejected{reason: "request body is invalid"}
	}

	sourceResult := submission.NewResponseSource(request.ResponseJSON)
	source, sourceMatched := sourceResult.(submission.ResponseSourceAccepted)
	if !sourceMatched {
		rejected := sourceResult.(submission.ResponseSourceRejected)
		return submissionRequestRejected{reason: rejected.Reason.Description()}
	}

	attachmentsResult := attachmentsFromRequest(request.Attachments)
	attachmentsAccepted, attachmentsMatched := attachmentsResult.(attachmentsRequestAccepted)
	if !attachmentsMatched {
		return submissionRequestRejected{reason: attachmentsResult.(attachmentsRequestRejected).reason}
	}

	return submissionRequestAccepted{command: submission.SubmitCommand{
		TaskID:         taskID,
		SubmitterID:    actor.ID,
		ResponseSource: source.Value,
		Attachments:    attachmentsAccepted.values,
	}}
}
func submissionToResponse(value submission.Submission) submissionResponse {
	errors := submissionValidationErrorsToResponse(value.Validation)
	return submissionResponse{
		ID:               value.ID.String(),
		TaskID:           value.TaskID.String(),
		SubmitterID:      value.SubmitterID.String(),
		State:            value.State.String(),
		ResponseJSON:     value.ResponseSource.String(),
		ReviewNote:       value.ReviewNote.String(),
		Attachments:      attachmentsToResponse(value.Attachments),
		ValidationErrors: errors,
		SensitiveFields:  submissionSensitiveFieldsToResponse(value.SensitiveFields),
	}
}
func submissionValidationErrorsToResponse(outcome submission.ValidationOutcome) []submissionValidationErrorResponse {
	failed, matched := outcome.(submission.ValidationFailed)
	if !matched {
		return []submissionValidationErrorResponse{}
	}
	errors := make([]submissionValidationErrorResponse, 0, len(failed.Errors))
	for errorIndex := range failed.Errors {
		validationError := failed.Errors[errorIndex]
		errors = append(errors, submissionValidationErrorResponse{Path: validationError.Path, Message: validationError.Message})
	}
	return errors
}

func submissionSensitiveFieldsToResponse(fields []submission.SensitiveField) []submissionSensitiveFieldResponse {
	values := make([]submissionSensitiveFieldResponse, 0, len(fields))
	for fieldIndex := range fields {
		field := fields[fieldIndex]
		values = append(values, submissionSensitiveFieldResponse{
			Path:       field.Path,
			Category:   field.Category,
			Retention:  field.Retention,
			Redaction:  field.Redaction,
			State:      field.State,
			RedactedAt: field.RedactedAt,
		})
	}
	return values
}
