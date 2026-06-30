package attachment

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestAttachmentAcceptsSmallDataURL(t *testing.T) {
	data := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png"))
	result := NewAttachment("receipt.png", "image/png", data)
	accepted, matched := result.(AttachmentAccepted)
	if !matched {
		t.Fatalf("result = %T, want AttachmentAccepted", result)
	}
	if accepted.Value.Name.String() != "receipt.png" {
		t.Fatalf("name = %q", accepted.Value.Name.String())
	}
	if accepted.Value.Content.SizeBytes() != 3 {
		t.Fatalf("size = %d, want 3", accepted.Value.Content.SizeBytes())
	}
	if accepted.Value.DataURL() != data {
		t.Fatalf("data URL round trip failed")
	}
}

func TestAttachmentRejectsTooLargeContent(t *testing.T) {
	data := "data:text/plain;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", MaxBytes+1)))
	result := NewAttachment("large.txt", "text/plain", data)
	rejected, matched := result.(AttachmentRejected)
	if !matched {
		t.Fatalf("result = %T, want AttachmentRejected", result)
	}
	if rejected.Reason.Description() != "attachment content is too large" {
		t.Fatalf("reason = %q", rejected.Reason.Description())
	}
}

func TestAttachmentRejectsUnsupportedContentType(t *testing.T) {
	data := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte("<p>x</p>"))
	result := NewAttachment("x.html", "text/html", data)
	rejected, matched := result.(AttachmentRejected)
	if !matched {
		t.Fatalf("result = %T, want AttachmentRejected", result)
	}
	if rejected.Reason.Description() != "attachment content type is not allowed" {
		t.Fatalf("reason = %q", rejected.Reason.Description())
	}
}

func TestAttachmentRejectsContentTypeMismatch(t *testing.T) {
	data := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png"))
	result := NewAttachment("receipt.jpg", "image/jpeg", data)
	rejected, matched := result.(AttachmentRejected)
	if !matched {
		t.Fatalf("result = %T, want AttachmentRejected", result)
	}
	if rejected.Reason.Description() != "attachment data URL is invalid" {
		t.Fatalf("reason = %q", rejected.Reason.Description())
	}
}
