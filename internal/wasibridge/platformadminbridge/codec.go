// Package platformadminbridge is the WASI bridge for internal/http's
// PlatformAdminService (a RuntimeState service, not a domain Store): hand-written
// per-type codecs (this file) plus a generated dispatcher and guest client
// (bridge_gen.go). It lets the mux running in a pooled guest reach the shared
// Postgres-backed platform-admin store on the host instead of a per-instance
// in-memory copy. internal/http is package httpserver.
package platformadminbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- httpserver.PlatformAdminRecord ----

type recordWire struct {
	UserID    string `json:"user_id"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

func encodeRecord(record httpserver.PlatformAdminRecord) recordWire {
	return recordWire{
		UserID:    corewire.EncodeUserID(record.UserID),
		Source:    record.Source,
		CreatedAt: corewire.EncodeTime(record.CreatedAt),
	}
}

func decodeRecord(wire recordWire) (httpserver.PlatformAdminRecord, error) {
	userID, err := corewire.DecodeUserID(wire.UserID)
	if err != nil {
		return httpserver.PlatformAdminRecord{}, err
	}
	createdAt, err := corewire.DecodeTime(wire.CreatedAt)
	if err != nil {
		return httpserver.PlatformAdminRecord{}, err
	}
	return httpserver.PlatformAdminRecord{UserID: userID, Source: wire.Source, CreatedAt: createdAt}, nil
}

func encodeRecords(values []httpserver.PlatformAdminRecord) []recordWire {
	encoded := make([]recordWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeRecord(values[index]))
	}
	return encoded
}

func decodeRecords(wires []recordWire) ([]httpserver.PlatformAdminRecord, error) {
	values := make([]httpserver.PlatformAdminRecord, 0, len(wires))
	for index := range wires {
		value, err := decodeRecord(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func decodeRecordPayload(wire *recordWire) (httpserver.PlatformAdminRecord, error) {
	if wire == nil {
		return httpserver.PlatformAdminRecord{}, fmt.Errorf("result is missing its platform admin record")
	}
	return decodeRecord(*wire)
}

// ---- result unions ----

type checkResultWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCheckResult(result httpserver.PlatformAdminCheckResult) checkResultWire {
	denied, matched := result.(httpserver.PlatformAdminDenied)
	if !matched {
		return checkResultWire{Variant: "allowed"}
	}
	reason := domainwire.EncodeDomainError(denied.Reason)
	return checkResultWire{Variant: "denied", Error: &reason}
}

func decodeCheckResult(wire checkResultWire) (httpserver.PlatformAdminCheckResult, error) {
	if wire.Variant == "denied" {
		return httpserver.PlatformAdminDenied{Reason: decodeReason(wire.Error)}, nil
	}
	return httpserver.PlatformAdminAllowed{}, nil
}

type recordsResultWire struct {
	Variant string                  `json:"variant"`
	Records []recordWire            `json:"records,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result httpserver.PlatformAdminListResult) recordsResultWire {
	switch typed := result.(type) {
	case httpserver.PlatformAdminsListed:
		return recordsResultWire{Variant: "listed", Records: encodeRecords(typed.Values)}
	case httpserver.PlatformAdminListRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return recordsResultWire{Variant: "rejected", Error: &reason}
	default:
		return recordsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown platform admin result %T", result))}
	}
}

func decodeListResult(wire recordsResultWire) (httpserver.PlatformAdminListResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeRecords(wire.Records)
		if err != nil {
			return nil, err
		}
		return httpserver.PlatformAdminsListed{Values: values}, nil
	case "rejected":
		return httpserver.PlatformAdminListRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown platform admin list variant %q", wire.Variant)
	}
}

type recordResultWire struct {
	Variant string                  `json:"variant"`
	Record  *recordWire             `json:"record,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeMutationResult(result httpserver.PlatformAdminMutationResult) recordResultWire {
	switch typed := result.(type) {
	case httpserver.PlatformAdminSaved:
		record := encodeRecord(typed.Value)
		return recordResultWire{Variant: "saved", Record: &record}
	case httpserver.PlatformAdminMutationRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return recordResultWire{Variant: "rejected", Error: &reason}
	default:
		return recordResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown platform admin result %T", result))}
	}
}

func decodeMutationResult(wire recordResultWire) (httpserver.PlatformAdminMutationResult, error) {
	switch wire.Variant {
	case "saved":
		record, err := decodeRecordPayload(wire.Record)
		if err != nil {
			return nil, err
		}
		return httpserver.PlatformAdminSaved{Value: record}, nil
	case "rejected":
		return httpserver.PlatformAdminMutationRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown platform admin mutation variant %q", wire.Variant)
	}
}

// ---- shared helpers ----

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "platform admin bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
