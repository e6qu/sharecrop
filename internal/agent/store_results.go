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

// CredentialWorkActivity is the read model of one credential's current work
// consumption: how much of today's UTC budget window is used and how many
// reservations attributed to the credential are active right now. It exists so
// a human watching their agent can see budget state next to the configured
// allowances; it is not part of Credential because it is a per-day aggregate,
// not a property of the credential itself.
type CredentialWorkActivity struct {
	CredentialID core.AgentCredentialID
	// TasksToday counts today's consumed daily-task-budget units
	// (reservations plus direct submissions made via the credential).
	TasksToday int64
	// CreditsSpentToday sums the credits spent via the credential today.
	CreditsSpentToday int64
	// ActiveReservations counts reservations established via the credential
	// that are still active.
	ActiveReservations int64
}

type WorkActivityStoreResult interface {
	workActivityStoreResult()
}

type WorkActivityStoreListed struct {
	Values []CredentialWorkActivity
}

type WorkActivityStoreRejected struct {
	Reason core.DomainError
}

func (WorkActivityStoreListed) workActivityStoreResult() {}

func (WorkActivityStoreRejected) workActivityStoreResult() {}

type UpdateWorkPolicyStoreResult interface {
	updateWorkPolicyStoreResult()
}

type UpdateWorkPolicyStoreUpdated struct {
	Value Credential
}

type UpdateWorkPolicyStoreRejected struct {
	Reason core.DomainError
}

func (UpdateWorkPolicyStoreUpdated) updateWorkPolicyStoreResult() {}

func (UpdateWorkPolicyStoreRejected) updateWorkPolicyStoreResult() {}
