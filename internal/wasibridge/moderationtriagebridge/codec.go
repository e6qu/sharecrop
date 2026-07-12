// Package moderationtriagebridge is the WASI bridge for internal/http's
// ModerationTriageService (a RuntimeState service, not a domain Store): hand-
// written per-type codecs (this file) plus a generated dispatcher and guest
// client (bridge_gen.go). It lets the mux running in a pooled guest reach the
// shared Postgres-backed moderation-triage store on the host instead of a
// per-instance in-memory copy. internal/http is package httpserver.
package moderationtriagebridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- audit.Event ----
//
// RecordOpen is the only method taking an audit.Event, and both the in-memory
// and db moderation-triage stores read only the event's ID and CreatedAt from it
// (the moderation report is keyed by the audit event that opened it). So the wire
// carries just those two fields and rebuilds a minimal event; if RecordOpen ever
// reads another field, this codec and the dual-run test must grow with it.

type eventWire struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

func encodeEvent(event audit.Event) eventWire {
	return eventWire{
		ID:        corewire.EncodeAuditEventID(event.ID),
		CreatedAt: corewire.EncodeTime(event.CreatedAt),
	}
}

func decodeEvent(wire eventWire) (audit.Event, error) {
	id, err := corewire.DecodeAuditEventID(wire.ID)
	if err != nil {
		return audit.Event{}, err
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return audit.Event{}, err
	}
	return audit.Event{ID: id, CreatedAt: createdAt}, nil
}

// ---- []core.AuditEventID ----

func encodeAuditEventIDs(ids []core.AuditEventID) []string {
	encoded := make([]string, 0, len(ids))
	for index := range ids {
		encoded = append(encoded, corewire.EncodeAuditEventID(ids[index]))
	}
	return encoded
}

func decodeAuditEventIDs(raw []string) ([]core.AuditEventID, error) {
	ids := make([]core.AuditEventID, 0, len(raw))
	for index := range raw {
		id, err := corewire.DecodeAuditEventID(raw[index])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ---- httpserver.ModerationTriageRecord ----

type recordWire struct {
	ReportID       string `json:"report_id"`
	State          string `json:"state"`
	ResolutionNote string `json:"resolution_note"`
	UpdatedBy      string `json:"updated_by"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func encodeRecord(record httpserver.ModerationTriageRecord) recordWire {
	return recordWire{
		ReportID:       corewire.EncodeAuditEventID(record.ReportID),
		State:          record.State,
		ResolutionNote: record.ResolutionNote,
		UpdatedBy:      record.UpdatedBy,
		CreatedAt:      corewire.EncodeTime(record.CreatedAt),
		UpdatedAt:      corewire.EncodeTime(record.UpdatedAt),
	}
}

func decodeRecord(wire recordWire) (httpserver.ModerationTriageRecord, error) {
	reportID, err := corewire.DecodeAuditEventID(wire.ReportID)
	if err != nil {
		return httpserver.ModerationTriageRecord{}, err
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return httpserver.ModerationTriageRecord{}, err
	}
	updatedAt, err := corewire.DecodeTime(wire.UpdatedAt)
	if err != nil {
		return httpserver.ModerationTriageRecord{}, err
	}
	return httpserver.ModerationTriageRecord{
		ReportID:       reportID,
		State:          wire.State,
		ResolutionNote: wire.ResolutionNote,
		UpdatedBy:      wire.UpdatedBy,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}, nil
}

func encodeRecords(values []httpserver.ModerationTriageRecord) []recordWire {
	encoded := make([]recordWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeRecord(values[index]))
	}
	return encoded
}

func decodeRecords(wires []recordWire) ([]httpserver.ModerationTriageRecord, error) {
	values := make([]httpserver.ModerationTriageRecord, 0, len(wires))
	for index := range wires {
		value, err := decodeRecord(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func decodeRecordPayload(wire *recordWire) (httpserver.ModerationTriageRecord, error) {
	if wire == nil {
		return httpserver.ModerationTriageRecord{}, fmt.Errorf("result is missing its moderation record")
	}
	return decodeRecord(*wire)
}

// ---- result unions ----

type recordResultWire struct {
	Variant string                  `json:"variant"`
	Record  *recordWire             `json:"record,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeMutationResult(result httpserver.ModerationTriageMutationResult) recordResultWire {
	switch typed := result.(type) {
	case httpserver.ModerationTriageSaved:
		record := encodeRecord(typed.Value)
		return recordResultWire{Variant: "saved", Record: &record}
	case httpserver.ModerationTriageMutationRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return recordResultWire{Variant: "rejected", Error: &reason}
	default:
		return recordResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown moderation result %T", result))}
	}
}

func decodeMutationResult(wire recordResultWire) (httpserver.ModerationTriageMutationResult, error) {
	switch wire.Variant {
	case "saved":
		record, err := decodeRecordPayload(wire.Record)
		if err != nil {
			return nil, err
		}
		return httpserver.ModerationTriageSaved{Value: record}, nil
	case "rejected":
		return httpserver.ModerationTriageMutationRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown moderation mutation variant %q", wire.Variant)
	}
}

type recordsResultWire struct {
	Variant string                  `json:"variant"`
	Records []recordWire            `json:"records,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result httpserver.ModerationTriageListResult) recordsResultWire {
	switch typed := result.(type) {
	case httpserver.ModerationTriageListed:
		return recordsResultWire{Variant: "listed", Records: encodeRecords(typed.Values)}
	case httpserver.ModerationTriageListRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return recordsResultWire{Variant: "rejected", Error: &reason}
	default:
		return recordsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown moderation result %T", result))}
	}
}

func decodeListResult(wire recordsResultWire) (httpserver.ModerationTriageListResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeRecords(wire.Records)
		if err != nil {
			return nil, err
		}
		return httpserver.ModerationTriageListed{Values: values}, nil
	case "rejected":
		return httpserver.ModerationTriageListRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown moderation list variant %q", wire.Variant)
	}
}

// ---- shared helpers ----

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "moderation triage bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
