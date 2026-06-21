package agent

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

type VerifyStoreResult interface {
	verifyStoreResult()
}

type VerifyStoreFound struct {
	Value Credential
}

type VerifyStoreRejected struct {
	Reason core.DomainError
}

func (VerifyStoreFound) verifyStoreResult() {}

func (VerifyStoreRejected) verifyStoreResult() {}

type ListStoreResult interface {
	listStoreResult()
}

type ListStoreListed struct {
	Values []Credential
}

type ListStoreRejected struct {
	Reason core.DomainError
}

func (ListStoreListed) listStoreResult() {}

func (ListStoreRejected) listStoreResult() {}

type RevokeStoreResult interface {
	revokeStoreResult()
}

type RevokeStoreRevoked struct {
	Value Credential
}

type RevokeStoreRejected struct {
	Reason core.DomainError
}

func (RevokeStoreRevoked) revokeStoreResult() {}

func (RevokeStoreRejected) revokeStoreResult() {}
