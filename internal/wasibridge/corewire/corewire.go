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

// EncodeCollectibleID / DecodeCollectibleID carry a core.CollectibleID.
func EncodeCollectibleID(id core.CollectibleID) string { return id.String() }

func DecodeCollectibleID(raw string) (core.CollectibleID, error) {
	created, matched := core.ParseCollectibleID(raw).(core.CollectibleIDCreated)
	if !matched {
		return core.CollectibleID{}, fmt.Errorf("invalid collectible id %q", raw)
	}
	return created.Value, nil
}

// EncodeTaskID / DecodeTaskID carry a core.TaskID.
func EncodeTaskID(id core.TaskID) string { return id.String() }

func DecodeTaskID(raw string) (core.TaskID, error) {
	created, matched := core.ParseTaskID(raw).(core.TaskIDCreated)
	if !matched {
		return core.TaskID{}, fmt.Errorf("invalid task id %q", raw)
	}
	return created.Value, nil
}

// EncodeAgentCredentialID / DecodeAgentCredentialID carry a core.AgentCredentialID.
func EncodeAgentCredentialID(id core.AgentCredentialID) string { return id.String() }

func DecodeAgentCredentialID(raw string) (core.AgentCredentialID, error) {
	created, matched := core.ParseAgentCredentialID(raw).(core.AgentCredentialIDCreated)
	if !matched {
		return core.AgentCredentialID{}, fmt.Errorf("invalid agent credential id %q", raw)
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

// EncodeOrgCredentialID / DecodeOrgCredentialID carry a core.OrgCredentialID.
func EncodeOrgCredentialID(id core.OrgCredentialID) string { return id.String() }

func DecodeOrgCredentialID(raw string) (core.OrgCredentialID, error) {
	created, matched := core.ParseOrgCredentialID(raw).(core.OrgCredentialIDCreated)
	if !matched {
		return core.OrgCredentialID{}, fmt.Errorf("invalid org credential id %q", raw)
	}
	return created.Value, nil
}

// EncodeTaskSeriesID / DecodeTaskSeriesID carry a core.TaskSeriesID.
func EncodeTaskSeriesID(id core.TaskSeriesID) string { return id.String() }

func DecodeTaskSeriesID(raw string) (core.TaskSeriesID, error) {
	created, matched := core.ParseTaskSeriesID(raw).(core.TaskSeriesIDCreated)
	if !matched {
		return core.TaskSeriesID{}, fmt.Errorf("invalid task series id %q", raw)
	}
	return created.Value, nil
}

// EncodeTaskReservationID / DecodeTaskReservationID carry a core.TaskReservationID.
func EncodeTaskReservationID(id core.TaskReservationID) string { return id.String() }

func DecodeTaskReservationID(raw string) (core.TaskReservationID, error) {
	created, matched := core.ParseTaskReservationID(raw).(core.TaskReservationIDCreated)
	if !matched {
		return core.TaskReservationID{}, fmt.Errorf("invalid task reservation id %q", raw)
	}
	return created.Value, nil
}

// EncodeSeriesCommentID / DecodeSeriesCommentID carry a core.SeriesCommentID.
func EncodeSeriesCommentID(id core.SeriesCommentID) string { return id.String() }

func DecodeSeriesCommentID(raw string) (core.SeriesCommentID, error) {
	created, matched := core.ParseSeriesCommentID(raw).(core.SeriesCommentIDCreated)
	if !matched {
		return core.SeriesCommentID{}, fmt.Errorf("invalid series comment id %q", raw)
	}
	return created.Value, nil
}

// EncodeTaskCommentID / DecodeTaskCommentID carry a core.TaskCommentID.
func EncodeTaskCommentID(id core.TaskCommentID) string { return id.String() }

func DecodeTaskCommentID(raw string) (core.TaskCommentID, error) {
	created, matched := core.ParseTaskCommentID(raw).(core.TaskCommentIDCreated)
	if !matched {
		return core.TaskCommentID{}, fmt.Errorf("invalid task comment id %q", raw)
	}
	return created.Value, nil
}

// EncodeTeamID / DecodeTeamID carry a core.TeamID.
func EncodeTeamID(id core.TeamID) string { return id.String() }

func DecodeTeamID(raw string) (core.TeamID, error) {
	created, matched := core.ParseTeamID(raw).(core.TeamIDCreated)
	if !matched {
		return core.TeamID{}, fmt.Errorf("invalid team id %q", raw)
	}
	return created.Value, nil
}

// EncodeOrganizationMembershipID / DecodeOrganizationMembershipID carry a
// core.OrganizationMembershipID.
func EncodeOrganizationMembershipID(id core.OrganizationMembershipID) string { return id.String() }

func DecodeOrganizationMembershipID(raw string) (core.OrganizationMembershipID, error) {
	created, matched := core.ParseOrganizationMembershipID(raw).(core.OrganizationMembershipIDCreated)
	if !matched {
		return core.OrganizationMembershipID{}, fmt.Errorf("invalid organization membership id %q", raw)
	}
	return created.Value, nil
}

// EncodeLedgerEntryID / DecodeLedgerEntryID carry a core.LedgerEntryID.
func EncodeLedgerEntryID(id core.LedgerEntryID) string { return id.String() }

func DecodeLedgerEntryID(raw string) (core.LedgerEntryID, error) {
	created, matched := core.ParseLedgerEntryID(raw).(core.LedgerEntryIDCreated)
	if !matched {
		return core.LedgerEntryID{}, fmt.Errorf("invalid ledger entry id %q", raw)
	}
	return created.Value, nil
}

// EncodeCreditAccountID / DecodeCreditAccountID carry a core.CreditAccountID.
func EncodeCreditAccountID(id core.CreditAccountID) string { return id.String() }

func DecodeCreditAccountID(raw string) (core.CreditAccountID, error) {
	created, matched := core.ParseCreditAccountID(raw).(core.CreditAccountIDCreated)
	if !matched {
		return core.CreditAccountID{}, fmt.Errorf("invalid credit account id %q", raw)
	}
	return created.Value, nil
}

// EncodeSubmissionID / DecodeSubmissionID carry a core.SubmissionID.
func EncodeSubmissionID(id core.SubmissionID) string { return id.String() }

func DecodeSubmissionID(raw string) (core.SubmissionID, error) {
	created, matched := core.ParseSubmissionID(raw).(core.SubmissionIDCreated)
	if !matched {
		return core.SubmissionID{}, fmt.Errorf("invalid submission id %q", raw)
	}
	return created.Value, nil
}

// EncodeSubmissionReceiptTokenID / DecodeSubmissionReceiptTokenID carry a
// core.SubmissionReceiptTokenID.
func EncodeSubmissionReceiptTokenID(id core.SubmissionReceiptTokenID) string { return id.String() }

func DecodeSubmissionReceiptTokenID(raw string) (core.SubmissionReceiptTokenID, error) {
	created, matched := core.ParseSubmissionReceiptTokenID(raw).(core.SubmissionReceiptTokenIDCreated)
	if !matched {
		return core.SubmissionReceiptTokenID{}, fmt.Errorf("invalid submission receipt token id %q", raw)
	}
	return created.Value, nil
}

// EncodeSubmissionCommentID / DecodeSubmissionCommentID carry a
// core.SubmissionCommentID.
func EncodeSubmissionCommentID(id core.SubmissionCommentID) string { return id.String() }

func DecodeSubmissionCommentID(raw string) (core.SubmissionCommentID, error) {
	created, matched := core.ParseSubmissionCommentID(raw).(core.SubmissionCommentIDCreated)
	if !matched {
		return core.SubmissionCommentID{}, fmt.Errorf("invalid submission comment id %q", raw)
	}
	return created.Value, nil
}

// EncodeTimePtr / DecodeTimePtr carry a nullable *time.Time as a string that is
// empty when the pointer is nil.
func EncodeTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return EncodeTime(*value)
}

func DecodeTimePtr(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := DecodeTime(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
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
