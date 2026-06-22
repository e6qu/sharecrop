package assets

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

type ListStoreListed struct {
	Values []Collectible
}

type ListStoreRejected struct {
	Reason core.DomainError
}

func (ListStoreListed) listStoreResult() {}

func (ListStoreRejected) listStoreResult() {}
