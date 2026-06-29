package notification

import "github.com/e6qu/sharecrop/internal/core"

type CreateStoreResult interface {
	createStoreResult()
}

type CreateStoreAccepted struct{}

type CreateStoreRejected struct {
	Reason core.DomainError
}

func (CreateStoreAccepted) createStoreResult() {}

func (CreateStoreRejected) createStoreResult() {}

type ListStoreResult interface {
	listStoreResult()
}

type ListStoreAccepted struct {
	Values []Notification
}

type ListStoreRejected struct {
	Reason core.DomainError
}

func (ListStoreAccepted) listStoreResult() {}

func (ListStoreRejected) listStoreResult() {}

type MarkReadStoreResult interface {
	markReadStoreResult()
}

type MarkReadStoreAccepted struct {
	Value Notification
}

type MarkReadStoreRejected struct {
	Reason core.DomainError
}

func (MarkReadStoreAccepted) markReadStoreResult() {}

func (MarkReadStoreRejected) markReadStoreResult() {}
