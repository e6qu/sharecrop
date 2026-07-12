// Package auditbridge is the WASI bridge for internal/audit's Store: a generated
// dispatcher and guest client (bridge_gen.go) over hand-written per-type codecs
// (this file). The codecs carry the domain knowledge - how each audit type maps
// to JSON and back - and are covered by round-trip tests; the generated file is
// pure plumbing (which method, which codec) so it can be regenerated from the
// Store interface and diffed in CI. Shared core types (ids, page, time) are
// serialized by internal/wasibridge/corewire, not duplicated here.
//
// The split is deliberate: adding or changing a Store method must regenerate the
// plumbing (caught by check-wasi-bridge), while a type's fields are checked by
// the compiler against these codecs and by the round-trip tests. Neither side
// carries business logic.
package auditbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/wasibridge/auditwire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// audit.Event, its Subject/Action/Metadata, and the Event codec live in the
// shared internal/wasibridge/auditwire package (also used by the
// moderation-triage bridge); the result unions below carry auditwire.EventWire.

// ---- audit.ListFilters and its three filter unions ----
//
// The no-filter variant is tagged "unfiltered" rather than the obvious wildcard
// word, which the project policy check forbids in source (even in a string).

type actionFilterWire struct {
	Variant string `json:"variant"`
	Value   string `json:"value,omitempty"`
}

func encodeActionFilter(filter audit.ActionFilter) actionFilterWire {
	if equals, matched := filter.(audit.ActionEquals); matched {
		return actionFilterWire{Variant: "equals", Value: equals.Value.String()}
	}
	return actionFilterWire{Variant: "unfiltered"}
}

func decodeActionFilter(wire actionFilterWire) (audit.ActionFilter, error) {
	switch wire.Variant {
	case "equals":
		return audit.ActionEquals{Value: audit.ActionFromString(wire.Value)}, nil
	case "unfiltered":
		return audit.AnyAction{}, nil
	default:
		return nil, fmt.Errorf("unknown action filter variant %q", wire.Variant)
	}
}

// stringFilterWire serves both the subject-kind and subject-id filters, which
// have the identical unfiltered/equals-a-string shape.
type stringFilterWire struct {
	Variant string `json:"variant"`
	Value   string `json:"value,omitempty"`
}

func encodeSubjectKindFilter(filter audit.SubjectKindFilter) stringFilterWire {
	if equals, matched := filter.(audit.SubjectKindEquals); matched {
		return stringFilterWire{Variant: "equals", Value: equals.Value}
	}
	return stringFilterWire{Variant: "unfiltered"}
}

func decodeSubjectKindFilter(wire stringFilterWire) (audit.SubjectKindFilter, error) {
	switch wire.Variant {
	case "equals":
		return audit.SubjectKindEquals{Value: wire.Value}, nil
	case "unfiltered":
		return audit.AnySubjectKind{}, nil
	default:
		return nil, fmt.Errorf("unknown subject kind filter variant %q", wire.Variant)
	}
}

func encodeSubjectIDFilter(filter audit.SubjectIDFilter) stringFilterWire {
	if equals, matched := filter.(audit.SubjectIDEquals); matched {
		return stringFilterWire{Variant: "equals", Value: equals.Value}
	}
	return stringFilterWire{Variant: "unfiltered"}
}

func decodeSubjectIDFilter(wire stringFilterWire) (audit.SubjectIDFilter, error) {
	switch wire.Variant {
	case "equals":
		return audit.SubjectIDEquals{Value: wire.Value}, nil
	case "unfiltered":
		return audit.AnySubjectID{}, nil
	default:
		return nil, fmt.Errorf("unknown subject id filter variant %q", wire.Variant)
	}
}

type listFiltersWire struct {
	Action      actionFilterWire `json:"action"`
	SubjectKind stringFilterWire `json:"subject_kind"`
	SubjectID   stringFilterWire `json:"subject_id"`
}

func encodeListFilters(filters audit.ListFilters) listFiltersWire {
	return listFiltersWire{
		Action:      encodeActionFilter(filters.Action),
		SubjectKind: encodeSubjectKindFilter(filters.SubjectKind),
		SubjectID:   encodeSubjectIDFilter(filters.SubjectID),
	}
}

