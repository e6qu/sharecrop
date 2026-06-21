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
