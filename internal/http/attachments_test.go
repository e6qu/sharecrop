package httpserver

import "testing"

func TestAttachmentsFromRequestRejectsTooManyAttachments(t *testing.T) {
	values := make([]attachmentRequest, 0, 6)
	for index := 0; index < 6; index++ {
		values = append(values, attachmentRequest{Name: "brief.txt", ContentType: "text/plain", DataURL: "data:text/plain;base64,aGVsbG8="})
	}

	result := attachmentsFromRequest(values)
	rejected, matched := result.(attachmentsRequestRejected)
	if !matched {
		t.Fatalf("result = %T, want attachmentsRequestRejected", result)
	}
	if rejected.reason != "too many attachments" {
		t.Fatalf("reason = %q", rejected.reason)
	}
}
