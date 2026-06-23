package core

import "testing"

func TestDefaultPageUsesDefaults(t *testing.T) {
	page := DefaultPage()
	if page.Limit() != 100 {
		t.Fatalf("limit = %d, want 100", page.Limit())
	}
	if page.Offset() != 0 {
		t.Fatalf("offset = %d, want 0", page.Offset())
	}
}

func TestNewPageAcceptsValidValues(t *testing.T) {
	result := NewPage(25, 50)
	accepted, matched := result.(PageAccepted)
	if !matched {
		t.Fatalf("result = %T, want PageAccepted", result)
	}
	if accepted.Value.Limit() != 25 {
		t.Fatalf("limit = %d, want 25", accepted.Value.Limit())
	}
	if accepted.Value.Offset() != 50 {
		t.Fatalf("offset = %d, want 50", accepted.Value.Offset())
	}
}

func TestNewPageClampsLimitToMaximum(t *testing.T) {
	result := NewPage(5000, 0)
	accepted, matched := result.(PageAccepted)
	if !matched {
		t.Fatalf("result = %T, want PageAccepted", result)
	}
	if accepted.Value.Limit() != 200 {
		t.Fatalf("limit = %d, want 200", accepted.Value.Limit())
	}
}

func TestNewPageRejectsNonPositiveLimit(t *testing.T) {
	result := NewPage(0, 0)
	rejected, matched := result.(PageRejected)
	if !matched {
		t.Fatalf("result = %T, want PageRejected", result)
	}
	if rejected.Reason.Code().String() != ErrorCodeInvalidArgument.String() {
		t.Fatalf("code = %q, want %q", rejected.Reason.Code().String(), ErrorCodeInvalidArgument.String())
	}
}

func TestNewPageRejectsNegativeOffset(t *testing.T) {
	result := NewPage(10, -1)
	rejected, matched := result.(PageRejected)
	if !matched {
		t.Fatalf("result = %T, want PageRejected", result)
	}
	if rejected.Reason.Code().String() != ErrorCodeInvalidArgument.String() {
		t.Fatalf("code = %q, want %q", rejected.Reason.Code().String(), ErrorCodeInvalidArgument.String())
	}
}
