// Package ledgerbridge is the WASI bridge for internal/ledger's Store: hand-
// written per-type codecs (this file) plus a generated dispatcher and guest
// client (bridge_gen.go). Shared core types (ids, page) are serialized by
// internal/wasibridge/corewire; the domain error by internal/wasibridge/domainwire.
//
// The ledger is the widest store: its commands carry nested selection unions
// (credit/tip/collectible/ban) and its accept/reject results carry nested payout
// and tip outcome unions. Amounts cross the wire as int64 base units and are
// rebuilt through their validating constructors.
package ledgerbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- scalar value types ----

func encodeCreditAmount(amount ledger.CreditAmount) int64 { return amount.Int64() }

func decodeCreditAmount(value int64) (ledger.CreditAmount, error) {
	accepted, matched := ledger.NewCreditAmount(value).(ledger.CreditAmountAccepted)
	if !matched {
		return ledger.CreditAmount{}, fmt.Errorf("invalid credit amount %d", value)
	}
	return accepted.Value, nil
}

func decodeSignedAmount(value int64) (ledger.SignedAmount, error) {
	accepted, matched := ledger.ParseSignedAmount(value).(ledger.SignedAmountAccepted)
	if !matched {
		return ledger.SignedAmount{}, fmt.Errorf("invalid signed amount %d", value)
	}
	return accepted.Value, nil
}

func encodeIdempotencyKey(key ledger.IdempotencyKey) string { return key.String() }

func decodeIdempotencyKey(raw string) (ledger.IdempotencyKey, error) {
	accepted, matched := ledger.NewIdempotencyKey(raw).(ledger.IdempotencyKeyAccepted)
	if !matched {
		return ledger.IdempotencyKey{}, fmt.Errorf("invalid idempotency key")
	}
	return accepted.Value, nil
}

func decodeEntryKind(raw string) (ledger.EntryKind, error) {
	accepted, matched := ledger.ParseEntryKind(raw).(ledger.EntryKindAccepted)
	if !matched {
		return ledger.EntryKind{}, fmt.Errorf("invalid entry kind %q", raw)
	}
	return accepted.Value, nil
}

func encodeReviewNote(note submission.ReviewNote) string { return note.String() }

func decodeReviewNote(raw string) (submission.ReviewNote, error) {
	accepted, matched := submission.NewStoredReviewNote(raw).(submission.ReviewNoteAccepted)
	if !matched {
		return submission.EmptyReviewNote(), fmt.Errorf("invalid review note")
	}
	return accepted.Value, nil
}

// ---- collectible id slices ----

func encodeCollectibleIDs(ids []core.CollectibleID) []string {
	encoded := make([]string, 0, len(ids))
	for index := range ids {
		encoded = append(encoded, corewire.EncodeCollectibleID(ids[index]))
	}
	return encoded
}

