package corewire

import (
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

func TestUserIDRoundTrip(t *testing.T) {
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	restored, err := DecodeUserID(EncodeUserID(created.Value))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored != created.Value {
		t.Errorf("user id did not round-trip")
	}
	if _, err := DecodeUserID("not-an-id"); err == nil {
		t.Errorf("DecodeUserID accepted a bad id")
	}
}

func TestAuditEventIDRoundTrip(t *testing.T) {
	created, matched := core.NewAuditEventID().(core.AuditEventIDCreated)
	if !matched {
		t.Fatalf("audit event id rejected")
	}
	restored, err := DecodeAuditEventID(EncodeAuditEventID(created.Value))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored != created.Value {
		t.Errorf("audit event id did not round-trip")
	}
	if _, err := DecodeAuditEventID("not-an-id"); err == nil {
		t.Errorf("DecodeAuditEventID accepted a bad id")
	}
}

func TestNotificationIDRoundTrip(t *testing.T) {
	created, matched := core.NewNotificationID().(core.NotificationIDCreated)
	if !matched {
		t.Fatalf("notification id rejected")
	}
	restored, err := DecodeNotificationID(EncodeNotificationID(created.Value))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored != created.Value {
		t.Errorf("notification id did not round-trip")
	}
	if _, err := DecodeNotificationID("not-an-id"); err == nil {
		t.Errorf("DecodeNotificationID accepted a bad id")
	}
}

func TestPageRoundTrip(t *testing.T) {
	accepted, matched := core.NewPage(20, 40).(core.PageAccepted)
	if !matched {
		t.Fatalf("page rejected")
	}
	restored, err := DecodePage(EncodePage(accepted.Value))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored.Limit() != 20 || restored.Offset() != 40 {
		t.Errorf("page: got limit=%d offset=%d, want 20/40", restored.Limit(), restored.Offset())
	}
}

func TestTimeRoundTrip(t *testing.T) {
	original := time.Date(2026, 7, 11, 12, 30, 0, 123456000, time.UTC)
	restored, err := DecodeTime(EncodeTime(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !restored.Equal(original) {
		t.Errorf("time: got %s, want %s", restored, original)
	}
	if _, err := DecodeTime("not-a-time"); err == nil {
		t.Errorf("DecodeTime accepted a bad timestamp")
	}
}
