package submission

import "github.com/e6qu/sharecrop/internal/core"

type CreateSubmissionStoreResult interface {
	createSubmissionStoreResult()
}

type CreateSubmissionStoreAccepted struct {
	Value Submission
}

type CreateSubmissionStoreRejected struct {
	Reason core.DomainError
}

func (CreateSubmissionStoreAccepted) createSubmissionStoreResult() {}

func (CreateSubmissionStoreRejected) createSubmissionStoreResult() {}

type FindReceiptStoreResult interface {
	findReceiptStoreResult()
}

type ReceiptFound struct {
	Value Submission
}

type ReceiptMissing struct {
	Reason core.DomainError
}

func (ReceiptFound) findReceiptStoreResult() {}

func (ReceiptMissing) findReceiptStoreResult() {}

type FindSubmissionStoreResult interface {
	findSubmissionStoreResult()
}

type FindSubmissionStoreAccepted struct {
	Value Submission
}

type FindSubmissionStoreRejected struct {
	Reason core.DomainError
}

func (FindSubmissionStoreAccepted) findSubmissionStoreResult() {}

func (FindSubmissionStoreRejected) findSubmissionStoreResult() {}

type ListSubmissionsStoreResult interface {
	listSubmissionsStoreResult()
}

type ListSubmissionsStoreAccepted struct {
	Values []Submission
}

type ListSubmissionsStoreRejected struct {
	Reason core.DomainError
}

func (ListSubmissionsStoreAccepted) listSubmissionsStoreResult() {}

func (ListSubmissionsStoreRejected) listSubmissionsStoreResult() {}