func decodeCollectibleIDs(raw []string) ([]core.CollectibleID, error) {
	ids := make([]core.CollectibleID, 0, len(raw))
	for index := range raw {
		id, err := corewire.DecodeCollectibleID(raw[index])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ---- ledger.TaskFund ----

type taskFundWire struct {
	TaskID          string `json:"task_id"`
	FunderAccountID string `json:"funder_account_id"`
	CreditAmount    int64  `json:"credit_amount"`
}

func encodeTaskFund(fund ledger.TaskFund) taskFundWire {
	return taskFundWire{
		TaskID:          corewire.EncodeTaskID(fund.TaskID),
		FunderAccountID: corewire.EncodeCreditAccountID(fund.FunderAccountID),
		CreditAmount:    encodeCreditAmount(fund.CreditAmount),
	}
}

func decodeTaskFund(wire *taskFundWire) (ledger.TaskFund, error) {
	if wire == nil {
		return ledger.TaskFund{}, fmt.Errorf("result is missing its task fund")
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return ledger.TaskFund{}, err
	}
	funderAccountID, err := corewire.DecodeCreditAccountID(wire.FunderAccountID)
	if err != nil {
		return ledger.TaskFund{}, err
	}
	amount, err := decodeCreditAmount(wire.CreditAmount)
	if err != nil {
		return ledger.TaskFund{}, err
	}
	return ledger.TaskFund{TaskID: taskID, FunderAccountID: funderAccountID, CreditAmount: amount}, nil
}

// ---- ledger.LedgerEntry ----

type taskReferenceWire struct {
	Variant string `json:"variant"`
	TaskID  string `json:"task_id,omitempty"`
}

func encodeTaskReference(ref ledger.TaskReference) taskReferenceWire {
	referenced, matched := ref.(ledger.TaskReferenced)
	if !matched {
		return taskReferenceWire{Variant: "none"}
	}
	return taskReferenceWire{Variant: "referenced", TaskID: corewire.EncodeTaskID(referenced.TaskID)}
}

func decodeTaskReference(wire taskReferenceWire) (ledger.TaskReference, error) {
	switch wire.Variant {
	case "none":
		return ledger.NoTaskReference{}, nil
	case "referenced":
		taskID, err := corewire.DecodeTaskID(wire.TaskID)
		if err != nil {
			return nil, err
		}
		return ledger.TaskReferenced{TaskID: taskID}, nil
	default:
		return nil, fmt.Errorf("unknown task reference variant %q", wire.Variant)
	}
}

type ledgerEntryWire struct {
	ID      string            `json:"id"`
	Kind    string            `json:"kind"`
	Amount  int64             `json:"amount"`
	TaskRef taskReferenceWire `json:"task_ref"`
}

func encodeLedgerEntry(entry ledger.LedgerEntry) ledgerEntryWire {
	return ledgerEntryWire{
		ID:      corewire.EncodeLedgerEntryID(entry.ID),
		Kind:    entry.Kind.String(),
		Amount:  entry.Amount.Int64(),
		TaskRef: encodeTaskReference(entry.TaskRef),
	}
}

func decodeLedgerEntry(wire ledgerEntryWire) (ledger.LedgerEntry, error) {
	id, err := corewire.DecodeLedgerEntryID(wire.ID)
	if err != nil {
		return ledger.LedgerEntry{}, err
	}
	kind, err := decodeEntryKind(wire.Kind)
	if err != nil {
		return ledger.LedgerEntry{}, err
	}
	amount, err := decodeSignedAmount(wire.Amount)
	if err != nil {
		return ledger.LedgerEntry{}, err
	}
	taskRef, err := decodeTaskReference(wire.TaskRef)
	if err != nil {
		return ledger.LedgerEntry{}, err
	}
	return ledger.LedgerEntry{ID: id, Kind: kind, Amount: amount, TaskRef: taskRef}, nil
}

func encodeLedgerEntries(entries []ledger.LedgerEntry) []ledgerEntryWire {
	encoded := make([]ledgerEntryWire, 0, len(entries))
	for index := range entries {
		encoded = append(encoded, encodeLedgerEntry(entries[index]))
	}
	return encoded
}

func decodeLedgerEntries(wires []ledgerEntryWire) ([]ledger.LedgerEntry, error) {
	entries := make([]ledger.LedgerEntry, 0, len(wires))
	for index := range wires {
		entry, err := decodeLedgerEntry(wires[index])
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ---- selection unions (command side) ----

// selectionWire is the tagged shape shared by every command-side selection
// union; a given union uses only the fields its variants carry.
type selectionWire struct {
	Variant string `json:"variant"`
	Amount  int64  `json:"amount,omitempty"`
	ID      string `json:"id,omitempty"`
}

func encodeCreditReviewSelection(selection ledger.CreditReviewSelection) selectionWire {
	switch typed := selection.(type) {
	case ledger.PartialCreditReviewSelection:
		return selectionWire{Variant: "partial", Amount: encodeCreditAmount(typed.Amount)}
	case ledger.NoCreditReviewSelection:
		return selectionWire{Variant: "none"}
	default:
		return selectionWire{Variant: "full"}
	}
}

func decodeCreditReviewSelection(wire selectionWire) (ledger.CreditReviewSelection, error) {
	switch wire.Variant {
	case "full":
		return ledger.FullCreditReviewSelection{}, nil
	case "none":
		return ledger.NoCreditReviewSelection{}, nil
	case "partial":
		amount, err := decodeCreditAmount(wire.Amount)
		if err != nil {
			return nil, err
		}
		return ledger.PartialCreditReviewSelection{Amount: amount}, nil
	default:
		return nil, fmt.Errorf("unknown credit review selection variant %q", wire.Variant)
	}
}

func encodeTipSelection(selection ledger.TipSelection) selectionWire {
	credit, matched := selection.(ledger.CreditTipSelection)
	if !matched {
		return selectionWire{Variant: "none"}
	}
	return selectionWire{Variant: "credit", Amount: encodeCreditAmount(credit.Amount)}
}

func decodeTipSelection(wire selectionWire) (ledger.TipSelection, error) {
	switch wire.Variant {
	case "none":
		return ledger.NoTipSelection{}, nil
	case "credit":
		amount, err := decodeCreditAmount(wire.Amount)
		if err != nil {
			return nil, err
		}
		return ledger.CreditTipSelection{Amount: amount}, nil
	default:
		return nil, fmt.Errorf("unknown tip selection variant %q", wire.Variant)
	}
}

func encodeCollectibleTipSelection(selection ledger.CollectibleTipSelection) selectionWire {
	selected, matched := selection.(ledger.CollectibleTipSelected)
	if !matched {
		return selectionWire{Variant: "none"}
	}
	return selectionWire{Variant: "selected", ID: corewire.EncodeCollectibleID(selected.ID)}
}

func decodeCollectibleTipSelection(wire selectionWire) (ledger.CollectibleTipSelection, error) {
	switch wire.Variant {
	case "none":
		return ledger.NoCollectibleTipSelection{}, nil
	case "selected":
		id, err := corewire.DecodeCollectibleID(wire.ID)
		if err != nil {
			return nil, err
		}
		return ledger.CollectibleTipSelected{ID: id}, nil
	default:
		return nil, fmt.Errorf("unknown collectible tip selection variant %q", wire.Variant)
	}
}

func encodeBanSelection(selection ledger.BanSelection) selectionWire {
	if _, matched := selection.(ledger.BanImplementorSelection); matched {
		return selectionWire{Variant: "ban"}
	}
	return selectionWire{Variant: "none"}
}

func decodeBanSelection(wire selectionWire) (ledger.BanSelection, error) {
	switch wire.Variant {
	case "none":
		return ledger.NoBanSelection{}, nil
	case "ban":
		return ledger.BanImplementorSelection{}, nil
	default:
		return nil, fmt.Errorf("unknown ban selection variant %q", wire.Variant)
	}
}

// ---- command structs ----

type fundCommandWire struct {
	EntryID        string `json:"entry_id"`
	FunderUserID   string `json:"funder_user_id"`
	TaskID         string `json:"task_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

func encodeFundCommand(command ledger.FundStoreCommand) fundCommandWire {
	return fundCommandWire{
		EntryID:        corewire.EncodeLedgerEntryID(command.EntryID),
		FunderUserID:   corewire.EncodeUserID(command.FunderUserID),
		TaskID:         corewire.EncodeTaskID(command.TaskID),
		Amount:         encodeCreditAmount(command.Amount),
		IdempotencyKey: encodeIdempotencyKey(command.IdempotencyKey),
	}
}

func decodeFundCommand(wire fundCommandWire) (ledger.FundStoreCommand, error) {
	entryID, err := corewire.DecodeLedgerEntryID(wire.EntryID)
	if err != nil {
		return ledger.FundStoreCommand{}, err
	}
	funderUserID, err := corewire.DecodeUserID(wire.FunderUserID)
	if err != nil {
		return ledger.FundStoreCommand{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return ledger.FundStoreCommand{}, err
	}
	amount, err := decodeCreditAmount(wire.Amount)
	if err != nil {
		return ledger.FundStoreCommand{}, err
	}
	key, err := decodeIdempotencyKey(wire.IdempotencyKey)
	if err != nil {
		return ledger.FundStoreCommand{}, err
	}
	return ledger.FundStoreCommand{
		EntryID:        entryID,
		FunderUserID:   funderUserID,
		TaskID:         taskID,
		Amount:         amount,
		IdempotencyKey: key,
	}, nil
}

type orgFundCommandWire struct {
	EntryID        string `json:"entry_id"`
	OrganizationID string `json:"organization_id"`
	TaskID         string `json:"task_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

func encodeOrgFundCommand(command ledger.OrganizationFundStoreCommand) orgFundCommandWire {
	return orgFundCommandWire{
		EntryID:        corewire.EncodeLedgerEntryID(command.EntryID),
		OrganizationID: corewire.EncodeOrganizationID(command.OrganizationID),
		TaskID:         corewire.EncodeTaskID(command.TaskID),
		Amount:         encodeCreditAmount(command.Amount),
		IdempotencyKey: encodeIdempotencyKey(command.IdempotencyKey),
	}
}

func decodeOrgFundCommand(wire orgFundCommandWire) (ledger.OrganizationFundStoreCommand, error) {
	entryID, err := corewire.DecodeLedgerEntryID(wire.EntryID)
	if err != nil {
		return ledger.OrganizationFundStoreCommand{}, err
	}
	organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
	if err != nil {
		return ledger.OrganizationFundStoreCommand{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return ledger.OrganizationFundStoreCommand{}, err
	}
	amount, err := decodeCreditAmount(wire.Amount)
	if err != nil {
		return ledger.OrganizationFundStoreCommand{}, err
	}
	key, err := decodeIdempotencyKey(wire.IdempotencyKey)
	if err != nil {
		return ledger.OrganizationFundStoreCommand{}, err
	}
	return ledger.OrganizationFundStoreCommand{
		EntryID:        entryID,
		OrganizationID: organizationID,
		TaskID:         taskID,
		Amount:         amount,
		IdempotencyKey: key,
	}, nil
}

type acceptCommandWire struct {
	PayoutEntryID    string        `json:"payout_entry_id"`
	RefundEntryID    string        `json:"refund_entry_id"`
	TipDebitEntryID  string        `json:"tip_debit_entry_id"`
	TipCreditEntryID string        `json:"tip_credit_entry_id"`
	RequesterUserID  string        `json:"requester_user_id"`
	TaskID           string        `json:"task_id"`
	SubmissionID     string        `json:"submission_id"`
	IdempotencyKey   string        `json:"idempotency_key"`
	CreditSelection  selectionWire `json:"credit_selection"`
	TipSelection     selectionWire `json:"tip_selection"`
	CollectibleTip   selectionWire `json:"collectible_tip"`
}

func encodeAcceptCommand(command ledger.AcceptStoreCommand) acceptCommandWire {
	return acceptCommandWire{
		PayoutEntryID:    corewire.EncodeLedgerEntryID(command.PayoutEntryID),
		RefundEntryID:    corewire.EncodeLedgerEntryID(command.RefundEntryID),
		TipDebitEntryID:  corewire.EncodeLedgerEntryID(command.TipDebitEntryID),
		TipCreditEntryID: corewire.EncodeLedgerEntryID(command.TipCreditEntryID),
		RequesterUserID:  corewire.EncodeUserID(command.RequesterUserID),
		TaskID:           corewire.EncodeTaskID(command.TaskID),
		SubmissionID:     corewire.EncodeSubmissionID(command.SubmissionID),
		IdempotencyKey:   encodeIdempotencyKey(command.IdempotencyKey),
		CreditSelection:  encodeCreditReviewSelection(command.CreditSelection),
		TipSelection:     encodeTipSelection(command.TipSelection),
		CollectibleTip:   encodeCollectibleTipSelection(command.CollectibleTip),
	}
}

func decodeAcceptCommand(wire acceptCommandWire) (ledger.AcceptStoreCommand, error) {
	ids, err := decodeReviewEntryIDs(wire.PayoutEntryID, wire.RefundEntryID, wire.TipDebitEntryID, wire.TipCreditEntryID)
	if err != nil {
		return ledger.AcceptStoreCommand{}, err
	}
	shared, err := decodeReviewCommon(wire.RequesterUserID, wire.TaskID, wire.SubmissionID, wire.IdempotencyKey)
	if err != nil {
		return ledger.AcceptStoreCommand{}, err
	}
	creditSelection, err := decodeCreditReviewSelection(wire.CreditSelection)
	if err != nil {
		return ledger.AcceptStoreCommand{}, err
	}
	tipSelection, err := decodeTipSelection(wire.TipSelection)
	if err != nil {
		return ledger.AcceptStoreCommand{}, err
	}
	collectibleTip, err := decodeCollectibleTipSelection(wire.CollectibleTip)
	if err != nil {
		return ledger.AcceptStoreCommand{}, err
	}
	return ledger.AcceptStoreCommand{
		PayoutEntryID:    ids.payout,
		RefundEntryID:    ids.refund,
		TipDebitEntryID:  ids.tipDebit,
		TipCreditEntryID: ids.tipCredit,
		RequesterUserID:  shared.requester,
		TaskID:           shared.taskID,
		SubmissionID:     shared.submissionID,
		IdempotencyKey:   shared.key,
		CreditSelection:  creditSelection,
		TipSelection:     tipSelection,
		CollectibleTip:   collectibleTip,
	}, nil
}

type requestChangesCommandWire struct {
	RequesterUserID string `json:"requester_user_id"`
	TaskID          string `json:"task_id"`
	SubmissionID    string `json:"submission_id"`
	ReviewNote      string `json:"review_note"`
}

func encodeRequestChangesCommand(command ledger.RequestChangesStoreCommand) requestChangesCommandWire {
	return requestChangesCommandWire{
		RequesterUserID: corewire.EncodeUserID(command.RequesterUserID),
		TaskID:          corewire.EncodeTaskID(command.TaskID),
		SubmissionID:    corewire.EncodeSubmissionID(command.SubmissionID),
		ReviewNote:      encodeReviewNote(command.ReviewNote),
	}
}

func decodeRequestChangesCommand(wire requestChangesCommandWire) (ledger.RequestChangesStoreCommand, error) {
	requesterUserID, err := corewire.DecodeUserID(wire.RequesterUserID)
	if err != nil {
		return ledger.RequestChangesStoreCommand{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return ledger.RequestChangesStoreCommand{}, err
	}
	submissionID, err := corewire.DecodeSubmissionID(wire.SubmissionID)
	if err != nil {
		return ledger.RequestChangesStoreCommand{}, err
	}
	reviewNote, err := decodeReviewNote(wire.ReviewNote)
	if err != nil {
		return ledger.RequestChangesStoreCommand{}, err
	}
	return ledger.RequestChangesStoreCommand{
		RequesterUserID: requesterUserID,
		TaskID:          taskID,
		SubmissionID:    submissionID,
		ReviewNote:      reviewNote,
	}, nil
}

type rejectCommandWire struct {
	PayoutEntryID    string        `json:"payout_entry_id"`
	TipDebitEntryID  string        `json:"tip_debit_entry_id"`
	TipCreditEntryID string        `json:"tip_credit_entry_id"`
	RequesterUserID  string        `json:"requester_user_id"`
	TaskID           string        `json:"task_id"`
	SubmissionID     string        `json:"submission_id"`
	IdempotencyKey   string        `json:"idempotency_key"`
	ReviewNote       string        `json:"review_note"`
	CreditSelection  selectionWire `json:"credit_selection"`
	TipSelection     selectionWire `json:"tip_selection"`
	BanSelection     selectionWire `json:"ban_selection"`
}

func encodeRejectCommand(command ledger.RejectStoreCommand) rejectCommandWire {
	return rejectCommandWire{
		PayoutEntryID:    corewire.EncodeLedgerEntryID(command.PayoutEntryID),
		TipDebitEntryID:  corewire.EncodeLedgerEntryID(command.TipDebitEntryID),
		TipCreditEntryID: corewire.EncodeLedgerEntryID(command.TipCreditEntryID),
		RequesterUserID:  corewire.EncodeUserID(command.RequesterUserID),
		TaskID:           corewire.EncodeTaskID(command.TaskID),
		SubmissionID:     corewire.EncodeSubmissionID(command.SubmissionID),
		IdempotencyKey:   encodeIdempotencyKey(command.IdempotencyKey),
		ReviewNote:       encodeReviewNote(command.ReviewNote),
		CreditSelection:  encodeCreditReviewSelection(command.CreditSelection),
		TipSelection:     encodeTipSelection(command.TipSelection),
		BanSelection:     encodeBanSelection(command.BanSelection),
	}
}

func decodeRejectCommand(wire rejectCommandWire) (ledger.RejectStoreCommand, error) {
	payoutEntryID, err := corewire.DecodeLedgerEntryID(wire.PayoutEntryID)
	if err != nil {
		return ledger.RejectStoreCommand{}, err
	}
	tipDebitEntryID, err := corewire.DecodeLedgerEntryID(wire.TipDebitEntryID)
	if err != nil {
		return ledger.RejectStoreCommand{}, err
	}
	tipCreditEntryID, err := corewire.DecodeLedgerEntryID(wire.TipCreditEntryID)
	if err != nil {
		return ledger.RejectStoreCommand{}, err
	}
	shared, err := decodeReviewCommon(wire.RequesterUserID, wire.TaskID, wire.SubmissionID, wire.IdempotencyKey)
	if err != nil {
		return ledger.RejectStoreCommand{}, err
	}
	reviewNote, err := decodeReviewNote(wire.ReviewNote)
	if err != nil {
		return ledger.RejectStoreCommand{}, err
	}
	creditSelection, err := decodeCreditReviewSelection(wire.CreditSelection)
	if err != nil {
		return ledger.RejectStoreCommand{}, err
	}
	tipSelection, err := decodeTipSelection(wire.TipSelection)
	if err != nil {
		return ledger.RejectStoreCommand{}, err
	}
	banSelection, err := decodeBanSelection(wire.BanSelection)
	if err != nil {
		return ledger.RejectStoreCommand{}, err
	}
	return ledger.RejectStoreCommand{
		PayoutEntryID:    payoutEntryID,
		TipDebitEntryID:  tipDebitEntryID,
		TipCreditEntryID: tipCreditEntryID,
		RequesterUserID:  shared.requester,
		TaskID:           shared.taskID,
		SubmissionID:     shared.submissionID,
		IdempotencyKey:   shared.key,
		ReviewNote:       reviewNote,
		CreditSelection:  creditSelection,
		TipSelection:     tipSelection,
		BanSelection:     banSelection,
	}, nil
}

type refundCommandWire struct {
	EntryID         string `json:"entry_id"`
	RequesterUserID string `json:"requester_user_id"`
	TaskID          string `json:"task_id"`
	IdempotencyKey  string `json:"idempotency_key"`
}

func encodeRefundCommand(command ledger.RefundStoreCommand) refundCommandWire {
	return refundCommandWire{
		EntryID:         corewire.EncodeLedgerEntryID(command.EntryID),
		RequesterUserID: corewire.EncodeUserID(command.RequesterUserID),
		TaskID:          corewire.EncodeTaskID(command.TaskID),
		IdempotencyKey:  encodeIdempotencyKey(command.IdempotencyKey),
	}
}

func decodeRefundCommand(wire refundCommandWire) (ledger.RefundStoreCommand, error) {
	entryID, err := corewire.DecodeLedgerEntryID(wire.EntryID)
	if err != nil {
		return ledger.RefundStoreCommand{}, err
	}
	requesterUserID, err := corewire.DecodeUserID(wire.RequesterUserID)
	if err != nil {
		return ledger.RefundStoreCommand{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return ledger.RefundStoreCommand{}, err
	}
	key, err := decodeIdempotencyKey(wire.IdempotencyKey)
	if err != nil {
		return ledger.RefundStoreCommand{}, err
	}
	return ledger.RefundStoreCommand{
		EntryID:         entryID,
		RequesterUserID: requesterUserID,
		TaskID:          taskID,
		IdempotencyKey:  key,
	}, nil
}

// reviewEntryIDs / reviewCommon collect the id fields the accept and reject
// commands share, so their decoders don't repeat the id-parsing boilerplate.
type reviewEntryIDs struct {
	payout    core.LedgerEntryID
	refund    core.LedgerEntryID
	tipDebit  core.LedgerEntryID
	tipCredit core.LedgerEntryID
}

func decodeReviewEntryIDs(payoutRaw, refundRaw, tipDebitRaw, tipCreditRaw string) (reviewEntryIDs, error) {
	payout, err := corewire.DecodeLedgerEntryID(payoutRaw)
	if err != nil {
		return reviewEntryIDs{}, err
	}
	refund, err := corewire.DecodeLedgerEntryID(refundRaw)
	if err != nil {
		return reviewEntryIDs{}, err
	}
	tipDebit, err := corewire.DecodeLedgerEntryID(tipDebitRaw)
	if err != nil {
		return reviewEntryIDs{}, err
	}
	tipCredit, err := corewire.DecodeLedgerEntryID(tipCreditRaw)
	if err != nil {
		return reviewEntryIDs{}, err
	}
	return reviewEntryIDs{payout: payout, refund: refund, tipDebit: tipDebit, tipCredit: tipCredit}, nil
}

type reviewCommon struct {
	requester    core.UserID
	taskID       core.TaskID
	submissionID core.SubmissionID
	key          ledger.IdempotencyKey
}

func decodeReviewCommon(requesterRaw, taskRaw, submissionRaw, keyRaw string) (reviewCommon, error) {
	requester, err := corewire.DecodeUserID(requesterRaw)
	if err != nil {
		return reviewCommon{}, err
	}
	taskID, err := corewire.DecodeTaskID(taskRaw)
	if err != nil {
		return reviewCommon{}, err
	}
	submissionID, err := corewire.DecodeSubmissionID(submissionRaw)
	if err != nil {
		return reviewCommon{}, err
	}
	key, err := decodeIdempotencyKey(keyRaw)
	if err != nil {
		return reviewCommon{}, err
	}
	return reviewCommon{requester: requester, taskID: taskID, submissionID: submissionID, key: key}, nil
}

// ---- nested outcome unions (result side) ----

type payoutOutcomeWire struct {
	Variant        string   `json:"variant"`
	WorkerUserID   string   `json:"worker_user_id,omitempty"`
	Amount         int64    `json:"amount,omitempty"`
	CollectibleIDs []string `json:"collectible_ids,omitempty"`
}

func encodePayoutOutcome(outcome ledger.PayoutOutcome) payoutOutcomeWire {
	switch typed := outcome.(type) {
	case ledger.CreditPayout:
		return payoutOutcomeWire{Variant: "credit", WorkerUserID: corewire.EncodeUserID(typed.WorkerUserID), Amount: encodeCreditAmount(typed.Amount)}
	case ledger.CollectiblePayout:
		return payoutOutcomeWire{Variant: "collectible", WorkerUserID: corewire.EncodeUserID(typed.WorkerUserID), CollectibleIDs: encodeCollectibleIDs(typed.CollectibleIDs)}
	case ledger.BundlePayout:
		return payoutOutcomeWire{Variant: "bundle", WorkerUserID: corewire.EncodeUserID(typed.WorkerUserID), Amount: encodeCreditAmount(typed.Amount), CollectibleIDs: encodeCollectibleIDs(typed.CollectibleIDs)}
	default:
		return payoutOutcomeWire{Variant: "none"}
	}
}

func decodePayoutOutcome(wire payoutOutcomeWire) (ledger.PayoutOutcome, error) {
	if wire.Variant == "none" {
		return ledger.NoPayout{}, nil
	}
	workerUserID, err := corewire.DecodeUserID(wire.WorkerUserID)
	if err != nil {
		return nil, err
	}
	switch wire.Variant {
	case "credit":
		amount, err := decodeCreditAmount(wire.Amount)
		if err != nil {
			return nil, err
		}
		return ledger.CreditPayout{WorkerUserID: workerUserID, Amount: amount}, nil
	case "collectible":
		ids, err := decodeCollectibleIDs(wire.CollectibleIDs)
		if err != nil {
			return nil, err
		}
		return ledger.CollectiblePayout{WorkerUserID: workerUserID, CollectibleIDs: ids}, nil
	case "bundle":
		amount, err := decodeCreditAmount(wire.Amount)
		if err != nil {
			return nil, err
		}
		ids, err := decodeCollectibleIDs(wire.CollectibleIDs)
		if err != nil {
			return nil, err
		}
		return ledger.BundlePayout{WorkerUserID: workerUserID, Amount: amount, CollectibleIDs: ids}, nil
	default:
		return nil, fmt.Errorf("unknown payout outcome variant %q", wire.Variant)
	}
}

type tipOutcomeWire struct {
	Variant       string `json:"variant"`
	WorkerUserID  string `json:"worker_user_id,omitempty"`
	Amount        int64  `json:"amount,omitempty"`
	CollectibleID string `json:"collectible_id,omitempty"`
}

func encodeTipOutcome(outcome ledger.TipOutcome) tipOutcomeWire {
	switch typed := outcome.(type) {
	case ledger.CreditTip:
		return tipOutcomeWire{Variant: "credit", WorkerUserID: corewire.EncodeUserID(typed.WorkerUserID), Amount: encodeCreditAmount(typed.Amount)}
	case ledger.CollectibleTip:
		return tipOutcomeWire{Variant: "collectible", WorkerUserID: corewire.EncodeUserID(typed.WorkerUserID), CollectibleID: corewire.EncodeCollectibleID(typed.CollectibleID)}
	case ledger.BundleTip:
		return tipOutcomeWire{Variant: "bundle", WorkerUserID: corewire.EncodeUserID(typed.WorkerUserID), Amount: encodeCreditAmount(typed.Amount), CollectibleID: corewire.EncodeCollectibleID(typed.CollectibleID)}
	default:
		return tipOutcomeWire{Variant: "none"}
	}
}

func decodeTipOutcome(wire tipOutcomeWire) (ledger.TipOutcome, error) {
	if wire.Variant == "none" {
		return ledger.NoTip{}, nil
	}
	workerUserID, err := corewire.DecodeUserID(wire.WorkerUserID)
	if err != nil {
		return nil, err
	}
	switch wire.Variant {
	case "credit":
		amount, err := decodeCreditAmount(wire.Amount)
		if err != nil {
			return nil, err
		}
		return ledger.CreditTip{WorkerUserID: workerUserID, Amount: amount}, nil
	case "collectible":
		id, err := corewire.DecodeCollectibleID(wire.CollectibleID)
		if err != nil {
			return nil, err
		}
		return ledger.CollectibleTip{WorkerUserID: workerUserID, CollectibleID: id}, nil
	case "bundle":
		amount, err := decodeCreditAmount(wire.Amount)
		if err != nil {
			return nil, err
		}
		id, err := corewire.DecodeCollectibleID(wire.CollectibleID)
		if err != nil {
			return nil, err
		}
		return ledger.BundleTip{WorkerUserID: workerUserID, Amount: amount, CollectibleID: id}, nil
	default:
		return nil, fmt.Errorf("unknown tip outcome variant %q", wire.Variant)
	}
}

// ---- result unions ----

// fundResultWire backs the fund and refund results, which each carry a task
// fund on success.
type fundResultWire struct {
	Variant string                  `json:"variant"`
	Fund    *taskFundWire           `json:"fund,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeFundResult(result ledger.FundResult) fundResultWire {
	switch typed := result.(type) {
	case ledger.TaskFunded:
		fund := encodeTaskFund(typed.Fund)
		return fundResultWire{Variant: "funded", Fund: &fund}
	case ledger.FundRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return fundResultWire{Variant: "rejected", Error: &reason}
	default:
		return fundResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown ledger result %T", result))}
	}
}

func decodeFundResult(wire fundResultWire) (ledger.FundResult, error) {
	switch wire.Variant {
	case "funded":
		fund, err := decodeTaskFund(wire.Fund)
		if err != nil {
			return nil, err
		}
		return ledger.TaskFunded{Fund: fund}, nil
	case "rejected":
		return ledger.FundRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown fund result variant %q", wire.Variant)
	}
}

func encodeRefundResult(result ledger.RefundResult) fundResultWire {
	switch typed := result.(type) {
	case ledger.TaskRefunded:
		fund := encodeTaskFund(typed.Fund)
		return fundResultWire{Variant: "refunded", Fund: &fund}
	case ledger.RefundRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return fundResultWire{Variant: "rejected", Error: &reason}
	default:
		return fundResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown ledger result %T", result))}
	}
}

func decodeRefundResult(wire fundResultWire) (ledger.RefundResult, error) {
	switch wire.Variant {
	case "refunded":
		fund, err := decodeTaskFund(wire.Fund)
		if err != nil {
			return nil, err
		}
		return ledger.TaskRefunded{Fund: fund}, nil
	case "rejected":
		return ledger.RefundRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown refund result variant %q", wire.Variant)
	}
}

// reviewedSubmissionWire backs the accept and reject results, which each carry a
// task/submission plus payout and tip outcomes on success.
type reviewedSubmissionWire struct {
	Variant      string                  `json:"variant"`
	TaskID       string                  `json:"task_id,omitempty"`
	SubmissionID string                  `json:"submission_id,omitempty"`
	Payout       *payoutOutcomeWire      `json:"payout,omitempty"`
	Tip          *tipOutcomeWire         `json:"tip,omitempty"`
	Error        *domainwire.DomainError `json:"error,omitempty"`
}

func reviewedSubmissionSuccess(variant string, taskID core.TaskID, submissionID core.SubmissionID, payout ledger.PayoutOutcome, tip ledger.TipOutcome) reviewedSubmissionWire {
	payoutWire := encodePayoutOutcome(payout)
	tipWire := encodeTipOutcome(tip)
	return reviewedSubmissionWire{
		Variant:      variant,
		TaskID:       corewire.EncodeTaskID(taskID),
		SubmissionID: corewire.EncodeSubmissionID(submissionID),
		Payout:       &payoutWire,
		Tip:          &tipWire,
	}
}

type reviewedSubmissionValue struct {
	taskID       core.TaskID
	submissionID core.SubmissionID
	payout       ledger.PayoutOutcome
	tip          ledger.TipOutcome
}

func decodeReviewedSubmission(wire reviewedSubmissionWire) (reviewedSubmissionValue, error) {
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return reviewedSubmissionValue{}, err
	}
	submissionID, err := corewire.DecodeSubmissionID(wire.SubmissionID)
	if err != nil {
		return reviewedSubmissionValue{}, err
	}
	if wire.Payout == nil || wire.Tip == nil {
		return reviewedSubmissionValue{}, fmt.Errorf("reviewed submission is missing its payout or tip")
	}
	payout, err := decodePayoutOutcome(*wire.Payout)
	if err != nil {
		return reviewedSubmissionValue{}, err
	}
	tip, err := decodeTipOutcome(*wire.Tip)
	if err != nil {
		return reviewedSubmissionValue{}, err
	}
	return reviewedSubmissionValue{taskID: taskID, submissionID: submissionID, payout: payout, tip: tip}, nil
}

func encodeAcceptResult(result ledger.AcceptResult) reviewedSubmissionWire {
	switch typed := result.(type) {
	case ledger.SubmissionAccepted:
		return reviewedSubmissionSuccess("accepted", typed.TaskID, typed.SubmissionID, typed.Payout, typed.Tip)
	case ledger.AcceptRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return reviewedSubmissionWire{Variant: "failed", Error: &reason}
	default:
		return reviewedSubmissionWire{Variant: "failed", Error: rejectionError(fmt.Sprintf("unknown ledger result %T", result))}
	}
}

func decodeAcceptResult(wire reviewedSubmissionWire) (ledger.AcceptResult, error) {
	switch wire.Variant {
	case "accepted":
		value, err := decodeReviewedSubmission(wire)
		if err != nil {
			return nil, err
		}
		return ledger.SubmissionAccepted{TaskID: value.taskID, SubmissionID: value.submissionID, Payout: value.payout, Tip: value.tip}, nil
	case "failed":
		return ledger.AcceptRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown accept result variant %q", wire.Variant)
	}
}

func encodeRejectResult(result ledger.RejectResult) reviewedSubmissionWire {
	switch typed := result.(type) {
	case ledger.SubmissionRejected:
		return reviewedSubmissionSuccess("rejected", typed.TaskID, typed.SubmissionID, typed.Payout, typed.Tip)
	case ledger.RejectRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return reviewedSubmissionWire{Variant: "failed", Error: &reason}
	default:
		return reviewedSubmissionWire{Variant: "failed", Error: rejectionError(fmt.Sprintf("unknown ledger result %T", result))}
	}
}

func decodeRejectResult(wire reviewedSubmissionWire) (ledger.RejectResult, error) {
	switch wire.Variant {
	case "rejected":
		value, err := decodeReviewedSubmission(wire)
		if err != nil {
			return nil, err
		}
		return ledger.SubmissionRejected{TaskID: value.taskID, SubmissionID: value.submissionID, Payout: value.payout, Tip: value.tip}, nil
	case "failed":
		return ledger.RejectRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown reject result variant %q", wire.Variant)
	}
}

type changesRequestedWire struct {
	Variant      string                  `json:"variant"`
	TaskID       string                  `json:"task_id,omitempty"`
	SubmissionID string                  `json:"submission_id,omitempty"`
	ReviewNote   string                  `json:"review_note,omitempty"`
	Error        *domainwire.DomainError `json:"error,omitempty"`
}

func encodeRequestChangesResult(result ledger.RequestChangesResult) changesRequestedWire {
	switch typed := result.(type) {
	case ledger.ChangesRequested:
		return changesRequestedWire{
			Variant:      "requested",
			TaskID:       corewire.EncodeTaskID(typed.TaskID),
			SubmissionID: corewire.EncodeSubmissionID(typed.SubmissionID),
			ReviewNote:   typed.ReviewNote,
		}
	case ledger.RequestChangesRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return changesRequestedWire{Variant: "rejected", Error: &reason}
	default:
		return changesRequestedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown ledger result %T", result))}
	}
}

func decodeRequestChangesResult(wire changesRequestedWire) (ledger.RequestChangesResult, error) {
	switch wire.Variant {
	case "requested":
		taskID, err := corewire.DecodeTaskID(wire.TaskID)
		if err != nil {
			return nil, err
		}
		submissionID, err := corewire.DecodeSubmissionID(wire.SubmissionID)
		if err != nil {
			return nil, err
		}
		return ledger.ChangesRequested{TaskID: taskID, SubmissionID: submissionID, ReviewNote: wire.ReviewNote}, nil
	case "rejected":
		return ledger.RequestChangesRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown request changes result variant %q", wire.Variant)
	}
}

type taskAllocatedWire struct {
	Variant string                  `json:"variant"`
	Amount  int64                   `json:"amount,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeTaskAllocatedResult(result ledger.TaskAllocatedResult) taskAllocatedWire {
	switch typed := result.(type) {
	case ledger.TaskAllocatedFound:
		return taskAllocatedWire{Variant: "found", Amount: typed.Amount}
	case ledger.TaskAllocatedRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return taskAllocatedWire{Variant: "rejected", Error: &reason}
	default:
		return taskAllocatedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown ledger result %T", result))}
	}
}

func decodeTaskAllocatedResult(wire taskAllocatedWire) (ledger.TaskAllocatedResult, error) {
	switch wire.Variant {
	case "found":
		return ledger.TaskAllocatedFound{Amount: wire.Amount}, nil
	case "rejected":
		return ledger.TaskAllocatedRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown task allocated result variant %q", wire.Variant)
	}
}

type balanceValueWire struct {
	Spendable int64 `json:"spendable"`
	Allocated int64 `json:"allocated"`
}

type balanceWire struct {
	Variant string                  `json:"variant"`
	Balance *balanceValueWire       `json:"balance,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeBalanceResult(result ledger.BalanceResult) balanceWire {
	switch typed := result.(type) {
	case ledger.BalanceFound:
		value := balanceValueWire{Spendable: typed.Value.Spendable(), Allocated: typed.Value.Allocated()}
		return balanceWire{Variant: "found", Balance: &value}
	case ledger.BalanceRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return balanceWire{Variant: "rejected", Error: &reason}
	default:
		return balanceWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown ledger result %T", result))}
	}
}

func decodeBalanceResult(wire balanceWire) (ledger.BalanceResult, error) {
	switch wire.Variant {
	case "found":
		if wire.Balance == nil {
			return nil, fmt.Errorf("balance result is missing its balance")
		}
		return ledger.BalanceFound{Value: ledger.NewBalance(wire.Balance.Spendable, wire.Balance.Allocated)}, nil
	case "rejected":
		return ledger.BalanceRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown balance result variant %q", wire.Variant)
	}
}

type entriesWire struct {
	Variant string                  `json:"variant"`
	Entries []ledgerEntryWire       `json:"entries,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListEntriesResult(result ledger.ListEntriesResult) entriesWire {
	switch typed := result.(type) {
	case ledger.EntriesListed:
		return entriesWire{Variant: "listed", Entries: encodeLedgerEntries(typed.Values)}
	case ledger.ListEntriesRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return entriesWire{Variant: "rejected", Error: &reason}
	default:
		return entriesWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown ledger result %T", result))}
	}
}

func decodeListEntriesResult(wire entriesWire) (ledger.ListEntriesResult, error) {
	switch wire.Variant {
	case "listed":
		entries, err := decodeLedgerEntries(wire.Entries)
		if err != nil {
			return nil, err
		}
		return ledger.EntriesListed{Values: entries}, nil
	case "rejected":
		return ledger.ListEntriesRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list entries result variant %q", wire.Variant)
	}
}

// ---- shared helpers ----

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "ledger bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
