package taskbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- taskResultWire: shared by create/find/change-state (single Task) ----

type taskResultWire struct {
	Variant string                  `json:"variant"`
	Task    *taskWire               `json:"task,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func taskSuccessWire(variant string, value task.Task) taskResultWire {
	encoded := encodeTask(value)
	return taskResultWire{Variant: variant, Task: &encoded}
}

func decodeTaskPayload(wire *taskWire) (task.Task, error) {
	if wire == nil {
		return task.Task{}, fmt.Errorf("result is missing its task")
	}
	return decodeTask(*wire)
}

func encodeCreateTaskResult(result task.CreateTaskStoreResult) taskResultWire {
	switch typed := result.(type) {
	case task.CreateTaskStoreAccepted:
		return taskSuccessWire("created", typed.Value)
	case task.CreateTaskStoreRejected:
		return taskResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return taskResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeCreateTaskResult(wire taskResultWire) (task.CreateTaskStoreResult, error) {
	if wire.Variant == "rejected" {
		return task.CreateTaskStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	value, err := decodeTaskPayload(wire.Task)
	if err != nil {
		return nil, err
	}
	return task.CreateTaskStoreAccepted{Value: value}, nil
}

func encodeFindTaskResult(result task.FindTaskStoreResult) taskResultWire {
	switch typed := result.(type) {
	case task.FindTaskStoreAccepted:
		return taskSuccessWire("found", typed.Value)
	case task.FindTaskStoreRejected:
		return taskResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return taskResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeFindTaskResult(wire taskResultWire) (task.FindTaskStoreResult, error) {
	if wire.Variant == "rejected" {
		return task.FindTaskStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	value, err := decodeTaskPayload(wire.Task)
	if err != nil {
		return nil, err
	}
	return task.FindTaskStoreAccepted{Value: value}, nil
}

func encodeChangeTaskStateResult(result task.ChangeTaskStateStoreResult) taskResultWire {
	switch typed := result.(type) {
	case task.ChangeTaskStateStoreAccepted:
		return taskSuccessWire("changed", typed.Value)
	case task.ChangeTaskStateStoreRejected:
		return taskResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return taskResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeChangeTaskStateResult(wire taskResultWire) (task.ChangeTaskStateStoreResult, error) {
	if wire.Variant == "rejected" {
		return task.ChangeTaskStateStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	value, err := decodeTaskPayload(wire.Task)
	if err != nil {
		return nil, err
	}
	return task.ChangeTaskStateStoreAccepted{Value: value}, nil
}

// ---- reservationResultWire: shared by create/change-state (single Reservation) ----

type reservationResultWire struct {
	Variant     string                  `json:"variant"`
	Reservation *reservationWire        `json:"reservation,omitempty"`
	Error       *domainwire.DomainError `json:"error,omitempty"`
}

func reservationSuccessWire(variant string, value task.Reservation) reservationResultWire {
	encoded := encodeReservation(value)
	return reservationResultWire{Variant: variant, Reservation: &encoded}
}

func decodeReservationPayload(wire *reservationWire) (task.Reservation, error) {
	if wire == nil {
		return task.Reservation{}, fmt.Errorf("result is missing its reservation")
	}
	return decodeReservation(*wire)
}

func encodeCreateReservationResult(result task.CreateReservationStoreResult) reservationResultWire {
	switch typed := result.(type) {
	case task.CreateReservationStoreAccepted:
		return reservationSuccessWire("created", typed.Value)
	case task.CreateReservationStoreRejected:
		return reservationResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return reservationResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeCreateReservationResult(wire reservationResultWire) (task.CreateReservationStoreResult, error) {
	if wire.Variant == "rejected" {
		return task.CreateReservationStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	value, err := decodeReservationPayload(wire.Reservation)
	if err != nil {
		return nil, err
	}
	return task.CreateReservationStoreAccepted{Value: value}, nil
}

func encodeChangeReservationStateResult(result task.ChangeReservationStateStoreResult) reservationResultWire {
	switch typed := result.(type) {
	case task.ChangeReservationStateStoreAccepted:
		return reservationSuccessWire("changed", typed.Value)
	case task.ChangeReservationStateStoreRejected:
		return reservationResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return reservationResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeChangeReservationStateResult(wire reservationResultWire) (task.ChangeReservationStateStoreResult, error) {
	if wire.Variant == "rejected" {
		return task.ChangeReservationStateStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	value, err := decodeReservationPayload(wire.Reservation)
	if err != nil {
		return nil, err
	}
	return task.ChangeReservationStateStoreAccepted{Value: value}, nil
}

// ---- seriesDetailResultWire: shared by find-series/series-mutation ----

type seriesDetailResultWire struct {
	Variant string                  `json:"variant"`
	Series  *seriesDetailWire       `json:"series,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func seriesDetailSuccessWire(variant string, value task.SeriesDetail) seriesDetailResultWire {
	encoded := encodeSeriesDetail(value)
	return seriesDetailResultWire{Variant: variant, Series: &encoded}
}