func decodeListFilters(wire listFiltersWire) (audit.ListFilters, error) {
	action, err := decodeActionFilter(wire.Action)
	if err != nil {
		return audit.ListFilters{}, err
	}
	subjectKind, err := decodeSubjectKindFilter(wire.SubjectKind)
	if err != nil {
		return audit.ListFilters{}, err
	}
	subjectID, err := decodeSubjectIDFilter(wire.SubjectID)
	if err != nil {
		return audit.ListFilters{}, err
	}
	return audit.ListFilters{Action: action, SubjectKind: subjectKind, SubjectID: subjectID}, nil
}

// ---- result unions ----
//
// Each result is a sealed union of a success payload and a DomainError
// rejection. A bridge/serialization failure (malformed wire) is returned as a
// decode error - distinct from a domain rejection, which is carried in the
// rejected variant.

type recordResultWire struct {
	Variant string                  `json:"variant"`
	Event   *auditwire.EventWire    `json:"event,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeRecordResult(result audit.RecordResult) recordResultWire {
	switch typed := result.(type) {
	case audit.EventRecorded:
		event := auditwire.EncodeEvent(typed.Value)
		return recordResultWire{Variant: "recorded", Event: &event}
	case audit.RecordRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return recordResultWire{Variant: "rejected", Error: &reason}
	default:
		return recordResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown audit result %T", result))}
	}
}

func decodeRecordResult(wire recordResultWire) (audit.RecordResult, error) {
	switch wire.Variant {
	case "recorded":
		if wire.Event == nil {
			return nil, fmt.Errorf("recorded record result is missing its event")
		}
		event, err := auditwire.DecodeEvent(*wire.Event)
		if err != nil {
			return nil, err
		}
		return audit.EventRecorded{Value: event}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected record result is missing its error")
		}
		return audit.RecordRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown record result variant %q", wire.Variant)
	}
}

type getResultWire struct {
	Variant string                  `json:"variant"`
	Event   *auditwire.EventWire    `json:"event,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeGetResult(result audit.GetResult) getResultWire {
	switch typed := result.(type) {
	case audit.EventFound:
		event := auditwire.EncodeEvent(typed.Value)
		return getResultWire{Variant: "found", Event: &event}
	case audit.GetRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return getResultWire{Variant: "rejected", Error: &reason}
	default:
		return getResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown audit result %T", result))}
	}
}

func decodeGetResult(wire getResultWire) (audit.GetResult, error) {
	switch wire.Variant {
	case "found":
		if wire.Event == nil {
			return nil, fmt.Errorf("found get result is missing its event")
		}
		event, err := auditwire.DecodeEvent(*wire.Event)
		if err != nil {
			return nil, err
		}
		return audit.EventFound{Value: event}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected get result is missing its error")
		}
		return audit.GetRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown get result variant %q", wire.Variant)
	}
}

type listResultWire struct {
	Variant string                  `json:"variant"`
	Events  []auditwire.EventWire   `json:"events,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result audit.ListResult) listResultWire {
	switch typed := result.(type) {
	case audit.EventsListed:
		events := make([]auditwire.EventWire, 0, len(typed.Values))
		for index := range typed.Values {
			events = append(events, auditwire.EncodeEvent(typed.Values[index]))
		}
		return listResultWire{Variant: "listed", Events: events}
	case audit.ListRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return listResultWire{Variant: "rejected", Error: &reason}
	default:
		return listResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown audit result %T", result))}
	}
}

func decodeListResult(wire listResultWire) (audit.ListResult, error) {
	switch wire.Variant {
	case "listed":
		events := make([]audit.Event, 0, len(wire.Events))
		for index := range wire.Events {
			event, err := auditwire.DecodeEvent(wire.Events[index])
			if err != nil {
				return nil, err
			}
			events = append(events, event)
		}
		return audit.EventsListed{Values: events}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected list result is missing its error")
		}
		return audit.ListRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list result variant %q", wire.Variant)
	}
}

// rejectionError encodes a defensive rejection for a result type outside its
// known union - a case the sealed interface should make unreachable, but which
// the exhaustive switches must still return something for.
func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
