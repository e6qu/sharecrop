// Package corewire holds the JSON wire codecs for the core value types that
// recur across every store bridge - typed ids, pagination, and timestamps -
// so each generated bridge serializes them identically instead of carrying its
// own copy. Store-specific types stay in that store's bridge package; this is
// only the shared vocabulary.
package corewire

import (
	"fmt"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

// EncodeUserID / DecodeUserID carry a core.UserID as its string form.
func EncodeUserID(id core.UserID) string { return id.String() }

func DecodeUserID(raw string) (core.UserID, error) {
	created, matched := core.ParseUserID(raw).(core.UserIDCreated)
	if !matched {
		return core.UserID{}, fmt.Errorf("invalid user id %q", raw)
	}
	return created.Value, nil
}

// EncodeAuditEventID / DecodeAuditEventID carry a core.AuditEventID.
func EncodeAuditEventID(id core.AuditEventID) string { return id.String() }

func DecodeAuditEventID(raw string) (core.AuditEventID, error) {
	created, matched := core.ParseAuditEventID(raw).(core.AuditEventIDCreated)
	if !matched {
		return core.AuditEventID{}, fmt.Errorf("invalid audit event id %q", raw)
	}
	return created.Value, nil
}

// EncodeNotificationID / DecodeNotificationID carry a core.NotificationID.
func EncodeNotificationID(id core.NotificationID) string { return id.String() }

func DecodeNotificationID(raw string) (core.NotificationID, error) {
	created, matched := core.ParseNotificationID(raw).(core.NotificationIDCreated)
	if !matched {
		return core.NotificationID{}, fmt.Errorf("invalid notification id %q", raw)
	}
	return created.Value, nil
}

// EncodeGuestID / DecodeGuestID carry a core.GuestID.
func EncodeGuestID(id core.GuestID) string { return id.String() }

func DecodeGuestID(raw string) (core.GuestID, error) {
	created, matched := core.ParseGuestID(raw).(core.GuestIDCreated)
	if !matched {
		return core.GuestID{}, fmt.Errorf("invalid guest id %q", raw)
	}
	return created.Value, nil
}

// EncodeRefreshTokenID / DecodeRefreshTokenID carry a core.RefreshTokenID.
func EncodeRefreshTokenID(id core.RefreshTokenID) string { return id.String() }

func DecodeRefreshTokenID(raw string) (core.RefreshTokenID, error) {
	created, matched := core.ParseRefreshTokenID(raw).(core.RefreshTokenIDCreated)
	if !matched {
		return core.RefreshTokenID{}, fmt.Errorf("invalid refresh token id %q", raw)
	}
	return created.Value, nil
}

// EncodeOrganizationID / DecodeOrganizationID carry a core.OrganizationID.
func EncodeOrganizationID(id core.OrganizationID) string { return id.String() }

func DecodeOrganizationID(raw string) (core.OrganizationID, error) {
	created, matched := core.ParseOrganizationID(raw).(core.OrganizationIDCreated)
	if !matched {
		return core.OrganizationID{}, fmt.Errorf("invalid organization id %q", raw)
	}
	return created.Value, nil
}

// EncodeString / DecodeString carry a plain string argument. DecodeString never
// errors; it exists so the generated bridge can treat a string argument with
// the same (encode, decode-with-error) shape as every other type.
func EncodeString(value string) string { return value }

func DecodeString(raw string) (string, error) { return raw, nil }

// PageWire is the wire form of core.Page (which hides its fields behind
// Limit/Offset getters).
type PageWire struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func EncodePage(page core.Page) PageWire {
	return PageWire{Limit: page.Limit(), Offset: page.Offset()}
}

func DecodePage(wire PageWire) (core.Page, error) {
	accepted, matched := core.NewPage(wire.Limit, wire.Offset).(core.PageAccepted)
	if !matched {
		return core.Page{}, fmt.Errorf("invalid page limit=%d offset=%d", wire.Limit, wire.Offset)
	}
	return accepted.Value, nil
}

// EncodeTime / DecodeTime carry a time.Time as RFC3339 with nanoseconds, in UTC.
func EncodeTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func DecodeTime(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: %w", raw, err)
	}
	return parsed.UTC(), nil
}
