// Package privacybridge is the WASI bridge for internal/http's PrivacyService (a
// RuntimeState service, not a domain Store): hand-written per-type codecs (this
// file) plus a generated dispatcher and guest client (bridge_gen.go). It lets the
// mux running in a pooled guest reach the shared Postgres-backed privacy store on
// the host instead of a per-instance in-memory copy. internal/http is package
// httpserver.
package privacybridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- submission.Submission ----
//
// RecordSensitiveFieldAccess is the only method taking a submission.Submission,
// and the store reads only the submission's ID and the Path of each sensitive
// field from it (it records one access event per sensitive field). So the wire
// carries just those, and rebuilds a minimal submission; if the store ever reads
// another field, this codec and the dual-run test must grow with it.

type submissionWire struct {
	ID              string   `json:"id"`
	SensitiveFields []string `json:"sensitive_fields,omitempty"`
}

func encodeSubmission(value submission.Submission) submissionWire {
	paths := make([]string, 0, len(value.SensitiveFields))
	for index := range value.SensitiveFields {
		paths = append(paths, value.SensitiveFields[index].Path)
	}
	return submissionWire{ID: corewire.EncodeSubmissionID(value.ID), SensitiveFields: paths}
}

func decodeSubmission(wire submissionWire) (submission.Submission, error) {
	id, err := corewire.DecodeSubmissionID(wire.ID)
	if err != nil {
		return submission.Submission{}, err
	}
	fields := make([]submission.SensitiveField, 0, len(wire.SensitiveFields))
	for index := range wire.SensitiveFields {
		fields = append(fields, submission.SensitiveField{Path: wire.SensitiveFields[index]})
	}
	return submission.Submission{ID: id, SensitiveFields: fields}, nil
}

// ---- httpserver.PrivacyRequestRecord ----

type recordWire struct {
	ID                 string `json:"id"`
	RequestedBy        string `json:"requested_by"`
	Kind               string `json:"kind"`
	State              string `json:"state"`
	ExportJSON         string `json:"export_json"`
	ResolutionNote     string `json:"resolution_note"`
	CreatedAt          string `json:"created_at"`
	ResolvedAt         string `json:"resolved_at"`
	RedactedFieldCount int    `json:"redacted_field_count"`
}

func encodeRecord(record httpserver.PrivacyRequestRecord) recordWire {
	return recordWire{
		ID:                 record.ID,
		RequestedBy:        corewire.EncodeUserID(record.RequestedBy),
		Kind:               record.Kind,
		State:              record.State,
		ExportJSON:         record.ExportJSON,
		ResolutionNote:     record.ResolutionNote,
		CreatedAt:          corewire.EncodeTime(record.CreatedAt),
		ResolvedAt:         corewire.EncodeTime(record.ResolvedAt),
		RedactedFieldCount: record.RedactedFieldCount,
	}
}

func decodeRecord(wire recordWire) (httpserver.PrivacyRequestRecord, error) {
	requestedBy, err := corewire.DecodeUserID(wire.RequestedBy)
	if err != nil {
		return httpserver.PrivacyRequestRecord{}, err
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return httpserver.PrivacyRequestRecord{}, err
	}
	resolvedAt, err := corewire.DecodeTime(wire.ResolvedAt)
	if err != nil {
		return httpserver.PrivacyRequestRecord{}, err
	}
	return httpserver.PrivacyRequestRecord{
		ID:                 wire.ID,
		RequestedBy:        requestedBy,
		Kind:               wire.Kind,
		State:              wire.State,
		ExportJSON:         wire.ExportJSON,
		ResolutionNote:     wire.ResolutionNote,
		CreatedAt:          createdAt,
		ResolvedAt:         resolvedAt,
		RedactedFieldCount: wire.RedactedFieldCount,
	}, nil
}

func encodeRecords(values []httpserver.PrivacyRequestRecord) []recordWire {
	encoded := make([]recordWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeRecord(values[index]))
	}
	return encoded
}

func decodeRecords(wires []recordWire) ([]httpserver.PrivacyRequestRecord, error) {
	values := make([]httpserver.PrivacyRequestRecord, 0, len(wires))
	for index := range wires {
		value, err := decodeRecord(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func decodeRecordPayload(wire *recordWire) (httpserver.PrivacyRequestRecord, error) {
	if wire == nil {
		return httpserver.PrivacyRequestRecord{}, fmt.Errorf("result is missing its privacy request record")
	}
	return decodeRecord(*wire)
}

// ---- result unions ----

type recordResultWire struct {
	Variant string                  `json:"variant"`
	Record  *recordWire             `json:"record,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeMutationResult(result httpserver.PrivacyMutationResult) recordResultWire {
	switch typed := result.(type) {
	case httpserver.PrivacyRequestSaved:
		record := encodeRecord(typed.Value)
		return recordResultWire{Variant: "saved", Record: &record}
	case httpserver.PrivacyRequestMutationRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return recordResultWire{Variant: "rejected", Error: &reason}
	default:
		return recordResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown privacy result %T", result))}
	}
}

func decodeMutationResult(wire recordResultWire) (httpserver.PrivacyMutationResult, error) {
	switch wire.Variant {
	case "saved":
		record, err := decodeRecordPayload(wire.Record)
		if err != nil {
			return nil, err
		}
		return httpserver.PrivacyRequestSaved{Value: record}, nil
	case "rejected":
		return httpserver.PrivacyRequestMutationRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown privacy mutation variant %q", wire.Variant)
	}
}

type recordsResultWire struct {
	Variant string                  `json:"variant"`
	Records []recordWire            `json:"records,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result httpserver.PrivacyListResult) recordsResultWire {
	switch typed := result.(type) {
	case httpserver.PrivacyRequestsListed:
		return recordsResultWire{Variant: "listed", Records: encodeRecords(typed.Values)}
	case httpserver.PrivacyRequestListRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return recordsResultWire{Variant: "rejected", Error: &reason}
	default:
		return recordsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown privacy result %T", result))}
	}
}

func decodeListResult(wire recordsResultWire) (httpserver.PrivacyListResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeRecords(wire.Records)
		if err != nil {
			return nil, err
		}
		return httpserver.PrivacyRequestsListed{Values: values}, nil
	case "rejected":
		return httpserver.PrivacyRequestListRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown privacy list variant %q", wire.Variant)
	}
}

type retentionResultWire struct {
	Variant            string                  `json:"variant"`
	RedactedFieldCount int                     `json:"redacted_field_count,omitempty"`
	Error              *domainwire.DomainError `json:"error,omitempty"`
}

func encodeRetentionResult(result httpserver.PrivacyRetentionResult) retentionResultWire {
	switch typed := result.(type) {
	case httpserver.PrivacyRetentionRun:
		return retentionResultWire{Variant: "run", RedactedFieldCount: typed.RedactedFieldCount}
	case httpserver.PrivacyRetentionRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return retentionResultWire{Variant: "rejected", Error: &reason}
	default:
		return retentionResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown privacy result %T", result))}
	}
}

func decodeRetentionResult(wire retentionResultWire) (httpserver.PrivacyRetentionResult, error) {
	switch wire.Variant {
	case "run":
		return httpserver.PrivacyRetentionRun{RedactedFieldCount: wire.RedactedFieldCount}, nil
	case "rejected":
		return httpserver.PrivacyRetentionRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown privacy retention variant %q", wire.Variant)
	}
}

// ---- shared helpers ----

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "privacy bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
