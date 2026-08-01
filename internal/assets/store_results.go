package assets

import "github.com/e6qu/sharecrop/internal/core"

type CreateStoreResult interface {
	createStoreResult()
}

type CreateStoreAccepted struct {
	// Value is the collectible as created, including store-assigned
	// provenance (the edition number for edition-kind mints).
	Value Collectible
}

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
