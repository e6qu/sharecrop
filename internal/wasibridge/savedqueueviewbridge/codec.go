// Package savedqueueviewbridge is the WASI bridge for internal/http's
// SavedQueueViewService (a RuntimeState service, not a domain Store): hand-
// written per-type codecs (this file) plus a generated dispatcher and guest
// client (bridge_gen.go). It lets the mux running in a pooled guest reach the
// shared Postgres-backed saved-queue-view store on the host instead of a
// per-instance in-memory copy. internal/http is package httpserver.
package savedqueueviewbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- httpserver.SavedQueueView ----

type savedQueueViewWire struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Query       string `json:"query"`
	StateFilter string `json:"state_filter"`
	TypeFilter  string `json:"type_filter"`
	Sort        string `json:"sort"`
}

func encodeView(view httpserver.SavedQueueView) savedQueueViewWire {
	return savedQueueViewWire{
		ID:          view.ID,
		UserID:      corewire.EncodeUserID(view.UserID),
		Scope:       view.Scope,
		Name:        view.Name,
		Query:       view.Query,
		StateFilter: view.StateFilter,
		TypeFilter:  view.TypeFilter,
		Sort:        view.Sort,
	}
}

func decodeView(wire savedQueueViewWire) (httpserver.SavedQueueView, error) {
	userID, err := corewire.DecodeUserID(wire.UserID)
	if err != nil {
		return httpserver.SavedQueueView{}, err
	}
	return httpserver.SavedQueueView{
		ID:          wire.ID,
		UserID:      userID,
		Scope:       wire.Scope,
		Name:        wire.Name,
		Query:       wire.Query,
		StateFilter: wire.StateFilter,
		TypeFilter:  wire.TypeFilter,
		Sort:        wire.Sort,
	}, nil
}

func encodeViews(values []httpserver.SavedQueueView) []savedQueueViewWire {
	encoded := make([]savedQueueViewWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeView(values[index]))
	}
	return encoded
}

func decodeViews(wires []savedQueueViewWire) ([]httpserver.SavedQueueView, error) {
	values := make([]httpserver.SavedQueueView, 0, len(wires))
	for index := range wires {
		value, err := decodeView(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func decodeViewPayload(wire *savedQueueViewWire) (httpserver.SavedQueueView, error) {
	if wire == nil {
		return httpserver.SavedQueueView{}, fmt.Errorf("result is missing its saved queue view")
	}
	return decodeView(*wire)
}

// ---- result unions ----

type viewsResultWire struct {
	Variant string                  `json:"variant"`
	Views   []savedQueueViewWire    `json:"views,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result httpserver.SavedQueueViewsListResult) viewsResultWire {
	switch typed := result.(type) {
	case httpserver.SavedQueueViewsListed:
		return viewsResultWire{Variant: "listed", Views: encodeViews(typed.Values)}
	case httpserver.SavedQueueViewsListRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return viewsResultWire{Variant: "rejected", Error: &reason}
	default:
		return viewsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown saved queue view result %T", result))}
	}
}

func decodeListResult(wire viewsResultWire) (httpserver.SavedQueueViewsListResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeViews(wire.Views)
		if err != nil {
			return nil, err
		}
		return httpserver.SavedQueueViewsListed{Values: values}, nil
	case "rejected":
		return httpserver.SavedQueueViewsListRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown saved queue view list variant %q", wire.Variant)
	}
}

type viewResultWire struct {
	Variant string                  `json:"variant"`
	View    *savedQueueViewWire     `json:"view,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeMutationResult(result httpserver.SavedQueueViewMutationResult) viewResultWire {
	switch typed := result.(type) {
	case httpserver.SavedQueueViewSaved:
		view := encodeView(typed.Value)
		return viewResultWire{Variant: "saved", View: &view}
	case httpserver.SavedQueueViewSaveRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return viewResultWire{Variant: "rejected", Error: &reason}
	default:
		return viewResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown saved queue view result %T", result))}
	}
}

func decodeMutationResult(wire viewResultWire) (httpserver.SavedQueueViewMutationResult, error) {
	switch wire.Variant {
	case "saved":
		view, err := decodeViewPayload(wire.View)
		if err != nil {
			return nil, err
		}
		return httpserver.SavedQueueViewSaved{Value: view}, nil
	case "rejected":
		return httpserver.SavedQueueViewSaveRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown saved queue view mutation variant %q", wire.Variant)
	}
}

// ---- shared helpers ----

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "saved queue view bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
