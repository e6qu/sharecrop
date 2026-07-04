package orgcred

import (
	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
)

// CreateStoreResult and its variants carry no credential-kind-specific
// payload (just accepted, or rejected with a reason), so they're shared
// directly with agent's identical store-create result rather than
// redeclaring the same three types here.
type CreateStoreResult = agent.CreateStoreResult

type CreateStoreAccepted = agent.CreateStoreAccepted

type CreateStoreRejected = agent.CreateStoreRejected

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
