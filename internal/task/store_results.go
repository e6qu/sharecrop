package task

import "github.com/e6qu/sharecrop/internal/core"

type CreateTaskStoreResult interface {
	createTaskStoreResult()
}

type CreateTaskStoreAccepted struct {
	Value Task
}

type CreateTaskStoreRejected struct {
	Reason core.DomainError
}

func (CreateTaskStoreAccepted) createTaskStoreResult() {}

func (CreateTaskStoreRejected) createTaskStoreResult() {}

type ChangeTaskStateStoreResult interface {
	changeTaskStateStoreResult()
}

type ChangeTaskStateStoreAccepted struct {
	Value Task
}

type ChangeTaskStateStoreRejected struct {
	Reason core.DomainError
}

func (ChangeTaskStateStoreAccepted) changeTaskStateStoreResult() {}

func (ChangeTaskStateStoreRejected) changeTaskStateStoreResult() {}

type FindTaskStoreResult interface {
	findTaskStoreResult()
}

type FindTaskStoreAccepted struct {
	Value Task
}

type FindTaskStoreRejected struct {
	Reason core.DomainError
}

func (FindTaskStoreAccepted) findTaskStoreResult() {}

func (FindTaskStoreRejected) findTaskStoreResult() {}

type ListTasksStoreResult interface {
	listTasksStoreResult()
}

type ListTasksStoreAccepted struct {
	Values []Task
}

type ListTasksStoreRejected struct {
	Reason core.DomainError
}

func (ListTasksStoreAccepted) listTasksStoreResult() {}

func (ListTasksStoreRejected) listTasksStoreResult() {}

type CreateCapabilityTokenStoreResult interface {
	createCapabilityTokenStoreResult()
}

type CreateCapabilityTokenStoreAccepted struct {
	Value CapabilityToken
}

type CreateCapabilityTokenStoreRejected struct {
	Reason core.DomainError
}

func (CreateCapabilityTokenStoreAccepted) createCapabilityTokenStoreResult() {}

func (CreateCapabilityTokenStoreRejected) createCapabilityTokenStoreResult() {}

type CreateReservationStoreResult interface {
	createReservationStoreResult()
}

type CreateReservationStoreAccepted struct {
	Value Reservation
}

type CreateReservationStoreRejected struct {
	Reason core.DomainError
}

func (CreateReservationStoreAccepted) createReservationStoreResult() {}

func (CreateReservationStoreRejected) createReservationStoreResult() {}

type ChangeReservationStateStoreResult interface {
	changeReservationStateStoreResult()
}

type ChangeReservationStateStoreAccepted struct {
	Value Reservation
}

type ChangeReservationStateStoreRejected struct {
	Reason core.DomainError
}

func (ChangeReservationStateStoreAccepted) changeReservationStateStoreResult() {}

func (ChangeReservationStateStoreRejected) changeReservationStateStoreResult() {}

type ListReservationsStoreResult interface {
	listReservationsStoreResult()
}

type ListReservationsStoreAccepted struct {
	Values []Reservation
}

type ListReservationsStoreRejected struct {
	Reason core.DomainError
}

func (ListReservationsStoreAccepted) listReservationsStoreResult() {}

func (ListReservationsStoreRejected) listReservationsStoreResult() {}

type SubmissionEligibilityStoreResult interface {
	submissionEligibilityStoreResult()
}

type SubmissionEligible struct{}

type SubmissionEligibilityRejected struct {
	Reason core.DomainError
}

func (SubmissionEligible) submissionEligibilityStoreResult() {}

func (SubmissionEligibilityRejected) submissionEligibilityStoreResult() {}