func encodeFindSeriesResult(result task.FindSeriesStoreResult) seriesDetailResultWire {
	switch typed := result.(type) {
	case task.FindSeriesStoreAccepted:
		return seriesDetailSuccessWire("found", typed.Value)
	case task.FindSeriesStoreRejected:
		return seriesDetailResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return seriesDetailResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeFindSeriesResult(wire seriesDetailResultWire) (task.FindSeriesStoreResult, error) {
	if wire.Variant == "rejected" {
		return task.FindSeriesStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	value, err := decodeSeriesDetail(wire.Series)
	if err != nil {
		return nil, err
	}
	return task.FindSeriesStoreAccepted{Value: value}, nil
}

func encodeSeriesMutationResult(result task.SeriesMutationStoreResult) seriesDetailResultWire {
	switch typed := result.(type) {
	case task.SeriesMutationStoreAccepted:
		return seriesDetailSuccessWire("accepted", typed.Value)
	case task.SeriesMutationStoreRejected:
		return seriesDetailResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return seriesDetailResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeSeriesMutationResult(wire seriesDetailResultWire) (task.SeriesMutationStoreResult, error) {
	if wire.Variant == "rejected" {
		return task.SeriesMutationStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	value, err := decodeSeriesDetail(wire.Series)
	if err != nil {
		return nil, err
	}
	return task.SeriesMutationStoreAccepted{Value: value}, nil
}

// ---- slice results ----

type listItemsResultWire struct {
	Variant string                  `json:"variant"`
	Items   []listItemWire          `json:"items,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListTasksResult(result task.ListTasksStoreResult) listItemsResultWire {
	switch typed := result.(type) {
	case task.ListTasksStoreAccepted:
		return listItemsResultWire{Variant: "listed", Items: encodeListItems(typed.Values)}
	case task.ListTasksStoreRejected:
		return listItemsResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return listItemsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeListTasksResult(wire listItemsResultWire) (task.ListTasksStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeListItems(wire.Items)
		if err != nil {
			return nil, err
		}
		return task.ListTasksStoreAccepted{Values: values}, nil
	case "rejected":
		return task.ListTasksStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list tasks result variant %q", wire.Variant)
	}
}

type seriesListResultWire struct {
	Variant string                  `json:"variant"`
	Series  []seriesWire            `json:"series,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListSeriesResult(result task.ListSeriesStoreResult) seriesListResultWire {
	switch typed := result.(type) {
	case task.ListSeriesStoreAccepted:
		return seriesListResultWire{Variant: "listed", Series: encodeSeriesList(typed.Values)}
	case task.ListSeriesStoreRejected:
		return seriesListResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return seriesListResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeListSeriesResult(wire seriesListResultWire) (task.ListSeriesStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeSeriesList(wire.Series)
		if err != nil {
			return nil, err
		}
		return task.ListSeriesStoreAccepted{Values: values}, nil
	case "rejected":
		return task.ListSeriesStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list series result variant %q", wire.Variant)
	}
}

type reservationsResultWire struct {
	Variant      string                  `json:"variant"`
	Reservations []reservationWire       `json:"reservations,omitempty"`
	Error        *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListReservationsResult(result task.ListReservationsStoreResult) reservationsResultWire {
	switch typed := result.(type) {
	case task.ListReservationsStoreAccepted:
		return reservationsResultWire{Variant: "listed", Reservations: encodeReservations(typed.Values)}
	case task.ListReservationsStoreRejected:
		return reservationsResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return reservationsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeListReservationsResult(wire reservationsResultWire) (task.ListReservationsStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeReservations(wire.Reservations)
		if err != nil {
			return nil, err
		}
		return task.ListReservationsStoreAccepted{Values: values}, nil
	case "rejected":
		return task.ListReservationsStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list reservations result variant %q", wire.Variant)
	}
}

// ---- comment results ----

type seriesCommentResultWire struct {
	Variant string                  `json:"variant"`
	Comment *seriesCommentWire      `json:"comment,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateSeriesCommentResult(result task.CreateSeriesCommentStoreResult) seriesCommentResultWire {
	switch typed := result.(type) {
	case task.CreateSeriesCommentStoreAccepted:
		comment := encodeSeriesComment(typed.Value)
		return seriesCommentResultWire{Variant: "created", Comment: &comment}
	case task.CreateSeriesCommentStoreRejected:
		return seriesCommentResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return seriesCommentResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeCreateSeriesCommentResult(wire seriesCommentResultWire) (task.CreateSeriesCommentStoreResult, error) {
	switch wire.Variant {
	case "created":
		if wire.Comment == nil {
			return nil, fmt.Errorf("result is missing its series comment")
		}
		comment, err := decodeSeriesComment(*wire.Comment)
		if err != nil {
			return nil, err
		}
		return task.CreateSeriesCommentStoreAccepted{Value: comment}, nil
	case "rejected":
		return task.CreateSeriesCommentStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create series comment result variant %q", wire.Variant)
	}
}

type seriesCommentsResultWire struct {
	Variant  string                  `json:"variant"`
	Comments []seriesCommentWire     `json:"comments,omitempty"`
	Error    *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListSeriesCommentsResult(result task.ListSeriesCommentsStoreResult) seriesCommentsResultWire {
	switch typed := result.(type) {
	case task.ListSeriesCommentsStoreAccepted:
		return seriesCommentsResultWire{Variant: "listed", Comments: encodeSeriesComments(typed.Values)}
	case task.ListSeriesCommentsStoreRejected:
		return seriesCommentsResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return seriesCommentsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeListSeriesCommentsResult(wire seriesCommentsResultWire) (task.ListSeriesCommentsStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeSeriesComments(wire.Comments)
		if err != nil {
			return nil, err
		}
		return task.ListSeriesCommentsStoreAccepted{Values: values}, nil
	case "rejected":
		return task.ListSeriesCommentsStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list series comments result variant %q", wire.Variant)
	}
}

type taskCommentResultWire struct {
	Variant string                  `json:"variant"`
	Comment *taskCommentWire        `json:"comment,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateTaskCommentResult(result task.CreateTaskCommentStoreResult) taskCommentResultWire {
	switch typed := result.(type) {
	case task.CreateTaskCommentStoreAccepted:
		comment := encodeTaskComment(typed.Value)
		return taskCommentResultWire{Variant: "created", Comment: &comment}
	case task.CreateTaskCommentStoreRejected:
		return taskCommentResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return taskCommentResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeCreateTaskCommentResult(wire taskCommentResultWire) (task.CreateTaskCommentStoreResult, error) {
	switch wire.Variant {
	case "created":
		if wire.Comment == nil {
			return nil, fmt.Errorf("result is missing its task comment")
		}
		comment, err := decodeTaskComment(*wire.Comment)
		if err != nil {
			return nil, err
		}
		return task.CreateTaskCommentStoreAccepted{Value: comment}, nil
	case "rejected":
		return task.CreateTaskCommentStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create task comment result variant %q", wire.Variant)
	}
}

type taskCommentsResultWire struct {
	Variant  string                  `json:"variant"`
	Comments []taskCommentWire       `json:"comments,omitempty"`
	Error    *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListTaskCommentsResult(result task.ListTaskCommentsStoreResult) taskCommentsResultWire {
	switch typed := result.(type) {
	case task.ListTaskCommentsStoreAccepted:
		return taskCommentsResultWire{Variant: "listed", Comments: encodeTaskComments(typed.Values)}
	case task.ListTaskCommentsStoreRejected:
		return taskCommentsResultWire{Variant: "rejected", Error: encodeReason(typed.Reason)}
	default:
		return taskCommentsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown task result %T", result))}
	}
}

func decodeListTaskCommentsResult(wire taskCommentsResultWire) (task.ListTaskCommentsStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeTaskComments(wire.Comments)
		if err != nil {
			return nil, err
		}
		return task.ListTaskCommentsStoreAccepted{Values: values}, nil
	case "rejected":
		return task.ListTaskCommentsStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list task comments result variant %q", wire.Variant)
	}
}

// ---- SubmissionEligibility (accept/reject) ----

type acceptedRejectedWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeSubmissionEligibilityResult(result task.SubmissionEligibilityStoreResult) acceptedRejectedWire {
	rejected, matched := result.(task.SubmissionEligibilityRejected)
	if !matched {
		return acceptedRejectedWire{Variant: "eligible"}
	}
	return acceptedRejectedWire{Variant: "rejected", Error: encodeReason(rejected.Reason)}
}

func decodeSubmissionEligibilityResult(wire acceptedRejectedWire) (task.SubmissionEligibilityStoreResult, error) {
	if wire.Variant == "rejected" {
		return task.SubmissionEligibilityRejected{Reason: decodeReason(wire.Error)}, nil
	}
	return task.SubmissionEligible{}, nil
}

// ---- shared helpers ----

func encodeReason(reason core.DomainError) *domainwire.DomainError {
	encoded := domainwire.EncodeDomainError(reason)
	return &encoded
}

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "task bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
